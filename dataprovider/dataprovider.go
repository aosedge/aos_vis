package dataprovider

import (
	"errors"
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
	adapter        dataadapter.DataAdapter //TODO: change to interface
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
	provider = &DataProvider{}
	provider.sensorDataChannel = make(chan []dataadapter.VisData, 100)
	provider.visDataStorage = createVisDataStorage()
	go provider.start()
	provider.adapter = dataadapter.GetVisDataAdapter()
	go provider.adapter.StartGettingData(provider.sensorDataChannel)

	return provider, nil
}

//TODO: add parameter to read or write
// IsPublicPath if path is public no authentication is required
func (dataprovider *DataProvider) IsPublicPath(path string) bool {
	if path == "Attribute.Vehicle.VehicleIdentification.VIN" {
		return true
	}
	if path == "Attribute.Vehicle.UserIdentification.Users" {
		return true
	}

	if path == "Signal.Drivetrain.InternalCombustionEngine.Power" {
		return true
	}
	if path == "Signal.Test.RPM" {
		return false
	}

	return true //TODO: currently make all public
}

// GetData returns VIS data
func (dataprovider *DataProvider) GetData(path string) (outData interface{}, err error) {
	var wasFound bool
	wasFound = false
	err = nil
	validID, err := createRegexpFromPath(path)
	if err != nil {
		log.Error("Incorrect path ", err)
		return outData, errors.New("404 Not found")
	}
	//var outputArray []map[string]interface{}
	m := make(map[string]interface{})

	for path, data := range dataprovider.visDataStorage {
		if validID.MatchString(path) == true {
			wasFound = true
			if data.isInitialized == false {
				//TODO: request data from adapter
			}
			//var m map[string]interface{}

			m[path] = data.data
			//		outputArray = append(outputArray, m)

			log.Debug("data = ", m[path])
		}
	}
	if wasFound == false {
		err = errors.New("404 Not found")
	}
	//TODO : return one value, or array or object
	//outData = outputArray
	outData = m
	return outData, err
}

// SetData sets VIS data
func (dataprovider *DataProvider) SetData(path string, inputData interface{}) (err error) {
	//TODO: prepare data and send set to adapter

	validID, err := createRegexpFromPath(path)
	if err != nil {
		log.Error("Incorrect path: ", err)
		return err
	}

	var visData []dataadapter.VisData

	for path := range dataprovider.visDataStorage {
		if validID.MatchString(path) == true {
			// array - try to find appropriate item to set
			arrayData, ok := inputData.([]interface{})
			if ok {
				for _, itemData := range arrayData {
					itemMap, ok := itemData.(map[string]interface{})
					if !ok {
						return errors.New("Wrong value format")
					}
					for dataPath, data := range itemMap {
						if strings.HasSuffix(path, dataPath) {
							visData = append(visData, dataadapter.VisData{Path: path, Data: data})
						}
					}
				}
			} else {
				// just value - set this for all matched items
				visData = append(visData, dataadapter.VisData{Path: path, Data: inputData})
			}
		}
	}

	return dataprovider.adapter.SetData(visData)
}

// Subscribe subscribes for data change
func (dataprovider *DataProvider) Subscribe(subsChan chan<- SubscriptionOutputData, path string) (id string, err error) {
	//TODO: add checking available path
	var subsElement subscriptionPare

	subsElement.value, err = createRegexpFromPath(path)
	if err != nil {
		log.Error("incorrect path ", err)
		return "", errors.New("404 Not found")
	}

	dataprovider.subscription.mutex.Lock()
	defer dataprovider.subscription.mutex.Unlock()

	dataprovider.currentSubsID++

	subsElement.subscriptionID = dataprovider.currentSubsID
	var wasFound bool
	for i := range dataprovider.subscription.ar {
		if dataprovider.subscription.ar[i].subsChan == subsChan {
			wasFound = true
			dataprovider.subscription.ar[i].ids = append(dataprovider.subscription.ar[i].ids, subsElement)
			log.Debug("Add subscription to available channel ID", dataprovider.currentSubsID, " path ", path)
		}
	}

	if wasFound == false {
		var subscripRootElement subscriptionElement
		subscripRootElement.subsChan = subsChan
		subscripRootElement.ids = append(subscripRootElement.ids, subsElement)
		dataprovider.subscription.ar = append(dataprovider.subscription.ar, subscripRootElement)
		log.Debug("Create new subscription ID", dataprovider.currentSubsID, " path ", path)
	}
	return strconv.FormatUint(dataprovider.currentSubsID, 10), nil
}

