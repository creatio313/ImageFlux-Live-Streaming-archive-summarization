package server

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	recommendedWebRTCHLSDurationSeconds = 2
	recommendedRTMPHLSDurationSeconds   = 2
	recommendedHLSStartTimeOffset       = -2
	defaultAudioBPS                     = 96000
	defaultVideoCodec                   = "h264_high"
)

var availableAudioBPS = []int{32000, 40000, 48000, 56000, 64000, 80000, 96000, 112000, 128000, 160000, 192000, 224000, 256000, 320000}

var allowedVideoCodecs = map[string]struct{}{
	"h264_high":     {},
	"h264_main":     {},
	"h264_baseline": {},
}

type CreateWebRTCChannelRequest struct {
	AuthWebhookURL  string `json:"auth_webhook_url,omitempty"`
	EventWebhookURL string `json:"event_webhook_url,omitempty"`
	Environment     *int   `json:"environment,omitempty"`
}

type CreateWebRTCChannelWithHLSRequest struct {
	HLS             []CreateHLSInput `json:"hls"`
	EncryptKeyURI   string           `json:"encrypt_key_uri,omitempty"`
	AuthWebhookURL  string           `json:"auth_webhook_url,omitempty"`
	EventWebhookURL string           `json:"event_webhook_url,omitempty"`
	Environment     *int             `json:"environment,omitempty"`
}

type CreateRTMPChannelRequest struct {
	HLS             []CreateHLSInput `json:"hls"`
	EncryptKeyURI   string           `json:"encrypt_key_uri,omitempty"`
	AuthWebhookURL  string           `json:"auth_webhook_url,omitempty"`
	EventWebhookURL string           `json:"event_webhook_url,omitempty"`
}

type CreateHLSInput struct {
	Video   *CreateHLSVideo   `json:"video,omitempty"`
	Audio   *CreateHLSAudio   `json:"audio,omitempty"`
	Archive *CreateHLSArchive `json:"archive,omitempty"`
}

type CreateHLSVideo struct {
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	FPS    int    `json:"fps,omitempty"`
	BPS    int    `json:"bps,omitempty"`
	Codec  string `json:"codec,omitempty"`
}

type CreateHLSAudio struct {
	BPS int `json:"bps,omitempty"`
}

type CreateHLSArchive struct {
	ArchiveDestinationID string `json:"archive_destination_id"`
}
//推奨値等をロジック側で追加するものについては、リクエストの型定義と分ける。
type createWebRTCChannelWithHLSPayload struct {
	HLS             []createHLSPayload `json:"hls"`
	EncryptKeyURI   string             `json:"encrypt_key_uri,omitempty"`
	AuthWebhookURL  string             `json:"auth_webhook_url,omitempty"`
	EventWebhookURL string             `json:"event_webhook_url,omitempty"`
	Environment     *int               `json:"environment,omitempty"`
}

type createRTMPChannelPayload struct {
	HLS             []createHLSPayload `json:"hls"`
	EncryptKeyURI   string             `json:"encrypt_key_uri,omitempty"`
	AuthWebhookURL  string             `json:"auth_webhook_url,omitempty"`
	EventWebhookURL string             `json:"event_webhook_url,omitempty"`
}

type createHLSPayload struct {
	DurationSeconds int               `json:"durationSeconds"`
	StartTimeOffset int               `json:"startTimeOffset"`
	Video           *CreateHLSVideo   `json:"video,omitempty"`
	Audio           *CreateHLSAudio   `json:"audio,omitempty"`
	Archive         *CreateHLSArchive `json:"archive,omitempty"`
}

type ListPlaylistURLsRequest struct {
	ChannelID string `json:"channel_id"`
}

type HLSPlaylist struct {
	ConnectionID string `json:"connection_id"`
	PlaylistURL  string `json:"playlist_url"`
}

//拡張性のため、応答値の型定義を持つ。現時点ではロジックに影響なし。
type CreateWebRTCChannelResponse struct {
	ChannelID string `json:"channel_id"`
	SoraURL   string `json:"sora_url"`
}

type CreateRTMPChannelResponse struct {
	ChannelID string `json:"channel_id"`
	IngestURL string `json:"ingest_url"`
	PlaylistURL string `json:"playlist_url"`
}

type ListPlaylistURLsResponse struct {
	ChannelID string        `json:"channel_id"`
	HLS       []HLSPlaylist `json:"hls"`
}

type ListChannelIDsResponse struct {
	ChannelIDs []string `json:"channel_ids"`
}

type DeleteChannelRequest struct {
	ChannelID string `json:"channel_id"`
}

func (s *Server) createWebRTCChannelWithHLS(w http.ResponseWriter, r *http.Request) {
	var req CreateWebRTCChannelWithHLSRequest
	if !s.decodeJSONBody(w, r, &req) {
		return
	}
	if len(req.HLS) == 0 {
		s.writeError(w, http.StatusBadRequest, "HLS設定は必須項目です。")
		return
	}

	payload := createWebRTCChannelWithHLSPayload{
		HLS:             make([]createHLSPayload, 0, len(req.HLS)),
		EncryptKeyURI:   req.EncryptKeyURI,
		AuthWebhookURL:  req.AuthWebhookURL,
		EventWebhookURL: req.EventWebhookURL,
		Environment:     req.Environment,
	}

	normalizedHLS, invalidIndex, err := normalizeHLSInputs(req.HLS, recommendedWebRTCHLSDurationSeconds)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("hls[%d].指定されたビデオコーデックは無効です。", invalidIndex))
		return
	}
	payload.HLS = append(payload.HLS, normalizedHLS...)

	if err := s.callImageFluxAPI(w, r, imagefluxCreateWebRTCChannelWithHLSTarget, payload); err != nil {
		s.writeImageFluxError(w, err)
	}
}

