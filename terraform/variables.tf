variable "access_token" {
  type        = string
  description = "プロジェクトのアクセストークン"
  sensitive   = true
}
variable "access_token_secret" {
  type        = string
  description = "プロジェクトのアクセストークンシークレット"
  sensitive   = true
}
variable "container_registry_image" {
  type        = string
  description = "コンテナレジストリのイメージパス。AppRunのデプロイに使用する。"
  default     = "creatio313-live-streaming.sakuracr.jp/webhook-handler:beta"
}
variable "email_address" {
  type        = string
  description = "シンプル通知を受け取るメールアドレス"
  sensitive   = true
}
variable "simple_mq_api_key" {
  type        = string
  description = "シンプルMQのAPIキー"
  sensitive   = true
}
variable "simple_mq_queue_name" {
  type        = string
  description = "シンプルMQのキュー名"
  default     = "imageflux-ls-archive-notification-queue"
}
variable "os_password" {
  type        = string
  description = "サーバーのOSパスワード"
  sensitive   = true
}
variable "ubuntu_icon" {
  type        = string
  description = "UbuntuアイコンのID"
  default     = "112901627751"
}
variable "zone" {
  type        = string
  description = "リソースを構築するゾーン"
  default     = "is1c"
}