data "sakura_archive" "ubuntu" {
  os_type = "ubuntu2404"
}

data "sakura_object_storage_site" "ishikari" {
  display_name = "石狩第1サイト"
}

resource "sakura_apprun_shared" "imageflux_live_streaming_webhook_handler" {
  name = "ImageFlux Live StreamingのWebhookハンドラ"

  max_scale       = 3
  min_scale       = 0
  port            = 8080
  timeout_seconds = 60

  components = [{
    name       = "ImageFlux Live StreamingのWebhookハンドラコンテナ"
    max_cpu    = "0.5"
    max_memory = "1Gi"
    deploy_source = {
      container_registry = {
        image = var.container_registry_image
      }
    }
    env = [{
      key   = "SIMPLEMQ_API_KEY"
      value = var.simple_mq_api_key
      },
      {
        key   = "SIMPLEMQ_QUEUE_NAME"
        value = var.simple_mq_queue_name
    }]
  }]
  traffics = [{
    version_index = 0
    percent       = 100
  }]
}

resource "sakura_disk" "archive_handling_server_disk" {
  name        = "アーカイブ処理サーバーのディスク"
  description = "アーカイブ処理サーバーのディスク。最低限のスペックで構築し、必要に応じてスケールアップする。"

  connector            = "virtio"
  encryption_algorithm = "aes256_xts"
  icon_id              = var.ubuntu_icon
  kms_key_id           = sakura_kms.archive_handling_server_encryption_key.id
  plan                 = "ssd"
  size                 = 20
  source_archive_id    = data.sakura_archive.ubuntu.id
  zone                 = var.zone
}

resource "sakura_kms" "archive_handling_server_encryption_key" {
  name        = "アーカイブ処理サーバー用暗号化キー"
  description = "アーカイブ処理サーバー用の暗号化キー。ディスクの暗号化に使用する。"
  key_origin  = "generated"
}

resource "sakura_object_storage_bucket" "imageflux_live_streaming_archive_bucket" {
  name    = "imageflux-live-streaming-archive-storage-bucket"
  site_id = data.sakura_object_storage_site.ishikari.id
}

resource "sakura_object_storage_permission" "imageflux_live_streaming_archive_bucket_r_permission" {
  name = "ImageFlux Live StreamingアーカイブバケットRead権限"
  bucket_controls = [{
    bucket    = sakura_object_storage_bucket.imageflux_live_streaming_archive_bucket.name
    can_read  = true
    can_write = false
  }]
  site_id = data.sakura_object_storage_site.ishikari.id
}

resource "sakura_object_storage_permission" "imageflux_live_streaming_archive_bucket_rw_permission" {
  name = "ImageFlux Live StreamingアーカイブバケットReadWrite権限"
  bucket_controls = [{
    bucket    = sakura_object_storage_bucket.imageflux_live_streaming_archive_bucket.name
    can_read  = true
    can_write = true
  }]
  site_id = data.sakura_object_storage_site.ishikari.id
}

resource "sakura_packet_filter" "minimum_filter" {
  name        = "最低限のパケットフィルタ"
  description = "アーカイブ処理サーバー用の最低限のパケットフィルタ"
  zone        = var.zone
}

resource "sakura_packet_filter_rules" "rules" {
  packet_filter_id = sakura_packet_filter.minimum_filter.id
  zone             = var.zone

  expression = [
    {
      description      = "Allow SSH access. Limit source IP addresses, if needed."
      destination_port = "22"
      protocol         = "tcp"
      source_network   = "0.0.0.0/0"
    },
    {
      protocol       = "udp"
      source_port    = "123"
      source_network = "0.0.0.0/0"
    },
    {
      protocol         = "udp"
      destination_port = "68"
    },
    {
      protocol = "icmp"
    },
    {
      protocol         = "tcp"
      destination_port = "32768-61000"
    },
    {
      protocol         = "udp"
      destination_port = "32768-61000"
    },
    {
      protocol = "fragment"
    },
    {
      protocol    = "ip"
      allow       = false
      description = "Deny all except above rules."
    }
  ]
}

resource "sakura_script" "ffmpeg_golang_install_script" {
  name    = "FFmpegとGoのインストールスクリプト"
  class   = "shell"
  content = file("scripts/install_ffmpeg_golang.sh")
  icon_id = var.ubuntu_icon
}

