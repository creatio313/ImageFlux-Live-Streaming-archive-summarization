package main

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// アーカイブのURLをさくらのウェブアクセラレータのドメイン+パスの形式で構築する
func buildArchiveURL(domain string, payload messageContent) (string, error) {
	archivePath := strings.Trim(path.Clean("/"+strings.TrimSpace(payload.Data.CurrentFilePath)), "/")
	if archivePath == "" || archivePath == "." {
		return "", fmt.Errorf("current_file_pathからアーカイブパスを取得できません: %q", payload.Data.CurrentFilePath)
	}

	// ドメインの正規化
	normalizedDomain := strings.TrimRight(strings.TrimSpace(domain), "/")
	if normalizedDomain == "" {
		return "", fmt.Errorf("archive.http_domainが設定されていません")
	}
	if !strings.HasPrefix(normalizedDomain, "http://") && !strings.HasPrefix(normalizedDomain, "https://") {
		normalizedDomain = "https://" + normalizedDomain
	}

	//結合して返却
	return normalizedDomain + "/" + archivePath, nil
}

// ffmpegでアーカイブURLから音声を抽出しつつ、指定秒数ごとにmp3分割する。
func extractMP3SegmentsFromArchive(ctx context.Context, archiveURL, workDir string) ([]string, error) {
	segmentPattern := filepath.Join(workDir, "part-%03d.mp3")
	args := []string{
		"-y",
		"-i", archiveURL,
		"-vn",
		"-f", "segment",
		"-segment_time", fmt.Sprintf("%d", int(maxAudioDurationSeconds)),
		"-reset_timestamps", "1",
		"-acodec", "libmp3lame",
		"-ar", "16000",
		"-ac", "1",
		segmentPattern,
	}
	if err := runCommand(ctx, "ffmpeg", args...); err != nil {
		return nil, fmt.Errorf("アーカイブからの音声分割生成に失敗しました: %w", err)
	}

	splitFiles, err := filepath.Glob(filepath.Join(workDir, "part-*.mp3"))
	if err != nil {
		return nil, fmt.Errorf("分割ファイルの列挙に失敗しました: %w", err)
	}
	if len(splitFiles) == 0 {
		return nil, fmt.Errorf("音声分割の結果が空です")
	}
	sort.Strings(splitFiles)

	return splitFiles, nil
}

func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 4096 {
			output = output[len(output)-4096:]
		}
		return fmt.Errorf("command=%s args=%v err=%w output=%s", name, args, err, string(output))
	}
	return nil
}
