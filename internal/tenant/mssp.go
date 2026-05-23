package tenant

type MSSPManager struct {
	tenantManager *TenantManager
}

type PartnerMetrics struct {
	TotalTenants       int     `json:"total_tenants"`
	ActiveTenants      int     `json:"active_tenants"`
	SuspendedTenants   int     `json:"suspended_tenants"`
	TotalRequests      int64   `json:"total_requests"`
	TotalBlocked      int64   `json:"total_blocked"`
	TotalRevenue       float64 `json:"total_revenue"`
	MRR                float64 `json:"mrr"`
	ChurnRate          float64 `json:"churn_rate"`
	TopTrafficTenant   string  `json:"top_traffic_tenant"`
	TopRevenueTenant   string  `json:"top_revenue_tenant"`
}

func (m *MSSPManager) GetPartnerMetrics(partnerID string) (*PartnerMetrics, error) {
	children := m.tenantManager.List(partnerID)
	
	metrics := &PartnerMetrics{
		TotalTenants: len(children),
	}
	
	for _, child := range children {
		if child.Status == "active" {
			metrics.ActiveTenants++
		} else if child.Status == "suspended" {
			metrics.SuspendedTenants++
		}
	}
	
	return metrics, nil
}

type BulkRuleDeploy struct {
	RuleID     string   `json:"rule_id"`
	TenantIDs  []string `json:"tenant_ids"`
	Action     string   `json:"action"`
}

func (m *MSSPManager) BulkDeployRule(deploy BulkRuleDeploy) map[string]error {
	results := make(map[string]error)
	
	for _, tenantID := range deploy.TenantIDs {
		err := m.tenantManager.ValidateQuota(tenantID, "rules", 0, 1)
		if err != nil {
			results[tenantID] = err
			continue
		}
		results[tenantID] = nil
	}
	
	return results
}