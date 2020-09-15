package gray_logs

import (
	"crypto/tls"
	"encoding/json"
	"time"

	"golibs/logging"

	"github.com/Devatoria/go-graylog"
)

func NewGrayLogs(conf Config, l logging.Logger) *grayLogsWriter {
	chanel := make(chan []logging.LogField, 1000)

	result := &grayLogsWriter{
		chanel:  chanel,
		host:    conf.HostName,
		version: conf.Version,
		conf:    conf,
		logger:  l,
	}

	err := result.connectToHost()
	if err != nil {
		l.ErrorF(ConnectionError.Wrap(err), "error during initialize connection to graylog host, will try to reconnect on next logs sending attempt")
	}
	result.start(conf.TimeOut)

	return result
}

type grayLogsWriter struct {
	chanel  chan []logging.LogField
	host    string
	version string
	logger  logging.Logger
	conf    Config
	writer  *graylog.Graylog
}

func (g *grayLogsWriter) connectToHost() (err error) {
	endPoint := graylog.Endpoint{
		Transport: graylog.TCP,
		Address:   g.conf.Addr,
		Port:      g.conf.Port,
	}

	if len(g.conf.Certificates) > 0 {
		tlsConf := &tls.Config{
			Certificates:                g.conf.Certificates,
			InsecureSkipVerify:          true,
			DynamicRecordSizingDisabled: true,
		}

		g.writer, err = graylog.NewGraylogTLS(endPoint, time.Second*time.Duration(g.conf.TimeOut), tlsConf)
	} else {
		g.logger.Warn("Missing graylog tls certificates, using unsafe tcp connection")
		g.writer, err = graylog.NewGraylog(endPoint)
	}
	if err != nil {
		g.writer = nil
		return
	}

	return
}

func (g *grayLogsWriter) start(timeOut int) {
	go func(chanel chan []logging.LogField) {
		for msg := range chanel {
			for {
				err := g.send(msg)
				if err != nil {
					g.logger.Error(err)
					g.writer = nil

					time.Sleep(time.Duration(timeOut) * time.Second)
					if timeOut < 60 {
						timeOut = timeOut * 2
					}
					continue
				}

				break
			}
		}
	}(g.chanel)
}

func (g *grayLogsWriter) send(message []logging.LogField) (err error) {
	for g.writer == nil {
		err = g.connectToHost()
		if err != nil {
			return
		}
	}

	data, msg := g.prepareFields(message)
	err = g.writer.Send(graylog.Message{
		Version:      g.version,
		Host:         g.host,
		FullMessage:  data,
		ShortMessage: msg,
	})

	return
}

func (g *grayLogsWriter) Print(_ string, fields []logging.LogField) {
	go func(chanel chan []logging.LogField, logFields []logging.LogField) {
		chanel <- logFields
	}(g.chanel, fields)
}

func (g *grayLogsWriter) prepareFields(fields []logging.LogField) (jsonData, insideMessage string) {
	entry := make(map[string]interface{}, len(fields)+1)
	for _, field := range fields {
		entry[field.Name] = field.Value
		if field.Name == logging.MessageFieldKey {
			insideMessage = field.Value.(string)
		}
	}
	entry["container_name"] = "backend"

	jsonLog, err := json.Marshal(entry)
	if err != nil {

	}

	jsonData = string(jsonLog)

	if insideMessage == "" {
		insideMessage = "empty message"
	}

	return
}
