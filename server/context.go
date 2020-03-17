package server

import (
	"encoding/json"
	"errors"
	"net/http"
)

const (
	ContentTypeHeader          = "Content-Type"
	ContentTypeApplicationJson = "application/json"
)

type Context struct {
	isAborted bool

	data    map[string]interface{}
	Request *http.Request
	Writer  *responseWriter

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

func (c *Context) Abort(status int) {
	c.Writer.WriteHeader(status)
	c.isAborted = true
	return
}

func (c *Context) IsAborted() bool {
	return c.isAborted
}

func (c *Context) Get(key string) (value interface{}, exists bool) {
	value, exists = c.data[key]
	return
}

func (c *Context) Set(key string, value interface{}) {
	c.data[key] = value
	return
}

func (c *Context) SendJson(data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		c.Writer.WriteHeader(http.StatusInternalServerError)
		c.Writer.status = http.StatusInternalServerError
		return
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	_, err = c.Writer.Write(jsonData)
	c.Writer.status = http.StatusOK

	if err != nil {
		panic(err)
	}
}

func (c *Context) BindJson(dest interface{}) (err error) {
	if c.Request != nil && c.Request.Body != nil {
		if !(c.Request.Header.Get(ContentTypeHeader) == ContentTypeApplicationJson) {
			err = errors.New("wrong content-type header")
		}

		err = json.NewDecoder(c.Request.Body).Decode(dest)
		if err != nil {
			return
		}
	}

	return
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (r *responseWriter) Status() int {
	return r.status
}
