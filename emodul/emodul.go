package emodul

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jaanek/hassext/httpclient"
	"github.com/jaanek/hassext/mq"
	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/oj"
	"github.com/zerodha/logf"
)

type data struct {
	sync.RWMutex
	val any
}

func (d *data) Write(val any) {
	d.Lock()
	defer d.Unlock()
	d.val = val
}

func (d *data) Get() any {
	d.RLock()
	defer d.RUnlock()
	return d.val
}

type EModul interface {
	Init() error
	Start(context.Context)
	FetchData() error
	DataUpdated() chan struct{}
	SetWorkingMode(mode WorkingMode) error
	BoilerFireUp() error
	BoilerDamping() error
	GetObject(string) any
	GetArray(string) []any
	GetInt64(string, *Errors) *int64
	GetString(string, *Errors) *string
	GetBool(string, *Errors) *bool
	ParseValve(uint) (*BuiltInValve, error)
}

type emodul struct {
	lo     logf.Logger
	mq     mq.MqttClient
	data   data
	http   httpclient.HttpClient
	params *HttpClientParams
	errors chan error
	update chan struct{}
	// parsed latest data
	mainValve *BuiltInValve
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
	Ignition        BoilerDevice = 251
	WorkingModePump BoilerDevice = 2011
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

type HttpControlData struct {
	Ido         BoilerDevice `json:"ido"`
	Params      uint         `json:"params"`
	ModuleIndex int          `json:"module_index"`
}

func NewEmodulClient(lo logf.Logger, mq mq.MqttClient, params *HttpClientParams) EModul {
	return &emodul{
		lo:        lo,
		mq:        mq,
		http:      httpclient.New(getApiDefaultRetryCheckPolicy(lo, params), emodulDefaultRetryWaitDelay),
		params:    params,
		errors:    make(chan error, 10),
		update:    make(chan struct{}, 1),
		mainValve: &BuiltInValve{},
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
		sensors := make([]Sensor, 0)
		sensors = append(sensors, mvTempSensor, mvSetTempSensor, mvReturnTempSensor)

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
			case <-m.DataUpdated():
				{
					// parse floor water main valve and trigger sensor updates if values have changed
					// at the end save the last values for the valve to compare against next time
					v, err := m.ParseValve(1012)
					if err != nil {
						m.lo.Error("Main valve parse", "error", err)
						continue
					}
					if isValueChanged(m.mainValve.currentTemp, v.currentTemp) {
						sensorPublish(m.lo, ctx, mvTempSensor, float32(*v.currentTemp)/10)
					}
					if isValueChanged(m.mainValve.setTemp, v.setTemp) {
						sensorPublish(m.lo, ctx, mvSetTempSensor, float32(*v.setTemp))
					}
					if isValueChanged(m.mainValve.returnTemp, v.returnTemp) {
						sensorPublish(m.lo, ctx, mvReturnTempSensor, float32(*v.returnTemp))
					}
					m.mainValve = v
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
		err := m.FetchData()
		if err != nil {
			werr := fmt.Errorf("fetch error %w", err)
			m.errors <- werr
			m.lo.Error("emodul", "error", werr)
		} else {
			m.update <- struct{}{}
		}

		// wait next tick
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func (m *emodul) FetchData() error {
	var url = m.params.ApiUrl + "/users/" + strconv.FormatInt(int64(m.params.UserId), 10) + "/modules/" + m.params.ModuleHash
	body, err := m.Get(url, func(req *httpclient.Request) {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.params.Token))
	}, HttpRespCallback)

	// Parse the response and write to the storage
	data, err := oj.Parse(body)
	if err != nil {
		return err
	}
	m.data.Write(data)

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

func (m *emodul) sendControlData(req HttpControlData, prefix string) error {
	data, err := json.Marshal([]HttpControlData{req})
	if err != nil {
		return err
	}
	body, err := m.Post(m.params.FrontendUrl+"/frontend/send_control_data", data, func(req *httpclient.Request) {
		m.params.SetCookies(req)
	}, func(resp *http.Response) {
		m.params.SaveCookies(resp)
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

func (m *emodul) DataUpdated() chan struct{} {
	return m.update
}

func (m *emodul) Get(url string, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	return Get(m.http, url, setReq, getResp)
}

func (m *emodul) Post(url string, data []byte, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	return Post(m.http, url, data, setReq, getResp)
}

func (m *emodul) GetObject(path string) any {
	dp := jp.MustParseString(path)
	result := dp.Get(m.data.Get())
	if len(result) > 0 {
		return result[0]
	}
	return nil
}

func (m *emodul) GetArray(path string) []any {
	dp := jp.MustParseString(path)
	return dp.Get(m.data.Get())
}

func (m *emodul) GetInt64(path string, errors *Errors) *int64 {
	val, ok := m.GetObject(path).(int64)
	if !ok {
		*errors = append(*errors, fmt.Errorf(fmt.Sprintf("path parse failed: %q", path)))
	}
	return &val
}

func (m *emodul) GetString(path string, errors *Errors) *string {
	val, ok := m.GetObject(path).(string)
	if !ok {
		*errors = append(*errors, fmt.Errorf(fmt.Sprintf("path parse failed: %q", path)))
	}
	return &val
}

func (m *emodul) GetBool(path string, errors *Errors) *bool {
	val, ok := m.GetObject(path).(bool)
	if !ok {
		*errors = append(*errors, fmt.Errorf(fmt.Sprintf("path parse failed: %q", path)))
	}
	return &val
}

func isValueChanged(oldValue *int64, newValue *int64) bool {
	if newValue == nil {
		return false
	}
	if oldValue == nil {
		return true
	}
	if *oldValue != *newValue {
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
