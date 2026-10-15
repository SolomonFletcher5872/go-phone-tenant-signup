package phonesaas

import (
	"context"
	"errors"
	"testing"
)

type stubIdentity struct {
	verifyCalls int
}

func (s *stubIdentity) StartPhoneSignup(context.Context, string, string) error { return nil }
func (s *stubIdentity) VerifyPhone(context.Context, string, string) error {
	s.verifyCalls++
	return nil
}

func TestLoginHonorsAccountLifecycle(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		wantErr      error
		wantVerifies int
	}{
		{name: "active account reaches phone verification", status: "active", wantVerifies: 1},
		{name: "suspended account is rejected before verification", status: "suspended", wantErr: ErrAccountSuspended},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity := &stubIdentity{}
			service := NewSignupService(identity)
			service.accounts["+15551234567"] = Account{ID: "acct-1", TenantName: "Northwind", Phone: "+15551234567", Status: tt.status}

			_, err := service.Login(context.Background(), "+15551234567", "123456")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Login() error = %v, want %v", err, tt.wantErr)
			}
			if identity.verifyCalls != tt.wantVerifies {
				t.Fatalf("VerifyPhone() calls = %d, want %d", identity.verifyCalls, tt.wantVerifies)
			}
		})
	}
}
