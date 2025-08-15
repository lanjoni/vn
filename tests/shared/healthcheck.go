package shared

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type HealthChecker struct {
	client   *http.Client
	timeouts TimeoutConfig
}

func NewHealthChecker() *HealthChecker {
	timeouts := GetOptimizedTimeouts()
	return &HealthChecker{
		client: &http.Client{
			Timeout: timeouts.HealthCheck,
		},
		timeouts: timeouts,
	}
}

func (hc *HealthChecker) WaitForServerReady(url string) error {
	return hc.WaitForServerReadyWithTimeout(url, hc.timeouts.TestExecution)
}

func (hc *HealthChecker) WaitForServerReadyWithTimeout(url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	healthURL := url + "/health"

	return RetryWithBackoff(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
		if err != nil {
			return err
		}

		resp, err := hc.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return &HealthCheckError{
				URL:        healthURL,
				StatusCode: resp.StatusCode,
			}
		}

		return nil
	}, hc.timeouts.MaxRetries, hc.timeouts.PollInterval)
}

func (hc *HealthChecker) WaitForCondition(condition func() bool, timeout time.Duration) bool {
	return WaitForCondition(condition, timeout, hc.timeouts.PollInterval)
}

func (hc *HealthChecker) WaitForHTTPResponse(url string, expectedStatus int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return RetryWithBackoff(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}

		resp, err := hc.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != expectedStatus {
			return &HealthCheckError{
				URL:        url,
				StatusCode: resp.StatusCode,
				Expected:   expectedStatus,
			}
		}

		return nil
	}, hc.timeouts.MaxRetries, hc.timeouts.PollInterval)
}

type HealthCheckError struct {
	URL        string
	StatusCode int
	Expected   int
}

func (e *HealthCheckError) Error() string {
	if e.Expected > 0 {
		return fmt.Sprintf("health check failed for %s: expected status %d, got %d",
			e.URL, e.Expected, e.StatusCode)
	}
	return fmt.Sprintf("health check failed for %s: got status %d", e.URL, e.StatusCode)
}
