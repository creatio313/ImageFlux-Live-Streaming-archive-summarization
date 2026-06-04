# 概要

ImageFlux Live Streamingのアーカイブ保存完了通知を受け取り、自動でさくらのAI Engineによる文字起こし・要約とその完了メール通知を行うシステム構成は「構成図.pptx」の通り。

# 構成

## docker

AppRun共用型で動作するDockerイメージ。
コンテナレジストリに対しbuild、pushして利用する。
ImageFluxのイベントWebhook通知をシンプルMQに追加する仕組み。

## preview_page

.m3u8のパスを入力すると、アーカイブをロードし、かつ同ディレクトリの文字起こしおよび要約データをWebに読み込む。

## server

シンプルMQからメッセージを取得し、mp3エンコード、さくらのAI Engine呼び出し、オブジェクトストレージ格納までを行う。
Go言語で構成。

## terraform

基盤を構築するTerraformソース。
シンプルMQ、サービスプリンシパル、さくらのAI Engine APIキーは事前に作成しておく必要がある。それ以外の基盤は自動構築。

```hcl
terraform init
terraform apply
```
