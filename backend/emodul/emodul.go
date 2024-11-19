package emodul

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/httpclient"
	"github.com/jaanek/hassext/mq"
	"github.com/ohler55/ojg/oj"
	"github.com/zerodha/logf"
)

type EModul interface {
	Init() error
	Start(context.Context)
	SetWorkingMode(mode WorkingMode) error
	BoilerFireUp() error
	BoilerDamping() error
	SetBufferTargetTemp(BufferTempReadingLocation, uint, bool) error
	SetBufferTargetTemps(uint, uint, bool) error
}

type emodul struct {
	lo         logf.Logger
	mq         mq.MqttClient
	http       httpclient.HttpClient
	params     *HttpClientParams
	errors     chan error
	moduleData data.Data
	menuData   data.Data
	dataUpdate chan struct{}
	// parsed latest data
	mainValve        *BuiltInValve
	topBufferTemp    *TemperatureSensor
	bottomBufferTemp *TemperatureSensor
}

type HttpClientParams struct {
	SkipRetryAuthorization bool
	ApiUrl                 string
	FrontendUrl            string
	Username               string
	Password               string
	UserId                 uint64
	Token                  string
	ModuleHash             string
	ModuleIndex            int
	Cookies                map[string]string
}

type BoilerDevice uint

const (
	Ignition         BoilerDevice = 251
	WorkingModePump  BoilerDevice = 2011
	TopBufferTemp    BoilerDevice = 3680
	BottomBufferTemp BoilerDevice = 3681
)

type WorkingMode = uint

const (
	HOUSE_HEATING   WorkingMode = 0
	BOILER_PRIORITY             = 1
	PARALLEL_PUMPS              = 2
	SUMMER_MODE                 = 3
)

type BoilerStartStopMode = uint

const (
	FIRE_UP BoilerStartStopMode = 0
	DAMPING                     = 1
)

type BufferTempReadingLocation = uint

const (
	BUFFER_TOP    BufferTempReadingLocation = 0
	BUFFER_BOTTOM BufferTempReadingLocation = 1
)

type HttpControlData struct {
	Ido         BoilerDevice `json:"ido"`
	Params      uint         `json:"params"`
	ModuleIndex int          `json:"module_index"`
}

func NewEmodulClient(lo logf.Logger, mq mq.MqttClient, params *HttpClientParams) EModul {
	return &emodul{
		lo:               lo,
		mq:               mq,
		http:             httpclient.New(nil, getApiDefaultRetryCheckPolicy(lo, params), emodulDefaultRetryWaitDelay, false),
		params:           params,
		errors:           make(chan error, 10),
		dataUpdate:       make(chan struct{}, 1),
		mainValve:        &BuiltInValve{},
		topBufferTemp:    &TemperatureSensor{},
		bottomBufferTemp: &TemperatureSensor{},
	}
}

func (m *emodul) Init() error {
	// api login
	token, userId, err := NewApiToken(m.lo, m.params)
	if err != nil || token == "" {
		return err
	}
	m.params.Token = token
	m.params.UserId = userId
	m.lo.Info("login", "user_id", userId, "access token", token)

	// frontend login
	loginRes, err := FrontendLogin(m.lo, m.params)
	if err != nil {
		return err
	}
	m.params.ModuleHash = loginRes.SelectedModuleHash
	m.params.ModuleIndex = loginRes.SelectedModuleIndex
	return nil
}