// Unsubscribe unsubscribes from data change
func (dataprovider *DataProvider) Unsubscribe(subsChan chan<- SubscriptionOutputData, subsID string) (err error) {
	var intSubs uint64
	intSubs, err = strconv.ParseUint(subsID, 10, 64)
	if err != nil {
		log.Error("Error cant convert subsID to int64")
		return err
	}
	dataprovider.subscription.mutex.Lock()
	defer dataprovider.subscription.mutex.Unlock()

	for i := range dataprovider.subscription.ar {
		if dataprovider.subscription.ar[i].subsChan == subsChan {
			wasFound := false
			for j, sID := range dataprovider.subscription.ar[i].ids {
				if intSubs == sID.subscriptionID {
					dataprovider.subscription.ar[i].ids = append(dataprovider.subscription.ar[i].ids[:j], dataprovider.subscription.ar[i].ids[j+1:]...)
					wasFound = true
					break
				}
			}
			if wasFound {
				if len(dataprovider.subscription.ar[i].ids) == 0 {
					dataprovider.subscription.ar = append(dataprovider.subscription.ar[:i], dataprovider.subscription.ar[i+1:]...)
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
func (dataprovider *DataProvider) UnsubscribeAll(subsChan chan<- SubscriptionOutputData) (err error) {
	dataprovider.subscription.mutex.Lock()
	defer dataprovider.subscription.mutex.Unlock()

	for i := range dataprovider.subscription.ar {
		if dataprovider.subscription.ar[i].subsChan == subsChan {
			dataprovider.subscription.ar = append(dataprovider.subscription.ar[:i], dataprovider.subscription.ar[i+1:]...)
			return nil
		}
	}
	//return errors.New("404 Not found")
	return nil
}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (dataprovider *DataProvider) start() {
	for {
		incomingData := <-dataprovider.sensorDataChannel
		dataprovider.processIncomingData(incomingData[:])
	}
}

func (dataprovider *DataProvider) processIncomingData(incomeData []dataadapter.VisData) {

	type notificationPair struct {
		id   uint64
		data map[string]interface{}
	}

	type notificationElement struct {
		subsChan         chan<- SubscriptionOutputData
		notificationData []notificationPair
	}

	dataprovider.subscription.mutex.Lock()
	defer dataprovider.subscription.mutex.Unlock()

	var notificationArray []notificationElement
	for _, data := range incomeData {
		//find element which receive in dataStorage

		//		log.Debug("process path = ", data.Path, " data= ", data.Data)
		wasChanged := false

		if data.Data == nil {
			continue
		}

		item := dataprovider.visDataStorage[data.Path]

		item.isInitialized = true
		item.data = data.Data

		valueType := reflect.TypeOf(data.Data).Kind()
		//		log.Debug("Value type ", valueType)

		if valueType == reflect.Array || valueType == reflect.Slice {
			//TODO: add array comeration
			wasChanged = true
			dataprovider.visDataStorage[data.Path] = item
		} else {
			if dataprovider.visDataStorage[data.Path].data != data.Data {
				log.Debug("Data for ", data.Path, " was changed from ", dataprovider.visDataStorage[data.Path].data, " to ", data.Data)
				dataprovider.visDataStorage[data.Path] = item
				wasChanged = true
			}
		}

		//		log.Debug("wasChanged = ", wasChanged)
		if wasChanged == true {
			log.Debug("process path = ", data.Path, " data= ", data.Data)
			// prepare data fro change notification
			notifyData := dataprovider.getNotificationElementsByPath(data.Path)
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
func (dataprovider *DataProvider) getNotificationElementsByPath(path string) (returnData []notificationData) {
	log.Debug("getNotificationElementsByPath path=", path)
	wasFound := false
	for i := range dataprovider.subscription.ar {
		for j := range dataprovider.subscription.ar[i].ids {
			if dataprovider.subscription.ar[i].ids[j].value.MatchString(path) {
				log.Debug("Find subscription element ID ", dataprovider.subscription.ar[i].ids[j].subscriptionID, " path= ", path)
				returnData = append(returnData, notificationData{
					subsChan: dataprovider.subscription.ar[i].subsChan,
					id:       dataprovider.subscription.ar[i].ids[j].subscriptionID})
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

func createRegexpFromPath(path string) (exp *regexp.Regexp, err error) {
	regexpStr := strings.Replace(path, ".", "[.]", -1)
	regexpStr = strings.Replace(regexpStr, "*", ".*?", -1)
	regexpStr = "^" + regexpStr
	log.Debug("filter =", regexpStr)
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
