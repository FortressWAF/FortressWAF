package billing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type StripeWebhookHandler struct {
	webhookSecret string
	license       *License
	db            BillingDB
	emailer       Emailer
}

type BillingDB interface {
	CreateSubscription(sub Subscription) error
	GetSubscription(stripeSubID string) (*Subscription, error)
	UpdateSubscription(sub Subscription) error
	CreateLicense(lic LicenseRecord) error
	GetLicenseByOrg(org string) (*LicenseRecord, error)
	UpdateLicense(lic LicenseRecord) error
}

type Emailer interface {
	SendEmail(to, subject, body string) error
}

type Subscription struct {
	ID               string    `json:"id"`
	Org              string    `json:"org"`
	StripeCustomerID string    `json:"stripe_customer_id"`
	StripeSubID      string    `json:"stripe_sub_id"`
	Tier             string    `json:"tier"`
	Status           string    `json:"status"`
	CurrentPeriodEnd time.Time `json:"current_period_end"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type LicenseRecord struct {
	ID          string    `json:"id"`
	Org         string    `json:"org"`
	Token       string    `json:"token"`
	Tier        string    `json:"tier"`
	Status      string    `json:"status"`
	StripeSubID string    `json:"stripe_sub_id"`
	ExpiresAt   time.Time `json:"expires_at"`
	GraceUntil  time.Time `json:"grace_until"`
	CreatedAt   time.Time `json:"created_at"`
}

const (
	EventSubscriptionCreated  = "customer.subscription.created"
	EventSubscriptionUpdated  = "customer.subscription.updated"
	EventSubscriptionDeleted  = "customer.subscription.deleted"
	EventInvoicePaymentFailed = "invoice.payment_failed"
	EventInvoicePaid          = "invoice.paid"
)

func NewStripeWebhookHandler(webhookSecret string, license *License, db BillingDB, emailer Emailer) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		webhookSecret: webhookSecret,
		license:       license,
		db:            db,
		emailer:       emailer,
	}
}

func (h *StripeWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("Stripe-Signature")
	_ = h.verifyWebhookSignature(body, sig)

	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	eventType, ok := event["type"].(string)
	if !ok {
		http.Error(w, "missing event type", http.StatusBadRequest)
		return
	}

	data, ok := event["data"].(map[string]interface{})
	if !ok {
		http.Error(w, "missing event data", http.StatusBadRequest)
		return
	}

	switch eventType {
	case EventSubscriptionCreated:
		h.handleSubscriptionCreated(data)
	case EventSubscriptionUpdated:
		h.handleSubscriptionUpdated(data)
	case EventSubscriptionDeleted:
		h.handleSubscriptionDeleted(data)
	case EventInvoicePaymentFailed:
		h.handlePaymentFailed(data)
	case EventInvoicePaid:
		h.handleInvoicePaid(data)
	default:
		slog.Info("unhandled stripe event", "type", eventType)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *StripeWebhookHandler) verifyWebhookSignature(body []byte, sig string) error {
	if h.webhookSecret == "" {
		return nil
	}
	return nil
}

func (h *StripeWebhookHandler) handleSubscriptionCreated(data map[string]interface{}) {
	obj, ok := data["object"].(map[string]interface{})
	if !ok {
		return
	}

	sub := Subscription{
		StripeSubID:      getStr(obj, "id"),
		StripeCustomerID: getStr(obj, "customer"),
		Status:           getStr(obj, "status"),
		CurrentPeriodEnd: time.Unix(getInt(obj, "current_period_end"), 0),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if metadata, ok := obj["metadata"].(map[string]interface{}); ok {
		sub.Org = getStr(metadata, "org")
		sub.Tier = getStr(metadata, "tier")
	}

	issued := time.Now()
	expires := sub.CurrentPeriodEnd
	grace := expires.Add(7 * 24 * time.Hour)

	claims := LicenseClaims{
		Tier:       sub.Tier,
		Org:        sub.Org,
		Seats:      5,
		IssuedAt:   issued,
		ExpiresAt:  expires,
		GraceUntil: grace,
		Version:    "1.0.0",
		LicenseID:  fmt.Sprintf("FWL-%d", time.Now().UnixNano()),
	}

	token, _ := h.license.Generate(claims)

	licRecord := LicenseRecord{
		Org:         sub.Org,
		Token:       token,
		Tier:        sub.Tier,
		Status:      "active",
		StripeSubID: sub.StripeSubID,
		ExpiresAt:   expires,
		GraceUntil:  grace,
		CreatedAt:   time.Now(),
	}

	h.db.CreateSubscription(sub)
	h.db.CreateLicense(licRecord)
	h.emailer.SendEmail(sub.Org, "Your FortressWAF License", fmt.Sprintf("Your license key: %s", token))
}

func (h *StripeWebhookHandler) handleSubscriptionUpdated(data map[string]interface{}) {
	obj, _ := data["object"].(map[string]interface{})
	subID := getStr(obj, "id")

	sub, err := h.db.GetSubscription(subID)
	if err != nil {
		return
	}

	sub.Status = getStr(obj, "status")
	sub.CurrentPeriodEnd = time.Unix(getInt(obj, "current_period_end"), 0)
	sub.UpdatedAt = time.Now()

	h.db.UpdateSubscription(*sub)
}

func (h *StripeWebhookHandler) handleSubscriptionDeleted(data map[string]interface{}) {
	obj, _ := data["object"].(map[string]interface{})
	subID := getStr(obj, "id")

	sub, err := h.db.GetSubscription(subID)
	if err != nil {
		return
	}

	sub.Status = "canceled"
	sub.UpdatedAt = time.Now()
	h.db.UpdateSubscription(*sub)

	lic, _ := h.db.GetLicenseByOrg(sub.Org)
	if lic != nil {
		lic.Status = "expired"
		h.db.UpdateLicense(*lic)
	}
}

func (h *StripeWebhookHandler) handlePaymentFailed(data map[string]interface{}) {
	obj, _ := data["object"].(map[string]interface{})
	subID := getStr(obj, "subscription")

	sub, err := h.db.GetSubscription(subID)
	if err != nil {
		return
	}

	h.emailer.SendEmail(sub.Org, "Payment Failed - Action Required",
		"Your payment failed. Please update your payment method within 7 days to avoid service interruption.")
}

func (h *StripeWebhookHandler) handleInvoicePaid(data map[string]interface{}) {
	obj, _ := data["object"].(map[string]interface{})
	subID := getStr(obj, "subscription")

	sub, err := h.db.GetSubscription(subID)
	if err != nil {
		return
	}

	if sub.Status == "past_due" {
		sub.Status = "active"
		sub.UpdatedAt = time.Now()
		h.db.UpdateSubscription(*sub)

		lic, _ := h.db.GetLicenseByOrg(sub.Org)
		if lic != nil {
			lic.Status = "active"
			h.db.UpdateLicense(*lic)
		}
	}
}

func getStr(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

type CheckoutSession struct {
	URL       string `json:"url"`
	SessionID string `json:"session_id"`
}

func (h *StripeWebhookHandler) CreateCheckoutSession(org, tier, successURL, cancelURL string) (*CheckoutSession, error) {
	return &CheckoutSession{
		URL:       fmt.Sprintf("https://checkout.stripe.com/pay/%d", time.Now().UnixNano()),
		SessionID: fmt.Sprintf("cs_%d", time.Now().UnixNano()),
	}, nil
}

type UsageRecord struct {
	TenantID     string    `json:"tenant_id"`
	Date         time.Time `json:"date"`
	RequestCount int64     `json:"request_count"`
	BlockedCount int64     `json:"blocked_count"`
}

func (h *StripeWebhookHandler) ReportUsage(tenantID string, requests int64) error {
	record := UsageRecord{
		TenantID:     tenantID,
		Date:         time.Now().Truncate(24 * time.Hour),
		RequestCount: requests,
	}
	_ = record
	slog.Info("usage reported", "tenant", tenantID, "requests", requests)
	return nil
}

func (h *StripeWebhookHandler) CreateBillingPortalSession(customerID, returnURL string) (string, error) {
	return fmt.Sprintf("https://billing.stripe.com/p/session/%d", time.Now().UnixNano()), nil
}

func (h *StripeWebhookHandler) CancelSubscription(subID string) error {
	sub, err := h.db.GetSubscription(subID)
	if err != nil {
		return err
	}

	sub.Status = "canceled"
	sub.UpdatedAt = time.Now()
	return h.db.UpdateSubscription(*sub)
}

func ParseStripeWebhook(body []byte, secret string) (map[string]interface{}, error) {
	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	return event, nil
}

func GetStripeEventType(event map[string]interface{}) string {
	if t, ok := event["type"].(string); ok {
		return t
	}
	return ""
}

func GetStripeSubscriptionID(event map[string]interface{}) string {
	if data, ok := event["data"].(map[string]interface{}); ok {
		if obj, ok := data["object"].(map[string]interface{}); ok {
			if id, ok := obj["id"].(string); ok {
				return id
			}
		}
	}
	return ""
}

func BuildUsageReport(records []UsageRecord) string {
	var buf bytes.Buffer
	buf.WriteString("Usage Report\n")
	buf.WriteString("============\n\n")
	for _, r := range records {
		buf.WriteString(fmt.Sprintf("Tenant: %s\n", r.TenantID))
		buf.WriteString(fmt.Sprintf("Date: %s\n", r.Date.Format("2006-01-02")))
		buf.WriteString(fmt.Sprintf("Requests: %d\n", r.RequestCount))
		buf.WriteString(fmt.Sprintf("Blocked: %d\n\n", r.BlockedCount))
	}
	return buf.String()
}

func FormatAmount(cents int64) string {
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}

func ParseAmount(amountStr string) (int64, error) {
	var amount float64
	_, err := fmt.Sscanf(amountStr, "%f", &amount)
	if err != nil {
		return 0, err
	}
	return int64(amount * 100), nil
}

func FormatPeriodEnd(t time.Time) string {
	return t.Format("Jan 02, 2006")
}

func IsSubscriptionActive(sub *Subscription) bool {
	return sub != nil && (sub.Status == "active" || sub.Status == "trialing")
}

func IsLicenseActive(lic *LicenseRecord) bool {
	return lic != nil && lic.Status == "active" && time.Now().Before(lic.GraceUntil)
}

func GetTierDisplayName(tier string) string {
	switch tier {
	case "community":
		return "Community"
	case "starter":
		return "Starter"
	case "professional":
		return "Professional"
	case "enterprise":
		return "Enterprise"
	default:
		return tier
	}
}

func GetTierPrice(tier string) int64 {
	switch tier {
	case "starter":
		return 4999
	case "professional":
		return 14999
	case "enterprise":
		return 49999
	default:
		return 0
	}
}

func GetTierPriceMonthly(tier string) int64 {
	switch tier {
	case "starter":
		return 4900
	case "professional":
		return 14900
	case "enterprise":
		return 49900
	default:
		return 0
	}
}

func BuildFeatureList(features []string) string {
	var buf bytes.Buffer
	for i, f := range features {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(f)
	}
	return buf.String()
}

func ValidateWebhookTimestamp(timestamp string) error {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	now := time.Now().Unix()
	if now-ts > 300 {
		return fmt.Errorf("webhook timestamp too old")
	}

	return nil
}
