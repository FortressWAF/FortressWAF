package compliance

func (ce *ComplianceEngine) getGDPRControls() []Control {
	return []Control{
		{ID: "GDPR-Art5-1-C", Framework: FrameworkGDPR, Name: "Purpose Limitation", Description: "Data collected for specified, explicit purposes"},
		{ID: "GDPR-Art5-1-D", Framework: FrameworkGDPR, Name: "Data Minimisation", Description: "Data adequate, relevant, limited to what is necessary"},
		{ID: "GDPR-Art5-1-E", Framework: FrameworkGDPR, Name: "Storage Limitation", Description: "Data kept in identifiable form no longer than necessary"},
		{ID: "GDPR-Art5-1-F", Framework: FrameworkGDPR, Name: "Integrity & Confidentiality", Description: "Data processed securely using appropriate technical measures"},
		{ID: "GDPR-Art6-1", Framework: FrameworkGDPR, Name: "Lawfulness", Description: "Processing has lawful basis"},
		{ID: "GDPR-Art7-1", Framework: FrameworkGDPR, Name: "Consent", Description: "Consent is freely given, specific, informed, and unambiguous"},
		{ID: "GDPR-Art12", Framework: FrameworkGDPR, Name: "Transparency", Description: "Provide privacy notices in clear, plain language"},
		{ID: "GDPR-Art15", Framework: FrameworkGDPR, Name: "Access", Description: "Data subjects can access their personal data"},
		{ID: "GDPR-Art16", Framework: FrameworkGDPR, Name: "Rectification", Description: "Data subjects can rectify inaccurate personal data"},
		{ID: "GDPR-Art17", Framework: FrameworkGDPR, Name: "Erasure", Description: "Data subjects can request erasure of their data"},
		{ID: "GDPR-Art20", Framework: FrameworkGDPR, Name: "Portability", Description: "Data provided in structured, machine-readable format"},
		{ID: "GDPR-Art25", Framework: FrameworkGDPR, Name: "Privacy by Design", Description: "Data protection by design and by default"},
		{ID: "GDPR-Art28", Framework: FrameworkGDPR, Name: "Processors", Description: "Data Processing Agreements with all processors"},
		{ID: "GDPR-Art30-1", Framework: FrameworkGDPR, Name: "ROPA", Description: "Maintain records of processing activities"},
		{ID: "GDPR-Art32", Framework: FrameworkGDPR, Name: "Security", Description: "Implement appropriate technical and organisational measures"},
		{ID: "GDPR-Art32-1-C", Framework: FrameworkGDPR, Name: "Encryption", Description: "AES-256 encryption at rest, TLS 1.2+ in transit"},
		{ID: "GDPR-Art32-2", Framework: FrameworkGDPR, Name: "Pseudonymisation", Description: "Use pseudonymisation where appropriate"},
		{ID: "GDPR-Art33", Framework: FrameworkGDPR, Name: "Breach Notification", Description: "Notify supervisory authority within 72 hours of breach"},
		{ID: "GDPR-Art34", Framework: FrameworkGDPR, Name: "Data Subject Notification", Description: "Notify data subjects of high-risk breaches"},
		{ID: "GDPR-Art35", Framework: FrameworkGDPR, Name: "DPIA", Description: "Conduct Data Protection Impact Assessments"},
		{ID: "GDPR-Art36", Framework: FrameworkGDPR, Name: "Consultation", Description: "Consult supervisory authority before processing if required"},
		{ID: "GDPR-Art37", Framework: FrameworkGDPR, Name: "DPO", Description: "Designate Data Protection Officer if required"},
	}
}