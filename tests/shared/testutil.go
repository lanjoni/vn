package shared

import (
	"testing"
	"time"

	"vn/tests/shared/testserver"
)

func WaitForServerReady(t *testing.T, server *testserver.TestServer) {
	t.Helper()

	healthChecker := NewHealthChecker()
	if err := healthChecker.WaitForServerReady(server.URL); err != nil {
		t.Logf("Server health check failed, but continuing with test: %v", err)
	}
}

func WaitForServerReadyWithTimeout(t *testing.T, server *testserver.TestServer, timeout time.Duration) {
	t.Helper()

	healthChecker := NewHealthChecker()
	if err := healthChecker.WaitForServerReadyWithTimeout(server.URL, timeout); err != nil {
		t.Logf("Server health check failed, but continuing with test: %v", err)
	}
}

func WaitForConditionInTest(t *testing.T, condition func() bool, timeout time.Duration, description string) bool {
	t.Helper()

	healthChecker := NewHealthChecker()
	success := healthChecker.WaitForCondition(condition, timeout)

	if !success {
		t.Logf("Condition '%s' was not met within %v", description, timeout)
	}

	return success
}
func StartTestMetrics(t *testing.T) *TestTimer {
	t.Helper()
	return StartTestTimer(t.Name())
}

func MeasureTestExecution(t *testing.T, testFunc func()) {
	t.Helper()

	timer := StartTestMetrics(t)
	defer func() {
		if r := recover(); r != nil {
			timer.SetSuccess(false)
			timer.Finish()
			panic(r)
		}
	}()

	testFunc()
	timer.SetSuccess(!t.Failed())
	timer.Finish()
}

func MeasureTestExecutionWithSetup(t *testing.T, setupFunc func(), testFunc func()) {
	t.Helper()

	timer := StartTestMetrics(t)
	defer func() {
		if r := recover(); r != nil {
			timer.SetSuccess(false)
			timer.Finish()
			panic(r)
		}
	}()

	timer.StartSetup()
	setupFunc()
	timer.EndSetup()

	testFunc()
	timer.SetSuccess(!t.Failed())
	timer.Finish()
}
