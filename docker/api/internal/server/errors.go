package server

import (
	"encoding/json"
	"errors"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message})
}

func (s *Server) writeImageFluxError(w http.ResponseWriter, err error) {
	var requestErr *requestError
	if errors.As(err, &requestErr) {
		s.writeError(w, requestErr.status, requestErr.message)
		return
	}
	s.writeError(w, http.StatusBadGateway, err.Error())
}

type requestError struct {
	status  int
	message string
}

func (err *requestError) Error() string {
	return err.message
}
