package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

//設定ファイルを読み、返すだけ。
func readConfigFromFile(configPath string) (appConfig, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return appConfig{}, fmt.Errorf("設定ファイルの読み取りに失敗しました: path=%q err=%w", configPath, err)
	}

	var cfg appConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return appConfig{}, fmt.Errorf("設定ファイルのJSON解析に失敗しました: path=%q err=%w", configPath, err)
	}

	return cfg, nil
}
//設定ファイルからサーバの初期化を行う。必要な値が不足している場合はエラーを返す。
func newArchiveHandlingServerFromConfig(cfg appConfig) (*archiveHandlingServer, error) {
	//サービスプリンシパルキー関連
	kid := strings.TrimSpace(cfg.ServicePrincipal.KeyKID)
	servicePrincipalID := strings.TrimSpace(cfg.ServicePrincipal.ResourceID)
	privateKeyPEM, err := loadPrivateKeyPEMFromFile(cfg.ServicePrincipal.PrivateKeyPEMPath)
	if err != nil {
		return nil, err
	}
	//シンプルMQのキュー
	simpleMQAPIKey := strings.TrimSpace(cfg.SimpleMQ.APIKey)
	queueName := strings.TrimSpace(cfg.SimpleMQ.QueueName)
	//さくらのウェブアクセラレータのドメイン
	archiveHTTPDomain := strings.TrimSpace(cfg.Archive.HTTPDomain)
	//AIエンジンの設定
	aiEngineAPIKey := strings.TrimSpace(cfg.AIEngine.APIKey)
	transcriptModel := strings.TrimSpace(cfg.AIEngine.TranscriptionModel)
	if transcriptModel == "" {
		transcriptModel = defaultTranscriptionModel
	}
	summaryModel := strings.TrimSpace(cfg.AIEngine.SummaryModel)
	if summaryModel == "" {
		summaryModel = defaultSummaryModel
	}
	//オブジェクトストレージの各種認証情報
	objectStorageEndpoint := strings.TrimSpace(cfg.ObjectStorage.Endpoint)
	objectStorageAccessKey := strings.TrimSpace(cfg.ObjectStorage.AccessKey)
	objectStorageSecretKey := strings.TrimSpace(cfg.ObjectStorage.SecretKey)
	objectStorageBucket := strings.TrimSpace(cfg.ObjectStorage.Bucket)
	objectStorageRegion := strings.TrimSpace(cfg.ObjectStorage.Region)
	notificationGroupID := strings.TrimSpace(cfg.Notification.GroupID)
	if objectStorageRegion == "" {
		objectStorageRegion = "jp-north-1"
	}

	if kid == "" {
		return nil, fmt.Errorf("service_principal.key_kidが設定されていません")
	}
	if servicePrincipalID == "" {
		return nil, fmt.Errorf("service_principal.resource_idが設定されていません")
	}
	if simpleMQAPIKey == "" {
		return nil, fmt.Errorf("simplemq.api_keyが設定されていません")
	}
	if queueName == "" {
		return nil, fmt.Errorf("simplemq.queue_nameが設定されていません")
	}
	if archiveHTTPDomain == "" {
		return nil, fmt.Errorf("archive.http_domainが設定されていません")
	}
	if aiEngineAPIKey == "" {
		return nil, fmt.Errorf("ai_engine.api_keyが設定されていません")
	}
	if objectStorageEndpoint == "" {
		return nil, fmt.Errorf("object_storage.endpointが設定されていません")
	}
	if objectStorageAccessKey == "" {
		return nil, fmt.Errorf("object_storage.access_keyが設定されていません")
	}
	if objectStorageSecretKey == "" {
		return nil, fmt.Errorf("object_storage.secret_keyが設定されていません")
	}
	if objectStorageBucket == "" {
		return nil, fmt.Errorf("object_storage.bucketが設定されていません")
	}
	if notificationGroupID == "" {
		return nil, fmt.Errorf("notification.group_idが設定されていません")
	}

	//オブジェクトストレージクライアントの初期化
	objectStorageClient, err := minio.New(objectStorageEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(objectStorageAccessKey, objectStorageSecretKey, ""),
		Secure: true,
		Region: objectStorageRegion,
	})
	if err != nil {
		return nil, fmt.Errorf("オブジェクトストレージクライアントの初期化に失敗しました: %w", err)
	}

	return &archiveHandlingServer{
		kid:                 kid,
		servicePrincipalID:  servicePrincipalID,
		privateKeyPEM:       privateKeyPEM,
		simpleMQAPIKey:      simpleMQAPIKey,
		queueName:           queueName,
		archiveHTTPDomain:   archiveHTTPDomain,
		aiEngineBaseURL:     aiEngineBaseURL,
		aiEngineAPIKey:      aiEngineAPIKey,
		transcriptModel:     transcriptModel,
		summaryModel:        summaryModel,
		notificationGroupID: notificationGroupID,
		objectStorageClient: objectStorageClient,
		objectStorageBucket: objectStorageBucket,
		httpClient:          &http.Client{},
	}, nil
}
//サービスプリンシパルキーを使うため、秘密鍵の中身を読み込む。
func loadPrivateKeyPEMFromFile(privateKeyPath string) (string, error) {
	normalizedPath := strings.TrimSpace(privateKeyPath)
	if normalizedPath == "" {
		normalizedPath = defaultPrivateKeyPEMPath
	}

	pemBytes, err := os.ReadFile(normalizedPath)
	if err != nil {
		return "", fmt.Errorf("秘密鍵PEMファイルの読み取りに失敗しました: path=%q err=%w", normalizedPath, err)
	}

	if len(strings.TrimSpace(string(pemBytes))) == 0 {
		return "", fmt.Errorf("秘密鍵PEMファイルが空です: path=%q", normalizedPath)
	}

	return string(pemBytes), nil
}

