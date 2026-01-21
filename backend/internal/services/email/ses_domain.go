package email

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

// VerificationStatus represents the verification status of a domain or email
type VerificationStatus string

const (
	VerificationPending    VerificationStatus = "Pending"
	VerificationSuccess    VerificationStatus = "Success"
	VerificationFailed     VerificationStatus = "Failed"
	VerificationTemporary  VerificationStatus = "TemporaryFailure"
	VerificationNotStarted VerificationStatus = "NotStarted"
)

// DomainInfo contains information about a verified domain
type DomainInfo struct {
	Domain             string
	VerificationStatus VerificationStatus
	VerificationToken  string
	DKIMEnabled        bool
	DKIMStatus         VerificationStatus
	DKIMTokens         []string
	MailFromDomain     string
	MailFromStatus     VerificationStatus
	CreatedAt          time.Time
	LastChecked        time.Time
}

// EmailIdentityInfo contains information about a verified email address
type EmailIdentityInfo struct {
	Email              string
	VerificationStatus VerificationStatus
	LastChecked        time.Time
}

// DKIMRecord represents a DKIM DNS record
type DKIMRecord struct {
	Name  string
	Type  string
	Value string
}

// VerificationRecord represents a DNS record for domain verification
type VerificationRecord struct {
	Name  string
	Type  string
	Value string
}

// DomainManager handles domain and identity verification in SES
type DomainManager struct {
	client *ses.Client
	config *SESConfig
	logger *SESLogger
	audit  *AuditLog
}

// NewDomainManager creates a new domain manager
func NewDomainManager(ctx context.Context, config *SESConfig) (*DomainManager, error) {
	client, err := config.CreateSESClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create SES client: %w", err)
	}

	return &DomainManager{
		client: client,
		config: config,
		logger: NewSESLogger(config),
		audit:  NewAuditLog(config),
	}, nil
}

// ============================================
// Domain Verification
// ============================================

// VerifyDomain initiates domain verification for SES
func (m *DomainManager) VerifyDomain(ctx context.Context, domain string) (*DomainInfo, error) {
	m.logger.Info("Initiating domain verification", "domain", domain)

	output, err := m.client.VerifyDomainIdentity(ctx, &ses.VerifyDomainIdentityInput{
		Domain: aws.String(domain),
	})
	if err != nil {
		m.logger.Error("Failed to initiate domain verification", "domain", domain, "error", err)
		return nil, fmt.Errorf("failed to verify domain: %w", err)
	}

	info := &DomainInfo{
		Domain:             domain,
		VerificationStatus: VerificationPending,
		VerificationToken:  aws.ToString(output.VerificationToken),
		CreatedAt:          time.Now(),
		LastChecked:        time.Now(),
	}

	m.logger.Info("Domain verification initiated",
		"domain", domain,
		"token", info.VerificationToken,
	)

	return info, nil
}

// GetDomainVerificationRecord returns the DNS TXT record needed for verification
func (m *DomainManager) GetDomainVerificationRecord(domain, token string) VerificationRecord {
	return VerificationRecord{
		Name:  fmt.Sprintf("_amazonses.%s", domain),
		Type:  "TXT",
		Value: token,
	}
}

