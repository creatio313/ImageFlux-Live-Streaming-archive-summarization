package server

import "net/http"

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !s.isOriginAllowed(origin) {
				s.writeError(w, http.StatusForbidden, "このドメインからの要求は許可されていません。")
				return
			}
			addCORSHeaders(w.Header(), origin)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) isOriginAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	for _, allowed := range s.allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

func addCORSHeaders(header http.Header, origin string) {
	header.Set("Access-Control-Allow-Origin", origin)
	header.Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, DELETE")
	header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Sora-Target")
	header.Set("Access-Control-Max-Age", "334")
	header.Add("Vary", "Origin")
}
