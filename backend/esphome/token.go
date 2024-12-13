package esphome

import (
	"net/http"
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
