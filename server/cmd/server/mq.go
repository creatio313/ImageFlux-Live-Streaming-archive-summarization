package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

//シンプルMQからメッセージを取得し、最新メッセージ1件を返す
func (c *archiveHandlingServer) receiveOne(ctx context.Context) (mqMessage, bool, error) {
	apiKey := strings.TrimSpace(c.simpleMQAPIKey)
	if apiKey == "" {
		return mqMessage{}, false, fmt.Errorf("シンプルMQ APIキーが設定されていません")
	}

	endpoint := fmt.Sprintf("%s/v1/queues/%s/messages", simpleMQBaseURL, url.PathEscape(c.queueName))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return mqMessage{}, false, fmt.Errorf("受信リクエストの生成に失敗しました: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return mqMessage{}, false, fmt.Errorf("受信API呼び出しに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return mqMessage{}, false, fmt.Errorf("受信APIレスポンスの読み取りに失敗しました: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mqMessage{}, false, fmt.Errorf("受信APIがエラーを返却しました: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var parsed simpleMQReceiveResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return mqMessage{}, false, fmt.Errorf("受信APIレスポンスのJSON解析に失敗しました: %w", err)
	}

	if len(parsed.Messages) == 0 {
		return mqMessage{}, false, nil
	}

	return parsed.Messages[0], true, nil
}

//シンプルMQの指定メッセージを削除する
func (c *archiveHandlingServer) deleteMessage(ctx context.Context, messageID string) error {
	apiKey := strings.TrimSpace(c.simpleMQAPIKey)
	if apiKey == "" {
		return fmt.Errorf("シンプルMQ APIキーが設定されていません")
	}

	endpoint := fmt.Sprintf("%s/v1/queues/%s/messages/%s", simpleMQBaseURL, url.PathEscape(c.queueName), url.PathEscape(messageID))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("削除リクエストの生成に失敗しました: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("削除API呼び出しに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("削除APIレスポンスの読み取りに失敗しました: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("削除APIがエラーを返却しました: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}