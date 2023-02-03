package emodul

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/jaanek/hassext/httpclient"
	"github.com/zerodha/logf"
)

type HttpLogin struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberMe"`
	LanguageId string `json:"languageId"`
	Remote     bool   `json:"remote"`
}
type HttpLoginResponse struct {
	Authenticated       bool        `json:"authenticated"`
	Modules             []ModulData `json:"modules"`
	SelectedModuleIndex int         `json:"selectedModuleIndex"`
	SelectedModuleHash  string      `json:"selectedModuleHash"`
}
type ModulData struct {
	Id               int    `json:"id"`
	Default          bool   `json:"default"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	Type             string `json:"type"`
	ControllerStatus string `json:"controllerStatus"`
	ModuleStatus     string `json:"moduleStatus"`
	Version          string `json:"version"`
	Company          string `json:"company"`
	// AdditionalInformation  string `json:"additionalInformation"`
	// PhoneNumber            string `json:"phoneNumber"`
	// ZipCode                string `json:"zipCode"`
	// Tag                    string `json:"tag"`
	// Country                string `json:"country"`
	// GmtId                  int    `json:"gmtId"`
	// GmtTime                string `json:"gmtTime"`
	// PostcodePolicyAccepted string `json:"postcodePolicyAccepted"`
	// Style                  string `json:"style"`
}

func HttpReqCallback(req *httpclient.Request) {}
func HttpRespCallback(resp *http.Response)    {}

func Get(client httpclient.HttpClient, url string, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	// GET data
	req, err := httpclient.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	setReq(req)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	getResp(res)
	return body, nil
}

func Post(client httpclient.HttpClient, url string, data []byte, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	req, err := httpclient.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	setReq(req)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	getResp(res)
	return body, nil
}

func (p *HttpClientParams) SaveCookies(resp *http.Response) {
	for _, cookie := range resp.Cookies() {
		p.Cookies[cookie.Name] = cookie.Value
	}
}

func (p *HttpClientParams) SetCookies(req *httpclient.Request) {
	for name, value := range p.Cookies {
		req.AddCookie(&http.Cookie{
			Name: name, Value: value, MaxAge: 60,
		})
		// fmt.Printf("Set cookie: %s: %s\n", name, value)
	}
}

func FrontendLogin(lo logf.Logger, params *HttpClientParams) (*HttpLoginResponse, error) {
	params.Cookies = make(map[string]string)
	client := httpclient.New(httpclient.DefaultRetryCheckPolicy(), httpclient.DefaultRetryWaitDelay)

	// get emodule start cookies by visiting front page
	body, err := Get(client, params.FrontendUrl+"/login", HttpReqCallback, func(resp *http.Response) {
		params.SaveCookies(resp)
	})

	// login
	data, err := json.Marshal(HttpLogin{
		Username:   params.Username,
		Password:   params.Password,
		RememberMe: false,
		LanguageId: "en",
		Remote:     false,
	})
	if err != nil {
		return nil, err
	}
	body, err = Post(client, params.FrontendUrl+"/frontend/login", data, func(req *httpclient.Request) {
		params.SetCookies(req)
	}, func(resp *http.Response) {
		params.SaveCookies(resp)
	})
	if err != nil {
		return nil, err
	}
	var loginBody = &HttpLoginResponse{}
	err = json.Unmarshal(body, loginBody)
	if err != nil {
		return nil, err
	}
	if !loginBody.Authenticated {
		return nil, fmt.Errorf("Login failed!")
	}
	return loginBody, nil
}