func (s *Server) createWebRTCChannel(w http.ResponseWriter, r *http.Request) {
	var req CreateWebRTCChannelRequest
	if !s.decodeJSONBody(w, r, &req) {
		return
	}

	if err := s.callImageFluxAPI(w, r, imagefluxCreateWebRTCChannelTarget, req); err != nil {
		s.writeImageFluxError(w, err)
	}
}

func (s *Server) createRTMPChannel(w http.ResponseWriter, r *http.Request) {
	var req CreateRTMPChannelRequest
	if !s.decodeJSONBody(w, r, &req) {
		return
	}
	if len(req.HLS) == 0 {
		s.writeError(w, http.StatusBadRequest, "HLS設定は必須項目です。")
		return
	}

	payload := createRTMPChannelPayload{
		HLS:             make([]createHLSPayload, 0, len(req.HLS)),
		EncryptKeyURI:   req.EncryptKeyURI,
		AuthWebhookURL:  req.AuthWebhookURL,
		EventWebhookURL: req.EventWebhookURL,
	}
	normalizedHLS, invalidIndex, err := normalizeHLSInputs(req.HLS, recommendedRTMPHLSDurationSeconds)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("hls[%d].指定されたビデオコーデックは無効です。", invalidIndex))
		return
	}
	payload.HLS = append(payload.HLS, normalizedHLS...)

	if err := s.callImageFluxAPI(w, r, imagefluxCreateRTMPChannelTarget, payload); err != nil {
		s.writeImageFluxError(w, err)
	}
}

func (s *Server) listPlaylistURLs(w http.ResponseWriter, r *http.Request) {
	var req ListPlaylistURLsRequest
	if !s.decodeJSONBody(w, r, &req) {
		return
	}
	if !s.requireNonEmptyString(w, req.ChannelID, "channel_id") {
		return
	}

	if err := s.callImageFluxAPI(w, r, imagefluxListPlaylistURLsTarget, req); err != nil {
		s.writeImageFluxError(w, err)
	}
}

func (s *Server) listChannels(w http.ResponseWriter, r *http.Request) {
	if err := s.callImageFluxAPI(w, r, imagefluxListChannelIDsTarget, map[string]any{}); err != nil {
		s.writeImageFluxError(w, err)
	}
}

func (s *Server) deleteChannel(w http.ResponseWriter, r *http.Request) {
	var req DeleteChannelRequest
	if !s.decodeJSONBody(w, r, &req) {
		return
	}
	if !s.requireNonEmptyString(w, req.ChannelID, "channel_id") {
		return
	}

	if err := s.callImageFluxAPI(w, r, imagefluxDeleteChannelTarget, req); err != nil {
		s.writeImageFluxError(w, err)
	}
}

func normalizeHLSInputs(inputs []CreateHLSInput, durationSeconds int) ([]createHLSPayload, int, error) {
	payload := make([]createHLSPayload, 0, len(inputs))
	for i, hls := range inputs {
		video, err := normalizeHLSVideo(hls.Video)
		if err != nil {
			return nil, i, err
		}
		audio := normalizeHLSAudio(hls.Audio)

		payload = append(payload, createHLSPayload{
			DurationSeconds: durationSeconds,
			StartTimeOffset: recommendedHLSStartTimeOffset,
			Video:           video,
			Audio:           audio,
			Archive:         hls.Archive,
		})
	}

	return payload, -1, nil
}

func normalizeHLSVideo(video *CreateHLSVideo) (*CreateHLSVideo, error) {
	if video == nil {
		return nil, nil
	}

	normalized := *video
	codec := strings.TrimSpace(normalized.Codec)
	if codec == "" {
		normalized.Codec = defaultVideoCodec
		return &normalized, nil
	}
	if _, ok := allowedVideoCodecs[codec]; !ok {
		return nil, fmt.Errorf("非対応のコーデックが指定されました。")
	}
	normalized.Codec = codec
	return &normalized, nil
}

func normalizeHLSAudio(audio *CreateHLSAudio) *CreateHLSAudio {
	if audio == nil {
		return nil
	}

	normalized := *audio
	if normalized.BPS == 0 {
		normalized.BPS = defaultAudioBPS
		return &normalized
	}
	normalized.BPS = normalizeAudioBitrate(normalized.BPS)
	return &normalized
}

func normalizeAudioBitrate(requested int) int {
	if requested <= availableAudioBPS[0] {
		return availableAudioBPS[0]
	}

	selected := availableAudioBPS[0]
	for _, bps := range availableAudioBPS {
		if requested < bps {
			return selected
		}
		selected = bps
	}
	return selected
}
