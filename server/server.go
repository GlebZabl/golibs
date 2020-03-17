package server

import (
	"fmt"
	"net/http"
	"strings"
)

type HandleFunc func(*Context)

type routNode struct {
	contextKey string
	method     string
	actions    []HandleFunc
}

func New(basePath string) *server {
	return &server{
		routing:  map[string]routNode{},
		basePath: basePath,
	}
}

type server struct {
	basePath    string
	middlewares []HandleFunc
	routing     map[string]routNode
}

func (s *server) Use(middleware HandleFunc) {
	s.middlewares = append(s.middlewares, middleware)
}

func (s *server) Post(path string, handler HandleFunc) {
	s.handle(http.MethodPost, path, handler)
}

func (s *server) Get(path string, handler HandleFunc) {
	s.handle(http.MethodGet, path, handler)
}

func (s *server) Run(port int) (err error) {
	return http.ListenAndServe(fmt.Sprintf(":%d", port), s)
}

func (s *server) handle(method, path string, handler HandleFunc) {
	actions := append([]HandleFunc{}, s.middlewares...)
	pathParts := strings.Split(path, "*")
	if len(pathParts) > 1 {
		for key := range s.routing {
			if strings.Contains(key, pathParts[0]) {
				panic(fmt.Sprintf("path already routed %s", pathParts[0]))
			}
		}
	}

	s.routing[s.basePath+pathParts[0]] = routNode{
		method:  method,
		actions: append(actions, handler),
	}
}

func (s *server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	context := &Context{
		data:    make(map[string]interface{}),
		Request: req,
		Writer: &responseWriter{
			ResponseWriter: res,
			status:         0,
		},
		index: -1,
	}

	node, ok := s.routing[req.URL.Path]
	if !ok {
		for key, action := range s.routing {
			if strings.Contains(req.URL.Path, key) {
				node = action
				context.Set(node.contextKey, strings.TrimLeft(req.URL.Path, key))
			}
		}

	}

	context.actions = node.actions

	context.Next()
}
