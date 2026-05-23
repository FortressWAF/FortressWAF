# Cloud Deployment (AWS/GCP/Azure)

FortressWAF can be deployed on major cloud platforms using platform-specific services and configurations.

## Amazon Web Services (AWS)

### AWS Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                    AWS Region                                    │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                           VPC (10.0.0.0/16)                                 │ │
│  │                                                                             │ │
│  │  ┌────────────────┐         ┌────────────────┐         ┌────────────────┐ │ │
│  │  │   Public Subnet │         │  Private Subnet │         │ Private Subnet │ │ │
│  │  │   (az-1)        │         │    (az-1)       │         │    (az-2)      │ │ │
│  │  │                 │         │                 │         │                │ │ │
│  │  │  ┌───────────┐ │         │  ┌───────────┐  │         │  ┌───────────┐ │ │ │
│  │  │  │ Fortress  │ │         │  │ PostgreSQL│  │         │  │PostgreSQL │ │ │ │
│  │  │  │   WAF     │ │         │  │  Primary  │  │         │  │  Replica  │ │ │ │
│  │  │  └───────────┘ │         │  └───────────┘  │         │  └───────────┘ │ │ │
│  │  │                 │         │                 │         │                │ │ │
│  │  │  ┌───────────┐ │         │  ┌───────────┐  │         │  ┌───────────┐ │ │ │
│  │  │  │   Redis   │ │         │  │   Redis    │  │         │  │   Redis   │ │ │ │
│  │  │  │  Cluster  │ │         │  │  Cluster   │  │         │  │  Cluster  │ │ │ │
│  │  │  └───────────┘ │         │  └───────────┘  │         │  └───────────┘ │ │ │
│  │  └────────────────┘         └────────────────┘         └────────────────┘ │ │
│  │                                                                             │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                        │                                         │
│                              ┌─────────┴─────────┐                               │
│                              │   Application    │                               │
│                              │    Load Balancer │                               │
│                              └─────────┬─────────┘                               │
│                                        │                                         │
│                              ┌─────────┴─────────┐                               │
│                              │    Auto Scaling   │                               │
│                              │      Group        │                               │
│                              └───────────────────┘                               │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### AWS EC2 Deployment

#### Launch Configuration

```hcl
# AWS EC2 Launch Configuration
resource "aws_launch_template" "fortresswaf" {
  name_prefix   = "fortresswaf-"
  image_id      = data.aws_ami.fortresswaf.id
  instance_type = "m5.xlarge"

  key_name = aws_key_pair.fortresswaf.key_name

  vpc_security_group_ids = [
    aws_security_group.fortresswaf.id
  ]

  user_data = templatefile("${path.module}/user_data.sh", {
    fw_version = var.fortresswaf_version
    fw_api_key = var.fortresswaf_api_key
  })

  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 1
  }

  root_block_device {
    volume_type           = "gp3"
    volume_size           = 100
    encrypted             = true
    delete_on_termination = false
  }

  monitoring = true
}

# Auto Scaling Group
resource "aws_autoscaling_group" "fortresswaf" {
  name                = "fortresswaf-asg"
  vpc_zone_identifier = aws_subnet.private.*.id
  min_size            = 2
  max_size            = 10
  desired_capacity    = 3

  launch_template {
    id      = aws_launch_template.fortresswaf.id
    version = "$Latest"
  }

  health_check_type         = "ELB"
  health_check_grace_period = 300

  tag {
    key                 = "Name"
    value               = "fortresswaf"
    propagate_at_launch = true
  }

  tag {
    key                 = "Environment"
    value               = var.environment
    propagate_at_launch = true
  }
}

# Scaling Policies
resource "aws_autoscaling_policy" "fortresswaf_scale_up" {
  name                   = "fortresswaf-scale-up"
  scaling_adjustment     = 1
  adjustment_type        = "ChangeInCapacity"
  cooldown               = 300
  autoscaling_group_name = aws_autoscaling_group.fortresswaf.name
}

resource "aws_cloudwatch_metric_alarm" "fortresswaf_cpu_high" {
  alarm_name          = "fortresswaf-cpu-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "CPUUtilization"
  namespace           = "AWS/EC2"
  period              = 300
  statistic           = "Average"
  threshold           = 70

  dimensions = {
    AutoScalingGroupName = aws_autoscaling_group.fortresswaf.name
  }

  alarm_actions = [aws_autoscaling_policy.fortresswaf_scale_up.arn]
}
```

