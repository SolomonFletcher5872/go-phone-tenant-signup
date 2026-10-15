package phonesaas

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrAccountMissing   = errors.New("account not found")
	ErrAccountSuspended = errors.New("account suspended")
	ErrSignupMissing    = errors.New("signup not started")
)

type PhoneIdentity interface {
	StartPhoneSignup(ctx context.Context, phone, locale string) error
	VerifyPhone(ctx context.Context, phone, code string) error
}

type Account struct {
	ID         string `json:"id"`
	TenantName string `json:"tenant_name"`
	Phone      string `json:"phone"`
	Status     string `json:"status"`
}

type SignupService struct {
	identity PhoneIdentity
	mu       sync.RWMutex
	pending  map[string]string
	accounts map[string]Account
}

func NewSignupService(identity PhoneIdentity) *SignupService {
	return &SignupService{identity: identity, pending: make(map[string]string), accounts: make(map[string]Account)}
}

func (s *SignupService) StartSignup(ctx context.Context, tenantName, phone, locale string) error {
	if tenantName == "" || phone == "" {
		return errors.New("tenant_name and phone are required")
	}
	if locale == "" {
		locale = "en"
	}
	if err := s.identity.StartPhoneSignup(ctx, phone, locale); err != nil {
		return err
	}
	s.mu.Lock()
	s.pending[phone] = tenantName
	s.mu.Unlock()
	return nil
}

func (s *SignupService) CompleteSignup(ctx context.Context, phone, code string) (Account, error) {
	s.mu.RLock()
	tenantName, pending := s.pending[phone]
	s.mu.RUnlock()
	if !pending {
		return Account{}, ErrSignupMissing
	}
	if err := s.identity.VerifyPhone(ctx, phone, code); err != nil {
		return Account{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	account := Account{ID: fmt.Sprintf("acct-%d", len(s.accounts)+1), TenantName: tenantName, Phone: phone, Status: "active"}
	s.accounts[phone] = account
	delete(s.pending, phone)
	return account, nil
}

func (s *SignupService) Login(ctx context.Context, phone, code string) (Account, error) {
	s.mu.RLock()
	account, found := s.accounts[phone]
	s.mu.RUnlock()
	if !found {
		return Account{}, ErrAccountMissing
	}
	if account.Status != "active" {
		return Account{}, ErrAccountSuspended
	}
	if err := s.identity.VerifyPhone(ctx, phone, code); err != nil {
		return Account{}, err
	}
	return account, nil
}

func (s *SignupService) SetAccountStatus(accountID, status string) (Account, error) {
	if status != "active" && status != "suspended" {
		return Account{}, errors.New("status must be active or suspended")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for phone, account := range s.accounts {
		if account.ID == accountID {
			account.Status = status
			s.accounts[phone] = account
			return account, nil
		}
	}
	return Account{}, ErrAccountMissing
}
