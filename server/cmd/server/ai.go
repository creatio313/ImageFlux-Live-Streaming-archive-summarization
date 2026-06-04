package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//文字起こしを呼ぶ関数。mp3をもとにAIエンジンのAPIを呼び出して、文字起こし結果をテキストで返す。
func (c *archiveHandlingServer) transcribeAudio(ctx context.Context, filePath string) (string, error) {
	//ファイル読み出し
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("文字起こし対象ファイルのオープンに失敗しました: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	filePart, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("fileパート作成に失敗しました: %w", err)
	}
	if _, err := io.Copy(filePart, file); err != nil {
		return "", fmt.Errorf("fileコピーに失敗しました: %w", err)
	}
	if err := writer.WriteField("model", c.transcriptModel); err != nil {
		return "", fmt.Errorf("モデル設定に失敗しました: %w", err)
	}
	if err := writer.WriteField("language", "ja"); err != nil {
		return "", fmt.Errorf("言語設定に失敗しました: %w", err)
	}
	if err := writer.WriteField("temperature", "0"); err != nil {
		return "", fmt.Errorf("温度設定に失敗しました: %w", err)
	}	
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("パラメータのクローズに失敗しました: %w", err)
	}

	//リクエスト生成
	endpoint := strings.TrimRight(c.aiEngineBaseURL, "/") + "/v1/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", fmt.Errorf("文字起こしリクエストの生成に失敗しました: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.aiEngineAPIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	//文字起こしAPIの呼び出し
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("文字起こしAPI呼び出しに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("文字起こしレスポンスの読み取りに失敗しました: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("文字起こしAPIがエラーを返却しました: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Model string `json:"model"`
		Text  string `json:"text"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("文字起こしレスポンスのJSON解析に失敗しました: %w", err)
	}
	if strings.TrimSpace(result.Text) == "" {
		return "", fmt.Errorf("文字起こしレスポンスにtextがありません")
	}

	return result.Text, nil
}
//文字起こし結果を要約する関数。文字起こし結果をもとにAIエンジンのAPIを呼び出して、要約結果をテキストで返す。
func (c *archiveHandlingServer) summarizeTranscript(ctx context.Context, transcript string) (string, error) {
	/***
		要約要求の構築。取り急ぎ重要度の高いパラメータのみ設定しているが、追加・削除可能。
		参考：https://manual.sakura.ad.jp/api/cloud/portal/?api=ai-engine-inference-api#operation/createResponse
	***/
	prompt := "文字起こしを要約せよ。出力形式は＜要約＞要約内容とすること。要約する文章は以下の通り：\n\n" + transcript
	instructions := "要約は日本語で、重要な点を5項目以内の箇条書きで記載すること。内容を正確に把握したうえで事実のみを扱い、推測や創作は禁止とする。"
	bodyMap := map[string]any{
		"model": c.summaryModel,
		"input": prompt,
		"instructions": instructions,
		"temperature": 0.2,
		"max_output_tokens": 800,
	}
	body, err := json.Marshal(bodyMap)
	if err != nil {
		return "", fmt.Errorf("要約リクエストJSON化に失敗しました: %w", err)
	}

	endpoint := strings.TrimRight(c.aiEngineBaseURL, "/") + "/v1/responses"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("要約リクエストの生成に失敗しました: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.aiEngineAPIKey)
	req.Header.Set("Content-Type", "application/json")

	//要約APIの呼び出し
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("要約API呼び出しに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("要約レスポンスの読み取りに失敗しました: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("要約APIがエラーを返却しました: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	summary := extractSummaryText(respBody)
	if strings.TrimSpace(summary) == "" {
		return "", fmt.Errorf("要約レスポンスから本文を抽出できませんでした: body=%s", string(respBody))
	}

	return summary, nil
}
//要約APIのレスポンスから要約テキストを抽出する関数。レスポンスの項目が多いため、最低限の項目を抽出している。
func extractSummaryText(respBody []byte) string {
	var response struct {
		Output     []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(respBody, &response); err == nil {
		//配列1個目に値がある場合はそれを返却（通常はこれ）
		if len(response.Output) > 0 && len(response.Output[0].Content) > 0 {
			if text := strings.TrimSpace(response.Output[0].Content[0].Text); text != "" {
				return text
			}
		}
		//変則的な配列の場合の予備ロジック（削除可）
		for _, out := range response.Output {
			for _, c := range out.Content {
				if text := strings.TrimSpace(c.Text); text != "" {
					return text
				}
			}
		}
	}
	return ""
}
