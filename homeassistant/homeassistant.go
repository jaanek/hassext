package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/httpclient"
	"github.com/ohler55/ojg/oj"
	"github.com/zerodha/logf"
)

type HomeAssistant interface {
	Start(context.Context)
	Notify(string, string, string) error
	Switch(string, SwitchAction) error
	Light(string, LightAction) error
	SetInputDateTime(string, time.Time, DateTimeOption) error
	SetInputBoolean(string, BooleanAction) error
	SetInputButton(string, ButtonAction) error
	SetInputTextValue(string, string) error
	SetInputNumberValue(string, uint64) error
	SetInputNumber(string, NumberAction) error
	Automation(string, AutomationAction) error
	Climate(string, ClimateAction) error
	ClimateSetHvacMode(string, ClimateHvacMode) error
	ClimateSetTemperature(string, float32, *ClimateHvacMode) error
	TimerStart(string, string) error
	Timer(string, TimerAction) error
	CounterConfigure(string, uint64, uint64, uint64, uint64, uint64) error
	Counter(string, CounterAction) error
	GetNordpoolPrices() NordpoolPrices
	Modbus
}

type homeassistant struct {
	lo             logf.Logger
	http           httpclient.HttpClient
	params         *HttpClientParams
	errors         chan error
	stateData      data.Data
	dataUpdate     chan struct{}
	nordpoolPrices NordpoolPrices
}

type HttpClientParams struct {
	ApiUrl string
	Token  string
}

func NewHomeAssistantClient(lo logf.Logger, params *HttpClientParams) HomeAssistant {
	return &homeassistant{
		lo:         lo,
		http:       httpclient.New(getApiDefaultRetryCheckPolicy(lo, params), defaultRetryWaitDelay),
		params:     params,
		errors:     make(chan error, 10),
		dataUpdate: make(chan struct{}, 1),
	}
}

func (m *homeassistant) Start(ctx context.Context) {
	// log errors if they happen
	go func() {
		for {
			select {
			case err := <-m.errors:
				m.lo.Error("Error while fetching homeassistant data", "error", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// parse sensor's data on updates
	go func() {
		// start listening updates
		for {
			select {
			case <-m.dataUpdate:
				{
					// get nordpool prices
					prices, err := m.parseNordpoolPrices()
					if err != nil {
						m.errors <- err
					} else {
						m.nordpoolPrices = *prices
						m.nordpoolPrices.Updated = time.Now()
						m.lo.Info("Nordpool", "prices", m.nordpoolPrices)
					}

					// emodul controller state
					entity, err := m.ParseEntityState("sensor.controller_state")
					if err != nil {
						m.errors <- err
					} else {
						m.lo.Info("Emodul", "controller state", entity)
					}
					// emodul operation mode
					entity, err = m.ParseEntityState("sensor.operation_modes")
					if err != nil {
						m.errors <- err
					} else {
						m.lo.Info("Emodul", "operation mode", entity)
					}
					// external temperature
					entity, err = m.ParseEntityState("sensor.external_temperature")
					if err != nil {
						m.errors <- err
					} else {
						m.lo.Info("Emodul", "external temperature", entity)
					}
					// komfovent operation mode
					entity, err = m.ParseEntityState("sensor.komfovent_operation_mode")
					if err != nil {
						m.errors <- err
					} else {
						m.lo.Info("Komfovent", "operation mode", entity)
					}

					// update states
					err = m.updateData()
					if err != nil {
						m.errors <- err
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// start fetching data
	ticker := time.NewTicker(10 * time.Second)
	for {
		// fetch data
		err := m.fetchData()
		if err != nil {
			werr := fmt.Errorf("fetch error %w", err)
			m.errors <- werr
			m.lo.Error("homeassistant", "error", werr)
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

func (m *homeassistant) fetchData() error {
	// fetch entity states
	body, err := m.Get(m.params.ApiUrl+"/states", func(req *httpclient.Request) {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.params.Token))
	}, httpclient.HttpRespCallback)
	if err != nil {
		return fmt.Errorf("http post error: %w", err)
	}
	data, err := oj.Parse(body)
	if err != nil {
		return err
	}
	m.stateData.Write(data)
	return nil
}

func (m *homeassistant) updateData() error {
	// t1 := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, t.Nanosecond(), t.Location())
	n := time.Now()
	t := time.Date(n.Year(), n.Month(), n.Day(), 3, 0, 0, 0, n.Location())
	err := m.SetInputDateTime("input_datetime.soojuspump_kyte_start", t, INPUT_TIME)
	if err != nil {
		return err
	}

	// set the heating allowed
	err = m.SetInputBoolean("input_boolean.katel_heating_allowed", BOOLEAN_TURN_ON)
	if err != nil {
		return err
	}
	return nil
}

func (m *homeassistant) callService(domain string, service string, input any) error {
	params, err := json.Marshal(input)
	if err != nil {
		return err
	}
	m.lo.Info("homeassistant callService", "request", params)
	var respErr error
	body, err := m.Post(m.params.ApiUrl+"/services/"+domain+"/"+service, params, func(req *httpclient.Request) {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.params.Token))
	}, func(resp *http.Response) {
		// https://developers.home-assistant.io/docs/api/rest/
		var ok = resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated
		if !ok {
			respErr = fmt.Errorf("HomeAssistant rest callService failed! Response code: %v, status: %v", resp.StatusCode, resp.Status)
		}
	})
	if err != nil || respErr != nil {
		return fmt.Errorf("http post error: %w, %w, body: %v", err, respErr, string(body))
	}
	m.lo.Info("callService", "response", string(body))
	return nil
}

func (m *homeassistant) Get(url string, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	return httpclient.Get(m.http, url, setReq, getResp)
}

func (m *homeassistant) Post(url string, data []byte, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	return httpclient.Post(m.http, url, data, setReq, getResp)
}