// CheckDomainVerification checks the verification status of a domain
func (m *DomainManager) CheckDomainVerification(ctx context.Context, domain string) (*DomainInfo, error) {
	output, err := m.client.GetIdentityVerificationAttributes(ctx, &ses.GetIdentityVerificationAttributesInput{
		Identities: []string{domain},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get verification status: %w", err)
	}

	attrs, ok := output.VerificationAttributes[domain]
	if !ok {
		return nil, fmt.Errorf("domain not found in verification attributes")
	}

	info := &DomainInfo{
		Domain:             domain,
		VerificationStatus: convertVerificationStatus(attrs.VerificationStatus),
		VerificationToken:  aws.ToString(attrs.VerificationToken),
		LastChecked:        time.Now(),
	}

	m.logger.Info("Domain verification status checked",
		"domain", domain,
		"status", info.VerificationStatus,
	)

	if info.VerificationStatus == VerificationSuccess {
		m.audit.LogDomainVerified(domain, "success")
	}

	return info, nil
}

// ListVerifiedDomains lists all verified domains
func (m *DomainManager) ListVerifiedDomains(ctx context.Context) ([]string, error) {
	var domains []string
	var nextToken *string

	for {
		output, err := m.client.ListIdentities(ctx, &ses.ListIdentitiesInput{
			IdentityType: types.IdentityTypeDomain,
			NextToken:    nextToken,
			MaxItems:     aws.Int32(100),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list identities: %w", err)
		}

		domains = append(domains, output.Identities...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return domains, nil
}

// DeleteDomain removes a domain from SES
func (m *DomainManager) DeleteDomain(ctx context.Context, domain string) error {
	_, err := m.client.DeleteIdentity(ctx, &ses.DeleteIdentityInput{
		Identity: aws.String(domain),
	})
	if err != nil {
		return fmt.Errorf("failed to delete domain: %w", err)
	}

	m.logger.Info("Domain deleted", "domain", domain)
	return nil
}

// ============================================
// DKIM Configuration
// ============================================

// EnableDKIM enables DKIM signing for a domain
func (m *DomainManager) EnableDKIM(ctx context.Context, domain string) (*DomainInfo, error) {
	m.logger.Info("Enabling DKIM for domain", "domain", domain)

	output, err := m.client.VerifyDomainDkim(ctx, &ses.VerifyDomainDkimInput{
		Domain: aws.String(domain),
	})
	if err != nil {
		m.logger.Error("Failed to enable DKIM", "domain", domain, "error", err)
		return nil, fmt.Errorf("failed to enable DKIM: %w", err)
	}

	info := &DomainInfo{
		Domain:      domain,
		DKIMEnabled: true,
		DKIMStatus:  VerificationPending,
		DKIMTokens:  output.DkimTokens,
		LastChecked: time.Now(),
	}

	m.logger.Info("DKIM enabled",
		"domain", domain,
		"tokens", info.DKIMTokens,
	)

	return info, nil
}

// GetDKIMRecords returns the DNS CNAME records needed for DKIM
func (m *DomainManager) GetDKIMRecords(domain string, tokens []string) []DKIMRecord {
	records := make([]DKIMRecord, len(tokens))
	for i, token := range tokens {
		records[i] = DKIMRecord{
			Name:  fmt.Sprintf("%s._domainkey.%s", token, domain),
			Type:  "CNAME",
			Value: fmt.Sprintf("%s.dkim.amazonses.com", token),
		}
	}
	return records
}

// CheckDKIMStatus checks the DKIM verification status
func (m *DomainManager) CheckDKIMStatus(ctx context.Context, domain string) (*DomainInfo, error) {
	output, err := m.client.GetIdentityDkimAttributes(ctx, &ses.GetIdentityDkimAttributesInput{
		Identities: []string{domain},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get DKIM status: %w", err)
	}

	attrs, ok := output.DkimAttributes[domain]
	if !ok {
		return nil, fmt.Errorf("domain not found in DKIM attributes")
	}

	info := &DomainInfo{
		Domain:      domain,
		DKIMEnabled: attrs.DkimEnabled,
		DKIMStatus:  convertDKIMStatus(attrs.DkimVerificationStatus),
		DKIMTokens:  attrs.DkimTokens,
		LastChecked: time.Now(),
	}

	m.logger.Info("DKIM status checked",
		"domain", domain,
		"enabled", info.DKIMEnabled,
		"status", info.DKIMStatus,
	)

	return info, nil
}

// SetDKIMSigningEnabled enables or disables DKIM signing for a domain
func (m *DomainManager) SetDKIMSigningEnabled(ctx context.Context, domain string, enabled bool) error {
	_, err := m.client.SetIdentityDkimEnabled(ctx, &ses.SetIdentityDkimEnabledInput{
		Identity:    aws.String(domain),
		DkimEnabled: enabled,
	})
	if err != nil {
		return fmt.Errorf("failed to set DKIM enabled: %w", err)
	}

	m.logger.Info("DKIM signing updated", "domain", domain, "enabled", enabled)
	return nil
}

// ============================================
// Email Address Verification
// ============================================

// VerifyEmailAddress initiates email address verification
func (m *DomainManager) VerifyEmailAddress(ctx context.Context, email string) error {
	m.logger.Info("Initiating email verification", "email", email)

	_, err := m.client.VerifyEmailIdentity(ctx, &ses.VerifyEmailIdentityInput{
		EmailAddress: aws.String(email),
	})
	if err != nil {
		m.logger.Error("Failed to initiate email verification", "email", email, "error", err)
		return fmt.Errorf("failed to verify email: %w", err)
	}

	m.logger.Info("Email verification initiated", "email", email)
	return nil
}

// CheckEmailVerification checks the verification status of an email address
func (m *DomainManager) CheckEmailVerification(ctx context.Context, email string) (*EmailIdentityInfo, error) {
	output, err := m.client.GetIdentityVerificationAttributes(ctx, &ses.GetIdentityVerificationAttributesInput{
		Identities: []string{email},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get verification status: %w", err)
	}

	attrs, ok := output.VerificationAttributes[email]
	if !ok {
		return &EmailIdentityInfo{
			Email:              email,
			VerificationStatus: VerificationNotStarted,
			LastChecked:        time.Now(),
		}, nil
	}

	return &EmailIdentityInfo{
		Email:              email,
		VerificationStatus: convertVerificationStatus(attrs.VerificationStatus),
		LastChecked:        time.Now(),
	}, nil
}

// ListVerifiedEmails lists all verified email addresses
func (m *DomainManager) ListVerifiedEmails(ctx context.Context) ([]string, error) {
	var emails []string
	var nextToken *string

	for {
		output, err := m.client.ListIdentities(ctx, &ses.ListIdentitiesInput{
			IdentityType: types.IdentityTypeEmailAddress,
			NextToken:    nextToken,
			MaxItems:     aws.Int32(100),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list identities: %w", err)
		}

		emails = append(emails, output.Identities...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return emails, nil
}

// DeleteEmailIdentity removes an email address from SES
func (m *DomainManager) DeleteEmailIdentity(ctx context.Context, email string) error {
	_, err := m.client.DeleteIdentity(ctx, &ses.DeleteIdentityInput{
		Identity: aws.String(email),
	})
	if err != nil {
		return fmt.Errorf("failed to delete email identity: %w", err)
	}

	m.logger.Info("Email identity deleted", "email", email)
	return nil
}

// ============================================
// Mail From Domain (Custom MAIL FROM)
// ============================================

// SetMailFromDomain configures a custom MAIL FROM domain
func (m *DomainManager) SetMailFromDomain(ctx context.Context, identity, mailFromDomain string) error {
	m.logger.Info("Setting MAIL FROM domain",
		"identity", identity,
		"mailFromDomain", mailFromDomain,
	)

	_, err := m.client.SetIdentityMailFromDomain(ctx, &ses.SetIdentityMailFromDomainInput{
		Identity:            aws.String(identity),
		MailFromDomain:      aws.String(mailFromDomain),
		BehaviorOnMXFailure: types.BehaviorOnMXFailureUseDefaultValue,
	})
	if err != nil {
		return fmt.Errorf("failed to set MAIL FROM domain: %w", err)
	}

	m.logger.Info("MAIL FROM domain configured",
		"identity", identity,
		"mailFromDomain", mailFromDomain,
	)
	return nil
}

// GetMailFromDomainRecords returns DNS records needed for MAIL FROM domain
func (m *DomainManager) GetMailFromDomainRecords(mailFromDomain, region string) []VerificationRecord {
	return []VerificationRecord{
		{
			Name:  mailFromDomain,
			Type:  "MX",
			Value: fmt.Sprintf("10 feedback-smtp.%s.amazonses.com", region),
		},
		{
			Name:  mailFromDomain,
			Type:  "TXT",
			Value: "v=spf1 include:amazonses.com ~all",
		},
	}
}

// CheckMailFromStatus checks the MAIL FROM domain status
func (m *DomainManager) CheckMailFromStatus(ctx context.Context, identity string) (*DomainInfo, error) {
	output, err := m.client.GetIdentityMailFromDomainAttributes(ctx, &ses.GetIdentityMailFromDomainAttributesInput{
		Identities: []string{identity},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get MAIL FROM status: %w", err)
	}

	attrs, ok := output.MailFromDomainAttributes[identity]
	if !ok {
		return nil, fmt.Errorf("identity not found")
	}

	return &DomainInfo{
		Domain:         identity,
		MailFromDomain: aws.ToString(attrs.MailFromDomain),
		MailFromStatus: convertMailFromStatus(attrs.MailFromDomainStatus),
		LastChecked:    time.Now(),
	}, nil
}

// ============================================
// Configuration Set Management
// ============================================

// CreateConfigurationSet creates a new configuration set
func (m *DomainManager) CreateConfigurationSet(ctx context.Context, name string) error {
	m.logger.Info("Creating configuration set", "name", name)

	_, err := m.client.CreateConfigurationSet(ctx, &ses.CreateConfigurationSetInput{
		ConfigurationSet: &types.ConfigurationSet{
			Name: aws.String(name),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create configuration set: %w", err)
	}

	m.logger.Info("Configuration set created", "name", name)
	return nil
}

// DeleteConfigurationSet deletes a configuration set
func (m *DomainManager) DeleteConfigurationSet(ctx context.Context, name string) error {
	_, err := m.client.DeleteConfigurationSet(ctx, &ses.DeleteConfigurationSetInput{
		ConfigurationSetName: aws.String(name),
	})
	if err != nil {
		return fmt.Errorf("failed to delete configuration set: %w", err)
	}

	m.logger.Info("Configuration set deleted", "name", name)
	return nil
}

// ListConfigurationSets lists all configuration sets
func (m *DomainManager) ListConfigurationSets(ctx context.Context) ([]string, error) {
	var sets []string
	var nextToken *string

	for {
		output, err := m.client.ListConfigurationSets(ctx, &ses.ListConfigurationSetsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list configuration sets: %w", err)
		}

		for _, set := range output.ConfigurationSets {
			sets = append(sets, aws.ToString(set.Name))
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return sets, nil
}

// ============================================
// Helper Functions
// ============================================

func convertVerificationStatus(status types.VerificationStatus) VerificationStatus {
	switch status {
	case types.VerificationStatusPending:
		return VerificationPending
	case types.VerificationStatusSuccess:
		return VerificationSuccess
	case types.VerificationStatusFailed:
		return VerificationFailed
	case types.VerificationStatusTemporaryFailure:
		return VerificationTemporary
	case types.VerificationStatusNotStarted:
		return VerificationNotStarted
	default:
		return VerificationNotStarted
	}
}

func convertDKIMStatus(status types.VerificationStatus) VerificationStatus {
	switch status {
	case types.VerificationStatusPending:
		return VerificationPending
	case types.VerificationStatusSuccess:
		return VerificationSuccess
	case types.VerificationStatusFailed:
		return VerificationFailed
	case types.VerificationStatusTemporaryFailure:
		return VerificationTemporary
	case types.VerificationStatusNotStarted:
		return VerificationNotStarted
	default:
		return VerificationNotStarted
	}
}

func convertMailFromStatus(status types.CustomMailFromStatus) VerificationStatus {
	switch status {
	case types.CustomMailFromStatusPending:
		return VerificationPending
	case types.CustomMailFromStatusSuccess:
		return VerificationSuccess
	case types.CustomMailFromStatusFailed:
		return VerificationFailed
	case types.CustomMailFromStatusTemporaryFailure:
		return VerificationTemporary
	default:
		return VerificationNotStarted
	}
}

// ============================================
// Domain Setup Wizard
// ============================================

// DomainSetupResult contains the results of a complete domain setup
type DomainSetupResult struct {
	Domain             string
	VerificationRecord VerificationRecord
	DKIMRecords        []DKIMRecord
	MailFromRecords    []VerificationRecord
	VerificationStatus VerificationStatus
	DKIMStatus         VerificationStatus
	MailFromStatus     VerificationStatus
	Instructions       []string
}

// SetupDomain performs a complete domain setup
func (m *DomainManager) SetupDomain(ctx context.Context, domain, mailFromSubdomain string) (*DomainSetupResult, error) {
	result := &DomainSetupResult{
		Domain:       domain,
		Instructions: make([]string, 0),
	}

	// Step 1: Verify domain
	domainInfo, err := m.VerifyDomain(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to verify domain: %w", err)
	}

	result.VerificationStatus = domainInfo.VerificationStatus
	result.VerificationRecord = m.GetDomainVerificationRecord(domain, domainInfo.VerificationToken)
	result.Instructions = append(result.Instructions,
		fmt.Sprintf("Add TXT record: %s = %s", result.VerificationRecord.Name, result.VerificationRecord.Value))

	// Step 2: Enable DKIM
	dkimInfo, err := m.EnableDKIM(ctx, domain)
	if err != nil {
		m.logger.Warn("Failed to enable DKIM, continuing", "error", err)
	} else {
		result.DKIMStatus = dkimInfo.DKIMStatus
		result.DKIMRecords = m.GetDKIMRecords(domain, dkimInfo.DKIMTokens)
		for _, record := range result.DKIMRecords {
			result.Instructions = append(result.Instructions,
				fmt.Sprintf("Add CNAME record: %s -> %s", record.Name, record.Value))
		}
	}

	// Step 3: Configure MAIL FROM domain
	if mailFromSubdomain != "" {
		mailFromDomain := fmt.Sprintf("%s.%s", mailFromSubdomain, domain)
		err = m.SetMailFromDomain(ctx, domain, mailFromDomain)
		if err != nil {
			m.logger.Warn("Failed to set MAIL FROM domain, continuing", "error", err)
		} else {
			result.MailFromRecords = m.GetMailFromDomainRecords(mailFromDomain, m.config.Region)
			for _, record := range result.MailFromRecords {
				result.Instructions = append(result.Instructions,
					fmt.Sprintf("Add %s record: %s = %s", record.Type, record.Name, record.Value))
			}
		}
	}

	result.Instructions = append(result.Instructions,
		"After adding all DNS records, verification can take up to 72 hours.")

	return result, nil
}

// CheckDomainSetupStatus checks the status of a domain setup
func (m *DomainManager) CheckDomainSetupStatus(ctx context.Context, domain string) (*DomainSetupResult, error) {
	result := &DomainSetupResult{
		Domain: domain,
	}

	// Check domain verification
	domainInfo, err := m.CheckDomainVerification(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to check domain verification: %w", err)
	}
	result.VerificationStatus = domainInfo.VerificationStatus

	// Check DKIM status
	dkimInfo, err := m.CheckDKIMStatus(ctx, domain)
	if err != nil {
		m.logger.Warn("Failed to check DKIM status", "error", err)
	} else {
		result.DKIMStatus = dkimInfo.DKIMStatus
	}

	// Check MAIL FROM status
	mailFromInfo, err := m.CheckMailFromStatus(ctx, domain)
	if err != nil {
		m.logger.Warn("Failed to check MAIL FROM status", "error", err)
	} else {
		result.MailFromStatus = mailFromInfo.MailFromStatus
	}

	return result, nil
}
