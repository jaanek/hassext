package jblbar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jaanek/hassext/httpclient"
	"github.com/zerodha/logf"
)

type JblBar interface {
	CallCommand(command Command, value *CommandPayload, result any) error
}

type jb struct {
	prefix string
	log    logf.Logger
	http   httpclient.HttpClient
	apiUrl string
}

func New(log logf.Logger, apiUrl string) JblBar {
	return &jb{
		prefix: "[jbl-bar] ",
		log:    log,
		http:   httpclient.New(getApiDefaultRetryCheckPolicy(log), defaultRetryWaitDelay, true),
		apiUrl: apiUrl, // "http://" + entry.GetAddr()
	}
}

type Command string

const (
	CommandSendAppController  Command = "sendAppController"
	CommandGetStreamingStatus Command = "getStreamingStatus"
)

type CommandPayload string

const (
	KeyPressedPower      CommandPayload = "power"
	KeyPressedSourceTV   CommandPayload = "source-tv"
	KeyPressedSourceHDMI CommandPayload = "source-hdmi"
	KeyPressedBluetooth  CommandPayload = "bluetooth"
	KeyPressedBassboost  CommandPayload = "bassboost"
	KeyPressedVolumeUp   CommandPayload = "volumeUp"
	KeyPressedVolumeDown CommandPayload = "volumeDown"
	KeyPressedMute       CommandPayload = "mute"
)

const (
	StreamingSourceIdle      = "IDLE"
	StreamingSourceTV        = "TV"
	StreamingSourceBluetooth = "BT"
	StreamingSourceHdmi      = "HDMI"
)

type CommandResultError struct {
	ErrorCode string `json:"error_code"`
}

type GetStreamingStatusResult struct {
	CommandResultError
	Status StreamingStatus `json:"status"`
}
type StreamingStatus struct {
	Source      string `json:"source"`
	IsStreaming string `json:"is_streaming"`
	IsAtmos     string `json:"is_atmos"`
}

// {"error_code":"0","status":{"source":"TV","is_streaming":"true","is_atmos":"false"}}

func (m *jb) CallCommand(command Command, value *CommandPayload, result any) error {
	var payloadStr = ""
	if value != nil {
		payload, err := json.Marshal(struct {
			KeyPressed CommandPayload `json:"key_pressed"`
		}{
			KeyPressed: *value,
		})
		if err != nil {
			return err
		}
		payloadStr = string(payload)
	}
	params := Values{
		"command": []string{string(command)},
	}
	if payloadStr != "" {
		params["payload"] = []string{payloadStr}
	}
	var data = []byte(params.Encode())
	var respErr error
	body, err := m.Post(m.apiUrl+"/httpapi.asp", data, func(req *httpclient.Request) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
		req.ContentLength = int64(len(data))
		fmt.Println("[REQUEST] callCommand", string(data))
	}, func(resp *http.Response) ([]byte, error) {
		body, e := readResult(resp, result)
		fmt.Println(fmt.Sprintf("[RESPONSE] callCommand Status code: %d, status: %s, result: %v", resp.StatusCode, resp.Status, string(body)))
		e = ValidateResponse(resp, e)
		if e != nil {
			return nil, e
		}
		return body, nil
	})
	if err != nil || respErr != nil {
		return fmt.Errorf("http post error: %w, %w, body: %v", err, respErr, string(body))
	}
	return nil
}

func (m *jb) Get(url string, setReq func(req *httpclient.Request), getResp func(resp *http.Response)) ([]byte, error) {
	return httpclient.Get(m.http, url, setReq, getResp)
}

func (m *jb) Post(url string, data []byte, setReq func(req *httpclient.Request), getResp func(resp *http.Response) ([]byte, error)) ([]byte, error) {
	return httpclient.Post(m.http, url, data, setReq, getResp)
}

type Values map[string][]string

// Encode encodes the values into “URL encoded” form
// ("bar=baz&foo=quux") sorted by key.
func (v Values) Encode() string {
	if len(v) == 0 {
		return ""
	}
	var buf strings.Builder
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		keyEscaped := k
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(v)
		}
	}
	return buf.String()
}

func readResult(resp *http.Response, value any) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return body, err
	}
	var e = json.NewDecoder(bytes.NewReader(body)).Decode(value)
	if e != nil {
		return body, e
	}
	return body, nil
}

func ValidateResponse(resp *http.Response, prevError error) error {
	if prevError != nil {
		return prevError
	}
	var ok = resp.StatusCode == http.StatusOK
	if !ok {
		return fmt.Errorf("Jb rest callCommand failed! Response code: %v, status: %v", resp.StatusCode, resp.Status)
	}
	return nil
}

func getApiDefaultRetryCheckPolicy(lo logf.Logger) httpclient.RetryCheck {
	return func(req *httpclient.Request, resp *http.Response, err error) (bool, error) {
		if err != nil {
			return true, err
		}
		if resp.StatusCode == 0 || resp.StatusCode >= 500 {
			return true, nil
		}
		// Request Throttling - Too many requests
		if resp.StatusCode == http.StatusTooManyRequests {
			return true, nil
		}
		return false, nil
	}
}

func defaultRetryWaitDelay(attemptNum int, resp *http.Response) (waitDelay time.Duration) {
	waitDelay = time.Second * 10

	// on "net/http: TLS handshake timeout" the resp is nil
	if resp == nil {
		return
	}

	// Request Throttling - Too many requests
	if resp.StatusCode == http.StatusTooManyRequests {
		waitDelay = time.Minute * 1
	} else if resp.StatusCode == http.StatusUnauthorized {
		waitDelay = time.Second * 1
	}
	return
}
