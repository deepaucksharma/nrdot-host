terraform {
  required_version = ">= 1.0"
  
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 5.0"
    }
  }
  
  backend "gcs" {
    bucket = "nrdot-terraform-state"
    prefix = "nrdot-host"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

# Enable required APIs
resource "google_project_service" "apis" {
  for_each = toset([
    "compute.googleapis.com",
    "container.googleapis.com",
    "pubsub.googleapis.com",
    "bigquery.googleapis.com",
    "bigtable.googleapis.com",
    "firestore.googleapis.com",
    "storage.googleapis.com",
    "secretmanager.googleapis.com",
    "cloudkms.googleapis.com",
    "monitoring.googleapis.com",
    "logging.googleapis.com",
    "cloudtrace.googleapis.com",
    "cloudfunctions.googleapis.com",
    "run.googleapis.com",
    "memcache.googleapis.com",
    "dlp.googleapis.com",
    "vertexai.googleapis.com"
  ])
  
  service                    = each.key
  disable_on_destroy         = false
  disable_dependent_services = false
}

# VPC Network
resource "google_compute_network" "nrdot" {
  name                    = "${var.name_prefix}-network"
  auto_create_subnetworks = false
  
  depends_on = [google_project_service.apis]
}

resource "google_compute_subnetwork" "nrdot" {
  name          = "${var.name_prefix}-subnet"
  network       = google_compute_network.nrdot.id
  region        = var.region
  ip_cidr_range = var.subnet_cidr
  
  private_ip_google_access = true
  
  log_config {
    aggregation_interval = "INTERVAL_5_SEC"
    flow_sampling        = 0.5
  }
}

# Cloud NAT for outbound connectivity
resource "google_compute_router" "nrdot" {
  name    = "${var.name_prefix}-router"
  network = google_compute_network.nrdot.id
  region  = var.region
}

resource "google_compute_router_nat" "nrdot" {
  name                               = "${var.name_prefix}-nat"
  router                             = google_compute_router.nrdot.name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"
}

# Firewall rules
resource "google_compute_firewall" "allow_internal" {
  name    = "${var.name_prefix}-allow-internal"
  network = google_compute_network.nrdot.name
  
  allow {
    protocol = "tcp"
    ports    = ["0-65535"]
  }
  
  allow {
    protocol = "udp"
    ports    = ["0-65535"]
  }
  
  allow {
    protocol = "icmp"
  }
  
  source_ranges = [var.subnet_cidr]
}

# Service Account
resource "google_service_account" "nrdot" {
  account_id   = "${var.name_prefix}-sa"
  display_name = "NRDOT Host Service Account"
}

# IAM roles for service account
resource "google_project_iam_member" "nrdot_roles" {
  for_each = toset([
    "roles/pubsub.editor",
    "roles/bigquery.dataEditor",
    "roles/bigtable.user",
    "roles/datastore.user",
    "roles/storage.objectAdmin",
    "roles/secretmanager.secretAccessor",
    "roles/cloudkms.cryptoKeyEncrypterDecrypter",
    "roles/monitoring.metricWriter",
    "roles/logging.logWriter",
    "roles/cloudtrace.agent",
    "roles/cloudfunctions.invoker"
  ])
  
  project = var.project_id
  role    = each.key
  member  = "serviceAccount:${google_service_account.nrdot.email}"
}

# KMS for encryption
resource "google_kms_key_ring" "nrdot" {
  name     = "${var.name_prefix}-keyring"
  location = var.region
}

resource "google_kms_crypto_key" "nrdot" {
  name     = "${var.name_prefix}-key"
  key_ring = google_kms_key_ring.nrdot.id
  
  rotation_period = "2592000s" # 30 days
  
  lifecycle {
    prevent_destroy = true
  }
}

# Pub/Sub topics and subscriptions
resource "google_pubsub_topic" "events" {
  name = "${var.name_prefix}-events"
  
  message_retention_duration = "604800s" # 7 days
  
  schema_settings {
    schema   = google_pubsub_schema.events.id
    encoding = "JSON"
  }
}

resource "google_pubsub_schema" "events" {
  name       = "${var.name_prefix}-events-schema"
  type       = "AVRO"
  definition = file("${path.module}/schemas/event.avsc")
}

resource "google_pubsub_subscription" "events" {
  name  = "${var.name_prefix}-events-sub"
  topic = google_pubsub_topic.events.name
  
  ack_deadline_seconds       = 60
  message_retention_duration = "604800s"
  retain_acked_messages      = false
  enable_exactly_once_delivery = true
  
  expiration_policy {
    ttl = ""
  }
  
  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }
  
  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dlq.id
    max_delivery_attempts = 5
  }
}

resource "google_pubsub_topic" "dlq" {
  name = "${var.name_prefix}-dlq"
}

