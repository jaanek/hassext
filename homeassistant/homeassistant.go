package homeassistant

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/httpclient"
	"github.com/ohler55/ojg/oj"
	"github.com/zerodha/logf"
)

type HomeAssistant interface {
	Notify
	Switch
	Light
	Climate
	Inputs
	Timer
	Counter
	Automation
	Modbus
	FetchData() error
	GetStateData() data.DataValue
}

type homeassistant struct {
	lo        logf.Logger
	http      httpclient.HttpClient
	params    *HttpClientParams
	stateData data.Data
}

type HttpClientParams struct {
	ApiUrl string
	Token  string
}

func NewHomeAssistantClient(lo logf.Logger, params *HttpClientParams) HomeAssistant {
	return &homeassistant{
		lo:     lo,
		http:   httpclient.New(getApiDefaultRetryCheckPolicy(lo, params), defaultRetryWaitDelay),
		params: params,
	}
}

func (m *homeassistant) GetStateData() data.DataValue {
	return m.stateData.Get()
}

func (m *homeassistant) FetchData() error {
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
