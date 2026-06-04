package main

import (
	"net/http"

	"github.com/minio/minio-go/v7"
)

const (
	// URL系
	servicePrincipalTokenEndpoint = "https://secure.sakura.ad.jp/cloud/api/iam/1.0/service-principals/oauth2/token"
	simpleMQBaseURL               = "https://simplemq.tk1b.api.sacloud.jp"
	aiEngineBaseURL               = "https://api.ai.sakura.ad.jp"
	simpleNotificationEndpointFmt = "https://secure.sakura.ad.jp/cloud/zone/is1a/api/cloud/1.1/commonserviceitem/%s/simplenotification/message"

	//既定設定値
	defaultConfigPath             = "./config.json"
	defaultPrivateKeyPEMPath      = "/app/service-principal-private-key.pem"
	defaultTranscriptionModel     = "whisper-large-v3-turbo"
	defaultSummaryModel           = "llm-jp-3.1-8x13b-instruct"
	maxAudioDurationSeconds       = 25 * 60
	maxAudioSizeBytes             = 30 * 1024 * 1024
)

type appConfig struct {
	ServicePrincipal servicePrincipalConfig `json:"service_principal"`
	SimpleMQ         simpleMQConfig         `json:"simplemq"`
	Archive          archiveConfig          `json:"archive"`
	AIEngine         aiEngineConfig         `json:"ai_engine"`
	ObjectStorage    objectStorageConfig    `json:"object_storage"`
	Notification     notificationConfig     `json:"notification"`
}

type servicePrincipalConfig struct {
	KeyKID            string `json:"key_kid"`
	ResourceID        string `json:"resource_id"`
	PrivateKeyPEMPath string `json:"private_key_pem_path"`
}

type simpleMQConfig struct {
	APIKey             string `json:"api_key"`
	QueueName          string `json:"queue_name"`
	PollInterval       string `json:"poll_interval"`
	ErrorRetryInterval string `json:"error_retry_interval"`
}

type archiveConfig struct {
	HTTPDomain string `json:"http_domain"`
}

type aiEngineConfig struct {
	APIKey             string `json:"api_key"`
	TranscriptionModel string `json:"transcription_model"`
	SummaryModel       string `json:"summary_model"`
}

type objectStorageConfig struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
}

type notificationConfig struct {
	GroupID string `json:"group_id"`
}

type archiveHandlingServer struct {
	kid                 string
	servicePrincipalID  string
	privateKeyPEM       string
	simpleMQAPIKey      string
	queueName           string
	archiveHTTPDomain   string
	aiEngineBaseURL     string
	aiEngineAPIKey      string
	transcriptModel     string
	summaryModel        string
	notificationGroupID string
	objectStorageClient *minio.Client
	objectStorageBucket string
	httpClient          *http.Client
}
//シンプルMQのメッセージ受信応答
type simpleMQReceiveResponse struct {
	Result   string      `json:"result"`
	Messages []mqMessage `json:"messages"`
}
//シンプルMQのメッセージ応答のうちメッセージ部分（配列要素）
type mqMessage struct {
	ID                  string `json:"id"`
	Content             string `json:"content"`
	CreatedAt           int64  `json:"created_at"`
	UpdatedAt           int64  `json:"updated_at"`
	ExpiresAt           int64  `json:"expires_at"`
	AcquiredAt          int64  `json:"acquired_at"`
	VisibilityTimeoutAt int64  `json:"visibility_timeout_at"`
}
//シンプルMQのメッセージ応答のうちメッセージ内容（ImageFluxのWebhookの内容に相当）
type messageContent struct {
	ChannelID string          `json:"channel_id"`
	Type      string          `json:"type"`
	Data      webhookFileInfo `json:"data"`
}
type webhookFileInfo struct {
	DestURI     string `json:"dest_uri"`
	FilePath    string `json:"file_path"`
	Size        int64  `json:"size"`
	FileType    string `json:"file_type"`
	AbsoluteURL string `json:"absolute_url"`
}

type simpleNotificationPayload struct {
	Message string `json:"Message"`
}