# BigQuery dataset
resource "google_bigquery_dataset" "nrdot" {
  dataset_id    = "${var.name_prefix}_events"
  friendly_name = "NRDOT Events"
  description   = "NRDOT event data"
  location      = var.region
  
  default_table_expiration_ms = 31536000000 # 365 days
  
  default_encryption_configuration {
    kms_key_name = google_kms_crypto_key.nrdot.id
  }
  
  access {
    role          = "OWNER"
    user_by_email = google_service_account.nrdot.email
  }
}

# Cloud Storage buckets
resource "google_storage_bucket" "data_lake" {
  name          = "${var.name_prefix}-data-lake-${var.project_id}"
  location      = var.region
  force_destroy = false
  
  uniform_bucket_level_access = true
  
  encryption {
    default_kms_key_name = google_kms_crypto_key.nrdot.id
  }
  
  lifecycle_rule {
    condition {
      age = 30
    }
    action {
      type          = "SetStorageClass"
      storage_class = "NEARLINE"
    }
  }
  
  lifecycle_rule {
    condition {
      age = 90
    }
    action {
      type          = "SetStorageClass"
      storage_class = "COLDLINE"
    }
  }
  
  lifecycle_rule {
    condition {
      age = 365
    }
    action {
      type = "Delete"
    }
  }
  
  versioning {
    enabled = true
  }
}

# Firestore database
resource "google_firestore_database" "nrdot" {
  provider = google-beta
  
  project     = var.project_id
  name        = "(default)"
  location_id = var.region
  type        = "FIRESTORE_NATIVE"
  
  concurrency_mode            = "OPTIMISTIC"
  app_engine_integration_mode = "DISABLED"
}

# Memorystore Redis instance
resource "google_redis_instance" "cache" {
  name           = "${var.name_prefix}-cache"
  memory_size_gb = var.redis_memory_gb
  region         = var.region
  
  tier = var.environment == "production" ? "STANDARD_HA" : "BASIC"
  
  redis_version = "REDIS_7_0"
  
  auth_enabled = true
  
  transit_encryption_mode = "SERVER_AUTHENTICATION"
  
  maintenance_policy {
    weekly_maintenance_window {
      day = "SUNDAY"
      start_time {
        hours   = 2
        minutes = 0
      }
    }
  }
}

# GKE Cluster for running NRDOT-HOST
resource "google_container_cluster" "nrdot" {
  name     = "${var.name_prefix}-cluster"
  location = var.region
  
  # Regional cluster for HA
  node_locations = var.zones
  
  network    = google_compute_network.nrdot.name
  subnetwork = google_compute_subnetwork.nrdot.name
  
  initial_node_count       = 1
  remove_default_node_pool = true
  
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }
  
  addons_config {
    horizontal_pod_autoscaling {
      disabled = false
    }
    http_load_balancing {
      disabled = false
    }
    gce_persistent_disk_csi_driver_config {
      enabled = true
    }
  }
  
  cluster_autoscaling {
    enabled = true
    resource_limits {
      resource_type = "cpu"
      minimum       = 10
      maximum       = 100
    }
    resource_limits {
      resource_type = "memory"
      minimum       = 40
      maximum       = 400
    }
  }
  
  maintenance_policy {
    daily_maintenance_window {
      start_time = "03:00"
    }
  }
}

resource "google_container_node_pool" "nrdot" {
  name       = "${var.name_prefix}-pool"
  cluster    = google_container_cluster.nrdot.id
  node_count = var.node_count
  
  autoscaling {
    min_node_count = var.min_node_count
    max_node_count = var.max_node_count
  }
  
  node_config {
    preemptible  = var.environment != "production"
    machine_type = var.machine_type
    
    service_account = google_service_account.nrdot.email
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
    
    labels = {
      environment = var.environment
    }
    
    tags = ["nrdot-node"]
    
    workload_metadata_config {
      mode = "GKE_METADATA"
    }
  }
  
  management {
    auto_repair  = true
    auto_upgrade = true
  }
}

# Outputs
output "network_name" {
  value = google_compute_network.nrdot.name
}

output "subnet_name" {
  value = google_compute_subnetwork.nrdot.name
}

output "service_account_email" {
  value = google_service_account.nrdot.email
}

output "gke_cluster_name" {
  value = google_container_cluster.nrdot.name
}

output "pubsub_topic" {
  value = google_pubsub_topic.events.name
}

output "pubsub_subscription" {
  value = google_pubsub_subscription.events.name
}

output "bigquery_dataset" {
  value = google_bigquery_dataset.nrdot.dataset_id
}

output "storage_bucket" {
  value = google_storage_bucket.data_lake.name
}

output "redis_host" {
  value = google_redis_instance.cache.host
}

output "redis_port" {
  value = google_redis_instance.cache.port
}

output "kms_key" {
  value = google_kms_crypto_key.nrdot.id
}