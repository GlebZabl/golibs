package logging

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golibs/errors"
)

const (
	debugLogLevel   = "DEBUG"
	warningLogLevel = "WARNING"
	infoLogLevel    = "INFO"
	errorLogLevel   = "ERROR"
	panicLogLevel   = "PANIC"

	LogLvlFieldKey        = "level"
	MessageFieldKey       = "message"
	TimeFieldKey          = "time"
	ErrorFieldKey         = "error"
	RequestIdFieldKey     = "request_id"
	PathLogKey            = "path"
	StatusCodeFieldKey    = "status_code"
	LatencyFieldKey       = "latency"
	MethodFieldKey        = "method"
	RequestFieldKey       = "request"
	ResponseFieldKey      = "response"
	ResponseErrorFieldKey = "response_error"
	RemoteAddressFieldKey = "remote_address"
	RequestUserUidKey     = "request_user_uid"

	jsonLogFormat  = "JSON"
	debugLogFormat = "DEBUG"
)

var defaultLogFields = map[string]bool{
	LogLvlFieldKey:        true,
	MessageFieldKey:       true,
	TimeFieldKey:          true,
	ErrorFieldKey:         true,
	RequestIdFieldKey:     true,
	PathLogKey:            true,
	StatusCodeFieldKey:    true,
	LatencyFieldKey:       true,
	MethodFieldKey:        true,
	RequestFieldKey:       true,
	ResponseFieldKey:      true,
	ResponseErrorFieldKey: true,
	RemoteAddressFieldKey: true,
	RequestUserUidKey:     true,
}

var levels = map[string]int{
	debugLogLevel:   0,
	infoLogLevel:    1,
	warningLogLevel: 2,
	errorLogLevel:   3,
	panicLogLevel:   4,
}

func NewLogger(config Config, printers []Printer) (logger Logger, err error) {

	var format string
	if config.Debug {
		format = debugLogFormat
	} else {
		format = jsonLogFormat
	}

	logger = buildLogger(printers, format, strings.ToUpper(config.LogLevel)).
		WithFields(map[string]interface{}{
			"env_name": config.EnvName,
			"branch":   config.Branch,
			"commit":   config.Commit,
		})

	return
}

func NewTestLogger(printers ...Printer) Logger {
	return &logger{
		printers:   printers,
		fields:     nil,
		level:      errorLogLevel,
		format:     debugLogFormat,
		errorHooks: nil,
	}
}

type Logger interface {
	Debug(msg string)
	DebugF(format string, args ...interface{})
	Info(msg string)
	InfoF(format string, args ...interface{})
	Warn(msg string)
	WarnF(format string, args ...interface{})
	Error(err error)
	ErrorF(err error, format string, args ...interface{})
	Panic(msg string)
	PanicF(format string, args ...interface{})

	AddPrinter(printer Printer)
	RegErrorHook(action errorHook)
	Replicate() Logger
	WithField(name string, value interface{}) Logger
	WithFields(map[string]interface{}) Logger
}

func buildLogger(printers []Printer, format, level string) *logger {
	return &logger{
		printers: printers,
		level:    level,
		format:   format,
	}
}

type logger struct {
	printers   []Printer
	fields     []LogField
	level      string
	format     string
	errorHooks []errorHook
}

type LogField struct {
	Name  string
	Value interface{}
}

func (l logger) Replicate() Logger {
	return &l
}

func (l *logger) RegErrorHook(action errorHook) {
	l.errorHooks = append(l.errorHooks, action)
}

func (l logger) WithField(name string, value interface{}) Logger {
	fieldsCopy := make([]LogField, len(l.fields))
	copy(fieldsCopy, l.fields)
	l.fields = fieldsCopy

	name = l.getCorrectFieldName(name, value)
	for i := range l.fields {
		if l.fields[i].Name == name {
			l.fields[i].Value = value
			return &l
		}
	}
	l.fields = append(l.fields, LogField{Name: name, Value: value})
	return &l
}

func (l logger) WithFields(fields map[string]interface{}) Logger {
	fieldsCopy := make([]LogField, len(l.fields))
	copy(fieldsCopy, l.fields)
	l.fields = fieldsCopy
	for name, value := range fields {
		found := false
		for i := range l.fields {
			if l.fields[i].Name == name {
				l.fields[i].Value = value
				found = true
				break
			}
		}

		if !found {
			l.fields = append(l.fields, LogField{Name: name, Value: value})
		}
	}

	return &l
}

func (l *logger) AddPrinter(printer Printer) {
	l.printers = append(l.printers, printer)
}

func (l *logger) Debug(msg string) {
	if levels[l.level] > levels[debugLogLevel] {
		return
	}

	log, fields := l.createLog(debugLogLevel, msg, l.fields)
	l.print(log, fields)
}

