terraform {
  required_version = ">= 1.0"
  
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 2.0"
    }
  }
  
  backend "azurerm" {
    resource_group_name  = "nrdot-terraform-state"
    storage_account_name = "nrdotterraformstate"
    container_name       = "tfstate"
    key                  = "nrdot-host.tfstate"
  }
}

provider "azurerm" {
  features {
    key_vault {
      purge_soft_delete_on_destroy = false
    }
  }
}

provider "azuread" {}

# Data sources
data "azurerm_client_config" "current" {}
data "azuread_client_config" "current" {}

# Resource Group
resource "azurerm_resource_group" "nrdot" {
  name     = "${var.name_prefix}-rg"
  location = var.location
  
  tags = var.tags
}

# Virtual Network
resource "azurerm_virtual_network" "nrdot" {
  name                = "${var.name_prefix}-vnet"
  address_space       = [var.vnet_cidr]
  location            = azurerm_resource_group.nrdot.location
  resource_group_name = azurerm_resource_group.nrdot.name
  
  tags = var.tags
}

resource "azurerm_subnet" "nrdot" {
  name                 = "${var.name_prefix}-subnet"
  resource_group_name  = azurerm_resource_group.nrdot.name
  virtual_network_name = azurerm_virtual_network.nrdot.name
  address_prefixes     = [var.subnet_cidr]
  
  service_endpoints = [
    "Microsoft.Storage",
    "Microsoft.KeyVault",
    "Microsoft.EventHub",
    "Microsoft.ServiceBus",
    "Microsoft.Sql",
    "Microsoft.ContainerRegistry"
  ]
}

# Network Security Group
resource "azurerm_network_security_group" "nrdot" {
  name                = "${var.name_prefix}-nsg"
  location            = azurerm_resource_group.nrdot.location
  resource_group_name = azurerm_resource_group.nrdot.name
  
  security_rule {
    name                       = "AllowInternalTraffic"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = var.subnet_cidr
    destination_address_prefix = var.subnet_cidr
  }
  
  tags = var.tags
}

resource "azurerm_subnet_network_security_group_association" "nrdot" {
  subnet_id                 = azurerm_subnet.nrdot.id
  network_security_group_id = azurerm_network_security_group.nrdot.id
}

# Managed Identity
resource "azurerm_user_assigned_identity" "nrdot" {
  name                = "${var.name_prefix}-identity"
  location            = azurerm_resource_group.nrdot.location
  resource_group_name = azurerm_resource_group.nrdot.name
  
  tags = var.tags
}

# Key Vault
resource "azurerm_key_vault" "nrdot" {
  name                = "${var.name_prefix}-kv-${random_string.suffix.result}"
  location            = azurerm_resource_group.nrdot.location
  resource_group_name = azurerm_resource_group.nrdot.name
  tenant_id           = data.azurerm_client_config.current.tenant_id
  sku_name            = "standard"
  
  purge_protection_enabled   = var.environment == "production"
  soft_delete_retention_days = 90
  
  access_policy {
    tenant_id = data.azurerm_client_config.current.tenant_id
    object_id = data.azurerm_client_config.current.object_id
    
    key_permissions = [
      "Create", "Get", "List", "Update", "Delete", "Purge", "Recover"
    ]
    
    secret_permissions = [
      "Set", "Get", "List", "Delete", "Purge", "Recover"
    ]
    
    certificate_permissions = [
      "Create", "Get", "List", "Update", "Delete", "Purge", "Recover"
    ]
  }
  
  access_policy {
    tenant_id = data.azurerm_client_config.current.tenant_id
    object_id = azurerm_user_assigned_identity.nrdot.principal_id
    
    key_permissions = [
      "Get", "UnwrapKey", "WrapKey"
    ]
    
    secret_permissions = [
      "Get", "List"
    ]
  }
  
  network_acls {
    default_action             = "Deny"
    bypass                     = "AzureServices"
    virtual_network_subnet_ids = [azurerm_subnet.nrdot.id]
  }
  
  tags = var.tags
}

resource "random_string" "suffix" {
  length  = 4
  special = false
  upper   = false
}

