package billing

import (
	"fmt"
)

type Features struct {
	MaxSites          int
	MaxRules          int
	MLEngine          bool
	MultiTenant       bool
	FIPSMode          bool
	ComplianceReports bool
	APIProtection     bool
	DLP               bool
	BotProtection     bool
	VirtualPatching   bool
	AdvancedAnalytics bool
	PrioritySupport   bool
	SLAMinutes        int
	MaxTenants        int
	MaxAPICallsPerDay int
}

var TierFeatures = map[string]Features{
	"community": {
		MaxSites:          1,
		MaxRules:          100,
		MLEngine:          false,
		MultiTenant:       false,
		FIPSMode:          false,
		ComplianceReports: false,
		APIProtection:     true,
		DLP:               false,
		BotProtection:     true,
		VirtualPatching:   true,
		AdvancedAnalytics: false,
		PrioritySupport:   false,
		SLAMinutes:        0,
		MaxTenants:        0,
		MaxAPICallsPerDay: 10000,
	},
	"starter": {
		MaxSites:          5,
		MaxRules:          500,
		MLEngine:          false,
		MultiTenant:       false,
		FIPSMode:          false,
		ComplianceReports: false,
		APIProtection:     true,
		DLP:               false,
		BotProtection:     true,
		VirtualPatching:   true,
		AdvancedAnalytics: false,
		PrioritySupport:   false,
		SLAMinutes:        0,
		MaxTenants:        0,
		MaxAPICallsPerDay: 50000,
	},
	"professional": {
		MaxSites:          25,
		MaxRules:          -1,
		MLEngine:          true,
		MultiTenant:       false,
		FIPSMode:          false,
		ComplianceReports: true,
		APIProtection:     true,
		DLP:               true,
		BotProtection:     true,
		VirtualPatching:   true,
		AdvancedAnalytics: true,
		PrioritySupport:   false,
		SLAMinutes:        240,
		MaxTenants:        0,
		MaxAPICallsPerDay: -1,
	},
	"enterprise": {
		MaxSites:          -1,
		MaxRules:          -1,
		MLEngine:          true,
		MultiTenant:       true,
		FIPSMode:          true,
		ComplianceReports: true,
		APIProtection:     true,
		DLP:               true,
		BotProtection:     true,
		VirtualPatching:   true,
		AdvancedAnalytics: true,
		PrioritySupport:   true,
		SLAMinutes:        60,
		MaxTenants:        -1,
		MaxAPICallsPerDay: -1,
	},
}

func GetFeatures(tier string) Features {
	if f, ok := TierFeatures[tier]; ok {
		return f
	}
	return TierFeatures["community"]
}

func CheckFeature(tier, featureName string) bool {
	f := GetFeatures(tier)
	switch featureName {
	case "ml_engine":
		return f.MLEngine
	case "multi_tenant":
		return f.MultiTenant
	case "fips_mode":
		return f.FIPSMode
	case "compliance_reports":
		return f.ComplianceReports
	case "api_protection":
		return f.APIProtection
	case "dlp":
		return f.DLP
	case "bot_protection":
		return f.BotProtection
	case "virtual_patching":
		return f.VirtualPatching
	case "advanced_analytics":
		return f.AdvancedAnalytics
	case "priority_support":
		return f.PrioritySupport
	default:
		return false
	}
}

func ValidateSiteLimit(tier string, currentCount int) error {
	f := GetFeatures(tier)
	if f.MaxSites == -1 {
		return nil
	}
	if currentCount >= f.MaxSites {
		return fmt.Errorf("site limit exceeded: %d/%d (tier: %s)", currentCount, f.MaxSites, tier)
	}
	return nil
}
