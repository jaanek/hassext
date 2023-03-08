package homeassistant

import (
	"context"
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
}

type EntityState struct {
	Id    string `json:"entity_id"`
	State string `json:"state"`
}

type homeassistant struct {
	lo         logf.Logger
	http       httpclient.HttpClient
	params     *HttpClientParams
	errors     chan error
	stateData  data.Data
	dataUpdate chan struct{}
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

func (m *homeassistant) Get(url string, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	return httpclient.Get(m.http, url, setReq, getResp)
}

func (m *homeassistant) Post(url string, data []byte, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	return httpclient.Post(m.http, url, data, setReq, getResp)
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
	state := entityData.GetString("$.state", &errors)
	if errors.HasAny() {
		return nil, errors.FirstError()
	}
	entity.State = *state
	return &entity, nil
}
