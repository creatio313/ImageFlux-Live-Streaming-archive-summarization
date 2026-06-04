package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) routes() {
	/***
	各API仕様はhttps://console.imageflux.jp/docs/live/apiを参照。
	***/
	// CORSのプリフライトリクエストに対するハンドラー。業務ロジックには関係しない。
	s.mux.HandleFunc("OPTIONS /api/imageflux/channels", s.preflight)
	s.mux.HandleFunc("OPTIONS /api/imageflux/channels/webrtc", s.preflight)
	s.mux.HandleFunc("OPTIONS /api/imageflux/channels/webrtc-to-hls", s.preflight)
	s.mux.HandleFunc("OPTIONS /api/imageflux/channels/rtmp-to-hls", s.preflight)
	s.mux.HandleFunc("OPTIONS /api/imageflux/channels/playlist-urls", s.preflight)
	s.mux.HandleFunc("OPTIONS /api/imageflux/archive-destinations", s.preflight)

	//死活確認用のエンドポイント
	s.mux.HandleFunc("GET /health", s.health)

	//WebRTC to HLS
	s.mux.HandleFunc("POST /api/imageflux/channels/webrtc-to-hls", s.createWebRTCChannelWithHLS)
	//WebRTC
	s.mux.HandleFunc("POST /api/imageflux/channels/webrtc", s.createWebRTCChannel)
	//RTMP to HLS
	s.mux.HandleFunc("POST /api/imageflux/channels/rtmp-to-hls", s.createRTMPChannel)

	//HLSプレイリスト取得
	s.mux.HandleFunc("POST /api/imageflux/channels/playlist-urls", s.listPlaylistURLs)

	//チャンネル管理
	s.mux.HandleFunc("GET /api/imageflux/channels", s.listChannels)
	s.mux.HandleFunc("DELETE /api/imageflux/channels", s.deleteChannel)

	//アーカイブ管理
	s.mux.HandleFunc("GET /api/imageflux/archive-destinations", s.listArchiveDestinations)
	s.mux.HandleFunc("POST /api/imageflux/archive-destinations", s.createArchiveDestination)
	s.mux.HandleFunc("DELETE /api/imageflux/archive-destinations", s.deleteArchiveDestination)
	s.mux.HandleFunc("/", s.notFound)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) preflight(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if !s.isOriginAllowed(origin) {
		s.writeError(w, http.StatusForbidden, "このドメインからの要求は許可されていません。")
		return
	}

	addCORSHeaders(w.Header(), origin)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) notFound(w http.ResponseWriter, _ *http.Request) {
	s.writeError(w, http.StatusNotFound, "リソースが見つかりません。")
}
