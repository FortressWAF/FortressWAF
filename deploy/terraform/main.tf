# Protected site definition
resource "fortresswaf_site" "api" {
  name        = "api-gateway"
  domains     = ["api.example.com", "api.internal.example.com"]
  upstreams   = ["http://backend-api:8080", "http://backend-api:8081"]
  tls_enabled = true
  auto_cert   = true
  auto_cert_email = "admin@example.com"

  request_modifiers {
    add_headers = {
      X-FortressWAF = "true"
    }
    remove_headers = ["X-Internal-Auth"]
  }

  response_modifiers {
    add_headers = {
      X-Content-Type-Options    = "nosniff"
      X-Frame-Options           = "DENY"
      Strict-Transport-Security = "max-age=31536000; includeSubDomains"
    }
    remove_headers = ["Server", "X-Powered-By"]
  }
}

resource "fortresswaf_site" "dashboard" {
  name        = "admin-dashboard"
  domains     = ["dashboard.example.com"]
  upstreams   = ["http://dashboard-app:3000"]
  tls_enabled = true
  auto_cert   = true
  auto_cert_email = "admin@example.com"

  rate_limit {
    enabled             = true
    requests_per_second = 50
    burst               = 100
    per_ip {
      enabled             = true
      requests_per_second = 5
      burst               = 10
    }
  }
}

# OWASP CRS rules enabled
resource "fortresswaf_rule_set" "owasp_crs" {
  name        = "owasp-crs-core"
  description = "OWASP Core Rule Set baseline"
  enabled     = true
  severity    = "critical"
  mode        = "block"

  rule_groups = [
    "sqli",
    "xss",
    "rce",
    "lfi",
    "protocol-attacks",
    "scanner-detection",
    "php-attacks",
    "java-attacks",
  ]
}

# SQL injection rule
resource "fortresswaf_rule" "sqli_union" {
  rule_set   = fortresswaf_rule_set.owasp_crs.id
  rule_id    = "FORTRESS-SQLI-001"
  name       = "SQL Injection - UNION SELECT"
  severity   = "critical"
  tags       = ["sqli", "owasp-a03", "pci-dss"]
  action     = "block"
  log        = true
  alert      = true

  condition {
    any = [
      {
        field    = "request.body"
        operator = "regex"
        pattern  = "(?i)(union\\s+(all\\s+)?select)"
      },
      {
        field    = "request.query"
        operator = "regex"
        pattern  = "(?i)(union\\s+(all\\s+)?select)"
      },
    ]
  }

  response {
    code = 403
    body = "{\"error\":\"Blocked by FortressWAF\",\"ref\":\"FORTRESS-SQLI-001\"}"
  }
}

# XSS rule
resource "fortresswaf_rule" "xss_script" {
  rule_set   = fortresswaf_rule_set.owasp_crs.id
  rule_id    = "FORTRESS-XSS-001"
  name       = "Cross-Site Scripting - Script Tag"
  severity   = "critical"
  tags       = ["xss", "owasp-a07", "pci-dss"]
  action     = "block"
  log        = true
  alert      = true

  condition {
    any = [
      {
        field    = "request.body"
        operator = "regex"
        pattern  = "(?i)<script[^>]*>.*?<\\/script>"
      },
      {
        field    = "request.query"
        operator = "regex"
        pattern  = "(?i)<script[^>]*>.*?<\\/script>"
      },
      {
        field    = "request.headers"
        operator = "regex"
        pattern  = "(?i)<script[^>]*>.*?<\\/script>"
      },
    ]
  }

  response {
    code = 403
    body = "{\"error\":\"Blocked by FortressWAF\",\"ref\":\"FORTRESS-XSS-001\"}"
  }
}

# Rate limiting policy
resource "fortresswaf_rate_limit_policy" "api_login" {
  name        = "api-login-rate-limit"
  description = "Rate limiting for authentication endpoints"
  enabled     = true
  scope       = "per_ip"
  rps         = 5
  burst       = 10

  match {
    path    = "/api/v1/login"
    methods = ["POST"]
  }

  match {
    path    = "/api/v1/register"
    methods = ["POST"]
  }

  match {
    path    = "/api/v1/password-reset"
    methods = ["POST"]
  }

  response {
    code    = 429
    body    = "{\"error\":\"Rate limit exceeded\",\"retry_after\":%d}"
    headers = {
      Retry-After = "%d"
    }
  }
}

