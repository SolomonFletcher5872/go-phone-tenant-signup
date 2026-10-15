package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	phonesaas "example.com/infrai-phone-saas"
)

type server struct {
	signups *phonesaas.SignupService
}

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	baseURL := os.Getenv("INFRAI_BASE_URL")
	service := phonesaas.NewSignupService(phonesaas.NewInfraiClient(baseURL, apiKey, nil))
	s := &server{signups: service}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /signup/code", s.startSignup)
	mux.HandleFunc("POST /signup/verify", s.completeSignup)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("POST /admin/accounts/{id}/status", s.setStatus)

	log.Println("phone SaaS listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func (s *server) startSignup(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TenantName string `json:"tenant_name"`
		Phone      string `json:"phone"`
		Locale     string `json:"locale"`
	}
	if !decode(w, r, &input) {
		return
	}
	if err := s.signups.StartSignup(r.Context(), input.TenantName, input.Phone, input.Locale); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "code_sent"})
}

func (s *server) completeSignup(w http.ResponseWriter, r *http.Request) {
	phone, code, ok := phoneCode(w, r)
	if !ok {
		return
	}
	account, err := s.signups.CompleteSignup(r.Context(), phone, code)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, account)
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	phone, code, ok := phoneCode(w, r)
	if !ok {
		return
	}
	account, err := s.signups.Login(r.Context(), phone, code)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "authenticated", "account": account})
}

func (s *server) setStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status string `json:"status"`
	}
	if !decode(w, r, &input) {
		return
	}
	account, err := s.signups.SetAccountStatus(r.PathValue("id"), input.Status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, account)
}

func phoneCode(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	var input struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if !decode(w, r, &input) {
		return "", "", false
	}
	if input.Phone == "" || input.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone and code are required"})
		return "", "", false
	}
	return input.Phone, input.Code, true
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	var apiErr *phonesaas.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
		status = apiErr.StatusCode
	}
	if errors.Is(err, phonesaas.ErrAccountMissing) {
		status = http.StatusNotFound
	}
	if errors.Is(err, phonesaas.ErrAccountSuspended) {
		status = http.StatusForbidden
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil && !strings.Contains(err.Error(), "closed") {
		log.Printf("encode response: %v", err)
	}
}