# Storage Account
resource "azurerm_storage_account" "nrdot" {
  name                     = "${var.name_prefix}sa${random_string.suffix.result}"
  resource_group_name      = azurerm_resource_group.nrdot.name
  location                 = azurerm_resource_group.nrdot.location
  account_tier             = "Standard"
  account_replication_type = var.environment == "production" ? "GRS" : "LRS"
  
  enable_https_traffic_only       = true
  min_tls_version                 = "TLS1_2"
  allow_nested_items_to_be_public = false
  
  blob_properties {
    delete_retention_policy {
      days = 30
    }
    
    container_delete_retention_policy {
      days = 30
    }
    
    versioning_enabled = true
  }
  
  identity {
    type = "SystemAssigned"
  }
  
  tags = var.tags
}

resource "azurerm_storage_container" "data_lake" {
  name                  = "data-lake"
  storage_account_name  = azurerm_storage_account.nrdot.name
  container_access_type = "private"
}

# Event Hub
resource "azurerm_eventhub_namespace" "nrdot" {
  name                = "${var.name_prefix}-ehns"
  location            = azurerm_resource_group.nrdot.location
  resource_group_name = azurerm_resource_group.nrdot.name
  sku                 = var.environment == "production" ? "Standard" : "Basic"
  capacity            = var.eventhub_capacity
  
  auto_inflate_enabled     = true
  maximum_throughput_units = 20
  
  identity {
    type = "SystemAssigned"
  }
  
  tags = var.tags
}

resource "azurerm_eventhub" "events" {
  name                = "events"
  namespace_name      = azurerm_eventhub_namespace.nrdot.name
  resource_group_name = azurerm_resource_group.nrdot.name
  partition_count     = var.eventhub_partitions
  message_retention   = 7
  
  capture_description {
    enabled = true
    encoding = "Avro"
    
    destination {
      name                = "EventHubArchive.AzureBlockBlob"
      archive_name_format = "{Namespace}/{EventHub}/{PartitionId}/{Year}/{Month}/{Day}/{Hour}/{Minute}/{Second}"
      blob_container_name = azurerm_storage_container.data_lake.name
      storage_account_id  = azurerm_storage_account.nrdot.id
    }
  }
}

# Service Bus
resource "azurerm_servicebus_namespace" "nrdot" {
  name                = "${var.name_prefix}-sbns"
  location            = azurerm_resource_group.nrdot.location
  resource_group_name = azurerm_resource_group.nrdot.name
  sku                 = var.environment == "production" ? "Premium" : "Standard"
  
  identity {
    type = "SystemAssigned"
  }
  
  tags = var.tags
}

resource "azurerm_servicebus_queue" "events" {
  name         = "events"
  namespace_id = azurerm_servicebus_namespace.nrdot.id
  
  enable_partitioning = true
  max_size_in_megabytes = 5120
  
  dead_lettering_on_message_expiration = true
  max_delivery_count                   = 3
}

# Cosmos DB
resource "azurerm_cosmosdb_account" "nrdot" {
  name                = "${var.name_prefix}-cosmos"
  location            = azurerm_resource_group.nrdot.location
  resource_group_name = azurerm_resource_group.nrdot.name
  offer_type          = "Standard"
  kind                = "GlobalDocumentDB"
  
  consistency_policy {
    consistency_level       = "Session"
    max_interval_in_seconds = 5
    max_staleness_prefix    = 100
  }
  
  geo_location {
    location          = azurerm_resource_group.nrdot.location
    failover_priority = 0
  }
  
  dynamic "geo_location" {
    for_each = var.environment == "production" ? [var.failover_location] : []
    content {
      location          = geo_location.value
      failover_priority = 1
    }
  }
  
  enable_automatic_failover = var.environment == "production"
  
  identity {
    type = "SystemAssigned"
  }
  
  tags = var.tags
}

resource "azurerm_cosmosdb_sql_database" "nrdot" {
  name                = "nrdot"
  resource_group_name = azurerm_cosmosdb_account.nrdot.resource_group_name
  account_name        = azurerm_cosmosdb_account.nrdot.name
  
  autoscale_settings {
    max_throughput = var.cosmos_max_throughput
  }
}

resource "azurerm_cosmosdb_sql_container" "events" {
  name                = "events"
  resource_group_name = azurerm_cosmosdb_account.nrdot.resource_group_name
  account_name        = azurerm_cosmosdb_account.nrdot.name
  database_name       = azurerm_cosmosdb_sql_database.nrdot.name
  partition_key_path  = "/tenantId"
  
  autoscale_settings {
    max_throughput = var.cosmos_max_throughput
  }
  
  default_ttl = 2592000 # 30 days
  
  indexing_policy {
    indexing_mode = "consistent"
    
    included_path {
      path = "/*"
    }
    
    excluded_path {
      path = "/\"_etag\"/?"
    }
  }
}

