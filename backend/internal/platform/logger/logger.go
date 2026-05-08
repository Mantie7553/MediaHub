package logger

import (
	"fmt"
	"log"
	"os"
)

var (
	infoLog  = log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	warnLog  = log.New(os.Stdout, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLog = log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
)

// print to console an info message
func Info(format string, args ...any) {
	infoLog.Output(2, fmt.Sprintf(format, args...))
}

// print to console a warning message
func Warn(format string, args ...any) {
	warnLog.Output(2, fmt.Sprintf(format, args...))
}

// print to console a warning message
func WarnDepth(depth int, format string, args ...any) {
	warnLog.Output(depth, fmt.Sprintf(format, args...))
}

// print to console an error message
func Error(format string, args ...any) {
	errorLog.Output(2, fmt.Sprintf(format, args...))
}

// print to console an error message
func ErrorDepth(depth int, format string, args ...any) {
	errorLog.Output(depth, fmt.Sprintf(format, args...))
}
