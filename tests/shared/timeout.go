package shared

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultServerStartSeconds   = 3
	defaultHTTPRequestSeconds   = 2
	defaultTestExecutionSeconds = 5
	defaultHealthCheckMillis    = 500
	defaultPollIntervalMillis   = 50
	defaultMaxRetries           = 3
	ciHTTPRequestSeconds        = 3
	ciTestExecutionSeconds      = 8
	slowServerStartSeconds      = 30
	slowPollIntervalMillis      = 200
)

type TimeoutConfig struct {
	ServerStart   time.Duration
	HTTPRequest   time.Duration
	TestExecution time.Duration
	HealthCheck   time.Duration
	PollInterval  time.Duration
	MaxRetries    int
}

func GetOptimizedTimeouts() TimeoutConfig {
	config := TimeoutConfig{
		ServerStart:   defaultServerStartSeconds * time.Second,
		HTTPRequest:   defaultHTTPRequestSeconds * time.Second,
		TestExecution: defaultTestExecutionSeconds * time.Second,
		HealthCheck:   defaultHealthCheckMillis * time.Millisecond,
		PollInterval:  defaultPollIntervalMillis * time.Millisecond,
		MaxRetries:    defaultMaxRetries,
	}

	if isCI() {
		config.ServerStart = 5 * time.Second
		config.HTTPRequest = ciHTTPRequestSeconds * time.Second
		config.TestExecution = ciTestExecutionSeconds * time.Second
		config.HealthCheck = 1 * time.Second
		config.PollInterval = 100 * time.Millisecond
		config.MaxRetries = 5
	}

	if isDebugMode() {
		config.ServerStart = 10 * time.Second
		config.HTTPRequest = 10 * time.Second
		config.TestExecution = slowServerStartSeconds * time.Second
		config.HealthCheck = 2 * time.Second
		config.PollInterval = slowPollIntervalMillis * time.Millisecond
		config.MaxRetries = 10
	}

	return config
}

func isCI() bool {
	return os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != ""
}

func isDebugMode() bool {
	debug := os.Getenv("VN_TEST_DEBUG")
	return debug == "1" || debug == "true"
}

func GetCustomTimeout(envVar string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(envVar); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}

func WaitForCondition(condition func() bool, timeout time.Duration, pollInterval time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(pollInterval)
	}

	return false
}

func RetryWithBackoff(operation func() error, maxRetries int, initialDelay time.Duration) error {
	var lastErr error
	delay := initialDelay

	for i := 0; i < maxRetries; i++ {
		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if i < maxRetries-1 {
			time.Sleep(delay)
			delay *= 2
			if delay > 5*time.Second {
				delay = 5 * time.Second
			}
		}
	}

	return lastErr
}