#### User Data Script

```bash
#!/bin/bash
set -e

# Install Docker
yum update -y
amazon-linux-extras install docker -y
systemctl enable docker
systemctl start docker

# Install Docker Compose
curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# Configure Docker daemon
mkdir -p /etc/docker
cat > /etc/docker/daemon.json << EOF
{
  "log-driver": "awslogs",
  "log-options": {
    "awslogs-group": "/fortresswaf/logs",
    "awslogs-region": "${aws_region}",
    "awslogs-stream-prefix": "fortresswaf"
  }
}
EOF

systemctl restart docker

# Pull FortressWAF
docker pull fortresswaf/fortresswaf:${fw_version}

# Write configuration
cat > /opt/fortresswaf/config.yaml << EOF
server:
  host: 0.0.0.0
  port: 8443

redis:
  host: ${redis_endpoint}
  port: 6379
  ssl: true

database:
  host: ${postgres_endpoint}
  port: 5432
  name: fortresswaf
  user: fortresswaf
  password: ${postgres_password}
  ssl_mode: require
EOF

# Start FortressWAF
docker run -d \
  --name fortresswaf \
  --restart unless-stopped \
  -p 8443:8443 \
  -v /opt/fortresswaf/config.yaml:/app/config/config.yaml:ro \
  fortresswaf/fortresswaf:${fw_version}
```

### AWS RDS PostgreSQL

```hcl
# RDS PostgreSQL for FortressWAF
resource "aws_db_instance" "fortresswaf" {
  identifier             = "fortresswaf-postgres"
  engine                 = "postgres"
  engine_version         = "15.3"
  instance_class         = "db.r6g.large"
  allocated_storage      = 100
  max_allocated_storage  = 500
  storage_encrypted      = true
  storage_type           = "gp3"

  db_name  = "fortresswaf"
  username = "fortresswaf"
  password = var.postgres_password

  vpc_security_group_ids = [aws_security_group.postgres.id]
  db_subnet_group_name   = aws_db_subnet_group.fortresswaf.name

  backup_retention_period = 30
  backup_window          = "03:00-04:00"
  maintenance_window     = "mon:04:00-mon:05:00"

  performance_insights_enabled = true
  monitoring_interval         = 60

  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]

  deletion_protection = true
  skip_final_snapshot = false
  final_snapshot_identifier = "fortresswaf-final-snapshot"
}
```

### AWS ElastiCache Redis

```hcl
# ElastiCache Redis Cluster
resource "aws_elasticache_cluster" "fortresswaf" {
  cluster_id           = "fortresswaf-redis"
  engine               = "redis"
  engine_version       = "7.0"
  node_type            = "cache.r6g.large"
  num_cache_nodes      = 3
  parameter_group_name = "default.redis7"
  port                 = 6379

  security_group_ids = [aws_security_group.redis.id]
  subnet_group_name  = aws_elasticache_subnet_group.fortresswaf.name

  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token_enabled         = true

  automatic_failover_enabled = true
  auto_minor_version_upgrade = true

  snapshot_retention_limit   = 30
  snapshot_window            = "03:00-05:00"
}
```

### AWS AMI Creation

```bash
# Create AMI from running instance
aws ec2 create-image \
  --instance-id i-1234567890abcdef0 \
  --name "fortresswaf-2.0.0" \
  --description "FortressWAF 2.0.0 AMI" \
  --no-reboot

# Share AMI with other accounts
aws ec2 modify-image-attribute \
  --image-id ami-1234567890abcdef0 \
  --attribute launchPermission \
  --operationType add \
  --user-ids 123456789012
```

