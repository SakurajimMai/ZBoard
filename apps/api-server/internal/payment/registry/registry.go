// Package registry provides a dynamic payment provider registry that loads
// configurations from the database. Admin changes take effect after Reload().
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/zboard/api-server/internal/payment"
	"github.com/zboard/api-server/internal/payment/creem"
	"github.com/zboard/api-server/internal/payment/epay"
	"github.com/zboard/api-server/internal/payment/nowpay"
	"github.com/zboard/api-server/internal/store"
)

// Registry holds all configured payment providers.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]payment.Provider
	store     *store.Store
}

func New(s *store.Store) *Registry {
	return &Registry{providers: make(map[string]payment.Provider), store: s}
}

// Register adds a provider manually (used for testing).
func (r *Registry) Register(p payment.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

// Get returns a provider by name.
func (r *Registry) Get(name string) (payment.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("payment provider %q not registered or not enabled", name)
	}
	return p, nil
}

// List returns all registered provider names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.providers))
	for k := range r.providers {
		out = append(out, k)
	}
	return out
}

// Reload reads all enabled payment_providers from the DB and rebuilds the
// in-memory provider map. Call this on startup and after admin changes.
func (r *Registry) Reload(ctx context.Context) error {
	rows, err := r.store.ListEnabledPaymentProviders(ctx)
	if err != nil {
		return fmt.Errorf("reload payment providers: %w", err)
	}
	newMap := make(map[string]payment.Provider, len(rows))
	for _, row := range rows {
		p, err := buildProvider(row)
		if err != nil {
			log.Printf("payment provider %q config error: %v (skipping)", row.Name, err)
			continue
		}
		newMap[row.Name] = p
	}
	r.mu.Lock()
	r.providers = newMap
	r.mu.Unlock()
	return nil
}

// buildProvider instantiates a Provider from a DB row's config_json.
func buildProvider(row store.PaymentProvider) (payment.Provider, error) {
	switch row.ProviderType {
	case "epay":
		var cfg struct {
			APIURL    string `json:"api_url"`
			PID       string `json:"pid"`
			SecretKey string `json:"secret_key"`
		}
		if err := json.Unmarshal([]byte(row.ConfigJSON), &cfg); err != nil {
			return nil, fmt.Errorf("parse epay config: %w", err)
		}
		if cfg.PID == "" || cfg.SecretKey == "" {
			return nil, fmt.Errorf("epay: pid and secret_key are required")
		}
		return epay.New(epay.Config{
			APIURL:    cfg.APIURL,
			PID:       cfg.PID,
			SecretKey: cfg.SecretKey,
		}), nil

	case "creem":
		var cfg struct {
			APIKey        string `json:"api_key"`
			WebhookSecret string `json:"webhook_secret"`
			APIURL        string `json:"api_url"`
		}
		if err := json.Unmarshal([]byte(row.ConfigJSON), &cfg); err != nil {
			return nil, fmt.Errorf("parse creem config: %w", err)
		}
		if cfg.APIKey == "" || cfg.WebhookSecret == "" {
			return nil, fmt.Errorf("creem: api_key and webhook_secret are required")
		}
		return creem.New(creem.Config{
			APIKey:        cfg.APIKey,
			WebhookSecret: cfg.WebhookSecret,
			APIURL:        cfg.APIURL,
		}), nil

	case "nowpayments":
		var cfg struct {
			APIKey    string `json:"api_key"`
			IPNSecret string `json:"ipn_secret"`
			APIURL    string `json:"api_url"`
		}
		if err := json.Unmarshal([]byte(row.ConfigJSON), &cfg); err != nil {
			return nil, fmt.Errorf("parse nowpayments config: %w", err)
		}
		if cfg.APIKey == "" || cfg.IPNSecret == "" {
			return nil, fmt.Errorf("nowpayments: api_key and ipn_secret are required")
		}
		return nowpay.New(nowpay.Config{
			APIKey:    cfg.APIKey,
			IPNSecret: cfg.IPNSecret,
			APIURL:    cfg.APIURL,
		}), nil

	default:
		return nil, fmt.Errorf("unknown provider_type: %s", row.ProviderType)
	}
}