func (l *logger) DebugF(format string, args ...interface{}) {
	l.Debug(fmt.Sprintf(format, args...))
}

func (l *logger) Info(msg string) {
	if levels[l.level] > levels[infoLogLevel] {
		return
	}

	log, fields := l.createLog(infoLogLevel, msg, l.fields)
	l.print(log, fields)
}

func (l *logger) InfoF(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

func (l *logger) Warn(msg string) {
	if levels[l.level] > levels[warningLogLevel] {
		return
	}

	log, fields := l.createLog(warningLogLevel, msg, l.fields)
	l.print(log, fields)
}

func (l *logger) WarnF(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

func (l *logger) Error(err error) {
	if levels[l.level] > levels[errorLogLevel] {
		return
	}

	log, fields := l.createLog(errorLogLevel, "Error occurred!",
		append(
			l.fields,
			LogField{
				Name:  ErrorFieldKey,
				Value: err.Error(),
			},
		),
	)

	l.print(log, fields)
	if !errors.IsType(err, errors.DoesNotExistErrorType, errors.AlreadyExistErrorType, errors.InconsistentErrorType,
		errors.ValidationErrorType, errors.ForbiddenErrorType) {
		l.triggerErrorHooks("Error occurred!", err.Error())
	}
}

func (l *logger) ErrorF(err error, format string, args ...interface{}) {
	if levels[l.level] > levels[errorLogLevel] {
		return
	}

	log, fields := l.createLog(errorLogLevel, fmt.Sprintf(format, args...),
		append(
			l.fields,
			LogField{
				Name:  ErrorFieldKey,
				Value: err.Error(),
			},
		),
	)

	l.print(log, fields)
	if !errors.IsType(err, errors.DoesNotExistErrorType, errors.AlreadyExistErrorType, errors.InconsistentErrorType,
		errors.ValidationErrorType, errors.ForbiddenErrorType) {
		l.triggerErrorHooks("Error occurred: "+fmt.Sprintf(format, args...), err.Error())
	}
}

func (l *logger) Panic(msg string) {
	if levels[l.level] > levels[panicLogLevel] {
		return
	}

	log, fields := l.createLog(panicLogLevel, msg, l.fields)

	l.print(log, fields)
	l.triggerErrorHooks("panic: "+msg, "")
}

func (l *logger) PanicF(format string, args ...interface{}) {
	l.Panic(fmt.Sprintf(format, args...))
}

func (l *logger) print(msg string, fields []LogField) {
	for _, printer := range l.printers {
		printer.Print(msg, fields)
	}
}

func (l *logger) createLog(level, message string, fields []LogField) (fullString string, fullFields []LogField) {
	fields = append(fields,
		LogField{
			Name:  LogLvlFieldKey,
			Value: level,
		}, LogField{
			Name:  MessageFieldKey,
			Value: message,
		}, LogField{
			Name:  TimeFieldKey,
			Value: time.Now().UTC().Format(time.RFC3339Nano),
		})

	switch l.format {
	case jsonLogFormat:
		return l.fieldsToJsonFormat(fields), fields
	case debugLogFormat:
		lvl := level
		switch level {
		case debugLogLevel:
			lvl = "\033[4;3m" + level + "\033[m"
		case infoLogLevel:
			lvl = "\033[0;34m" + level + "\033[m"
		case warningLogLevel:
			lvl = "\033[0;33m" + level + "\033[m"
		case errorLogLevel:
			lvl = "\033[0;31m" + level + "\033[m"
		case panicLogLevel:
			lvl = "\033[41m" + level + "\033[40m\033[m"
		}

		return l.fieldsToDebugFormat(fields, lvl, message), fields
	default:
		panic("wrong logs format!")
	}
}

func (l *logger) fieldsToJsonFormat(fields []LogField) string {
	entry := make(map[string]interface{}, len(fields))
	for _, field := range fields {
		entry[field.Name] = field.Value
	}

	jsonLog, err := json.Marshal(entry)
	if err != nil {

	}
	return string(jsonLog)
}

func (l *logger) fieldsToDebugFormat(fields []LogField, level, message string) string {
	return fmt.Sprintf("%s	%s	%s	%s", time.Now().UTC().Format(time.RFC3339), level, message, l.fieldsToJsonFormat(fields))
}

func (l *logger) triggerErrorHooks(msg, err string) {
	var requestId string
	for _, field := range l.fields {
		if field.Name == RequestIdFieldKey {
			requestId = field.Value.(string)
		}
	}

	for _, hook := range l.errorHooks {
		hook(msg, err, requestId)
	}
}

func (l *logger) getCorrectFieldName(key string, value interface{}) string {
	if !defaultLogFields[key] {
		return key
	}

	if _, ok := value.(string); ok {
		return key
	}

	return "custom_" + key
}

type errorHook func(msg, err, requestId string)
