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
	SetInputDateTime(string, time.Time, DateTimeOption) error
	Automation(string, AutomationAction) error
	GetNordpoolPrices() NordpoolPrices
}

type EntityState struct {
	Id    string `json:"entity_id"`
	State string `json:"state"`
}

type NordpoolPrices struct {
	EntityState
	Updated      time.Time
	CurrentPrice float64   `json:"currentPrice"`
	Average      float64   `json:"average"`
	Peak         float64   `json:"peak"`
	Min          float64   `json:"min"`
	Max          float64   `json:"max"`
	Unit         string    `json:"unit"`
	Currency     string    `json:"currency"`
	Country      string    `json:"country"`
	Today        []float64 `json:"today"`
	Tomorrow     []float64 `json:"tomorrow"`
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
	return nil
}

type DateTimeOption string

const (
	INPUT_DATE     DateTimeOption = "date"
	INPUT_TIME     DateTimeOption = "time"
	INPUT_DATETIME DateTimeOption = "datetime"
)

func (m *homeassistant) SetInputDateTime(entityId string, time time.Time, format DateTimeOption) error {
	var req = struct {
		EntityId string  `json:"entity_id"`
		Date     *string `json:"date,omitempty"`     // 2019-04-20
		Time     *string `json:"time,omitempty"`     // 05:04:20
		DateTime *string `json:"datetime,omitempty"` // 2019-04-20 05:04:20
		// Timestamp *uint64 `json:"timestamp"` // 9223372036854776000
	}{
		EntityId: entityId,
	}

	switch format {
	case INPUT_DATE:
		t := time.Format("2006-01-02")
		req.Date = &t
	case INPUT_TIME:
		t := time.Format("15:04:05")
		req.Time = &t
	case INPUT_DATETIME:
		t := time.Format("2006-01-02 15:04:05")
		req.DateTime = &t
	default:
		return fmt.Errorf("Unknown datetime format option: %v", format)
	}

	err := m.callService("input_datetime", "set_datetime", req)
	if err != nil {
		return err
	}
	return nil
}

type AutomationAction string

const (
	AUTOMATION_TRIGGER  AutomationAction = "trigger"
	AUTOMATION_TOGGLE   AutomationAction = "toggle"
	AUTOMATION_TURN_ON  AutomationAction = "turn_on"
	AUTOMATION_TURN_OFF AutomationAction = "turn_off"
	AUTOMATION_RELOAD   AutomationAction = "reload"
)

func (m *homeassistant) Automation(entityId string, action AutomationAction) error {
	// automation.kaivita_soojuspump
	var req = struct {
		EntityId string `json:"entity_id"`
	}{
		EntityId: entityId,
	}

	err := m.callService("automation", string(action), req)
	if err != nil {
		return err
	}
	return nil
}

func (m *homeassistant) GetNordpoolPrices() NordpoolPrices {
	return m.nordpoolPrices
}

func (m *homeassistant) callService(domain string, service string, input any) error {
	params, err := json.Marshal(input)
	if err != nil {
		return err
	}
	m.lo.Info("homeassistant callService", "request", params)
	body, err := m.Post(m.params.ApiUrl+"/services/"+domain+"/"+service, params, func(req *httpclient.Request) {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.params.Token))
	}, httpclient.HttpRespCallback)
	if err != nil {
		return fmt.Errorf("http post error: %w", err)
	}

	// validate response
	m.lo.Info("callService", "response", string(body))
	return nil
}

func (m *homeassistant) Get(url string, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	return httpclient.Get(m.http, url, setReq, getResp)
}

func (m *homeassistant) Post(url string, data []byte, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	return httpclient.Post(m.http, url, data, setReq, getResp)
}

func (m *homeassistant) parseNordpoolPrices() (*NordpoolPrices, error) {
	entityId := "sensor.nordpool_mwh_ee_eur_3_10_02"
	var prices NordpoolPrices = NordpoolPrices{
		EntityState: EntityState{
			Id: entityId,
		},
	}
	var errors data.Errors

	// get prices
	entityPath := fmt.Sprintf("$[?(@.entity_id == \"%v\")]", "sensor.nordpool_mwh_ee_eur_3_10_02")
	json := m.stateData.GetObject(entityPath)
	if json == nil {
		return nil, fmt.Errorf("Entity state not found! entity_id: %v", entityId)
	}
	entityData := data.Data{Value: json}
	prices.State = entityData.GetString("$.state", &errors)
	if errors.HasAny() {
		return nil, errors.FirstError()
	}
	prices.CurrentPrice = entityData.GetFloat64("$.attributes.current_price", &errors)
	prices.Average = entityData.GetFloat64("$.attributes.average", &errors)
	prices.Peak = entityData.GetFloat64("$.attributes.peak", &errors)
	prices.Min = entityData.GetFloat64("$.attributes.min", &errors)
	prices.Max = entityData.GetFloat64("$.attributes.max", &errors)
	prices.Unit = entityData.GetString("$.attributes.unit", &errors)
	prices.Currency = entityData.GetString("$.attributes.currency", &errors)
	prices.Country = entityData.GetString("$.attributes.country", &errors)
	if errors.HasAny() {
		return nil, errors.FirstError()
	}
	todayData := entityData.GetArray("$.attributes.today.*")
	for _, t := range todayData {
		v, ok := t.(float64)
		if ok {
			prices.Today = append(prices.Today, v)
		}
	}
	tomorrowData := entityData.GetArray("$.attributes.tomorrow.*")
	for _, t := range tomorrowData {
		v, ok := t.(float64)
		if ok {
			prices.Tomorrow = append(prices.Tomorrow, v)
		}
	}
	return &prices, nil
}

func (m *homeassistant) ParseEntityState(entityId string) (*EntityState, error) {
	var entity EntityState = EntityState{
		Id: entityId,
	}
	var errors data.Errors

	// get sensor data
	entityPath := fmt.Sprintf("$[?(@.entity_id == \"%v\")]", entityId)
	json := m.stateData.GetObject(entityPath)
	if json == nil {
		return nil, fmt.Errorf("Entity state not found! entity_id: %v", entityId)
	}
	entityData := data.Data{Value: json}
	entity.State = entityData.GetString("$.state", &errors)
	if errors.HasAny() {
		return nil, errors.FirstError()
	}
	return &entity, nil
}
