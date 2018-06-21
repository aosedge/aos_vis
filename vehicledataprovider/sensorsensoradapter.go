package vehicledataprovider

import (
	"io/ioutil"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// SensorSensorAdapter adapter for read data from sensorsender
type SensorSensorAdapter struct {
	url string
}

type sensorSenderData struct {
	Driver    string `json:"driver"`
	Vin       string `json:"vin"`
	Telemetry struct {
		AcStat           int         `json:"ac_stat"`
		AccMode          bool        `json:"acc_mode"`
		AirtempOutsd     int         `json:"airtemp_outsd"`
		AudModeAdv       bool        `json:"aud_mode_adv"`
		Aus              bool        `json:"aus"`
		AutoStat         int         `json:"auto_stat"`
		Autodfgstat      int         `json:"autodfgstat"`
		Avgfuellvl       int         `json:"avgfuellvl"`
		BattVolt         int         `json:"batt_volt"`
		BrkStat          int         `json:"brk_stat"`
		CellVr           int         `json:"cell_vr"`
		CruiseTgl        bool        `json:"cruise_tgl"`
		DefrostSel       bool        `json:"defrost_sel"`
		DnArwStepRq      int         `json:"dn_arw_step_rq"`
		DrLkStat         int         `json:"dr_lk_stat"`
		DrvAjar          bool        `json:"drv_ajar"`
		DrvSeatbelt      int         `json:"drv_seatbelt"`
		EblStat          int         `json:"ebl_stat"`
		Engcooltemp      int         `json:"engcooltemp"`
		Engoiltemp       int         `json:"engoiltemp"`
		Engrpm           int         `json:"engrpm"`
		Engstyle         int         `json:"engstyle"`
		FgAjar           bool        `json:"fg_ajar"`
		FlHsStat         int         `json:"fl_hs_stat"`
		FlVsStat         int         `json:"fl_vs_stat"`
		FrHsStat         int         `json:"fr_hs_stat"`
		FrVsStat         int         `json:"fr_vs_stat"`
		FtDrvAtcTemp     int         `json:"ft_drv_atc_temp"`
		FtDrvMtcTemp     int         `json:"ft_drv_mtc_temp"`
		FtHvacBlwFnSp    int         `json:"ft_hvac_blw_fn_sp"`
		FtHvacCtrlStat   int         `json:"ft_hvac_ctrl_stat"`
		FtHvacMdStat     int         `json:"ft_hvac_md_stat"`
		FtPsgAtcTemp     int         `json:"ft_psg_atc_temp"`
		FtPsgMtcTemp     int         `json:"ft_psg_mtc_temp"`
		GasRange         int         `json:"gas_range"`
		Gr               int         `json:"gr"`
		HazardStatus     bool        `json:"hazard_status"`
		HibmlvrStat      int         `json:"hibmlvr_stat"`
		HlStat           int         `json:"hl_stat"`
		HrnswPsd         bool        `json:"hrnsw_psd"`
		Hrnswpsd         bool        `json:"hrnswpsd"`
		HswStat          bool        `json:"hsw_stat"`
		LRAjar           bool        `json:"l_r_ajar"`
		Lrw              int         `json:"lrw"`
		MaxAcsts         int         `json:"max_acsts"`
		MenuRq           int         `json:"menu_rq"`
		Odo              int         `json:"odo"`
		OilPress         int         `json:"oil_press"`
		PresetCfg        int         `json:"preset_cfg"`
		Prkbrkstat       int         `json:"prkbrkstat"`
		PrndStat         int         `json:"prnd_stat"`
		PsgAjar          bool        `json:"psg_ajar"`
		PsgOdsStat       int         `json:"psg_ods_stat"`
		PsgSeatbelt      int         `json:"psg_seatbelt"`
		RRAjar           bool        `json:"r_r_ajar"`
		RecircStat       int         `json:"recirc_stat"`
		Reserved1        bool        `json:"reserved_1"`
		Reserved2        int         `json:"reserved_2"`
		Reserved3        int         `json:"reserved_3"`
		Reserved4        int         `json:"reserved_4"`
		Reserved5        int         `json:"reserved_5"`
		RlHeatStat       int         `json:"rl_heat_stat"`
		RlVentOff        bool        `json:"rl_vent_off"`
		RrDrUnlkd        bool        `json:"rr_dr_unlkd"`
		RrHeatStat       int         `json:"rr_heat_stat"`
		RrVentOff        bool        `json:"rr_vent_off"`
		RtArwRstRq       int         `json:"rt_arw_rst_rq"`
		SMinusB          bool        `json:"s_minus_b"`
		SPlusB           bool        `json:"s_plus_b"`
		Seek             int         `json:"seek"`
		StwLvrStat       int         `json:"stw_lvr_stat"`
		StwTemp          int         `json:"stw_temp"`
		SyncStat         bool        `json:"sync_stat"`
		Tirepressfl      int         `json:"tirepressfl"`
		Tirepressfr      int         `json:"tirepressfr"`
		Tirepressrl      int         `json:"tirepressrl"`
		Tirepressrr      int         `json:"tirepressrr"`
		Tirepressspr     int         `json:"tirepressspr"`
		TurnindLtOn      bool        `json:"turnind_lt_on"`
		TurnindRtOn      bool        `json:"turnind_rt_on"`
		TurnindlvrStat   int         `json:"turnindlvr_stat"`
		UpArwRq          int         `json:"up_arw_rq"`
		VcBodyStyle      int         `json:"vc_body_style"`
		VcCountry        int         `json:"vc_country"`
		VcModelYear      int         `json:"vc_model_year"`
		VcVehLine        int         `json:"vc_veh_line"`
		VehIntTemp       int         `json:"veh_int_temp"`
		VehSpeed         int         `json:"veh_speed"`
		Vehspddisp       int         `json:"vehspddisp"`
		Vol              int         `json:"vol"`
		Wa               bool        `json:"wa"`
		WhUp             bool        `json:"wh_up"`
		Wprsw6Posn       int         `json:"wprsw6posn"`
		WprwashRSwPosnV3 int         `json:"wprwash_r_sw_posn_v3"`
		WprwashswPsd     int         `json:"wprwashsw_psd"`
		Lat              float64     `json:"lat"`
		Lon              float64     `json:"lon"`
		Vin              interface{} `json:"vin"`
		MoveToRectangle  bool        `json:"move_to_rectangle"`
		InRectangle      bool        `json:"in_rectangle"`
		RectangleLong0   float64     `json:"rectangle_long0"`
		RectangleLat0    float64     `json:"rectangle_lat0"`
		RectangleLong1   float64     `json:"rectangle_long1"`
		RectangleLat1    float64     `json:"rectangle_lat1"`
	} `json:"telemetry"`
}

/*******************************************************************************
 * Public
 ******************************************************************************/

// NewSensorSensorAdapter Create SensorSensorAdapter
func NewSensorSensorAdapter(url string) (sensorAdapter *SensorSensorAdapter) {
	sensorAdapter = new(SensorSensorAdapter)
	sensorAdapter.url = url
	return sensorAdapter
}

// StartGettingData start getting data with interval
func (sensorAdapter *SensorSensorAdapter) StartGettingData(period uint, dataChan chan<- VisData) {
	ticker := time.NewTicker(time.Duration(period) * time.Second)
	interrupt := make(chan os.Signal, 1) //TODO redo
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Info("send GET request")
			data, err := sensorAdapter.readDataFromSensors()
			if err != nil {
				log.Warning("Can't read data")
				continue
			}
			jsonStr := string(data)
			visData, err := sensorAdapter.convertDataToVisFormat(&jsonStr)
			if err != nil {
				log.Warning("Can't convert to vis data")
				continue
			}
			for _, data := range visData {
				dataChan <- data
			}
		case <-interrupt:
			log.Info("interrupt")
			break
		}
	}
}

// Stop TODO
func (sensorAdapter *SensorSensorAdapter) Stop() {

}

/*******************************************************************************
 * Private
 ******************************************************************************/

func (sensorAdapter *SensorSensorAdapter) readDataFromSensors() (data []byte, err error) {
	res, err := http.Get(sensorAdapter.url)
	if err != nil {
		log.Error("Error HTTP GET to ", sensorAdapter.url, err)
		return data, err
	}
	data, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Error("Error read from HTTP body responce ", err)
		return data, err
	}
	log.Info("%s", data)
	return data, nil
}

//TODO
func (sensorAdapter *SensorSensorAdapter) convertDataToVisFormat(str *string) ([]VisData, error) {
	log.Info("Do some action")
	return nil, nil
}
