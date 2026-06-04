package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const imagefluxLiveStreamingAPIBaseURL = "https://live-api.imageflux.jp/"

const (
	//チャンネル作成
	imagefluxCreateWebRTCChannelWithHLSTarget = "ImageFlux_20200316.CreateMultistreamChannelWithHLS"
	imagefluxCreateWebRTCChannelTarget        = "ImageFlux_20200316.CreateMultistreamChannel"
	imagefluxCreateRTMPChannelTarget          = "ImageFlux_20250901.CreateRTMPChannel"
	//HLSプレイリスト取得
	imagefluxListPlaylistURLsTarget           = "ImageFlux_20200207.ListPlaylistURLs"
	//チャンネル管理
	imagefluxDeleteChannelTarget              = "ImageFlux_20180501.DeleteChannel"
	imagefluxListChannelIDsTarget             = "ImageFlux_20200729.ListChannelIDs"
	//アーカイブ管理
	imagefluxListArchiveDestinationsTarget    = "ImageFlux_20190205.ListArchiveDestinations"
	imagefluxCreateArchiveDestinationTarget   = "ImageFlux_20190205.CreateArchiveDestination"
	imagefluxDeleteArchiveDestinationTarget   = "ImageFlux_20190205.DeleteArchiveDestination"
)

func (s *Server) callImageFluxAPI(w http.ResponseWriter, r *http.Request, target string, payload any) error {
	//ImageFlux APIへのリクエストを組み立てて送信する。
	accessToken, ok := s.requireBearerToken(w, r)
	if !ok {
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	downstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, imagefluxLiveStreamingAPIBaseURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}

	downstreamReq.Header.Set("Content-Type", "application/json")
	downstreamReq.Header.Set("X-Sora-Target", target)
	downstreamReq.Header.Set("Authorization", "Bearer "+accessToken)

	//ImageFlux APIからのレスポンスをそのままクライアントに転送する。
	downstreamResp, err := s.client.Do(downstreamReq)
	if err != nil {
		return err
	}
	defer downstreamResp.Body.Close()

	for key, values := range downstreamResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	if contentType := downstreamResp.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.WriteHeader(downstreamResp.StatusCode)
	_, err = io.Copy(w, downstreamResp.Body)
	return err
}

func extractBearerToken(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) < 7 || !strings.EqualFold(trimmed[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(trimmed[7:])
}
