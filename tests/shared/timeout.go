package shared

import (
	"os"
	"strconv"
	"time"
)

type TimeoutConfig struct {
	ServerStart    time.Duration
	HTTPRequest    time.Duration
	TestExecution  time.Duration
	HealthCheck    time.Duration
	PollInterval   time.Duration
	MaxRetries     int
}

func GetOptimizedTimeouts() TimeoutConfig {
	config := TimeoutConfig{
		ServerStart:   3 * time.Second,
		HTTPRequest:   2 * time.Second,
		TestExecution: 5 * time.Second,
		HealthCheck:   500 * time.Millisecond,
		PollInterval:  50 * time.Millisecond,
		MaxRetries:    3,
	}

	if isCI() {
		config.ServerStart = 5 * time.Second
		config.HTTPRequest = 3 * time.Second
		config.TestExecution = 8 * time.Second
		config.HealthCheck = 1 * time.Second
		config.PollInterval = 100 * time.Millisecond
		config.MaxRetries = 5
	}

	if isDebugMode() {
		config.ServerStart = 10 * time.Second
		config.HTTPRequest = 10 * time.Second
		config.TestExecution = 30 * time.Second
		config.HealthCheck = 2 * time.Second
		config.PollInterval = 200 * time.Millisecond
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