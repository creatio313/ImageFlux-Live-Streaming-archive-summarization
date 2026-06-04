#!/bin/bash
#
# @sacloud-once
# @sacloud-name "FFmpeg & Golang"
# @sacloud-desc FFmpeg と 最新版のGolang をインストールします。
#
# @sacloud-require-archive distro-ubuntu

_motd() {
  LOG=$(ls /root/.sacloud-api/notes/*log 2>/dev/null || echo "/var/log/sacloud-startup.log")
  case $1 in
    start)
    echo -e "\n#-- Startup-script is \\033[0;32mrunning\\033[0;39m. --#\n\nPlease check the log file: ${LOG}\n" > /etc/motd
    ;;
    fail)
    echo -e "\n#-- Startup-script \\033[0;31mfailed\\033[0;39m. --#\n\nPlease check the log file: ${LOG}\n" > /etc/motd
    exit 1
    ;;
    end)
    cp -f /dev/null /etc/motd
    ;;
  esac
}

_motd start
set -eux
trap '_motd fail' ERR

cd /root

# 1. パッケージの更新と必須ツール(FFmpeg等)のインストール
apt-get update && apt-get upgrade -y
apt-get install -y ca-certificates curl wget ffmpeg

# 2. 最新のGo (1.26.4) を公式から直接ダウンロードしてインストール
GO_VERSION="1.26.4"
curl -fsSL -o go.tar.gz "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
rm -rf /usr/local/go
tar -C /usr/local -xzf go.tar.gz
rm go.tar.gz

# 3. 実行ファイルへのシンボリックリンクを作成（全ユーザーで確実に使えるようにする）
ln -s /usr/local/go/bin/go /usr/local/bin/go
ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Complete!
_motd end