## Google Cloud Platform (GCP)

### GCP Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                    GCP Project                                   │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                              VPC Network                                    │ │
│  │                                                                             │ │
│  │  ┌────────────────────┐         ┌────────────────────┐                    │ │
│  │  │    Zone: us-central1-a      │    Zone: us-central1-b      │            │ │
│  │  │                                                                 │            │ │
│  │  │  ┌──────────────┐ │         │  ┌──────────────┐ │                    │ │
│  │  │  │  Managed     │ │         │  │  Managed     │ │                    │ │
│  │  │  │  Instance    │ │◀───────▶│  │  Instance    │ │                    │ │
│  │  │  │  Group       │ │         │  │  Group       │ │                    │ │
│  │  │  └──────────────┘ │         │  └──────────────┘ │                    │ │
│  │  │         │          │         │         │          │                    │ │
│  │  │         ▼          │         │         ▼          │                    │ │
│  │  │  ┌──────────────┐ │         │  ┌──────────────┐ │                    │ │
│  │  │  │  FortressWAF │ │         │  │  FortressWAF │ │                    │ │
│  │  │  │  Container   │ │         │  │  Container   │ │                    │ │
│  │  │  └──────────────┘ │         │  └──────────────┘ │                    │ │
│  │  └────────────────────┘         └────────────────────┘                    │ │
│  │                                                                             │ │
│  │  ┌────────────────────────────────────────────────────────────────────┐   │ │
│  │  │                       Cloud SQL (PostgreSQL)                       │   │ │
│  │  └────────────────────────────────────────────────────────────────────┘   │ │
│  │                                                                             │ │
│  │  ┌────────────────────────────────────────────────────────────────────┐   │ │
│  │  │                        Memorystore (Redis)                         │   │ │
│  │  └────────────────────────────────────────────────────────────────────┘   │ │
│  │                                                                             │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### GCP Deployment with Container Optimized OS

```yaml
# compute.tf
# Managed Instance Group
resource "google_compute_instance_template" "fortresswaf" {
  name        = "fortresswaf-template"
  machine_type = "n2-standard-4"
  zone         = "us-central1-a"

  disk {
    source_image = "cos-cloud/global/images/family/cos-stable"
    auto_delete  = true
    boot         = true
  }

  network_interface {
    network    = google_compute_network.fortresswaf.id
    subnetwork = google_compute_subnetwork.fortresswaf.id
    access_config {
      // Public IP for testing, use Cloud NAT for production
    }
  }

  metadata = {
    google-logging-enabled = "true"
    startup-script = templatefile("${path.module}/startup.sh", {
      fw_version    = var.fortresswaf_version
      postgres_host = google_sql_database_instance.fortresswaf.ip_address
      redis_host    = google_redis_instance.fortresswaf.host
    })
  }

  tags = ["fortresswaf", "https-server"]

  service_account {
    scopes = ["cloud-platform"]
  }
}

resource "google_compute_health_check" "fortresswaf" {
  name               = "fortresswaf-health-check"
  check_interval_sec  = 30
  timeout_sec        = 10
  healthy_threshold   = 2
  unhealthy_threshold = 3

  https_health_check {
    port         = 8443
    request_path = "/health"
  }
}

resource "google_compute_region_instance_group_manager" "fortresswaf" {
  name               = "fortresswaf-migm"
  base_instance_name = "fortresswaf"
  region             = "us-central1"

  version {
    instance_template = google_compute_instance_template.fortresswaf.id
  }

  update_policy {
    type                  = "PROACTIVE"
    minimal_action        = "REPLACE"
    max_surge_fixed       = 1
    max_unavailable_fixed = 0
    min_ready_sec         = 60
  }

  target_pools        = [google_compute_target_pool.fortresswaf.id]
  health_check        = google_compute_health_check.fortresswaf.id
  wait_for_instances  = true
}

resource "google_compute_autoscaler" "fortresswaf" {
  name   = "fortresswaf-autoscaler"
  region = "us-central1"
  target = google_compute_region_instance_group_manager.fortresswaf.id

  autoscaling_policy {
    min_replicas    = 2
    max_replicas    = 10
    cooldown_period = 60

    cpu_utilization_target = 0.7
  }
}
```

