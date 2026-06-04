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
	return buildSiblingObjectKey(payload.Data.DestURI, "transcript.txt")
}

func buildSummaryObjectKey(payload messageContent) (string, error) {
	return buildSiblingObjectKey(payload.Data.DestURI, "summary.txt")
}

func (c *archiveHandlingServer) uploadTranscriptText(ctx context.Context, objectKey, transcript string) error {
	return c.uploadTextObject(ctx, objectKey, transcript)
}

func (c *archiveHandlingServer) uploadSummaryText(ctx context.Context, objectKey, summary string) error {
	return c.uploadTextObject(ctx, objectKey, summary)
}

func buildSiblingObjectKey(destURI, fileName string) (string, error) {
	s3URI := strings.TrimSpace(destURI)
	if !strings.HasPrefix(s3URI, "s3://") {
		return "", fmt.Errorf("dest_uriはs3://形式である必要があります: %q", destURI)
	}

	rest := strings.TrimPrefix(s3URI, "s3://")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return "", fmt.Errorf("dest_uriの形式が不正です: %q", destURI)
	}

	objectPath := strings.Trim(path.Clean("/"+parts[1]), "/")
	if objectPath == "" || objectPath == "." {
		return "", fmt.Errorf("dest_uriからオブジェクトパスを取得できません: %q", destURI)
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