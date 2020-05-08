package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	timeout = 4 * time.Second
)

// HTTPError represents a wrapper of http response error
type HTTPError struct {
	Status     string
	StatusCode int
}

type HTTPClient struct {
	*http.Client
}

func InitHTTPClient() *HTTPClient {
	return &HTTPClient{
		/* Ref:
		1. https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
		2. https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
		3. https://medium.com/@simonfrey/go-as-in-golang-standard-net-http-config-will-break-your-production-environment-1360871cb72b
		*/
		&http.Client{
			// Instead of using default timeout "0", set timeout to prevent from malicious service trying to blocking requests (and goroutines) indefinitely,
			Timeout: timeout,
		},
	}
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("***** [HTTP::ERROR] *****[Status:%s] [StatusCode:%d]", e.Status, e.StatusCode)
}

func (hc HTTPClient) HTTPRequest(method, url string, headers map[string]string, reqBody []byte) ([]byte, error) {
	switch strings.ToUpper(method) {
	case "GET":
		return hc.HTTPGet(url, headers)
	case "POST":
		return hc.HTTPPost(url, headers, reqBody)
	case "DELETE":
		return hc.HTTPDelete(url, headers)
	default:
		return nil, fmt.Errorf("net/http: invalid method %q", method)
	}
}

// HTTPGet implement HTTP GET request
func (hc HTTPClient) HTTPGet(url string, headers map[string]string) ([]byte, error) {
	request, _ := http.NewRequest("GET", url, nil)
	log.Printf("***** HTTPGet *****[URL:%s] [HEADERS:%s]", url, headers)

	if len(headers) > 0 {
		for key, value := range headers {
			request.Header.Set(key, value)
		}
	}

	response, httpErr := hc.Do(request)
	if httpErr != nil {
		log.Errorf("***** HTTPGet::[FAIL] *****[URL:%s] [HEADERS:%s] [Error:%v] ", url, headers, httpErr)
		return nil, httpErr
	}
	if response.StatusCode != 200 {
		log.Errorf("***** HTTPGet::[FAIL] *****[URL:%s] [HEADERS:%s] [StatusCode:%d] [RESPONSE:%s] ", url, headers, response.StatusCode, response.Status)
		return nil, HTTPError{response.Status, response.StatusCode}
	}
	// without closing the response body, the connection may remain open and cause resource leak.
	defer response.Body.Close()

	resBody, ioErr := ioutil.ReadAll(response.Body)
	if ioErr != nil {
		log.Errorf("***** HTTPGet::[FAIL] *****ReadAll Execution [Error:%v] ", ioErr)
		return nil, ioErr
	}

	return resBody, nil
}

// HTTPPost implement HTTP POST request
func (hc HTTPClient) HTTPPost(url string, headers map[string]string, reqBody []byte) ([]byte, error) {
	request, _ := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	log.Printf("***** HTTPPost *****[URL:%s] [HEADERS:%s] [BODY:%s] ", url, headers, string(reqBody))

	if len(headers) > 0 {
		for key, value := range headers {
			request.Header.Set(key, value)
		}
	}

	response, httpErr := hc.Do(request)
	if httpErr != nil {
		log.Errorf("***** HTTPPost::[FAIL] *****[URL:%s] [HEADERS:%s] [BODY:%s] [Error:%v] ", url, headers, string(reqBody), httpErr)
		return nil, httpErr
	}
	if response.StatusCode != 200 {
		log.Errorf("***** HTTPPost::[FAIL] *****[URL:%s] [HEADERS:%s] [BODY:%s] [StatusCode:%d] [RESPONSE:%s] ", url, headers, string(reqBody), response.StatusCode, response.Status)
		return nil, HTTPError{response.Status, response.StatusCode}
	}
	defer response.Body.Close()

	resBody, ioErr := ioutil.ReadAll(response.Body)
	if ioErr != nil {
		log.Errorf("***** HTTPPost::[FAIL] *****ReadAll Execution [Error:%v] ", ioErr)
		return nil, ioErr
	}

	log.Infof("***** HTTPPost::[SUCCESS] *****[URL:%s] [RESPONSE:%s] ", url, resBody)
	return resBody, nil
}

// HTTPDelete implement HTTP DELETE request
func (hc HTTPClient) HTTPDelete(url string, headers map[string]string) ([]byte, error) {
	request, _ := http.NewRequest("DELETE", url, bytes.NewReader([]byte{}))
	log.Printf("***** HTTPDelete *****[URL:%s] [HEADERS:%s] ", url, headers)

	if len(headers) > 0 {
		for key, value := range headers {
			request.Header.Set(key, value)
		}
	}

	response, httpErr := hc.Do(request)
	if httpErr != nil {
		log.Errorf("***** HTTPDelete::[FAIL] *****[URL:%s] [HEADERS:%s] [Error:%v] ", url, headers, httpErr)
		return nil, httpErr
	}
	if response.StatusCode != 200 {
		log.Errorf("***** HTTPDelete::[FAIL] *****[URL:%s] [HEADERS:%s] [StatusCode:%d] [RESPONSE:%s] ", url, headers, response.StatusCode, response.Status)
		return nil, HTTPError{response.Status, response.StatusCode}
	}
	defer response.Body.Close()

	resBody, ioErr := ioutil.ReadAll(response.Body)
	if ioErr != nil {
		log.Errorf("***** HTTPDelete::[FAIL] *****ReadAll Execution [Error:%v] ", ioErr)
		return nil, ioErr
	}

	log.Infof("***** HTTPDelete::[SUCCESS] *****[URL:%s] [RESPONSE:%s] ", url, resBody)
	return resBody, nil
}