resource "sakura_server" "archive_handling_server" {
  name        = "アーカイブ処理サーバ"
  description = "アーカイブ処理サーバ。アーカイブの文字起こし、要約、通知を行う。"

  core             = 1
  disks            = [sakura_disk.archive_handling_server_disk.id]
  icon_id          = var.ubuntu_icon
  interface_driver = "virtio"
  memory           = 2
  tags             = ["@keyboard-us"]
  zone             = var.zone

  disk_edit_parameter = {
    hostname            = "ubuntuhost"
    password_wo         = var.os_password
    password_wo_version = 1
    disable_pw_auth     = true

    ssh_key_ids = [sakura_ssh_key.archive_handling_server_sshkey.id]
    script = [{
      id = sakura_script.ffmpeg_golang_install_script.id
    }]
  }

  network_interface = [{
    upstream         = "shared"
    packet_filter_id = sakura_packet_filter.minimum_filter.id
  }]
}

# アーカイブ処理サーバーのSSHログイン用秘密鍵を生成
resource "tls_private_key" "temporary_ssh_key" {
  algorithm = "RSA"
  rsa_bits  = 4096
}
// 秘密鍵をファイルに保存
resource "local_sensitive_file" "private_key_file" {
  content  = tls_private_key.temporary_ssh_key.private_key_pem
  filename = ".ssh/id_rsa.pem"
}
// 生成したSSH公開鍵をさくらのクラウドのSSHキーリソースに登録
resource "sakura_ssh_key" "archive_handling_server_sshkey" {
  name        = "アーカイブ処理サーバーSSHキー"
  description = "アーカイブ処理サーバー用のSSHキー。.ssh/ディレクトリに保存された秘密鍵とペア。"
  public_key  = tls_private_key.temporary_ssh_key.public_key_openssh
}

resource "sakura_simple_notification_destination" "notification_target_for_imageflux_live_streaming_archive_notifier" {
  name        = "通知先メールアドレス"
  description = "ImageFlux Live Streamingアーカイブ・要約完了時の通知先メールアドレス"
  type        = "email"
  value       = var.email_address
}

resource "sakura_simple_notification_group" "notification_group_for_imageflux_live_streaming_archive_notifier" {
  name         = "ImageFlux Live Streamingアーカイブ通知グループ"
  description  = "ImageFlux Live Streamingアーカイブ・要約完了時の通知グループ"
  destinations = [sakura_simple_notification_destination.notification_target_for_imageflux_live_streaming_archive_notifier.id]
}

resource "sakura_webaccel" "imageflux_live_streaming_archive_domain" {
  name             = "ImageFlux Live Streamingアーカイブ配信ドメイン"
  domain_type      = "subdomain"
  request_protocol = "https-redirect"
  cors_rules = [
    {
      allow_all = true
    }
  ]

  origin_parameters = {
    type                   = "bucket"
    access_key_wo          = sakura_object_storage_permission.imageflux_live_streaming_archive_bucket_r_permission.access_key
    secret_access_key_wo   = sakura_object_storage_permission.imageflux_live_streaming_archive_bucket_r_permission.secret_key
    bucket_name            = sakura_object_storage_bucket.imageflux_live_streaming_archive_bucket.name
    credentials_wo_version = 1
    use_document_index     = true
    endpoint               = join("", ["s3.", data.sakura_object_storage_site.ishikari.endpoint])
    region                 = data.sakura_object_storage_site.ishikari.region
  }

  logging = {
    enabled                = true
    bucket_name            = sakura_object_storage_bucket.imageflux_live_streaming_archive_bucket.name
    access_key_wo          = sakura_object_storage_permission.imageflux_live_streaming_archive_bucket_rw_permission.access_key
    secret_access_key_wo   = sakura_object_storage_permission.imageflux_live_streaming_archive_bucket_rw_permission.secret_key
    credentials_wo_version = 1
    endpoint               = join("", ["s3.", data.sakura_object_storage_site.ishikari.endpoint])
    region                 = data.sakura_object_storage_site.ishikari.region
  }
  default_cache_ttl = 334
  normalize_ae      = "gzip"
}
resource "sakura_webaccel_activation" "imageflux_live_streaming_archive_domain_activation" {
  site_id = sakura_webaccel.imageflux_live_streaming_archive_domain.id
  enabled = true
}