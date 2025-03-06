package colored_logging_test

import (
	log "colored_logging"
	"os"
	"runtime"
	"testing"
)

func TestDebug(t *testing.T) {
	logger := log.New(os.Stdout).WithDebug()
	logger.Info("info")
	logger.Warn("warning")
	logger.Error("error")
	logger.Debug("debug")
}
func TestNoDebug(t *testing.T) {
	logger := log.New(os.Stdout)
	logger.Info("info")
	logger.Warn("warning")
	logger.Error("error")
	logger.Debug("debug")
}

func inner_func() {
	logger := log.New(os.Stdout).WithDebug().WithLogFile("test.log")
	logger.Info("info")
	logger.Warn("warning")
	logger.Error("error")
	logger.Debug("debug")
}

func TestLogFile(t *testing.T) {
	inner_func()
	runtime.GC() // log file closes automatically
}
