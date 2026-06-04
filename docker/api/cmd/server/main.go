package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

const (
	archiveCreatedType = "imageflux.archive_created"
	targetFileType     = "m3u8"
)

const simpleMQEnqueueEndpointFmt = "https://simplemq.tk1b.api.sacloud.jp/v1/queues/%s/messages"

// アーカイブ作成時のイベントWebhook通知の構造
type WebhookPayload struct {
	ChannelID string      `json:"channel_id"`
	Type      string      `json:"type"`
	Data      WebhookData `json:"data"`
}

// アーカイブ作成時のイベントWebhook通知のうち、dataの子項目
type WebhookData struct {
	DestURI     string `json:"dest_uri"`
	FilePath    string `json:"file_path"`
	Size        int64  `json:"size"`
	FileType    string `json:"file_type"`
	AbsoluteURL string `json:"absolute_url"`
}

// シンプルMQに送信するメッセージの構造
type simpleMQSendRequest struct {
	Content string `json:"content"`
}

// シンプルMQクライアント（サーバ本体）の構造体
type simpleMQClient struct {
	apiKey     string
	queueName  string
	httpClient *http.Client
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// シンプルMQクライアントの初期化。環境変数から必要な情報を読み取る。
	notifier, err := newSimpleMQClientFromEnv()
	if err != nil {
		log.Fatalf("シンプルMQクライアントの初期化に失敗しました: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /", webhookHandler(notifier))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("サーバが%sポートで起動しました。", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("サーバの起動に失敗しました: %v", err)
	}
}
/***
メインロジック
***/
//環境変数を読み込み、サーバを初期化する関数
func newSimpleMQClientFromEnv() (*simpleMQClient, error) {
	apiKey := os.Getenv("SIMPLEMQ_API_KEY")
	queueName := os.Getenv("SIMPLEMQ_QUEUE_NAME")

	if apiKey == "" {
		return nil, fmt.Errorf("環境変数SIMPLEMQ_API_KEYが設定されていません")
	}
	if queueName == "" {
		return nil, fmt.Errorf("環境変数SIMPLEMQ_QUEUE_NAMEが設定されていません")
	}

	return &simpleMQClient{
		apiKey:     apiKey,
		queueName:  queueName,
		httpClient: &http.Client{},
	}, nil
}
//Webhook通知を処理するHTTPハンドラー関数
func webhookHandler(notifier *simpleMQClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("リクエストボディの読み取りに失敗しました: %v", err)
			http.Error(w, "リクエストボディの読み取りに失敗しました", http.StatusBadRequest)
			return
		}

		// フィルタリング結果のいかんを問わず、Webhook通知の内容を標準出力する。
		log.Printf("received webhook raw body: %s", string(body))

		// JSONの構造が不正な場合はエラーを返す。
		var payload WebhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("無効なJSONです: %v", err)
			http.Error(w, "無効なJSONです", http.StatusBadRequest)
			return
		}

		if payload.Type != archiveCreatedType {
			log.Printf("type値%qはフィルタリング対象外のため、後続の処理を省略します。", payload.Type)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"matched": false,
			})
			return
		}

		// フィルタリング対象のtype値の場合は内容を変数化。
		channelID := payload.ChannelID
		eventType := payload.Type
		destURI := payload.Data.DestURI
		filePath := payload.Data.FilePath
		size := payload.Data.Size
		fileType := payload.Data.FileType
		absoluteURL := payload.Data.AbsoluteURL

		log.Printf(
			"Webhookを処理します: チャンネルID：%q タイプ：%q アーカイブ保存先URI：%q ファイルパス：%q ファイルサイズ：%d ファイル形式：%q 復元URL（欠損時のみ）：%q",
			channelID,
			eventType,
			destURI,
			filePath,
			size,
			fileType,
			absoluteURL,
		)

		// m3u8ファイルの作成イベントであれば、シンプルMQにメッセージを追加する。それ以外は何もしない。
		fileTypeMatched := fileType == targetFileType
		matched := eventType == archiveCreatedType && fileTypeMatched
		if matched {
			log.Printf("シンプルMQメッセージ追加要件に合致しました。")

			enqueuePayload, err := json.Marshal(payload)
			if err != nil {
				log.Printf("シンプルMQメッセージのJSON化に失敗しました: %v", err)
				http.Error(w, "シンプルMQメッセージのJSON化に失敗しました", http.StatusInternalServerError)
				return
			}

			if err := notifier.send(r.Context(), enqueuePayload); err != nil {
				log.Printf("シンプルMQへのメッセージ追加に失敗しました: %v", err)
				http.Error(w, "シンプルMQへのメッセージ追加に失敗しました", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("シンプルMQメッセージ追加要件に合致しませんでした。")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"matched": matched,
		})
	}
}
func (c *simpleMQClient) send(ctx context.Context, message []byte) error {
	payloadBody, err := json.Marshal(simpleMQSendRequest{Content: base64.StdEncoding.EncodeToString(message)})
	if err != nil {
		return fmt.Errorf("シンプルMQメッセージ送信リクエストのJSON化に失敗しました: %w", err)
	}

	endpoint := fmt.Sprintf(simpleMQEnqueueEndpointFmt, url.PathEscape(c.queueName))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBody))
	if err != nil {
		return fmt.Errorf("シンプルMQメッセージ追加リクエストの生成に失敗しました: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// シンプルMQのメッセージ追加APIを呼び出す
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("シンプルMQメッセージ追加API呼び出しに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("シンプルMQメッセージ追加APIレスポンスの読み取りに失敗しました: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("シンプルMQメッセージ追加APIがエラーを返却しました: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	log.Printf("シンプルMQへのメッセージ追加に成功しました: status=%d", resp.StatusCode)
	return nil
}