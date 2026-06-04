package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (c *archiveHandlingServer) sendCompletionNotification(ctx context.Context, payload messageContent, archiveURL, transcriptKey, summaryKey string) error {
	//アクセストークンの発行
	accessToken, err := c.issueAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("通知用アクセストークンの発行に失敗しました: %w", err)
	}

	message := fmt.Sprintf(
		"アーカイブが保存され、文字起こしと要約が完了しました。\nチャンネルID: %s\nタイプ: %s\nm3u8: %s\n文字起こし: s3://%s/%s\n要約: s3://%s/%s",
		payload.ChannelID,
		payload.Type,
		archiveURL,
		c.objectStorageBucket,
		transcriptKey,
		c.objectStorageBucket,
		summaryKey,
	)

	body, err := json.Marshal(simpleNotificationPayload{Message: message})
	if err != nil {
		return fmt.Errorf("通知メッセージJSON化に失敗しました: %w", err)
	}

	endpoint := fmt.Sprintf(simpleNotificationEndpointFmt, url.PathEscape(c.notificationGroupID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("通知リクエスト生成に失敗しました: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	//シンプル通知APIの呼び出し
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("シンプル通知API呼び出しに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("通知APIレスポンスの読み取りに失敗しました: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("通知APIがエラーを返却しました: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}