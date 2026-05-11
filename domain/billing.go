package domain

import (
	"errors"
	"fmt"
)

var (
	ErrBillingProviderNotConfigured   = errors.New("billing provider is not configured")
	ErrBillingProviderMisconfigured   = errors.New("billing provider is misconfigured")
	ErrBillingPlanInvalid             = errors.New("billing plan is invalid")
	ErrBillingWebhookSignatureInvalid = errors.New("billing webhook signature is invalid")
)

type BillingPlan string

const (
	BillingPlanFree BillingPlan = "free"
	BillingPlanPro  BillingPlan = "pro"
)

func (p BillingPlan) Validate() error {
	switch p {
	case BillingPlanFree, BillingPlanPro:
		return nil
	case "":
		return fmt.Errorf("%w: empty", ErrBillingPlanInvalid)
	default:
		return fmt.Errorf("%w: %s", ErrBillingPlanInvalid, p)
	}
}

type BillingCheckoutSession struct {
	URL string `json:"url"`
}

type BillingPortalSession struct {
	URL string `json:"url"`
}

type BillingCustomerRef struct {
	ExternalCustomerID string `json:"external_customer_id"`
}

type ProviderWebhookEvent struct {
	EventID                string      `json:"event_id"`
	EventType              string      `json:"event_type"`
	ExternalCustomerID     string      `json:"external_customer_id"`
	ExternalSubscriptionID string      `json:"external_subscription_id"`
	Plan                   BillingPlan `json:"plan"`
}
