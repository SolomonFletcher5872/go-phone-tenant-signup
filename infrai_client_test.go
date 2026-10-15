package phonesaas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestStartPhoneSignupUsesSMSContract(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/v1/auth/phone/send_code":
			if !reflect.DeepEqual(body, map[string]any{"phone": "+15551234567", "purpose": "signup", "locale": "en"}) {
				t.Errorf("send_code body = %v", body)
			}
			_, _ = w.Write([]byte(`{"ok":true,"data":{"unexpected":"auth-result"}}`))
		case "/v1/sms/otp":
			if !reflect.DeepEqual(body, map[string]any{"to": "+15551234567"}) {
				t.Errorf("sms.otp body = %v", body)
			}
			_, _ = w.Write([]byte(`{"ok":true,"data":{}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	if err := NewInfraiClient(server.URL, "test-key", server.Client()).StartPhoneSignup(context.Background(), "+15551234567", "en"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"/v1/auth/phone/send_code", "/v1/sms/otp"}) {
		t.Errorf("calls = %v", calls)
	}
}
