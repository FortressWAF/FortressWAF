variable "fortress_endpoint" {
  description = "FortressWAF admin API endpoint"
  type        = string
  default     = "https://waf.example.com:8443"
}

variable "fortress_api_key" {
  description = "API key for FortressWAF admin authentication"
  type        = string
  sensitive   = true
}

variable "slack_webhook_url" {
  description = "Slack webhook URL for security alerts"
  type        = string
  default     = ""
  sensitive   = true
}

variable "blocklist_ips" {
  description = "List of IP ranges to permanently block"
  type        = list(string)
  default     = []
}

variable "environment" {
  description = "Deployment environment tag"
  type        = string
  default     = "production"
}

variable "enable_ml_detection" {
  description = "Enable ML-based threat detection"
  type        = bool
  default     = true
}

variable "default_action" {
  description = "Default action when no rule matches"
  type        = string
  default     = "allow"
}

variable "log_level" {
  description = "Logging verbosity level"
  type        = string
  default     = "info"
}

variable "rate_limit_global_rps" {
  description = "Global rate limit requests per second"
  type        = number
  default     = 1000
}

variable "rate_limit_global_burst" {
  description = "Global rate limit burst size"
  type        = number
  default     = 2000
}

variable "rate_limit_per_ip_rps" {
  description = "Per-IP rate limit requests per second"
  type        = number
  default     = 100
}

variable "rate_limit_per_ip_burst" {
  description = "Per-IP rate limit burst size"
  type        = number
  default     = 200
}

variable "tls_min_version" {
  description = "Minimum TLS version"
  type        = string
  default     = "1.2"
}

variable "anomaly_threshold" {
  description = "Anomaly score threshold (0 to disable anomaly scoring)"
  type        = number
  default     = 10
}

variable "enable_pci_dss" {
  description = "Enable PCI DSS compliance mode"
  type        = bool
  default     = false
}

variable "enable_gdpr" {
  description = "Enable GDPR compliance mode"
  type        = bool
  default     = false
}

variable "log_retention_days" {
  description = "Audit log retention period in days"
  type        = number
  default     = 90
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default = {
    Terraform   = "true"
    Environment = "production"
  }
}
