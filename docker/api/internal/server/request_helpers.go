package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

const maxJSONBodyBytes int64 = 1 << 20

func (s *Server) decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)).Decode(dst); err != nil {
		s.writeError(w, http.StatusBadRequest, "無効なJSON内容です。")
		return false
	}
	return true
}

func (s *Server) requireNonEmptyString(w http.ResponseWriter, value string, field string) bool {
	if strings.TrimSpace(value) == "" {
		s.writeError(w, http.StatusBadRequest, field+"は必須項目です。")
		return false
	}
	return true
}

func (s *Server) requireBearerToken(w http.ResponseWriter, r *http.Request) (string, bool) {
	token := extractBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		s.writeError(w, http.StatusUnauthorized, "APIトークンは必須項目です。")
		return "", false
	}
	return token, true
}
