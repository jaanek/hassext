package httpclient

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

func New(tlsConfig *tls.Config, retry RetryCheck, waitDelay RetryWaitDelay, debugWire bool) HttpClient {
	var hc = &httpClient{
		client: http.Client{
			Timeout: time.Second * 5 * 60,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
		RetryMax:       3,
		RetryCheck:     retry,
		RetryWaitDelay: waitDelay,
	}
	if debugWire {
		hc.client.Transport = &loggingTransport{
			Transport: hc.client.Transport, // http.DefaultTransport,
		}
	}
	return hc
}

func DefaultRetryCheckPolicy() RetryCheck {
	return func(req *Request, resp *http.Response, err error) (bool, error) {
		if err != nil {
			return true, err
		}
		if resp.StatusCode == 0 || resp.StatusCode >= 500 {
			return true, nil
		}
		return false, nil
	}
}

func DefaultRetryWaitDelay(attemptNum int, resp *http.Response) (waitDelay time.Duration) {
	waitDelay = time.Minute

	// on "net/http: TLS handshake timeout" the resp is nil
	if resp == nil {
		return
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		waitDelay = time.Minute * 1
	} else if resp.StatusCode == http.StatusUnauthorized {
		waitDelay = time.Second * 1
	}
	return

	// if no response were given then wait for the default delay
	// if resp == nil || resp.Body == nil {
	//	return
	// }

	// // Check the if json response and get the wait delay from there
	// body, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	//	log.Printf("Error reading wait delay response: %v, %v, response body: %v", err, resp.StatusCode, string(body))
	//	return
	// }

	// apiResult := &common.WaitDelayApiResult{}
	// err = json.Unmarshal(body, apiResult)
	// if err != nil {
	//	log.Printf("Error reading wait delay json from response: %v, %v, response body: %v", err, resp.StatusCode, string(body))
	//	return
	// }
	// log.Printf("Wait delay from response: %v, %v", apiResult.Retry, apiResult.Result)
	// if apiResult.Retry > 0 {
	//	waitDelay = time.Duration(apiResult.Retry) * time.Second
	// }
}

type loggingTransport struct {
	Transport http.RoundTripper
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to log it
	reqBody, _ := httputil.DumpRequestOut(req, true)
	fmt.Printf("Request:\n%s\n", string(reqBody))

	// Perform the actual request
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Clone the response to log it
	respBody, _ := httputil.DumpResponse(resp, true)
	fmt.Printf("Response:\n%s\n", string(respBody))

	return resp, nil
}

type HttpClient interface {
	Post(url, contentType string, body io.ReadSeeker) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
	Do(req *Request) (*http.Response, error)
}

type httpClient struct {
	client         http.Client
	RetryCheck     RetryCheck
	RetryMax       int
	RetryWaitDelay RetryWaitDelay
	ErrorHandler   ErrorHandler
}

type RetryCheck func(req *Request, resp *http.Response, err error) (bool, error)
type RetryWaitDelay func(attemptNum int, resp *http.Response) time.Duration
type ErrorHandler func(resp *http.Response, err error, numTries int) (*http.Response, error)

// Request wraps the metadata needed to create HTTP requests.
type Request struct {
	body io.ReadSeeker
	*http.Request
}

func NewRequest(method, url string, body io.ReadSeeker) (*Request, error) {
	var rcBody io.ReadCloser
	if body != nil {
		rcBody = io.NopCloser(body)
	}

	httpReq, err := http.NewRequest(method, url, rcBody)
	if err != nil {
		return nil, err
	}

	return &Request{body, httpReq}, nil
}

func (c *httpClient) Do(req *Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 1; ; i++ {
		log.Printf("[DEBUG] %s %s", req.Method, req.URL)

		// Always rewind the request body when non-nil
		if req.body != nil {
			if _, err := req.body.Seek(0, 0); err != nil {
				c.client.CloseIdleConnections()
				return nil, fmt.Errorf("failed to seek body %w", err)
			}
		}

		// Attempt the request
		resp, err = c.client.Do(req.Request)
		if err != nil {
			log.Printf("[ERR] %s %s request failed: %v", req.Method, req.URL, err)
		}
		var code int // HTTP response code
		if resp != nil {
			code = resp.StatusCode
		}

		// Check if we should continue with retries
		checkOk, checkErr := c.RetryCheck(req, resp, err)
		if !checkOk {
			if checkErr != nil {
				err = checkErr
			}
			c.client.CloseIdleConnections()
			return resp, err
		}
		waitDelay := c.RetryWaitDelay(i, resp)

		// We are going to retry, consume any response to reuse the connection
		if err == nil && resp != nil {
			c.drainBody(resp.Body)
		}

		// Check if any retries left
		remain := c.RetryMax - i
		if remain == 0 {
			break
		}

		// Wait specified delay
		desc := fmt.Sprintf("%s %s", req.Method, req.URL)
		if code > 0 {
			desc = fmt.Sprintf("%s (status: %d)", desc, code)
		}
		log.Printf("[DEBUG] %s: retrying in %s (%d left)", desc, waitDelay, remain)
		time.Sleep(waitDelay)
	}

	if c.ErrorHandler != nil {
		c.client.CloseIdleConnections()
		return c.ErrorHandler(resp, err, c.RetryMax)
	}

	// By default, when max retries done, we close the response body and return an error without
	// returning the response
	if resp != nil {
		resp.Body.Close()
	}
	c.client.CloseIdleConnections()
	return nil, fmt.Errorf("%s %s giving up after %d attempts", req.Method, req.URL, c.RetryMax)
}

// We need to consume response bodies to maintain http connections, but
// limit the size we consume to respReadLimit.
var respReadLimit = int64(4096)

func (c *httpClient) drainBody(body io.ReadCloser) {
	defer body.Close()
	_, err := io.Copy(io.Discard, io.LimitReader(body, respReadLimit))
	if err != nil {
		log.Printf("[ERR] error draining response body: %v", err)
	}
}

func (c *httpClient) Post(url, contentType string, body io.ReadSeeker) (resp *http.Response, err error) {
	req, err := NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

func (c *httpClient) Get(url string) (resp *http.Response, err error) {
	req, err := NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}
