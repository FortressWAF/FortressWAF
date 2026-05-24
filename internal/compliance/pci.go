package compliance

func (ce *ComplianceEngine) getPCIControls() []Control {
	return []Control{
		{ID: "PCI-6.3.3", Framework: FrameworkPCI, Name: "Security Vulnerabilities", Description: "Protect against newly discovered vulnerabilities within 7 days"},
		{ID: "PCI-6.4", Framework: FrameworkPCI, Name: "WAF Deployment", Description: "Ensure all public-facing web applications are protected by a WAF"},
		{ID: "PCI-6.5", Framework: FrameworkPCI, Name: "Injection Flaws", Description: "Protect against injection flaws including SQLi"},
		{ID: "PCI-6.5.1", Framework: FrameworkPCI, Name: "SQLi Protection", Description: "Block SQL injection attacks"},
		{ID: "PCI-6.5.2", Framework: FrameworkPCI, Name: "XSS Protection", Description: "Block cross-site scripting attacks"},
		{ID: "PCI-6.5.3", Framework: FrameworkPCI, Name: "Authentication", Description: "Broken authentication and session management"},
		{ID: "PCI-6.5.4", Framework: FrameworkPCI, Name: "IDOR Protection", Description: "Protect against insecure direct object references"},
		{ID: "PCI-6.5.5", Framework: FrameworkPCI, Name: "CSRF Protection", Description: "Protect against cross-site request forgery"},
		{ID: "PCI-6.5.6", Framework: FrameworkPCI, Name: "Session Timeout", Description: "Inactive session timeout"},
		{ID: "PCI-6.5.7", Framework: FrameworkPCI, Name: "URL Redirects", Description: "Protect against open redirects"},
		{ID: "PCI-6.5.8", Framework: FrameworkPCI, Name: "File Uploads", Description: "Validate all uploaded files"},
		{ID: "PCI-6.5.9", Framework: FrameworkPCI, Name: "Encoding", Description: "Protect against OS command injection"},
		{ID: "PCI-6.5.10", Framework: FrameworkPCI, Name: "Buffer Overflows", Description: "Protect against buffer overflows"},
		{ID: "PCI-6.6", Framework: FrameworkPCI, Name: "App-layer Firewall", Description: "Address all threats to public-facing web apps"},
		{ID: "PCI-8.2", Framework: FrameworkPCI, Name: "User Auth", Description: "Authenticate all access to system components"},
		{ID: "PCI-8.3", Framework: FrameworkPCI, Name: "MFA", Description: "Incorporate multi-factor authentication"},
		{ID: "PCI-10.1", Framework: FrameworkPCI, Name: "Audit Logging", Description: "Implement audit trails for all system components"},
		{ID: "PCI-10.2", Framework: FrameworkPCI, Name: "User Identification", Description: "All individual user access to cardholder data"},
		{ID: "PCI-10.3", Framework: FrameworkPCI, Name: "Audit Timing", Description: "Record audit trail entries for all system components"},
		{ID: "PCI-10.4", Framework: FrameworkPCI, Name: "Time Synchronization", Description: "Synchronize internal time clocks"},
		{ID: "PCI-10.5", Framework: FrameworkPCI, Name: "Log Retention", Description: "Retain audit trail history for at least 1 year"},
		{ID: "PCI-10.6", Framework: FrameworkPCI, Name: "Log Review", Description: "Review audit logs and security events daily"},
		{ID: "PCI-10.7", Framework: FrameworkPCI, Name: "Log Integrity", Description: "Protect audit trail files from unauthorized modifications"},
		{ID: "PCI-32", Framework: FrameworkPCI, Name: "Data Retention", Description: "Limit data storage amount and retention time"},
		{ID: "PCI-3.4", Framework: FrameworkPCI, Name: "Encryption at Rest", Description: "Render PAN unreadable anywhere it is stored"},
	}
}