### GCP Cloud SQL

```yaml
# cloudsql.tf
resource "google_sql_database_instance" "fortresswaf" {
  name             = "fortresswaf-postgres"
  database_version = "POSTGRES_15"
  region           = "us-central1"

  settings {
    tier = "db-n1-standard-4"

    availability_type = "REGIONAL"  # High availability
    disk_type        = "PD_SSD"
    disk_size        = 100
    disk_autoresize  = true

    ip_configuration {
      ipv4_enabled    = true
      private_network = google_compute_network.fortresswaf.id
      require_ssl     = true
    }

    backup_configuration {
      enabled                        = true
      start_time                      = "03:00"
      point_in_time_recovery_enabled = true
      backup_retention_settings {
        retained_backups = 30
        retention_unit   = "COUNT"
      }
    }

    maintenance_window {
      day          = 7   # Sunday
      hour         = 4   # 4 AM
      update_track = "stable"
    }

    insights_config {
      query_insights_enabled  = true
      query_string_length     = 1024
      record_application_tags = true
      record_client_address   = false
    }
  }

  deletion_protection = true
}
```

### GCP Memorystore

```yaml
# memorystore.tf
resource "google_redis_instance" "fortresswaf" {
  name           = "fortresswaf-redis"
  memory_size_gb = 4
  redis_version  = "redis_7_0"
  location_id    = "us-central1"
  tier           = "STANDARD_HA"

  connectivity {
    network    = google_compute_network.fortresswaf.id
    direct_peering_enabled = false
    private_service_connect_enabled = true
  }

  transit_encryption_mode = "SERVER_AUTHENTICATION"

  auth_enabled = true
}
```

## Microsoft Azure

### Azure Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                  Azure Region                                   │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                           Virtual Network (10.0.0.0/16)                      │ │
│  │                                                                             │ │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │ │
│  │  │                        Availability Zone 1                           │   │ │
│  │  │  ┌─────────────────────────────────────────────────────────────┐   │   │ │
│  │  │  │              Virtual Machine Scale Set                      │   │   │ │
│  │  │  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐            │   │ │
│  │  │  │  │  Fortress   │ │  Fortress   │ │  Fortress   │            │   │ │
│  │  │  │  │  WAF VM    │ │  WAF VM    │ │  WAF VM    │            │   │ │
│  │  │  │  └─────────────┘ └─────────────┘ └─────────────┘            │   │ │
│  │  │  └─────────────────────────────────────────────────────────────┘   │   │ │
│  │  └─────────────────────────────────────────────────────────────────────┘   │ │
│  │                                                                             │ │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │ │
│  │  │                      Azure Database for PostgreSQL                    │   │ │
│  │  │                      (Flexible Server, HA)                           │   │ │
│  │  └─────────────────────────────────────────────────────────────────────┘   │ │
│  │                                                                             │ │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │ │
│  │  │                         Azure Cache for Redis                          │   │ │
│  │  │                         (Premium Tier, Clustering)                    │   │ │
│  │  └─────────────────────────────────────────────────────────────────────┘   │ │
│  │                                                                             │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Azure Virtual Machine Scale Set

