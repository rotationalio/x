/*
Package out implements simple hierarchical logging functionality for debugging and
logging. The package can write to any configured logger, but generally writes to stdout
(hence the name of the package) for use with system logging. The log level specifies the
verbosity of the output. For example if the level is set to Info, then Debug and Trace
messages will become no-ops and ignored by the logger.

This package also provides a Caution log level - caution messages are only printed if
a specific threshold of messages has been reached. This helps to reduce the number of
repeated messages (e.g. connection down) that occur in logging while still giving
effective debugging and systems administration feedback to the user.
*/
package out

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func init() {
	logger = log.New(os.Stdout, "", 0)
	cautionCounter = new(counter)
	cautionCounter.init()
}

// Logging levels for specify the verbosity of log output. The higher the level, the
// less verbose the output is. The out will log messages >= to the specified level.
const (
	LevelTrace uint8 = iota
	LevelDebug
	LevelInfo
	LevelCaution
	LevelStatus
	LevelWarn
	LevelSilent
)

// DefaultCautionThreshold for issuing caution output to the logger after accumulating messages.
const DefaultCautionThreshold = 80

// This package acts as a module level logger, so the variables are all at the top level.
var (
	logLevel        = LevelInfo
	logger          *log.Logger
	cautionCounter  *counter
	logLevelStrings = [...]string{
		"trace", "debug", "info", "status", "warn", "silent",
	}
)

//===========================================================================
// Interact with debug output
//===========================================================================

// Init the output logger to stdout with the prefix and log options
func Init(prefix string, flag int) {
	logger = log.New(os.Stdout, prefix, flag)
}

// LogLevel returns a string representation of the current level.
func LogLevel() string {
	return logLevelStrings[logLevel]
}

// SetLogLevel modifies the log level for messages at runtime. Ensures that
// the highest level that can be set is the trace level. This function is
// often called from outside of the package in an init() function to define
// how logging is handled in the console.
func SetLogLevel(level uint8) {
	if level > LevelSilent {
		level = LevelSilent
	}

	logLevel = level
}

// SetLogger to write output to.
func SetLogger(lg *log.Logger) {
	logger = lg
}

// SetCautionThreshold to the specified number of messages before print.
func SetCautionThreshold(threshold uint) {
	cautionCounter.threshold = threshold
}

//===========================================================================
// Debugging output functions
//===========================================================================

// Print to the standard logger at the specified level. Arguments are handled
// in the manner of log.Printf, but a newline is appended.
func print(level uint8, msg string, a ...interface{}) {
	if logLevel <= level {
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		logger.Printf(msg, a...)
	}
}

// Warn prints to the standard logger if level is warn or greater; arguments
// are handled in the manner of log.Printf, but a newline is appended.
func Warn(msg string, a ...interface{}) {
	print(LevelWarn, msg, a...)
}

// Warne is a helper function to simply warn about an error received.
func Warne(err error) {
	Warn(err.Error())
}

// Status prints to the standard logger if level is status or greater;
// arguments are handled in the manner of log.Printf, but a newline is appended.
func Status(msg string, a ...interface{}) {
	print(LevelStatus, msg, a...)
}

// Caution prints to the standard logger if the level is caution or greater and if the
// number of times caution has been called with the same message has reached the
// threshold. This reduces the number of repeated log output messages while still
// allowing the system to report valuable information.
func Caution(msg string, a ...interface{}) {
	if logLevel > LevelCaution {
		// Don't waste memory if the log level is set above caution.
		return
	}

	msg = fmt.Sprintf(msg, a...)
	if cautionCounter.log(msg) {
		print(LevelCaution, msg)
	}
}

// Info prints to the standard logger if level is info or greater; arguments
// are handled in the manner of log.Printf, but a newline is appended.
func Info(msg string, a ...interface{}) {
	print(LevelInfo, msg, a...)
}

// Debug prints to the standard logger if level is debug or greater;
// arguments are handled in the manner of log.Printf, but a newline is appended.
func Debug(msg string, a ...interface{}) {
	print(LevelDebug, msg, a...)
}

// Trace prints to the standard logger if level is trace or greater;
// arguments are handled in the manner of log.Printf, but a newline is appended.
func Trace(msg string, a ...interface{}) {
	print(LevelTrace, msg, a...)
}
