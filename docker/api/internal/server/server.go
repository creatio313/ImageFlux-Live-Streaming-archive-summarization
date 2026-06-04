package server

import (
	"net/http"
)

type Server struct {
	client         *http.Client
	allowedOrigins []string
	mux            *http.ServeMux
}

func New(client *http.Client, allowedOrigins []string) *Server {
	if client == nil {
		client = &http.Client{}
	}

	s := &Server{
		client:         client,
		allowedOrigins: allowedOrigins,
		mux:            http.NewServeMux(),
	}

	s.routes()
	return s
}

func (s *Server) Router() http.Handler {
	return s.withCORS(s.mux)
}
