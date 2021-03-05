package main

import (
	"fmt"
	"log/syslog"
	"strings"
)

const DEFAULT_LEVEL = "notice"
const DEFAULT_FACILITY = "user"

// facility converts a syslog facility string into a int
func facility(keyword string) syslog.Priority {
	switch strings.ToLower(keyword) {
	case "user":
		return syslog.LOG_USER
	case "mail":
		return syslog.LOG_MAIL
	case "daemon":
		return syslog.LOG_DAEMON
	case "auth":
		return syslog.LOG_AUTH
	case "authpriv":
		return syslog.LOG_AUTHPRIV
	case "local0":
		return syslog.LOG_LOCAL0
	case "local1":
		return syslog.LOG_LOCAL1
	case "local2":
		return syslog.LOG_LOCAL2
	case "local3":
		return syslog.LOG_LOCAL3
	case "local4":
		return syslog.LOG_LOCAL4
	case "local5":
		return syslog.LOG_LOCAL5
	case "local6":
		return syslog.LOG_LOCAL6
	case "local7":
		return syslog.LOG_LOCAL7
	}
	return syslog.LOG_LOCAL5
}

// severity converts a syslog severity string into a int.
func severity(keyword string) syslog.Priority {
	switch strings.ToLower(keyword) {
	case "emerg", "panic":
		return syslog.LOG_EMERG
	case "alert":
		return syslog.LOG_ALERT
	case "crit":
		return syslog.LOG_CRIT
	case "err", "error":
		return syslog.LOG_ERR
	case "warning", "warn":
		return syslog.LOG_WARNING
	case "notice":
		return syslog.LOG_NOTICE
	case "info":
		return syslog.LOG_INFO
	case "debug":
		return syslog.LOG_DEBUG
	}
	return syslog.LOG_INFO
}

func (source *Source) Debug(message string) {
	if source.LogLevel >= syslog.LOG_DEBUG {
		source.Logger.Debug(message)
	}
}

func (source *Source) Info(message string) {
	if source.LogLevel >= syslog.LOG_INFO {
		source.Logger.Info(message)
	}
}

func (source *Source) Notice(message string) {
	if source.LogLevel >= syslog.LOG_NOTICE {
		source.Logger.Notice(message)
	}
}

// Warning
func (source *Source) Warning(message string) {
	if source.LogLevel >= syslog.LOG_WARNING {
		source.Logger.Warning(message)
	}
}

func (source *Source) Err(message string) {
	if source.LogLevel >= syslog.LOG_ERR {
		source.Logger.Err(message)
	}
}

func (source *Source) Crit(message string) {
	if source.LogLevel >= syslog.LOG_CRIT {
		source.Logger.Crit(message)
	}
}

func (source *Source) Alert(message string) {
	if source.LogLevel >= syslog.LOG_ALERT {
		source.Logger.Alert(message)
	}
}

func (source *Source) Emerg(message string) {
	if source.LogLevel >= syslog.LOG_EMERG {
		source.Logger.Emerg(message)
	}
}

func (source *Source) Debugf(format string, args ...interface{}) {
	source.Debug(fmt.Sprintf(format, args...))
}

func (source *Source) Infof(format string, args ...interface{}) {
	source.Info(fmt.Sprintf(format, args...))
}

func (source *Source) Noticef(format string, args ...interface{}) {
	source.Notice(fmt.Sprintf(format, args...))
}

func (source *Source) Warningf(format string, args ...interface{}) {
	source.Warning(fmt.Sprintf(format, args...))
}

func (source *Source) Errf(format string, args ...interface{}) {
	source.Err(fmt.Sprintf(format, args...))
}

func (source *Source) Critf(format string, args ...interface{}) {
	source.Crit(fmt.Sprintf(format, args...))
}

func (source *Source) Alertf(format string, args ...interface{}) {
	source.Alert(fmt.Sprintf(format, args...))
}

func (source *Source) Emergf(format string, args ...interface{}) {
	source.Emerg(fmt.Sprintf(format, args...))
}