func (m *emodul) Start(ctx context.Context) {
	// log errors if they happen
	go func() {
		for {
			select {
			case err := <-m.errors:
				m.lo.Error("Error while fetching emodul data", "error", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// parse sensor's data on updates
	go func() {
		// init sensors
		mvTempSensor := NewMqttTemperatureSensor(m.lo, m.mq, DeviceFloorWaterMainValve, "floorwatermainvalve1", "Floor water main valve temperature", "hassext/floor-water-main-valve-current-temp")
		mvSetTempSensor := NewMqttTemperatureSensor(m.lo, m.mq, DeviceFloorWaterMainValve, "floorwatermainvalvesettemp1", "Floor water main valve set temperature", "hassext/floor-water-main-valve-set-temp")
		mvReturnTempSensor := NewMqttTemperatureSensor(m.lo, m.mq, DeviceFloorWaterMainValve, "floorwatermainvalveboilerreturntemp1", "Floor water main valve boiler return temperature", "hassext/floor-water-main-valve-boiler-return-temp")
		sensorBufferTopTemp := NewMqttTemperatureSensor(m.lo, m.mq, DeviceBoilerBufferTank, "boilerbuffertank-toptemp", "Boiler buffer tank top temperature", "hassext/boiler-buffer-tank-top-temp")
		sensorBuffeTopTargetTemp := NewMqttTemperatureSensor(m.lo, m.mq, DeviceBoilerBufferTank, "boilerbuffertank-toptargettemp", "Boiler buffer tank top target temperature", "hassext/boiler-buffer-tank-top-target-temp")
		sensorBufferBottomTemp := NewMqttTemperatureSensor(m.lo, m.mq, DeviceBoilerBufferTank, "boilerbuffertank-bottomtemp", "Boiler buffer tank bottom temperature", "hassext/boiler-buffer-tank-bottom-temp")
		sensorBufferBottomTargetTemp := NewMqttTemperatureSensor(m.lo, m.mq, DeviceBoilerBufferTank, "boilerbuffertank-bottomtargettemp", "Boiler buffer tank bottom target temperature", "hassext/boiler-buffer-tank-bottom-target-temp")
		sensors := make([]Sensor, 0)
		sensors = append(
			sensors,
			mvTempSensor,
			mvSetTempSensor,
			mvReturnTempSensor,
			sensorBufferTopTemp,
			sensorBuffeTopTargetTemp,
			sensorBufferBottomTemp,
			sensorBufferBottomTargetTemp,
		)

		// send configs
		for _, sensor := range sensors {
			err := sensor.PublishConfig(ctx)
			if err != nil {
				m.lo.Error("Sensor config mqtt publish", "error", err)
			}
		}

		// start listening updates
		for {
			select {
			case <-m.dataUpdate:
				{
					moduleData := m.moduleData.Get()
					menuData := m.menuData.Get()

					// parse floor water main valve and trigger sensor updates if values have changed
					// at the end save the last values for the valve to compare against next time
					v, err := ParseValve(1012, moduleData)
					if err != nil {
						m.lo.Error("Main valve parse", "error", err)
						continue
					}
					if isValueChanged(m.mainValve.currentTemp, v.currentTemp) {
						sensorPublish(m.lo, ctx, mvTempSensor, float32(v.currentTemp)/10)
					}
					if isValueChanged(m.mainValve.setTemp, v.setTemp) {
						sensorPublish(m.lo, ctx, mvSetTempSensor, float32(v.setTemp))
					}
					if isValueChanged(m.mainValve.returnTemp, v.returnTemp) {
						sensorPublish(m.lo, ctx, mvReturnTempSensor, float32(v.returnTemp))
					}
					m.mainValve = v

					// parse buffer tank temperatures
					topBufferTemp, err := ParseTempSensor(1018, moduleData, menuData)
					if err != nil {
						m.lo.Error("top buffer parsing", "error", err)
					} else {
						if isValueChanged(m.topBufferTemp.currentTemp, topBufferTemp.currentTemp) {
							sensorPublish(m.lo, ctx, sensorBufferTopTemp, float32(topBufferTemp.currentTemp)/10)
						}
						if isValueChanged(m.topBufferTemp.targetTemp, topBufferTemp.targetTemp) {
							sensorPublish(m.lo, ctx, sensorBuffeTopTargetTemp, float32(topBufferTemp.targetTemp))
						}
						m.topBufferTemp = topBufferTemp
					}
					bottomBufferTemp, err := ParseTempSensor(1019, moduleData, menuData)
					if err != nil {
						m.lo.Error("bottom buffer parsing", "error", err)
					} else {
						if isValueChanged(m.bottomBufferTemp.currentTemp, bottomBufferTemp.currentTemp) {
							sensorPublish(m.lo, ctx, sensorBufferBottomTemp, float32(bottomBufferTemp.currentTemp)/10)
						}
						if isValueChanged(m.bottomBufferTemp.targetTemp, bottomBufferTemp.targetTemp) {
							sensorPublish(m.lo, ctx, sensorBufferBottomTargetTemp, float32(bottomBufferTemp.targetTemp))
						}
						m.bottomBufferTemp = bottomBufferTemp
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// start fetching data
	ticker := time.NewTicker(1 * time.Minute)
	for {
		// fetch data
		err := m.fetchData()
		if err != nil {
			werr := fmt.Errorf("fetch error %w", err)
			m.errors <- werr
			m.lo.Error("emodul", "error", werr)
		} else {
			m.dataUpdate <- struct{}{}
		}

		// wait next tick
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (m *emodul) fetchData() error {
	// fetch module data
	var url = m.params.ApiUrl + "/users/" + strconv.FormatInt(int64(m.params.UserId), 10) + "/modules/" + m.params.ModuleHash
	body, err := m.Get(url, func(req *httpclient.Request) {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.params.Token))
	}, httpclient.HttpGetRespCallback)
	if err != nil {
		return fmt.Errorf("http post error: %w", err)
	}
	data, err := oj.Parse(body)
	if err != nil {
		return err
	}
	m.moduleData.Write(data)

	// fetch menu data
	body, err = m.Get(m.params.FrontendUrl+"/frontend/menu_main?module_index="+strconv.Itoa(m.params.ModuleIndex), func(req *httpclient.Request) {
		m.params.SetCookies(req)
	}, func(resp *http.Response) {
		m.params.SaveCookies(resp)
	})
	if err != nil {
		return fmt.Errorf("http get error: %w", err)
	}
	data, err = oj.Parse(body)
	if err != nil {
		return err
	}
	m.menuData.Write(data)
	return nil
}

func (m *emodul) SetWorkingMode(mode WorkingMode) error {
	req := HttpControlData{
		Ido:         WorkingModePump,
		Params:      mode,
		ModuleIndex: m.params.ModuleIndex,
	}
	return m.sendControlData(req, "SetWorkingMode "+strconv.Itoa(int(mode)))
}

func (m *emodul) BoilerFireUp() error {
	req := HttpControlData{
		Ido:         Ignition,
		Params:      FIRE_UP,
		ModuleIndex: m.params.ModuleIndex,
	}
	return m.sendControlData(req, "BoilerFireUp")
}

func (m *emodul) BoilerDamping() error {
	req := HttpControlData{
		Ido:         Ignition,
		Params:      DAMPING,
		ModuleIndex: m.params.ModuleIndex,
	}
	return m.sendControlData(req, "BoilerDamping")
}

func (m *emodul) SetBufferTargetTemp(location BufferTempReadingLocation, temp uint, waitDone bool) error {
	var device BoilerDevice
	var locationName string
	switch location {
	case BUFFER_TOP:
		device = TopBufferTemp
		locationName = "top"
	case BUFFER_BOTTOM:
		device = BottomBufferTemp
		locationName = "bottom"
	default:
		return fmt.Errorf("Unknown buffer target temp reading location. Arg: %v", location)
	}
	m.lo.Info("SetBufferTargetTemp setting buffer target temp", "location", locationName, "device", device, "params", temp)
	req := HttpControlData{
		Ido:         device,
		Params:      temp,
		ModuleIndex: m.params.ModuleIndex,
	}
	err := m.sendControlData(req, "SetBufferTargetTemp")
	if err != nil {
		return err
	}

	// wait here until target temp takes effect, otherwise another call to the same function overrides the current one
	if waitDone {
		ticker := time.NewTicker(10 * time.Second)
		ctxT, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
	loop:
		for {
			m.lo.Info("Waiting SetBufferTargetTemp to take effect ...", "location", locationName)
			err := m.fetchData()
			if err != nil {
				return err
			}
			time.Sleep(2 * time.Second)
			t := int64(temp)
			switch location {
			case BUFFER_TOP:
				if isEqualInt64(m.topBufferTemp.targetTemp, t) {
					break loop
				}
				m.lo.Info("Waiting SetBufferTargetTemp", "location", locationName, "target temp", m.topBufferTemp.targetTemp, "param", temp)
			case BUFFER_BOTTOM:
				if isEqualInt64(m.bottomBufferTemp.targetTemp, t) {
					break loop
				}
				m.lo.Info("Waiting SetBufferTargetTemp", "location", locationName, "target temp", m.bottomBufferTemp.targetTemp, "param", temp)
			}

			select {
			case <-ticker.C:
				continue
			case <-ctxT.Done():
				break loop
			}
		}
		switch location {
		case BUFFER_TOP:
			m.lo.Info("Done SetBufferTargetTemp", "location", locationName, "target temp", m.topBufferTemp.targetTemp, "param", temp)
		case BUFFER_BOTTOM:
			m.lo.Info("Done SetBufferTargetTemp", "location", locationName, "target temp", m.bottomBufferTemp.targetTemp, "param", temp)
		}
	}
	return nil
}

func (m *emodul) SetBufferTargetTemps(top uint, bottom uint, waitDone bool) error {
	if m.bottomBufferTemp.min <= int64(bottom) && m.bottomBufferTemp.max >= int64(bottom) {
		err := m.SetBufferTargetTemp(BUFFER_BOTTOM, bottom, waitDone)
		if err != nil {
			return err
		}
		err = m.SetBufferTargetTemp(BUFFER_TOP, top, waitDone)
		if err != nil {
			return err
		}
	} else {
		err := m.SetBufferTargetTemp(BUFFER_TOP, top, waitDone)
		if err != nil {
			return err
		}
		err = m.SetBufferTargetTemp(BUFFER_BOTTOM, bottom, waitDone)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *emodul) sendControlData(req HttpControlData, prefix string) error {
	data, err := json.Marshal([]HttpControlData{req})
	if err != nil {
		return err
	}
	body, err := m.Post(m.params.FrontendUrl+"/frontend/send_control_data", data, func(req *httpclient.Request) {
		m.params.SetCookies(req)
	}, func(resp *http.Response) ([]byte, error) {
		m.params.SaveCookies(resp)
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("http post error: %w", err)
	}

	// validate response
	if string(body) != "1" {
		m.lo.Warn(prefix+" returned unknown http result code", "code", string(body))
		return fmt.Errorf("unknown http result code: %s", body)
	}
	m.lo.Info(prefix + " success")
	return nil
}

func (m *emodul) Get(url string, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	return httpclient.Get(m.http, url, setReq, getResp)
}

func (m *emodul) Post(url string, data []byte, setReq func(req *httpclient.Request), getResp func(resp *http.Response) ([]byte, error)) ([]byte, error) {
	return httpclient.Post(m.http, url, data, setReq, getResp)
}

func isValueChanged(oldValue int64, newValue int64) bool {
	if oldValue != newValue {
		return true
	}
	return false
}

func isEqualInt64(oldValue int64, newValue int64) bool {
	if oldValue == newValue {
		return true
	}
	return false
}

func sensorPublish(lo logf.Logger, ctx context.Context, sensor Sensor, value float32) {
	err := sensor.PublishData(ctx, value)
	if err != nil {
		lo.Error("Sensor mqtt publish ", "error", err)
	}
}
