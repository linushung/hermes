package server

import (
	"crypto/x509"
	"net/http"
	"net/url"
	"regexp"
	"time"

	rhttp "github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	defaultRetryMax              = 2
	defaultMaxConnsPerHost       = 100
	defaultMaxIdleConns          = 75
	defaultMaxIdleConnsPerHost   = 75
	defaultIdleConnTimeout       = 45 * time.Second
	defaultExpectContinueTimeout = 1 * time.Second
)

// RetryHTTPClient represent a wrapper of retryablehttp.Client
type RetryHTTPClient struct {
	*rhttp.Client
}

// Ref: https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
// InitRetryClient return a retryable HTTP client with default config of Hermes service
func InitRetryClient() *RetryHTTPClient {
	rc := rhttp.NewClient()
	// Replace default timeout "0" for http.client
	rc.HTTPClient.Timeout = timeout
	rc.Logger = log.New()
	rc.RetryMax = defaultRetryMax
	rc.CheckRetry = defaultRetryPolicy
	//rc.Backoff = rhttp.LinearJitterBackoff

	// Replace default config for http.Transport
	t := rc.HTTPClient.Transport.(*http.Transport)
	t.MaxConnsPerHost = defaultMaxConnsPerHost
	t.MaxIdleConns = defaultMaxIdleConns
	t.MaxIdleConnsPerHost = defaultMaxIdleConnsPerHost
	t.IdleConnTimeout = defaultIdleConnTimeout
	//t.ExpectContinueTimeout = defaultExpectContinueTimeout

	return &RetryHTTPClient{rc}
}

//Ref: rhttp.DefaultRetryPolicy()
func defaultRetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// A regular expression to match the error returned by net/http when the
	// configured number of redirects is exhausted. This error isn't typed
	// specifically so we resort to matching on the error string.
	redirectsErrorRe := regexp.MustCompile(`stopped after \d+ redirects\z`)

	// do not retry on context.Canceled or context.DeadlineExceeded
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		if v, ok := err.(*url.Error); ok {
			// Don't retry if the error was due to too many redirects.
			if redirectsErrorRe.MatchString(v.Error()) {
				return false, nil
			}

			// Don't retry if the error was due to TLS cert verification failure.
			if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
				return false, nil
			}

			// Don't retry if the error was due to Timeout!
			// Error: [net/http: request canceled (Client.Timeout exceeded while awaiting headers)]
			if v.Timeout() {
				return false, nil
			}
		}

		// The error is likely recoverable so retry.
		return true, nil
	}

	// Check the response code. We retry on 500-range responses to allow
	// the server time to recover, as 500's are typically not permanent
	// errors and may relate to outages on the server side. This will catch
	// invalid response codes as well, like 0 and 999.
	if resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != 501) {
		return true, nil
	}

	return false, nil
}
