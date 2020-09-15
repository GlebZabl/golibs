package server

import (
	"encoding/json"
	"net/http"
)

const (
	ContentTypeHeader          = "Content-Type"
	ContentTypeApplicationJson = "application/json"
)

type Context struct {
	isAborted bool

	request        *http.Request
	responseWriter responseWriter

	extraData    map[string]interface{}
	requestBody  interface{}
	requestId    string
	requesterUid string

	index   int
	actions []HandleFunc
}

func (c *Context) Next() {
	for c.index < len(c.actions)-1 {
		c.index++
		if !c.isAborted {
			c.actions[c.index](c)
		}
	}
}

func (c *Context) SendJson(data interface{}) {
	c.responseWriter.statusCode = http.StatusOK
	c.responseWriter.responseBody = data
}

func (c *Context) SendJsonWithStatus(data interface{}, statusCode int) {
	c.responseWriter.statusCode = statusCode
	c.responseWriter.responseBody = data
}

func (c *Context) AbortWithPayload(data interface{}, statusCode int) {
	c.isAborted = true
	c.responseWriter.statusCode = statusCode
	c.responseWriter.responseBody = data
}

func (c *Context) Abort() {
	c.isAborted = true
}

func (c *Context) AbortWithCode(status int) {
	c.responseWriter.statusCode = status
	c.isAborted = true
	return
}

func (c *Context) Get(key string) (value interface{}, exists bool) {
	value, exists = c.extraData[key]
	return
}

func (c *Context) Set(key string, value interface{}) {
	c.extraData[key] = value
	return
}

func (c *Context) BindJson(dest interface{}) (err error) {
	if c.request != nil && c.request.Body != nil {
		if !(c.request.Header.Get(ContentTypeHeader) == ContentTypeApplicationJson) {
			err = BindingError.New("wrong content-type header")
		}

		err = json.NewDecoder(c.request.Body).Decode(dest)
		if err != nil {
			err = BindingError.Wrap(err)
			return
		}
	}

	return
}

func (c *Context) ResponseWriter() http.ResponseWriter {
	return &c.responseWriter
}

func (c *Context) Request() *http.Request {
	return c.request
}

func (c *Context) StatusCode() int {
	return c.responseWriter.statusCode
}

func (c *Context) ResponseBody() interface{} {
	return c.responseWriter.responseBody
}

func (c *Context) RequestId() string {
	return c.requestId
}

func (c *Context) SetRequestId(requestId string) {
	c.requestId = requestId
	return
}

func (c *Context) RequesterUid() string {
	return c.requesterUid
}

func (c *Context) SetRequesterUid(uid string) {
	c.requesterUid = uid

	return
}

func (c *Context) done() {
	c.responseWriter.done()
	return
}

type responseWriter struct {
	writer        http.ResponseWriter
	statusCode    int
	responseBytes []byte
	responseBody  interface{}
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode

}

func (r *responseWriter) Header() http.Header {
	return r.writer.Header()
}

func (r *responseWriter) Write(data []byte) (int, error) {
	r.responseBytes = append(r.responseBytes, data...)
	return len(data), nil
}

func (r *responseWriter) Status() int {
	return r.statusCode
}

func (r *responseWriter) done() {
	var err error
	var response []byte
	switch {
	case len(r.responseBytes) > 0:
		response = r.responseBytes
	case r.responseBody != nil:
		response, err = json.Marshal(r.responseBody)
		if err != nil {
			r.writer.WriteHeader(http.StatusInternalServerError)
			panic(err)
		}
		if r.writer.Header().Get("Content-Type") == "" {
			r.writer.Header().Set("Content-Type", "application/json")
		}
	}

	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	r.writer.WriteHeader(r.statusCode)

	if response != nil {
		_, err = r.writer.Write(response)
		if err != nil {
			panic(err)
		}
	}
}
