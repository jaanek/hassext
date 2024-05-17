package httpclient

import (
	"bytes"
	"io"
	"net/http"
)

func HttpReqCallback(req *Request)            {}
func HttpGetRespCallback(resp *http.Response) {}
func HttpRespCallback(resp *http.Response) ([]byte, error) {
	return nil, nil
}

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
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	getResp(res)
	return body, nil
}

func Post(client HttpClient, url string, data []byte, setReq func(req *Request), getResp func(resp *http.Response) ([]byte, error)) ([]byte, error) {
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
	body, err := getResp(res)
	if err != nil {
		return nil, err
	}
	if body == nil {
		body, err = io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}
