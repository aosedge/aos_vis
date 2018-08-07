package dataprovider

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"gitpct.epam.com/epmd-aepr/aos_vis/config"
	"gitpct.epam.com/epmd-aepr/aos_vis/dataadapter"
)

/*******************************************************************************
 * Types
 ******************************************************************************/

//SubscriptionOutputData struct to inform aboute data change by subscription
type SubscriptionOutputData struct {
	ID      string
	OutData interface{}
}

type visInternalData struct {
	data          interface{}
	id            int32
	isInitialized bool
}

type subscriptionElement struct {
	subsChan chan<- SubscriptionOutputData
	ids      []subscriptionPare
}

type subscriptionPare struct {
	subscriptionID uint64
	value          *regexp.Regexp
}

// DataProvider interface for geeting vehicle data
type DataProvider struct {
	sensorDataChannel chan []dataadapter.VisData
	subscription      struct {
		ar    []subscriptionElement
		mutex sync.Mutex
	}
	visDataStorage map[string]visInternalData
	currentSubsID  uint64

	adapterMap map[string]dataadapter.DataAdapter
}

type notificationData struct {
	subsChan chan<- SubscriptionOutputData
	id       uint64
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// New returns pointer to DataProvider
func New(config *config.Config) (provider *DataProvider, err error) {
	log.Debug("Create data provider")

	provider = &DataProvider{}
	provider.sensorDataChannel = make(chan []dataadapter.VisData, 100)
	provider.visDataStorage = createVisDataStorage()
	go provider.start()

	provider.adapterMap = make(map[string]dataadapter.DataAdapter)

	adapter, _ := dataadapter.NewTestAdapter()

	pathes, err := adapter.GetPathList()
	if err != nil {
		return nil, err
	}

	for _, path := range pathes {
		if _, ok := provider.adapterMap[path]; ok {
			log.WithField("path", path).Warningf("Path already in adapter map")
		} else {
			log.WithFields(log.Fields{"path": path, "adaptor": adapter.GetName()}).Debug("Add path")

			provider.adapterMap[path] = adapter
		}
	}

	return provider, nil
}

// IsPathPublic if path is public no authentication is required
func (provider *DataProvider) IsPathPublic(path string) (result bool, err error) {
	if _, ok := provider.adapterMap[path]; !ok {
		return result, fmt.Errorf("Path %s doesn't exits", path)
	}

	result, err = provider.adapterMap[path].IsPathPublic(path)

	log.WithFields(log.Fields{"path": path, "result": result}).Debug("Is path public")

	return result, err
}

// GetData returns VIS data
func (provider *DataProvider) GetData(path string) (data interface{}, err error) {
	log.WithField("path", path).Debug("Get data")

	filter, err := createRegexpFromPath(path)
	if err != nil {
		return data, err
	}

	// Create map of pathes grouped by adapter
	adapterDataMap := make(map[dataadapter.DataAdapter][]string)

	for path, adapter := range provider.adapterMap {
		if filter.MatchString(path) {
			if adapterDataMap[adapter] == nil {
				adapterDataMap[adapter] = make([]string, 0, 10)
			}

			if adapterDataMap[adapter] == nil {
				adapterDataMap[adapter] = make([]string, 0, 10)
			}
			adapterDataMap[adapter] = append(adapterDataMap[adapter], path)
		}
	}

	// Create common data array
	commonData := make(map[string]interface{})

	for adapter, pathList := range adapterDataMap {
		result, err := adapter.GetData(pathList)
		if err != nil {
			return data, err
		}
		for path, value := range result {
			log.WithFields(log.Fields{"adapter": adapter.GetName(), "value": value, "path": path}).Debug("Data from adapter")

			commonData[path] = value
		}
	}

	if len(commonData) == 0 {
		return data, errors.New("The specified data path does not exist")
	}

	return convertData(commonData), nil
}

// SetData sets VIS data
func (provider *DataProvider) SetData(path string, data interface{}) (err error) {
	log.WithFields(log.Fields{"path": path, "data": data}).Debug("Set data")

	filter, err := createRegexpFromPath(path)
	if err != nil {
		return err
	}

	// Create map from data. According to VIS spec data could be array of map,
	// map or simple value. Convert array of map to map and keep map as is.
	suffixMap := make(map[string]interface{})

	switch data.(type) {
	// convert array of map to map
	case []map[string]interface{}:
		for _, arrayItem := range data.([]map[string]interface{}) {
			for path, value := range arrayItem {
				suffixMap[path] = value
			}
		}

	// keep map as is
	case map[string]interface{}:
		suffixMap = data.(map[string]interface{})
	}

	// adapterDataMap contains VIS data grouped by adapters
	adapterDataMap := make(map[dataadapter.DataAdapter]map[string]interface{})

	for path, adapter := range provider.adapterMap {
		if filter.MatchString(path) {
			var value interface{}
			if len(suffixMap) != 0 {
				// if there is suffix map, try to find proper path by suffix
				for suffix, v := range suffixMap {
					if strings.HasSuffix(path, suffix) {
						value = v
						break
					}
				}
			} else {
				// For simple value set data
				value = data
			}

			if value != nil {
				// Set data to adapterDataMap
				log.WithFields(log.Fields{"adapter": adapter.GetName(), "value": value, "path": path}).Debug("Set data to adapter")

				if adapterDataMap[adapter] == nil {
					adapterDataMap[adapter] = make(map[string]interface{})
				}
				adapterDataMap[adapter][path] = value
			}
		}
	}

	// If adapterMap is empty: no path found
	if len(adapterDataMap) == 0 {
		return errors.New("The server is unable to fulfil the client request because the request is malformed")
	}

	// Everything ok: try to set to adapter
	for adapter, visData := range adapterDataMap {
		if err = adapter.SetData(visData); err != nil {
			return err
		}
	}

	return nil
}

// Subscribe subscribes for data change
func (provider *DataProvider) Subscribe(subsChan chan<- SubscriptionOutputData, path string) (id string, err error) {
	//TODO: add checking available path
	var subsElement subscriptionPare

	subsElement.value, err = createRegexpFromPath(path)
	if err != nil {
		log.Error("incorrect path ", err)
		return "", errors.New("404 Not found")
	}

	provider.subscription.mutex.Lock()
	defer provider.subscription.mutex.Unlock()

	provider.currentSubsID++

	subsElement.subscriptionID = provider.currentSubsID
	var wasFound bool
	for i := range provider.subscription.ar {
		if provider.subscription.ar[i].subsChan == subsChan {
			wasFound = true
			provider.subscription.ar[i].ids = append(provider.subscription.ar[i].ids, subsElement)
			log.Debug("Add subscription to available channel ID", provider.currentSubsID, " path ", path)
		}
	}

	if wasFound == false {
		var subscripRootElement subscriptionElement
		subscripRootElement.subsChan = subsChan
		subscripRootElement.ids = append(subscripRootElement.ids, subsElement)
		provider.subscription.ar = append(provider.subscription.ar, subscripRootElement)
		log.Debug("Create new subscription ID", provider.currentSubsID, " path ", path)
	}
	return strconv.FormatUint(provider.currentSubsID, 10), nil
}

// Unsubscribe unsubscribes from data change
func (provider *DataProvider) Unsubscribe(subsChan chan<- SubscriptionOutputData, subsID string) (err error) {
	var intSubs uint64
	intSubs, err = strconv.ParseUint(subsID, 10, 64)
	if err != nil {
		log.Error("Error cant convert subsID to int64")
		return err
	}
	provider.subscription.mutex.Lock()
	defer provider.subscription.mutex.Unlock()

	for i := range provider.subscription.ar {
		if provider.subscription.ar[i].subsChan == subsChan {
			wasFound := false
			for j, sID := range provider.subscription.ar[i].ids {
				if intSubs == sID.subscriptionID {
					provider.subscription.ar[i].ids = append(provider.subscription.ar[i].ids[:j], provider.subscription.ar[i].ids[j+1:]...)
					wasFound = true
					break
				}
			}
			if wasFound {
				if len(provider.subscription.ar[i].ids) == 0 {
					provider.subscription.ar = append(provider.subscription.ar[:i], provider.subscription.ar[i+1:]...)
					log.Debug("Remove channel subscription")
				}
				return nil
			}
			break
		}
	}

	return errors.New("404 Not found")
}

// UnsubscribeAll unsubscribes from all data changes
func (provider *DataProvider) UnsubscribeAll(subsChan chan<- SubscriptionOutputData) (err error) {
	provider.subscription.mutex.Lock()
	defer provider.subscription.mutex.Unlock()

	for i := range provider.subscription.ar {
		if provider.subscription.ar[i].subsChan == subsChan {
			provider.subscription.ar = append(provider.subscription.ar[:i], provider.subscription.ar[i+1:]...)
			return nil
		}
	}
	//return errors.New("404 Not found")
	return nil
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (provider *DataProvider) start() {
	for {
		incomingData := <-provider.sensorDataChannel
		provider.processIncomingData(incomingData[:])
	}
}

func (provider *DataProvider) processIncomingData(incomeData []dataadapter.VisData) {

	type notificationPair struct {
		id   uint64
		data map[string]interface{}
	}

	type notificationElement struct {
		subsChan         chan<- SubscriptionOutputData
		notificationData []notificationPair
	}

	provider.subscription.mutex.Lock()
	defer provider.subscription.mutex.Unlock()

	var notificationArray []notificationElement
	for _, data := range incomeData {
		//find element which receive in dataStorage

		//		log.Debug("process path = ", data.Path, " data= ", data.Data)
		wasChanged := false

		if data.Data == nil {
			continue
		}

		item := provider.visDataStorage[data.Path]

		item.isInitialized = true
		item.data = data.Data

		valueType := reflect.TypeOf(data.Data).Kind()
		//		log.Debug("Value type ", valueType)

		if valueType == reflect.Array || valueType == reflect.Slice {
			//TODO: add array comeration
			wasChanged = true
			provider.visDataStorage[data.Path] = item
		} else {
			if provider.visDataStorage[data.Path].data != data.Data {
				log.Debug("Data for ", data.Path, " was changed from ", provider.visDataStorage[data.Path].data, " to ", data.Data)
				provider.visDataStorage[data.Path] = item
				wasChanged = true
			}
		}

		//		log.Debug("wasChanged = ", wasChanged)
		if wasChanged == true {
			log.Debug("process path = ", data.Path, " data= ", data.Data)
			// prepare data fro change notification
			notifyData := provider.getNotificationElementsByPath(data.Path)
			for _, notifyElement := range notifyData {
				//go across all channels
				wasChFound := false
				for j := range notificationArray {
					if notificationArray[j].subsChan == notifyElement.subsChan {
						wasIDFound := false
						///go across all Id
						for k := range notificationArray[j].notificationData {
							if notificationArray[j].notificationData[k].id == notifyElement.id {
								wasIDFound = true
								notificationArray[j].notificationData[k].data[data.Path] = data.Data
								log.Debug("Add notification to abaliable ID ")
								break
							}
						}
						if wasIDFound == false {
							//Add noe ID to available channel
							log.Debug("Create new notification element ID for available channel ")
							pair := notificationPair{id: notifyElement.id, data: make(map[string]interface{})}
							pair.data[data.Path] = data.Data
							notificationArray[j].notificationData = append(notificationArray[j].notificationData, pair)
						}
						wasChFound = true
					}
				}
				if wasChFound == false {
					//add new channel
					log.Debug("Create new channel")
					pair := notificationPair{id: notifyElement.id, data: make(map[string]interface{})}
					pair.data[data.Path] = data.Data
					pairStore := []notificationPair{pair}

					notificationArray = append(notificationArray, notificationElement{subsChan: notifyElement.subsChan, notificationData: pairStore})

				}
			}
		}
	}

	for i := range notificationArray {
		for j := range notificationArray[i].notificationData {
			dataTosend := &(notificationArray[i].notificationData[j])
			sendElement := SubscriptionOutputData{ID: strconv.FormatUint(dataTosend.id, 10), OutData: dataTosend.data}
			log.Debug("Send id ", sendElement.ID, " Data: ", sendElement.OutData)
			notificationArray[i].subsChan <- sendElement
		}
	}
}
func (provider *DataProvider) getNotificationElementsByPath(path string) (returnData []notificationData) {
	log.Debug("getNotificationElementsByPath path=", path)
	wasFound := false
	for i := range provider.subscription.ar {
		for j := range provider.subscription.ar[i].ids {
			if provider.subscription.ar[i].ids[j].value.MatchString(path) {
				log.Debug("Find subscription element ID ", provider.subscription.ar[i].ids[j].subscriptionID, " path= ", path)
				returnData = append(returnData, notificationData{
					subsChan: provider.subscription.ar[i].subsChan,
					id:       provider.subscription.ar[i].ids[j].subscriptionID})
				wasFound = true
				break
			}
		}
	}
	if wasFound == false {
		log.Debug("No subscription for ", path)
	}

	return returnData
}

func createVisDataStorage() map[string]visInternalData {
	var storage map[string]visInternalData

	storage = make(map[string]visInternalData)

	storage["Attribute.Vehicle.UserIdentification.Users"] = visInternalData{id: 8888, data: []string{"User1"}, isInitialized: true}
	storage["Attribute.Vehicle.VehicleIdentification.VIN"] = visInternalData{id: 39, data: "1234567890QWERTYU", isInitialized: true}

	//TODO: addede temporary for test
	storage["Signal.Drivetrain.InternalCombustionEngine.RPM"] = visInternalData{id: 58, data: 2372, isInitialized: true}
	storage["Signal.Drivetrain.InternalCombustionEngine.Power"] = visInternalData{id: 65, data: 60, isInitialized: true}
	storage["Signal.Test.RPM"] = visInternalData{id: 9000, data: 60, isInitialized: true}

	storage["Attribute.Emulator.rectangle_long0"] = visInternalData{id: 9001, isInitialized: false}
	storage["Attribute.Emulator.rectangle_lat0"] = visInternalData{id: 9002, isInitialized: false}
	storage["Attribute.Emulator.rectangle_long1"] = visInternalData{id: 9003, isInitialized: false}
	storage["Attribute.Emulator.rectangle_lat1"] = visInternalData{id: 9004, isInitialized: false}
	storage["Attribute.Emulator.to_rectangle"] = visInternalData{id: 9005, isInitialized: false}

	return storage
}

func getParentPath(path string) (parent string) {
	return path[:strings.LastIndex(path, ".")]
}

func createRegexpFromPath(path string) (exp *regexp.Regexp, err error) {
	regexpStr := strings.Replace(path, ".", "[.]", -1)
	regexpStr = strings.Replace(regexpStr, "*", ".*?", -1)
	regexpStr = "^" + regexpStr
	exp, err = regexp.Compile(regexpStr)

	return exp, err
}

func isArraysEqual(arr1, arr2 []interface{}) (result bool) {
	if arr1 == nil && arr2 == nil {
		return true
	}

	if arr1 == nil || arr2 == nil {
		return false
	}

	if len(arr1) != len(arr2) {
		return false
	}

	for i := range arr1 {
		if arr1[i] != arr2[i] {
			return false
		}
	}
	return true
}

func convertData(data map[string]interface{}) (result interface{}) {
	// Group by parent map[parent] -> (map[path] -> value)
	parentDataMap := make(map[string]map[string]interface{})

	for path, value := range data {
		parent := getParentPath(path)
		if parentDataMap[parent] == nil {
			parentDataMap[parent] = make(map[string]interface{})
		}
		parentDataMap[parent][path] = value
	}

	// make array from map
	dataArray := make([]map[string]interface{}, 0, len(parentDataMap))
	for _, value := range parentDataMap {
		dataArray = append(dataArray, value)
	}

	// VIS defines 3 forms of returning result:
	// * simple value if it is one signal
	// * map[path]value if result belongs to same parent
	// * []map[path]value if result belongs to different parents
	//
	// TODO: It is unclear from spec how to combine results in one map.
	// By which criteria we should put data to one map or to array element.
	// For now it is combined by parent node.

	if len(dataArray) == 1 {
		if len(dataArray[0]) == 1 {
			for _, value := range dataArray[0] {
				// return simple value
				return value
			}
		}
		// return map of same parent
		return dataArray[0]
	}
	// return array of different parents
	return dataArray
}