```yaml
# azuredeploy.tf
resource "azurerm_virtual_network" "fortresswaf" {
  name                = "fortresswaf-vnet"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.fortresswaf.location
  resource_group_name = azurerm_resource_group.fortresswaf.name
}

resource "azurerm_subnet" "fortresswaf" {
  name                 = "fortresswaf-subnet"
  resource_group_name  = azurerm_resource_group.fortresswaf.name
  virtual_network_name = azurerm_virtual_network.fortresswaf.name
  address_prefixes     = ["10.0.1.0/24"]
}

resource "azurerm_bastion_subnet" "fortresswaf" {
  name                 = "AzureBastionSubnet"
  resource_group_name  = azurerm_resource_group.fortresswaf.name
  virtual_network_name = azurerm_virtual_network.fortresswaf.name
  address_prefixes     = ["10.0.0.0/24"]
}

resource "azurerm_virtual_machine_scale_set" "fortresswaf" {
  name                = "fortresswaf-vmss"
  location            = azurerm_resource_group.fortresswaf.location
  resource_group_name = azurerm_resource_group.fortresswaf.name
  sku                 = "Standard_D4s_v3"
  instances           = 3
  admin_username      = "fortresswaf"

  admin_ssh_key {
    username   = "fortresswaf"
    public_key = tls_private_key.fortresswaf.public_key_openssh
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts-gen2"
    version   = "latest"
  }

  os_disk {
    storage_account_type = "StandardSSD_LRS"
    caching              = "ReadWrite"
  }

  network_interface {
    name                      = "fortresswaf-nic"
    primary                  = true
    subnet_id                = azurerm_subnet.fortresswaf.id
    ip_configuration {
      name      = "internal"
      primary   = true
      subnet_id = azurerm_subnet.fortresswaf.id
      load_balancer_backend_address_pools_ids = [
        azurerm_lb_backend_address_pool.fortresswaf.id
      ]
    }
  }

  custom_data = base64encode(templatefile("${path.module}/cloud-init.yaml", {
    fw_version    = var.fortresswaf_version
    postgres_host = azurerm_postgresql_flexible_server.fortresswaf.fqdn
    redis_host    = azurerm_redis_cache.fortresswaf.host_name
  }))

  automatic_os_upgrade_policy {
    disable_automatic_rollback = false
    enable_automatic_upgrades  = true
  }

  rolling_upgrade_policy {
    max_batch_instance_percent = 20
    max_unhealthy_instance_percent = 20
    max_unhealthy_upgraded_instance_percent = 20
    pause_time_between_batches = "PT0S"
  }

  upgrade_mode = "Automatic"
}

resource "azurerm_lb" "fortresswaf" {
  name                = "fortresswaf-lb"
  location            = azurerm_resource_group.fortresswaf.location
  resource_group_name = azurerm_resource_group.fortresswaf.name
  sku                 = "Standard"

  frontend_ip_configuration {
    name                 = "PublicIPAddress"
    public_ip_address_id = azurerm_public_ip.fortresswaf.id
  }

  backend_address_pool {
    name = "fortresswaf-backend-pool"
  }

  probe {
    name                = "https-probe"
    protocol            = "Https"
    port                = 8443
    path                = "/health"
    interval_in_seconds = 30
    number_of_probes     = 3
  }

  load_balancing_rule {
    name                           = "https-rule"
    protocol                       = "Tcp"
    frontend_port                  = 443
    backend_port                   = 8443
    frontend_ip_configuration_name = "PublicIPAddress"
    backend_address_pool_name      = "fortresswaf-backend-pool"
    probe_id                       = azurerm_lb_probe.fortresswaf.id
  }
}
```

### Azure Database for PostgreSQL

