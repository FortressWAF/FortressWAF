package tenant

import (
	"fmt"
	"strings"
)

type IsolationManager struct {
	tenantManager *TenantManager
}

func NewIsolationManager(tm *TenantManager) *IsolationManager {
	return &IsolationManager{tenantManager: tm}
}

func (im *IsolationManager) GetPostgresSchema(tenantID string) string {
	safe := sanitizeIdentifier(tenantID)
	return fmt.Sprintf("tenant_%s", safe)
}

func (im *IsolationManager) GetRedisNamespace(tenantID string) string {
	return fmt.Sprintf("tenant:%s:", tenantID)
}

func (im *IsolationManager) GetS3Bucket(tenantID string) string {
	return fmt.Sprintf("fortresswaf-logs-%s", tenantID)
}

func (im *IsolationManager) ValidateTenantAccess(requestingTenantID, resourceTenantID string) error {
	if requestingTenantID == resourceTenantID {
		return nil
	}
	
	reqTenant, err := im.tenantManager.Get(requestingTenantID)
	if err != nil {
		return err
	}
	
	if reqTenant.Tier == "enterprise" && reqTenant.Branding.ProductName != "FortressWAF" {
		resourceTenant, err := im.tenantManager.Get(resourceTenantID)
		if err != nil {
			return err
		}
		if resourceTenant.ParentID == requestingTenantID {
			return nil
		}
	}
	
	return fmt.Errorf("access denied: tenant %s cannot access tenant %s resources", requestingTenantID, resourceTenantID)
}

func (im *IsolationManager) BuildScopedQuery(tenantID, baseQuery string) string {
	schema := im.GetPostgresSchema(tenantID)
	return fmt.Sprintf("SET search_path TO %s; %s", schema, baseQuery)
}

func (im *IsolationManager) GetAPIKeyScope(apiKey string) (string, error) {
	parts := strings.Split(apiKey, "-")
	if len(parts) >= 2 && parts[0] == "fw" {
		return parts[1], nil
	}
	return "", fmt.Errorf("invalid API key format")
}

func sanitizeIdentifier(id string) string {
	var result strings.Builder
	for _, c := range id {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result.WriteRune(c)
		} else {
			result.WriteRune('_')
		}
	}
	return result.String()
}