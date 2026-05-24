package tenant

import (
	"fmt"
	"sync"
	"time"
)

type Tenant struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Org       string         `json:"org"`
	Status    string         `json:"status"` // active, suspended, canceled
	Tier      string         `json:"tier"`
	Plan      string         `json:"plan"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Settings  TenantSettings `json:"settings"`
	Quotas    Quotas         `json:"quotas"`
	Branding  Branding       `json:"branding"`
	ParentID  string         `json:"parent_id,omitempty"` // for MSSP hierarchy
	Tags      []string       `json:"tags"`
}

type TenantSettings struct {
	Timezone           string `json:"timezone"`
	Language           string `json:"language"`
	DateFormat         string `json:"date_format"`
	LogRetentionDays   int    `json:"log_retention_days"`
	MaxLogAge          int    `json:"max_log_age"`
	SIEMExport         bool   `json:"siem_export"`
	TwoFactorAuth      bool   `json:"two_factor_auth"`
	SessionTimeoutMins int    `json:"session_timeout_mins"`
}

type Quotas struct {
	MaxSites          int   `json:"max_sites"`
	MaxRules          int   `json:"max_rules"`
	MaxUsers          int   `json:"max_users"`
	MaxAPIKeys        int   `json:"max_api_keys"`
	MaxAPICallsPerDay int64 `json:"max_api_calls_per_day"`
	MaxStorageGB      int   `json:"max_storage_gb"`
}

type Branding struct {
	ProductName     string `json:"product_name"`
	LogoURL         string `json:"logo_url"`
	FaviconURL      string `json:"favicon_url"`
	PrimaryColor    string `json:"primary_color"`
	SecondaryColor  string `json:"secondary_color"`
	DashboardDomain string `json:"dashboard_domain"`
	CustomEmailFrom string `json:"custom_email_from"`
	SupportEmail    string `json:"support_email"`
	SupportURL      string `json:"support_url"`
	HidePoweredBy   bool   `json:"hide_powered_by"`
	TermsURL        string `json:"terms_url"`
	PrivacyURL      string `json:"privacy_url"`
}

type TenantManager struct {
	mu      sync.RWMutex
	tenants map[string]*Tenant
}

func NewTenantManager() *TenantManager {
	return &TenantManager{
		tenants: make(map[string]*Tenant),
	}
}

func (tm *TenantManager) Create(t *Tenant) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if t.ID == "" {
		t.ID = fmt.Sprintf("tenant-%d", time.Now().UnixNano())
	}
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	t.Status = "active"

	tm.tenants[t.ID] = t
	return nil
}

func (tm *TenantManager) Get(id string) (*Tenant, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if t, ok := tm.tenants[id]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("tenant not found: %s", id)
}

func (tm *TenantManager) List(parentID string) []*Tenant {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var result []*Tenant
	for _, t := range tm.tenants {
		if parentID == "" && t.ParentID == "" {
			result = append(result, t)
		} else if t.ParentID == parentID {
			result = append(result, t)
		}
	}
	return result
}

func (tm *TenantManager) Update(t *Tenant) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, ok := tm.tenants[t.ID]; !ok {
		return fmt.Errorf("tenant not found: %s", t.ID)
	}
	t.UpdatedAt = time.Now()
	tm.tenants[t.ID] = t
	return nil
}

func (tm *TenantManager) Delete(id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, ok := tm.tenants[id]; !ok {
		return fmt.Errorf("tenant not found: %s", id)
	}
	delete(tm.tenants, id)
	return nil
}

func (tm *TenantManager) Suspend(id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if t, ok := tm.tenants[id]; ok {
		t.Status = "suspended"
		t.UpdatedAt = time.Now()
		return nil
	}
	return fmt.Errorf("tenant not found: %s", id)
}

func (tm *TenantManager) ValidateQuota(tenantID string, resource string, current, limit int) error {
	tm.mu.RLock()
	t, ok := tm.tenants[tenantID]
	tm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("tenant not found")
	}

	switch resource {
	case "sites":
		if current >= t.Quotas.MaxSites && t.Quotas.MaxSites != -1 {
			return fmt.Errorf("site quota exceeded: %d/%d", current, t.Quotas.MaxSites)
		}
	case "rules":
		if current >= t.Quotas.MaxRules && t.Quotas.MaxRules != -1 {
			return fmt.Errorf("rule quota exceeded: %d/%d", current, t.Quotas.MaxRules)
		}
	case "users":
		if current >= t.Quotas.MaxUsers && t.Quotas.MaxUsers != -1 {
			return fmt.Errorf("user quota exceeded: %d/%d", current, t.Quotas.MaxUsers)
		}
	}
	return nil
}

func (tm *TenantManager) GetBranding(tenantID string) *Branding {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if t, ok := tm.tenants[tenantID]; ok {
		return &t.Branding
	}
	return nil
}
