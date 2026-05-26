-- ============================================================================
-- FortressWAF - PostgreSQL Initialization Script
-- ============================================================================
-- This script runs automatically on first database creation.
-- It sets up the core schema, extensions, and indexes.
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- ============================================================================
-- Core Tables
-- ============================================================================

-- Tenants (multi-tenant support)
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    slug VARCHAR(255) NOT NULL UNIQUE,
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Users
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(50) NOT NULL DEFAULT 'viewer',
    mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_secret VARCHAR(255),
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

-- Sites (protected upstream applications)
CREATE TABLE IF NOT EXISTS sites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    domains TEXT[] NOT NULL DEFAULT '{}',
    upstreams TEXT[] NOT NULL DEFAULT '{}',
    tls_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    tls_cert TEXT,
    tls_key TEXT,
    auto_cert BOOLEAN NOT NULL DEFAULT FALSE,
    settings JSONB DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Security events / audit log
CREATE TABLE IF NOT EXISTS security_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    site_id UUID REFERENCES sites(id) ON DELETE SET NULL,
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'info',
    source_ip INET NOT NULL,
    source_country VARCHAR(10),
    request_method VARCHAR(10),
    request_path TEXT,
    request_headers JSONB,
    request_body TEXT,
    rule_id VARCHAR(255),
    rule_name VARCHAR(255),
    action VARCHAR(20) NOT NULL DEFAULT 'block',
    score NUMERIC(5,2) DEFAULT 0,
    ml_scores JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Blocked IPs
CREATE TABLE IF NOT EXISTS blocked_ips (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    ip_address INET NOT NULL,
    reason VARCHAR(255),
    blocked_by VARCHAR(50) NOT NULL DEFAULT 'manual',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, ip_address)
);

-- Whitelisted IPs
CREATE TABLE IF NOT EXISTS whitelisted_ips (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    ip_address INET NOT NULL,
    description VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, ip_address)
);

-- Custom rules
CREATE TABLE IF NOT EXISTS custom_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rule_definition JSONB NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'medium',
    action VARCHAR(20) NOT NULL DEFAULT 'block',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    priority INT NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- API keys
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    scopes TEXT[] NOT NULL DEFAULT '{"read"}',
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Compliance audit trail
CREATE TABLE IF NOT EXISTS audit_trail (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id VARCHAR(255),
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- Indexes for Performance
-- ============================================================================

-- Security events: query by time range + tenant
CREATE INDEX IF NOT EXISTS idx_security_events_tenant_created
    ON security_events (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_security_events_source_ip
    ON security_events (source_ip);
CREATE INDEX IF NOT EXISTS idx_security_events_severity
    ON security_events (severity, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_security_events_rule
    ON security_events (rule_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_security_events_type
    ON security_events (event_type, created_at DESC);

-- Blocked IPs: quick lookup
CREATE INDEX IF NOT EXISTS idx_blocked_ips_ip
    ON blocked_ips (ip_address);
CREATE INDEX IF NOT EXISTS idx_blocked_ips_expiry
    ON blocked_ips (expires_at)
    WHERE expires_at IS NOT NULL;

-- Whitelisted IPs: quick lookup
CREATE INDEX IF NOT EXISTS idx_whitelisted_ips_ip
    ON whitelisted_ips (ip_address);

-- Users: auth lookups
CREATE INDEX IF NOT EXISTS idx_users_email
    ON users (email);

-- Sites: domain lookup
CREATE INDEX IF NOT EXISTS idx_sites_domains
    ON sites USING GIN (domains);

-- Audit trail: compliance queries
CREATE INDEX IF NOT EXISTS idx_audit_trail_tenant_created
    ON audit_trail (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_trail_action
    ON audit_trail (action, created_at DESC);

-- ============================================================================
-- Partitioning for security_events (optional, for high-volume deployments)
-- Uncomment for time-based partitioning
-- ============================================================================
-- CREATE TABLE security_events_partitioned (
--     LIKE security_events INCLUDING ALL
-- ) PARTITION BY RANGE (created_at);

-- ============================================================================
-- Default Data
-- ============================================================================

-- Insert default tenant
INSERT INTO tenants (name, slug, plan, status)
VALUES ('Default', 'default', 'enterprise', 'active')
ON CONFLICT (slug) DO NOTHING;

-- Insert default admin user (password: admin - CHANGE THIS!)
-- bcrypt hash of 'admin'
INSERT INTO users (tenant_id, email, password_hash, name, role)
SELECT t.id, 'admin@fortresswaf.local', '$2a$12$LQv3c1yqBo9SkvXS7QTJPeNgH.4v4MRRwOfE.FZQ3.BtP./.67RqG', 'Admin', 'admin'
FROM tenants t
WHERE t.slug = 'default'
ON CONFLICT (tenant_id, email) DO NOTHING;

-- ============================================================================
-- Functions
-- ============================================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply triggers
DO $$
DECLARE
    tbl TEXT;
BEGIN
    FOR tbl IN SELECT unnest(ARRAY['tenants', 'users', 'sites', 'custom_rules'])
    LOOP
        EXECUTE format('
            DROP TRIGGER IF EXISTS trigger_update_%I_updated_at ON %I;
            CREATE TRIGGER trigger_update_%I_updated_at
                BEFORE UPDATE ON %I
                FOR EACH ROW
                EXECUTE FUNCTION update_updated_at_column();
        ', tbl, tbl, tbl, tbl);
    END LOOP;
END;
$$;

-- Cleanup expired blocked IPs
CREATE OR REPLACE FUNCTION cleanup_expired_blocked_ips()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM blocked_ips
    WHERE expires_at IS NOT NULL AND expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
