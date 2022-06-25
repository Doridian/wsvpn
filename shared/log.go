package shared

import (
	"fmt"
	"log"
	"os"
)

func makeLoggerPrefix(logType string, prefix string) string {
	return fmt.Sprintf("[%s-%s]", logType, prefix)
}

func MakeLogger(logType string, prefix string) *log.Logger {
	return log.New(os.Stderr, makeLoggerPrefix(logType, prefix), log.Lmsgprefix)
}

func UpdateLogger(logger *log.Logger, logType string, prefix string) {
	logger.SetPrefix(makeLoggerPrefix(logType, prefix))
}
