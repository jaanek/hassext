package hass

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/jaanek/hassext/httpclient"
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
	mq     MqttClient
	data   data
	http   httpclient.HttpClient
	url    string
	token  string
	errors chan error
	update chan struct{}
	// sensors
	mainValve *BuiltInValve
}

func NewEmodulClient(lo logf.Logger, mq MqttClient, url, token string) EModul {
	return &emodul{
		lo:     lo,
		mq:     mq,
		http:   httpclient.New(httpclient.DefaultRetryCheckPolicy(), httpclient.DefaultRetryWaitDelay),
		url:    url,
		token:  token,
		errors: make(chan error, 10),
		update: make(chan struct{}, 1),
	}
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

	// parse sensor's data on data updates
	go func() {
		// send configs
		config := SensorMqttConfig{
			Device: SensorMqttConfigDevice{
				Identifiers:  []string{"hassext_floorwatermainvalve1"},
				Manufacturer: "Katel",
				Model:        "Temperature sensor",
				Name:         "Floor water main valve",
			},
			DeviceClass:       "temperature",
			UniqueId:          "hassext_floorwatermainvalve1",
			UnitOfMeasurement: "°C",
			ValueTemplate:     "{{ value_json.temperature }}",
			EnabledByDefault:  true,
			Name:              "Floor water main valve temperature",
			StateClass:        "measurement",
			StateTopic:        "hassext/floor-water-main-valve-current-temp",
		}
		bytes, err := json.Marshal(config)
		if err != nil {
			m.lo.Error("Valve config serialize error", "error", err)
			return
		}
		fmt.Printf("Publish config %+v\n", string(bytes))
		err = m.mq.Publish(ctx, 10*time.Second, "homeassistant/sensor/floorwatermainvalve1/temperature/config", bytes)
		if err != nil {
			m.lo.Error("Valve config mqtt publish", "error", err)
			return
		}

		// start listening updates
		for {
			select {
			case <-m.DataUpdated():
				{
					// parse built-in valve
					v, err := m.ParseValve(1012)
					if err != nil {
						m.lo.Error("Valve parse", "error", err)
						continue
					}
					// trigger currentTemp update if changed
					if m.mainValve == nil || (*m.mainValve.currentTemp != *v.currentTemp) {
						data := struct {
							Temperature float32 `json:"temperature"`
						}{
							Temperature: float32(*v.currentTemp) / 10,
						}
						bytes, err := json.Marshal(data)
						if err != nil {
							m.lo.Error("Valve serialize error", "error", err)
							continue
						}
						fmt.Printf("Publish data %+v, value: %v\n", string(bytes), *v.currentTemp)
						err = m.mq.Publish(ctx, 10*time.Second, "hassext/floor-water-main-valve-current-temp", bytes)
						if err != nil {
							m.lo.Error("Valve mqtt publish", "error", err)
						}
					}
					m.mainValve = v
					fmt.Printf("Valve data updated %+v\n", m.mainValve)
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
	req, err := httpclient.NewRequest("GET", m.url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.token))
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
