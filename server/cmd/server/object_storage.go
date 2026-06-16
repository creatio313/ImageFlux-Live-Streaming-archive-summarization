package main

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
)

/***
将来的な処理分岐に備え、アップロード関数を文字起こし用と要約用で分けているが、現状は両者ともテキストオブジェクトをアップロードするだけの同一処理になっている。
***/
func buildTranscriptObjectKey(payload messageContent) (string, error) {
	return buildSiblingObjectKey(payload.Data.CurrentFilePath, "transcript.txt")
}

func buildSummaryObjectKey(payload messageContent) (string, error) {
	return buildSiblingObjectKey(payload.Data.CurrentFilePath, "summary.txt")
}

func (c *archiveHandlingServer) uploadTranscriptText(ctx context.Context, objectKey, transcript string) error {
	return c.uploadTextObject(ctx, objectKey, transcript)
}

func (c *archiveHandlingServer) uploadSummaryText(ctx context.Context, objectKey, summary string) error {
	return c.uploadTextObject(ctx, objectKey, summary)
}

func buildSiblingObjectKey(currentFilePath, fileName string) (string, error) {
	objectPath := strings.Trim(path.Clean("/"+strings.TrimSpace(currentFilePath)), "/")
	if objectPath == "" || objectPath == "." {
		return "", fmt.Errorf("current_file_pathからオブジェクトパスを取得できません: %q", currentFilePath)
	}

	//親フォルダの取得
	dir := strings.Trim(path.Dir(objectPath), "/")
	if dir == "" || dir == "." {
		return fileName, nil
	}

	return path.Join(dir, fileName), nil
}

func (c *archiveHandlingServer) uploadTextObject(ctx context.Context, objectKey, content string) error {
	if c.objectStorageClient == nil {
		return fmt.Errorf("オブジェクトストレージクライアントが初期化されていません")
	}

	reader := strings.NewReader(content)
	info, err := c.objectStorageClient.PutObject(
		ctx,
		c.objectStorageBucket,
		objectKey,
		reader,
		int64(len([]byte(content))),
		minio.PutObjectOptions{ContentType: "text/plain; charset=utf-8"},
	)
	if err != nil {
		return fmt.Errorf("テキスト成果物のアップロードに失敗しました: bucket=%q key=%q err=%w", c.objectStorageBucket, objectKey, err)
	}
	if info.Size <= 0 {
		return fmt.Errorf("テキスト成果物のアップロード結果が不正です: bucket=%q key=%q size=%d", c.objectStorageBucket, objectKey, info.Size)
	}

	return nil
}