```yaml
# postgresql.tf
resource "azurerm_postgresql_flexible_server" "fortresswaf" {
  name                = "fortresswaf-postgres"
  location            = azurerm_resource_group.fortresswaf.location
  resource_group_name = azurerm_resource_group.fortresswaf.name

  sku_name   = "B_Standard_B4ms"
  tier       = "Burstable"
  version    = "15"
  storage_mb = 32768

  admin_username  = "fortresswaf"
  password        = var.postgres_password
  authentication  = {
    password_auth_type = "required"
    active_directory_auth = "Enabled"
    tenant_id = data.azurerm_client_config.current.tenant_id
  }

  backup_retention_days = 30
  geo_redundant_backup  = "Enabled"

  high_availability {
    mode                      = "ZoneRedundant"
    standby_availability_zone = "2"
  }

  network {
    private_dns_zone_id = azurerm_private_dns_zone.fortresswaf.id
    subnet_id            = azurerm_subnet.postgres.id
  }

  maintenance_window {
    day_of_week  = 0
    start_hour   = 4
    start_minute = 0
  }
}
```

### Azure Cache for Redis

```yaml
# redis.tf
resource "azurerm_redis_cache" "fortresswaf" {
  name                = "fortresswaf-redis"
  location            = azurerm_resource_group.fortresswaf.location
  resource_group_name = azurerm_resource_group.fortresswaf.name
  sku_name            = "Premium"
  family             = "P"
  capacity           = 2

  enable_non_ssl_port           = false
  minimum_tls_version           = "1.2"

  redis_configuration {
    maxmemory_reserved      = 100
    maxmemory_delta        = "50"
    maxmemory_policy       = "allkeys-lru"
    rdb_backup_enabled     = true
    rdb_backup_frequency   = 60
    rdb_backup_max_cooldown = 720
  }

  private_static_ip_address = "10.0.1.10"
  subnet_id                  = azurerm_subnet.redis.id

  shard_count = 3

  tags = {
    Environment = var.environment
  }
}
```

## Cloud-Specific Configurations

### AWS-Specific Settings

```yaml
# config.yaml additions for AWS
server:
  # Use instance metadata service
  imds:
    enabled: true
    token_ttl: 21600

cloud:
  provider: aws
  region: us-east-1
  # Use IAM roles for credentials
  iam_role: true
```

### GCP-Specific Settings

```yaml
# config.yaml additions for GCP
server:
  # Use GCP metadata service
  gcp_metadata:
    enabled: true

cloud:
  provider: gcp
  project: my-project-id
  # Use service account for credentials
  service_account: default
```

### Azure-Specific Settings

```yaml
# config.yaml additions for Azure
server:
  # Use Azure Instance Metadata Service
  azure:
    enabled: true

cloud:
  provider: azure
  subscription_id: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
  tenant_id: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
  # Use managed identity
  managed_identity: true
```

## Auto-Scaling Configuration

### AWS Auto Scaling

```yaml
# CloudWatch scaling policy
{
  "MetricName": "CPUUtilization",
  "Namespace": "AWS/EC2",
  "Statistic": "Average",
  "Period": 300,
  "EvaluationPeriods": 2,
  "Threshold": 70,
  "ComparisonOperator": "GreaterThanThreshold",
  "ScalingAdjustment": 1
}
```

### GCP Autoscaler

```yaml
# Managed instance group autoscaler
{
  "autoscalingMode": "SCALE_UP",
  "coolDownPeriod": "60s",
  "cpuUtilization": {
    "utilizationTarget": 0.7
  },
  "maxNumReplicas": 10,
  "minNumReplicas": 2
}
```

### Azure Autoscale Settings

```yaml
# Azure Monitor autoscale
{
  "name": "fortresswaf-autoscale",
  "type": "Microsoft.Insights/autoScaleSettings",
  "properties": {
    "enabled": true,
    "targetResourceUri": "/subscriptions/xxx/resourceGroups/xxx/providers/microsoft.compute/virtualMachineScaleSets/fortresswaf",
    "profiles": [{
      "name": "default",
      "capacity": {
        "minimum": "2",
        "maximum": "10",
        "default": "3"
      },
      "rules": [{
        "metricTrigger": {
          "metricName": "Percentage CPU",
          "operator": "GreaterThan",
          "threshold": 70,
          "scaleDirection": "Increase",
          "scaleAction": { "value": "1", "type": "ChangeCount" }
        }
      }]
    }]
  }
}
```
