output "apprun_public_url" {
  description = "AppRun公開URL"
  value       = sakura_apprun_shared.imageflux_live_streaming_webhook_handler.public_url
}

output "object_storage_bucket_name" {
  description = "バケット名"
  value       = sakura_object_storage_bucket.imageflux_live_streaming_archive_bucket.name
}

output "object_storage_access_key" {
  description = "バケットのアクセスキー"
  value       = nonsensitive(sakura_object_storage_permission.imageflux_live_streaming_archive_bucket_rw_permission.access_key)
}

output "object_storage_secret_access_key" {
  description = "バケットのシークレットアクセスキー"
  value       = nonsensitive(sakura_object_storage_permission.imageflux_live_streaming_archive_bucket_rw_permission.secret_key)
}

output "server_ip" {
  description = "サーバーのIPアドレス"
  value       = sakura_server.archive_handling_server.ip_address
}

output "simple_notification_group_id" {
  description = "シンプル通知グループID"
  value       = sakura_simple_notification_group.notification_group_for_imageflux_live_streaming_archive_notifier.id
}

output "webaccel_public_url" {
  description = "さくらのウェブアクセラレータ公開URL"
  value       = sakura_webaccel.imageflux_live_streaming_archive_domain.subdomain
}