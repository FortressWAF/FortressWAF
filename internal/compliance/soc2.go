package compliance

func (ce *ComplianceEngine) getSOC2Controls() []Control {
	return []Control{
		{ID: "SOC2-CC1.1", Framework: FrameworkSOC2, Name: "Control Environment", Description: "Entity demonstrates commitment to integrity and ethical values"},
		{ID: "SOC2-CC2.1", Framework: FrameworkSOC2, Name: "Information & Communication", Description: "Entity obtains relevant quality information"},
		{ID: "SOC2-CC2.2", Framework: FrameworkSOC2, Name: "Internal Communication", Description: "Entity internally communicates information including objectives and responsibilities"},
		{ID: "SOC2-CC3.1", Framework: FrameworkSOC2, Name: "Risk Assessment", Description: "Entity specifies objectives with sufficient clarity"},
		{ID: "SOC2-CC4.1", Framework: FrameworkSOC2, Name: "Monitoring", Description: "Entity selects and develops ongoing evaluations"},
		{ID: "SOC2-CC5.1", Framework: FrameworkSOC2, Name: "Control Activities", Description: "Entity selects and develops control activities"},
		{ID: "SOC2-CC5.2", Framework: FrameworkSOC2, Name: "Technology Controls", Description: "Entity deploys control activities through technology"},
		{ID: "SOC2-CC6.1", Framework: FrameworkSOC2, Name: "Logical Access", Description: "Logical access controls prevent unauthorized access"},
		{ID: "SOC2-CC6.2", Framework: FrameworkSOC2, Name: "MFA", Description: "Multi-factor authentication is implemented"},
		{ID: "SOC2-CC6.3", Framework: FrameworkSOC2, Name: "Unique IDs", Description: "New access requires unique user IDs"},
		{ID: "SOC2-CC6.4", Framework: FrameworkSOC2, Name: "Access Removal", Description: "Access is removed upon termination"},
		{ID: "SOC2-CC6.6", Framework: FrameworkSOC2, Name: "Security Events", Description: "Security events are logged and monitored"},
		{ID: "SOC2-CC7.1", Framework: FrameworkSOC2, Name: "Vulnerability Management", Description: "System vulnerabilities are identified and remediated"},
		{ID: "SOC2-CC7.2", Framework: FrameworkSOC2, Name: "Security Monitoring", Description: "System monitoring processes detect security events"},
		{ID: "SOC2-CC7.3", Framework: FrameworkSOC2, Name: "Incident Response", Description: "Security incidents are identified and responded to"},
		{ID: "SOC2-CC7.4", Framework: FrameworkSOC2, Name: "Disaster Recovery", Description: "Availability commitments and requirements are established"},
		{ID: "SOC2-CC8.1", Framework: FrameworkSOC2, Name: "Change Management", Description: "Changes are authorized, tested, and approved"},
		{ID: "SOC2-CC9.1", Framework: FrameworkSOC2, Name: "Data Transmission", Description: "Data transmitted to/from third parties is encrypted"},
		{ID: "SOC2-A1.1", Framework: FrameworkSOC2, Name: "Availability", Description: "Availability commitments and system requirements are established"},
	}
}
