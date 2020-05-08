package server

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/linushung/hermes/internal/pkg/configs"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
)

var (
	once     sync.Once
	instance *CircuitBreakerManager
	/*
		Timeout value has to be considered with timeout of http.Client in order for consistent response. Set
		this value a little less than http.Client makes http request mainly handle by hystrix
	*/
	// Timeout is how long to wait for command to complete, in milliseconds
	defaultTimeout = 5000
	// MaxConcurrent is how many commands of the same type can run at the same time
	defaultMaxConcurrent = 50
	// VolumeThreshold is the minimum number of requests needed before a circuit can be tripped due to health
	defaultVolumeThreshold = hystrix.DefaultVolumeThreshold
	// SleepWindow is how long, in milliseconds, to wait after a circuit opens before testing for recovery
	defaultSleepWindow = hystrix.DefaultSleepWindow
	// ErrorPercentThreshold causes circuits to open once the rolling measure of errors exceeds this percent of requests
	defaultErrorPercentThreshold = hystrix.DefaultErrorPercentThreshold
)

const (
	DefaultHandler = "GeneralEventHandler"
)

type circuitBreakerConfig struct {
	Timeout                int  `mapstructure:"timeout"`
	MaxConcurrentRequests  int  `mapstructure:"maxconcurrentrequests"`
	RequestVolumeThreshold int  `mapstructure:"requestvolumethreshold"`
	SleepWindow            int  `mapstructure:"sleepwindow"`
	ErrorPercentThreshold  int  `mapstructure:"errorpercentthreshold"`
	Retryable              bool `mapstructure:"retryable"`
}

// CircuitBreakerManager defines the basic configuration of Hystrix Circuit Breaker
type CircuitBreakerManager struct {
	Register map[string]*circuitBreakerConfig `mapstructure:"registers"`
	HTTPClient
	RetryHTTPClient
}

func GetCircuitBreakerMgr() *CircuitBreakerManager {
	return instance
}

func initHystrixStreamServer() {
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	go http.ListenAndServe(net.JoinHostPort("", "8092"), hystrixStreamHandler)
}

func InitCircuitBreakerMgr() {
	once.Do(func() {
		cbm := &CircuitBreakerManager{}
		if err := configs.GetConfigUnmarshalKey("circuitbreaker", cbm); err != nil {
			log.Fatalf("***** [INIT:CIRCUITBREAKER][FAIL] ***** Failed to init Circuit Breaker configuration:: %v ......", err)
			os.Exit(1)
		}

		cbm.Register[DefaultHandler] = &circuitBreakerConfig{
			Timeout:                defaultTimeout,
			MaxConcurrentRequests:  defaultMaxConcurrent,
			RequestVolumeThreshold: defaultVolumeThreshold,
			SleepWindow:            defaultSleepWindow,
			ErrorPercentThreshold:  defaultErrorPercentThreshold,
			Retryable:              false,
		}
		for r, c := range cbm.Register {
			hystrix.ConfigureCommand(r, hystrix.CommandConfig{
				Timeout:                c.Timeout,
				MaxConcurrentRequests:  c.MaxConcurrentRequests,
				RequestVolumeThreshold: c.RequestVolumeThreshold,
				SleepWindow:            c.SleepWindow,
				ErrorPercentThreshold:  c.ErrorPercentThreshold,
			})
		}

		hc := InitHTTPClient()
		rc := InitRetryClient()
		initHystrixStreamServer()
		instance = &CircuitBreakerManager{cbm.Register, *hc, *rc}
		log.Infof("***** [INIT:CIRCUITBREAKER] ***** Initialise circuit breaker manager with %d registers ......", len(instance.Register))
	})
}

// CBHTTPGet makes HTTP GET request with Hystrix circuit breaker
func (cbm *CircuitBreakerManager) CBHTTPGet(register, url, headers string, retryable bool) ([]byte, error) {
	if _, ok := cbm.Register[register]; ok {
		register = DefaultHandler
	}

	resTube := make(chan []byte, 1)
	runFunc := cbm.runFuncGenerator("GET", url, headers, nil, retryable, resTube)
	errTube := hystrix.Go(strings.ToLower(register), runFunc, fallbackFunc)

	select {
	case res := <-resTube:
		return res, nil
	case err := <-errTube:
		log.Errorf("***** [CIRCUITBREAKER][FAIL] ***** Error:: %#v", err.Error())
		return nil, err
	}
}

// CBHTTPPost makes HTTP POST request with Hystrix circuit breaker
func (cbm *CircuitBreakerManager) CBHTTPPost(register, url, headers string, reqBody []byte) ([]byte, error) {
	if _, ok := cbm.Register[register]; !ok {
		register = DefaultHandler
	}

	resTube := make(chan []byte, 1)
	retryable := cbm.Register[register].Retryable
	runFunc := cbm.runFuncGenerator("POST", url, headers, reqBody, retryable, resTube)

	errTube := hystrix.Go(strings.ToLower(register), runFunc, fallbackFunc)

	select {
	case res := <-resTube:
		return res, nil
	case err := <-errTube:
		log.Errorf("***** [CIRCUITBREAKER][FAIL] ***** Error:: %#v", err.Error())
		return nil, err
	}
}

func (cbm CircuitBreakerManager) runFuncGenerator(method, url, headers string, reqBody []byte, retryable bool, resTube chan []byte) func() error {
	if retryable {
		return cbm.retryableRunFunc(method, url, headers, reqBody, resTube)
	}

	return cbm.cbRunFunc(method, url, headers, reqBody, resTube)
}

func (cbm CircuitBreakerManager) cbRunFunc(method, url, headers string, reqBody []byte, resTube chan []byte) func() error {
	return func() error {
		res, httpErr := cbm.HTTPClient.HTTPRequest(method, url, map[string]string{"Content-Type": headers}, reqBody)
		if httpErr != nil {
			// Return error to fallbackFunc
			return httpErr
		}

		resTube <- res
		return nil
	}
}

func (cbm CircuitBreakerManager) retryableRunFunc(method, url, headers string, reqBody []byte, resTube chan []byte) func() error {
	return func() error {
		req, err := retryablehttp.NewRequest(method, url, reqBody)
		if err != nil {
			log.Errorf("***** [CIRCUITBREAKER][FAIL] ***** Cannot create request")
			return nil
		}

		req.Header.Set("Content-Type", headers)
		res, httpErr := cbm.RetryHTTPClient.Do(req)
		if httpErr != nil {
			// Return error to fallbackFunc
			return httpErr
		}
		if res.StatusCode != 200 {
			return fmt.Errorf("***** [HTTP::ERROR] *****[Status:%s] [StatusCode:%d]", res.Status, res.StatusCode)
		}

		resBody, ioErr := ioutil.ReadAll(res.Body)
		if ioErr != nil {
			log.Errorf("***** HTTPPost::[FAIL] *****ReadAll Execution [Error:%v] ", ioErr)
			return nil
		}

		resTube <- resBody
		return nil
	}
}

func fallbackFunc(err error) error {
	// Return error to errTube
	return err
}
