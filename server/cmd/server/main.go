package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("config", defaultConfigPath, "設定ファイルパス(JSON)")
	flag.Parse()

	cfg, err := readConfigFromFile(*configPath)
	if err != nil {
		log.Fatalf("設定ファイルの読み込みに失敗しました: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// サーバの初期化
	server, err := newArchiveHandlingServerFromConfig(cfg)
	if err != nil {
		log.Fatalf("アーカイブ処理サーバの初期化に失敗しました: %v", err)
	}

	// シンプルMQからのメッセージ受信頻度の設定
	pollInterval := parseDurationOrDefault(cfg.SimpleMQ.PollInterval, 5*time.Second)
	errorRetryInterval := parseDurationOrDefault(cfg.SimpleMQ.ErrorRetryInterval, 60*time.Second)

	log.Printf("シンプルMQコンシューマを開始します。queue=%q poll_interval=%s", server.queueName, pollInterval)
	if err := runConsumerLoop(ctx, server, pollInterval, errorRetryInterval); err != nil && ctx.Err() == nil {
		log.Fatalf("コンシューマループが異常終了しました: %v", err)
	}

	log.Printf("シンプルMQコンシューマを終了します。")
}

// シンプルMQ確認頻度設定をパースするだけ。
func parseDurationOrDefault(value string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(value)
	if v == "" {
		return fallback
	}

	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		log.Printf("duration値%qが不正のためデフォルト値%sを使用します", v, fallback)
		return fallback
	}

	return d
}

/*
**
メインループ
**
*/
func runConsumerLoop(ctx context.Context, server *archiveHandlingServer, pollInterval, errorRetryInterval time.Duration) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		//シンプルMQからメッセージを受信する。エラーおよび空の場合は指定時間後に再度受信を試みる。
		msg, ok, err := server.receiveOne(ctx)
		if err != nil {
			log.Printf("メッセージ受信に失敗しました: %v", err)
			if !sleepWithContext(ctx, errorRetryInterval) {
				return nil
			}
			continue
		}
		if !ok {
			if !sleepWithContext(ctx, pollInterval) {
				return nil
			}
			continue
		}

		// メッセージの処理
		if err := handleMessage(ctx, server, msg); err != nil {
			log.Printf("シンプルMQメッセージ処理に失敗しました。次回再取得に任せます: id=%s err=%v", msg.ID, err)
			if !sleepWithContext(ctx, errorRetryInterval) {
				return nil
			}
			continue
		}

		if err := server.deleteMessage(ctx, msg.ID); err != nil {
			log.Printf("シンプルMQメッセージ削除に失敗しました。重複処理を避けるため確認が必要です: id=%s err=%v", msg.ID, err)
			if !sleepWithContext(ctx, errorRetryInterval) {
				return nil
			}
			continue
		}
	}
}

/*
**
メイン処理
**
*/
func handleMessage(ctx context.Context, server *archiveHandlingServer, msg mqMessage) error {
	//メッセージ内容を取得する
	decoded, err := base64.StdEncoding.DecodeString(msg.Content)
	if err != nil {
		return fmt.Errorf("content(base64)のデコードに失敗しました: %w", err)
	}

	var payload messageContent
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return fmt.Errorf("メッセージJSONのパースに失敗しました: %w", err)
	}

	log.Printf(
		"受信メッセージ: id=%s channel_id=%q type=%q file_path=%q",
		msg.ID,
		payload.ChannelID,
		payload.Type,
		payload.Data.CurrentFilePath,
	)

	//m3u8へのアクセスURLを構築する
	archiveURL, err := buildArchiveURL(server.archiveHTTPDomain, payload)
	if err != nil {
		return fmt.Errorf("アーカイブURLの構築に失敗しました: %w", err)
	}

	//アーカイブ音声処理用の作業ディレクトリを作成する
	workDir, err := os.MkdirTemp("", "archive-audio-*")
	if err != nil {
		return fmt.Errorf("作業ディレクトリの作成に失敗しました: %w", err)
	}
	defer os.RemoveAll(workDir)

	// アーカイブURLから音声を抽出しつつ分割する。
	segments, err := extractMP3SegmentsFromArchive(ctx, archiveURL, workDir)
	if err != nil {
		return err
	}

	/***
		文字起こし
	***/
	//分割した音声ファイルを順番に文字起こしする。結果をtranscriptParts変数に追加していく。
	transcriptParts := make([]string, 0, len(segments))
	for i, segmentPath := range segments {
		text, err := server.transcribeAudio(ctx, segmentPath)
		if err != nil {
			return fmt.Errorf("文字起こしに失敗しました(part %d/%d): %w", i+1, len(segments), err)
		}
		transcriptParts = append(transcriptParts, text)
		log.Printf("文字起こし結果(part %d/%d): %s", i+1, len(segments), text)
	}

	transcriptText := strings.TrimSpace(strings.Join(transcriptParts, "\n\n"))
	if transcriptText == "" {
		return fmt.Errorf("文字起こし結果が空です")
	}

	//文字起こし結果をオブジェクトストレージにアップロードする
	objectKey, err := buildTranscriptObjectKey(payload)
	if err != nil {
		return fmt.Errorf("文字起こしオブジェクトキーの構築に失敗しました: %w", err)
	}
	if err := server.uploadTranscriptText(ctx, objectKey, transcriptText); err != nil {
		return err
	}
	log.Printf("文字起こし結果をアップロードしました: bucket=%q key=%q", server.objectStorageBucket, objectKey)

	summaryText, err := server.summarizeTranscript(ctx, transcriptText)
	if err != nil {
		return fmt.Errorf("要約生成に失敗しました: %w", err)
	}

	summaryKey, err := buildSummaryObjectKey(payload)
	if err != nil {
		return fmt.Errorf("要約オブジェクトキーの構築に失敗しました: %w", err)
	}
	if err := server.uploadSummaryText(ctx, summaryKey, summaryText); err != nil {
		return err
	}
	log.Printf("要約結果をアップロードしました: bucket=%q key=%q", server.objectStorageBucket, summaryKey)

	if err := server.sendCompletionNotification(ctx, payload, archiveURL, objectKey, summaryKey); err != nil {
		return err
	}
	log.Printf("シンプル通知APIで完了通知を送信しました: notification_group_id=%q", server.notificationGroupID)

	return nil
}

// 指定時間待機するが、コンテキストがキャンセルされたら早期に戻る。
func sleepWithContext(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
