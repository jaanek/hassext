package emodul

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/jaanek/hassext/httpclient"
	"github.com/zerodha/logf"
)

func getApiDefaultRetryCheckPolicy(lo logf.Logger, params *HttpClientParams) httpclient.RetryCheck {
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

		// if unauthorized then try to get a new access token and try again
		skipRetryAuthorization := params != nil && params.SkipRetryAuthorization
		if resp.StatusCode == http.StatusUnauthorized && !skipRetryAuthorization {
			url := req.URL.Scheme + "://" + req.URL.Host + req.URL.Path

			// if we are not dealing with api url then relogin frontend
			if strings.HasPrefix(url, params.ApiUrl) {
				// new api token
				token, userId, err := NewApiToken(lo, params)
				if err != nil || token == "" {
					return true, err
				}
				lo.Info("login", "user_id", userId, "access token", token)
				// update current api instance so that other requests succeed
				params.Token = token
				params.UserId = userId
				// update current request that got unauthorized so that next try succeeds
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
				return true, nil
			} else if strings.HasPrefix(url, params.FrontendUrl) {
				loginRes, err := FrontendLogin(lo, params)
				if err != nil {
					return true, err
				}
				req.Header.Del("Cookie")
				params.SetCookies(req)
				params.ModuleHash = loginRes.SelectedModuleHash
				params.ModuleIndex = loginRes.SelectedModuleIndex
				lo.Info("frontend login", "success", loginRes.Authenticated, "module index", params.ModuleIndex, "module hash", params.ModuleHash)
				return true, nil
			} else {
				lo.Warn("login url unknown", "url", req.URL.RequestURI())
			}
		}
		return false, nil
	}
}

func emodulDefaultRetryWaitDelay(attemptNum int, resp *http.Response) (waitDelay time.Duration) {
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

func NewApiToken(lo logf.Logger, params *HttpClientParams) (string, uint64, error) {
	// payload
	up := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: params.Username,
		Password: params.Password,
	}
	payload, err := json.Marshal(up)
	if err != nil {
		return "", 0, err
	}
	client := httpclient.New(httpclient.DefaultRetryCheckPolicy(), httpclient.DefaultRetryWaitDelay, false)
	req, err := httpclient.NewRequest("POST", params.ApiUrl+"/authentication", bytes.NewReader(payload))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", 0, err
	}

	// parse the response
	tokenResp := struct {
		Authenticated    bool   `json:"authenticated"`
		UserId           uint64 `json:"user_id"`
		AccessGoogleHome bool   `json:"access_google_home"`
		Token            string `json:"token"`
	}{}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", 0, err
	}
	if !tokenResp.Authenticated {
		return "", 0, fmt.Errorf("authenticated: %v", tokenResp.Authenticated)
	}
	return tokenResp.Token, tokenResp.UserId, nil
}
