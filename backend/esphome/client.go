package esphome

import (
	"fmt"
	"net/http"

	"github.com/jaanek/hassext/httpclient"
	"github.com/zerodha/logf"
)

type RestClient interface {
}

type client struct {
	lo     logf.Logger
	http   httpclient.HttpClient
	params *HttpClientParams
	data   UponorControllerData
	// stateData data.Data
}

type HttpClientParams struct {
	Host string
}

func NewRestClient(lo logf.Logger, params *HttpClientParams) RestClient {
	return &client{
		lo:     lo,
		http:   httpclient.New(nil, getApiDefaultRetryCheckPolicy(lo, params), defaultRetryWaitDelay, false),
		params: params,
	}
}

func (m *client) GetData() *UponorControllerData {
	return &m.data
}

type UponorWaspVar struct {
	VarName  string `json:"waspVarName"`
	VarValue string `json:"waspVarValue"`
}

type UponorControllerData struct {
	ResultCode string `json:"result"`
	Output     struct {
		Vars []UponorWaspVar
	}
}

func (m *client) FetchData() error {
	m.data = UponorControllerData{}
	var input = []byte("{}")
	_, err := httpclient.Post(m.http, "http://"+m.params.Host+"/JNAP/", input, func(req *httpclient.Request) {
		req.Header.Set("x-jnap-action", "http://phyn.com/jnap/uponorsky/GetAttributes")
		req.ContentLength = int64(len(input))
	}, func(resp *http.Response) ([]byte, error) {
		body, e := httpclient.ReadJsonResult(resp, &m.data)
		fmt.Println(fmt.Sprintf("[RESPONSE] Status code: %d, status: %s, result: %v", resp.StatusCode, resp.Status, string(body)))
		if e != nil {
			return nil, e
		}
		return body, nil
	})
	if err != nil {
		return fmt.Errorf("http post error: %w", err)
	}
	// m.stateData.Write(result)
	return nil
}
