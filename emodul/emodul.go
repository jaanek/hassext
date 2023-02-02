package emodul

import (
	"context"
	"fmt"
	"io/ioutil"
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
	Url                    string
	Username               string
	Password               string
	ModuleId               string
	UserId                 uint64
	Token                  string
}

func NewEmodulClient(lo logf.Logger, mq mq.MqttClient, params *HttpClientParams) EModul {
	return &emodul{
		lo:        lo,
		mq:        mq,
		http:      httpclient.New(getEmodulDefaultRetryCheckPolicy(lo, params), emodulDefaultRetryWaitDelay),
		params:    params,
		errors:    make(chan error, 10),
		update:    make(chan struct{}, 1),
		mainValve: &BuiltInValve{},
	}
}

func (m *emodul) Init() error {
	token, userId, err := newEmodulAccessToken(m.lo, m.params)
	if err != nil || token == "" {
		return err
	}
	m.params.Token = token
	m.params.UserId = userId
	m.lo.Info("login", "user_id", userId, "access token", token)
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
	// GET data
	req, err := httpclient.NewRequest("GET", m.params.Url+"/users/"+strconv.FormatInt(int64(m.params.UserId), 10)+"/modules/"+m.params.ModuleId, nil)
	if err != nil {
		return err
	}
	if m.params.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.params.Token))
	}
	res, err := m.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// Parse the response and write to the storage
	data, err := oj.Parse(body)
	if err != nil {
		return err
	}
	m.data.Write(data)

	return nil
}

func (m *emodul) DataUpdated() chan struct{} {
	return m.update
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
