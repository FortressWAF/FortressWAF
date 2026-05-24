package compliance

func (ce *ComplianceEngine) getHIPAAControls() []Control {
	return []Control{
		{ID: "HIPAA-164.308(a)(1)", Framework: FrameworkHIPAA, Name: "Security Management Process", Description: "Risk analysis and risk management implemented"},
		{ID: "HIPAA-164.308(a)(3)", Framework: FrameworkHIPAA, Name: "Workforce Security", Description: "Implement access authorization and management"},
		{ID: "HIPAA-164.308(a)(4)", Framework: FrameworkHIPAA, Name: "Information Access", Description: "Implement access authorization for ePHI"},
		{ID: "HIPAA-164.308(a)(5)", Framework: FrameworkHIPAA, Name: "Security Awareness", Description: "Implement security awareness and training program"},
		{ID: "HIPAA-164.308(a)(6)", Framework: FrameworkHIPAA, Name: "Security Incident", Description: "Implement security incident procedures"},
		{ID: "HIPAA-164.308(a)(7)", Framework: FrameworkHIPAA, Name: "Contingency", Description: "Establish data backup and disaster recovery plans"},
		{ID: "HIPAA-164.310(a)", Framework: FrameworkHIPAA, Name: "Access Control", Description: "Implement access control measures"},
		{ID: "HIPAA-164.310(b)", Framework: FrameworkHIPAA, Name: "Audit Controls", Description: "Implement hardware, software, procedures for audit trails"},
		{ID: "HIPAA-164.310(c)", Framework: FrameworkHIPAA, Name: "Integrity Controls", Description: "Implement electronic mechanisms to authenticate ePHI"},
		{ID: "HIPAA-164.310(d)", Framework: FrameworkHIPAA, Name: "Transmission Security", Description: "Implement encryption and integrity controls for ePHI transmission"},
		{ID: "HIPAA-164.312(a)", Framework: FrameworkHIPAA, Name: "Technical Safeguards", Description: "Implement technical policies for ePHI access"},
		{ID: "HIPAA-164.312(b)", Framework: FrameworkHIPAA, Name: "Audit Trail", Description: "Record and examine activity in systems containing ePHI"},
		{ID: "HIPAA-164.312(c)", Framework: FrameworkHIPAA, Name: "Integrity", Description: "Implement mechanisms to authenticate ePHI and protect from improper alteration"},
		{ID: "HIPAA-164.312(e)", Framework: FrameworkHIPAA, Name: "Transmission Security", Description: "Implement encryption and access controls for ePHI transmission"},
	}
}
