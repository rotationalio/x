package out

// BadgerLogger implements the badger.Logger to ensure database log messages get
// appropriately logged at the level specified (and use the internal logger).
type BadgerLogger struct{}

// Errorf writes a log message to out.Warn
func (l *BadgerLogger) Errorf(msg string, a ...interface{}) {
	Warn(msg, a...)
}

// Warningf writes a log message to out.Caution
func (l *BadgerLogger) Warningf(msg string, a ...interface{}) {
	Caution(msg, a...)
}

// Infof writes a message to out.Info
func (l *BadgerLogger) Infof(msg string, a ...interface{}) {
	Info(msg, a...)
}

// Debugf writes a message to out.Debug
func (l *BadgerLogger) Debugf(msg string, a ...interface{}) {
	Debug(msg, a...)
}
