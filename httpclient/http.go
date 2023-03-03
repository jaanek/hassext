package httpclient

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

func HttpReqCallback(req *Request)         {}
func HttpRespCallback(resp *http.Response) {}

func Get(client HttpClient, url string, setReq func(req *Request), getResp func(resp *http.Response)) ([]byte, error) {
	// GET data
	req, err := NewRequest("GET", url, nil)
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

func Post(client HttpClient, url string, data []byte, setReq func(req *Request), getResp func(resp *http.Response)) ([]byte, error) {
	req, err := NewRequest("POST", url, bytes.NewReader(data))
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
