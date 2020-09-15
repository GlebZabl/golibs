package loki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golibs/logging"
	"golibs/models"
)

func NewLoki(conf Config) *loki {
	client := http.DefaultClient
	client.Timeout = time.Second * time.Duration(conf.TimeOutSec)

	chanel := make(chan []logging.LogField, 100)
	lk := loki{
		url:           conf.Url,
		containerName: conf.ContainerName,
		client:        http.Client{},
		chanel:        chanel,
	}

	lk.start()

	return &lk
}

type loki struct {
	url           string
	containerName string
	client        http.Client
	chanel        chan []logging.LogField
}

func (l *loki) Print(_ string, fields []logging.LogField) {
	go func(chanel chan []logging.LogField, fields []logging.LogField) {
		chanel <- fields
	}(l.chanel, fields)
}

func (l *loki) start() {
	go func(chanel chan []logging.LogField) {
		for logs := range chanel {
			timeout := 1
			for {
				err := l.send(logs)
				if err != nil {
					time.Sleep(time.Duration(timeout) * time.Second)
					if timeout < 60 {
						timeout = timeout * 2
					}
					continue
				}

				break
			}
		}
	}(l.chanel)
}

func (l *loki) send(fields []logging.LogField) (err error) {
	logFields := make(map[string]interface{}, len(fields))
	var level, ts, message, method, path, statusCode, responseStatus, latency, requestId, errorMsg, responseError, requesterAddr string
	for _, field := range fields {
		switch field.Name {
		case logging.LogLvlFieldKey:
			level = strings.ToLower(strings.Replace(field.Value.(string), `"`, `'`, -1))
		case logging.MessageFieldKey:
			message = strings.Replace(field.Value.(string), `"`, `'`, -1)
		case logging.MethodFieldKey:
			method = strings.Replace(field.Value.(string), `"`, `'`, -1)
		case logging.PathLogKey:
			path = strings.Replace(field.Value.(string), `"`, `'`, -1)
		case logging.StatusCodeFieldKey:
			statusCode = strings.Replace(field.Value.(string), `"`, `'`, -1)
		case logging.ResponseFieldKey:
			if response, ok := field.Value.(models.Response); ok {
				responseStatus = strings.Replace(response.Status, `"`, `'`, -1)
				responseError = strings.Replace(response.ErrorCode, `"`, `'`, -1)
			}
		case logging.LatencyFieldKey:
			latency = strconv.FormatFloat(field.Value.(float64), 'f', -1, 64)
		case logging.RequestIdFieldKey:
			requestId = strings.Replace(field.Value.(string), `"`, `'`, -1)
		case logging.ErrorFieldKey:
			errorMsg = strings.Replace(field.Value.(string), `"`, `'`, -1)
		case logging.RemoteAddressFieldKey:
			requesterAddr = strings.Replace(field.Value.(string), `"`, `'`, -1)
		case logging.TimeFieldKey:
			ts = strings.Replace(field.Value.(string), `"`, `'`, -1)
		}

		logFields[field.Name] = field.Value
	}

	labels := map[string]string{
		"container_name":              l.containerName,
		logging.LogLvlFieldKey:        level,
		logging.MessageFieldKey:       message,
		logging.MethodFieldKey:        method,
		logging.PathLogKey:            path,
		logging.StatusCodeFieldKey:    statusCode,
		logging.ResponseFieldKey:      responseStatus,
		logging.LatencyFieldKey:       latency,
		logging.RequestIdFieldKey:     requestId,
		logging.ErrorFieldKey:         errorMsg,
		logging.ResponseErrorFieldKey: responseError,
		logging.RemoteAddressFieldKey: requesterAddr,
		logging.TimeFieldKey:          ts,
	}

	jsonFields, err := json.Marshal(logFields)
	if err != nil {
		fmt.Println(fmt.Sprintf(
			`{"error":"%s","message":"error during marshaling log fields for loki request","time":"%s"}`,
			err.Error(), time.Now().UTC().Format(time.RFC3339)),
		)
		err = nil
		return
	}

	data := dto{
		Streams: []stream{
			{
				Stream: labels,
				Values: [][]string{
					{
						strconv.FormatInt(time.Now().UnixNano(), 10),
						string(jsonFields),
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println(fmt.Sprintf(
			`{"error":"%s","message":"error during marshaling loki request","time":"%s"}`,
			err.Error(), time.Now().UTC().Format(time.RFC3339)),
		)
		err = nil
		return
	}

	request, err := http.NewRequest(http.MethodPost, l.url, bytes.NewReader(jsonData))
	if err != nil {
		fmt.Println(fmt.Sprintf(
			`{"error":"%s","message":"error during building loki request","time":"%s"}`,
			err.Error(), time.Now().UTC().Format(time.RFC3339)),
		)
		err = nil
		return
	}

	request.Header.Set("content-type", "application/json")

	response, err := l.client.Do(request)
	if err != nil {
		fmt.Println(fmt.Sprintf(
			`{"error":"%s","message":"error during loki request","time":"%s"}`,
			err.Error(), time.Now().UTC().Format(time.RFC3339)),
		)
		return
	}

	if response.StatusCode != 204 {
		fmt.Println(fmt.Sprintf(
			`{"error":"http status error during sending logs to loki, status: %d","time":"%s"`,
			response.StatusCode, time.Now().UTC().Format(time.RFC3339)),
		)
		return
	}

	return
}

type dto struct {
	Streams []stream `json:"streams"`
}

type stream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}