# Global rate limiting
resource "fortresswaf_rate_limit_policy" "global" {
  name        = "global-rate-limit"
  description = "Global per-IP rate limit"
  enabled     = true
  scope       = "per_ip"
  rps         = 100
  burst       = 200
}

# Virtual patch for Log4Shell
resource "fortresswaf_virtual_patch" "log4shell" {
  name        = "Log4Shell Virtual Patch"
  cve         = "CVE-2021-44228"
  severity    = "critical"
  enabled     = true
  action      = "block"
  description = "Mitigates Apache Log4j2 JNDI injection vulnerability"

  affected_paths = ["*"]

  condition {
    any = [
      {
        field    = "request.body"
        operator = "regex"
        pattern  = "(?i)\\$\\{jndi:(ldap|ldaps|rmi|dns|nis|iiop|corba|nds|http):"
      },
      {
        field    = "request.query"
        operator = "regex"
        pattern  = "(?i)\\$\\{jndi:(ldap|ldaps|rmi|dns|nis|iiop|corba|nds|http):"
      },
      {
        field    = "request.headers"
        operator = "regex"
        pattern  = "(?i)\\$\\{jndi:(ldap|ldaps|rmi|dns|nis|iiop|corba|nds|http):"
      },
    ]
  }

  response {
    code = 403
    body = "{\"error\":\"Blocked by FortressWAF\",\"ref\":\"CVE-2021-44228\"}"
  }
}

# Virtual patch for Spring4Shell
resource "fortresswaf_virtual_patch" "spring4shell" {
  name        = "Spring4Shell Virtual Patch"
  cve         = "CVE-2022-22965"
  severity    = "critical"
  enabled     = true
  action      = "block"
  description = "Mitigates Spring Framework RCE vulnerability"

  affected_paths = ["*"]

  condition {
    any = [
      {
        field    = "request.query"
        operator = "regex"
        pattern  = "(?i)class\\.module\\.classLoader"
      },
      {
        field    = "request.body"
        operator = "regex"
        pattern  = "(?i)class\\.module\\.classLoader"
      },
    ]
  }

  response {
    code = 403
    body = "{\"error\":\"Blocked by FortressWAF\",\"ref\":\"CVE-2022-22965\"}"
  }
}

# IP allowlist for internal tools
resource "fortresswaf_ip_list" "internal_monitoring" {
  name        = "internal-monitoring"
  description = "Internal monitoring IPs"
  type        = "allowlist"
  entries     = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
}

# IP blocklist for known bad actors
resource "fortresswaf_ip_list" "blocklist" {
  name        = "known-bad-actors"
  description = "Known malicious IP ranges"
  type        = "blocklist"
  entries     = var.blocklist_ips
}

# Alerting configuration
resource "fortresswaf_alert_config" "slack_alerts" {
  name          = "slack-security-alerts"
  enabled       = true
  min_severity  = "high"
  cooldown      = 300

  slack {
    enabled    = true
    webhook_url = var.slack_webhook_url
    channel    = "#security-alerts"
    username   = "FortressWAF"
  }
}

# Custom rule for blocking specific paths
resource "fortresswaf_rule" "block_admin" {
  name     = "Block Admin Exposure"
  rule_id  = "FORTRESS-CUSTOM-001"
  severity = "high"
  tags     = ["custom", "security"]
  action   = "block"
  log      = true
  alert    = true

  condition {
    any = [
      {
        field    = "request.path"
        operator = "starts_with"
        pattern  = "/wp-admin"
      },
      {
        field    = "request.path"
        operator = "starts_with"
        pattern  = "/administrator"
      },
      {
        field    = "request.path"
        operator = "starts_with"
        pattern  = "/phpmyadmin"
      },
    ]
  }

  response {
    code = 404
    body = "{\"error\":\"Not Found\"}"
  }
}