# Redis Cache
resource "azurerm_redis_cache" "nrdot" {
  name                = "${var.name_prefix}-redis"
  location            = azurerm_resource_group.nrdot.location
  resource_group_name = azurerm_resource_group.nrdot.name
  capacity            = var.redis_capacity
  family              = var.redis_family
  sku_name            = var.redis_sku
  
  enable_non_ssl_port = false
  minimum_tls_version = "1.2"
  
  redis_configuration {
    enable_authentication = true
  }
  
  tags = var.tags
}

# Log Analytics Workspace
resource "azurerm_log_analytics_workspace" "nrdot" {
  name                = "${var.name_prefix}-logs"
  location            = azurerm_resource_group.nrdot.location
  resource_group_name = azurerm_resource_group.nrdot.name
  sku                 = "PerGB2018"
  retention_in_days   = var.log_retention_days
  
  tags = var.tags
}

# Application Insights
resource "azurerm_application_insights" "nrdot" {
  name                = "${var.name_prefix}-insights"
  location            = azurerm_resource_group.nrdot.location
  resource_group_name = azurerm_resource_group.nrdot.name
  workspace_id        = azurerm_log_analytics_workspace.nrdot.id
  application_type    = "web"
  
  tags = var.tags
}

# AKS Cluster
resource "azurerm_kubernetes_cluster" "nrdot" {
  name                = "${var.name_prefix}-aks"
  location            = azurerm_resource_group.nrdot.location
  resource_group_name = azurerm_resource_group.nrdot.name
  dns_prefix          = var.name_prefix
  
  default_node_pool {
    name                = "default"
    node_count          = var.node_count
    vm_size             = var.node_vm_size
    vnet_subnet_id      = azurerm_subnet.nrdot.id
    enable_auto_scaling = true
    min_count           = var.min_node_count
    max_count           = var.max_node_count
    
    upgrade_settings {
      max_surge = "33%"
    }
  }
  
  identity {
    type = "SystemAssigned"
  }
  
  network_profile {
    network_plugin    = "azure"
    network_policy    = "azure"
    load_balancer_sku = "standard"
  }
  
  oms_agent {
    log_analytics_workspace_id = azurerm_log_analytics_workspace.nrdot.id
  }
  
  azure_policy_enabled = true
  
  tags = var.tags
}

# Role Assignments
resource "azurerm_role_assignment" "nrdot_storage" {
  scope                = azurerm_storage_account.nrdot.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_user_assigned_identity.nrdot.principal_id
}

resource "azurerm_role_assignment" "nrdot_eventhub" {
  scope                = azurerm_eventhub_namespace.nrdot.id
  role_definition_name = "Azure Event Hubs Data Owner"
  principal_id         = azurerm_user_assigned_identity.nrdot.principal_id
}

resource "azurerm_role_assignment" "nrdot_servicebus" {
  scope                = azurerm_servicebus_namespace.nrdot.id
  role_definition_name = "Azure Service Bus Data Owner"
  principal_id         = azurerm_user_assigned_identity.nrdot.principal_id
}

# Outputs
output "resource_group_name" {
  value = azurerm_resource_group.nrdot.name
}

output "vnet_name" {
  value = azurerm_virtual_network.nrdot.name
}

output "subnet_id" {
  value = azurerm_subnet.nrdot.id
}

output "identity_id" {
  value = azurerm_user_assigned_identity.nrdot.id
}

output "identity_client_id" {
  value = azurerm_user_assigned_identity.nrdot.client_id
}

output "key_vault_name" {
  value = azurerm_key_vault.nrdot.name
}

output "storage_account_name" {
  value = azurerm_storage_account.nrdot.name
}

output "eventhub_namespace" {
  value = azurerm_eventhub_namespace.nrdot.name
}

output "cosmos_endpoint" {
  value = azurerm_cosmosdb_account.nrdot.endpoint
}

output "redis_hostname" {
  value = azurerm_redis_cache.nrdot.hostname
}

output "aks_cluster_name" {
  value = azurerm_kubernetes_cluster.nrdot.name
}

output "app_insights_key" {
  value     = azurerm_application_insights.nrdot.instrumentation_key
  sensitive = true
}