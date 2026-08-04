package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sampo/internal/diagnostics"
	"sampo/internal/domain"
)

type fakeApplication struct{}

func (fakeApplication) EnrollFilesystem(_ context.Context, name, root string) (domain.Provider, error) {
	return domain.Provider{ID: "provider-1", Kind: domain.ProviderFilesystem, DisplayName: name, RootLocator: root}, nil
}
func (fakeApplication) Scan(context.Context, string) (domain.ScanSummary, error) {
	return domain.ScanSummary{}, nil
}
func (fakeApplication) Providers(context.Context) ([]domain.Provider, error) { return nil, nil }
func (fakeApplication) Search(context.Context, string, int) ([]domain.SearchResult, error) {
	return nil, nil
}
func (fakeApplication) Stats(context.Context) (domain.CatalogueStats, error) {
	return domain.CatalogueStats{}, nil
}

func TestGatewayRequiresOneTimeSessionOriginAndCSRF(t *testing.T) {
	const baseURL = "http://127.0.0.1:43123"
	gateway, err := New(fakeApplication{}, context.Background(), log.New(io.Discard, "", 0), diagnostics.NewManager(t.TempDir(), diagnostics.BuildEnvironment(nil)))
	if err != nil {
		t.Fatal(err)
	}
	handler, err := gateway.Handler(baseURL)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, baseURL+"/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", response.Code)
	}

	badOrigin := jsonRequest(t, http.MethodPost, baseURL+"/session/bootstrap", map[string]string{"token": gateway.bootstrapSecret})
	badOrigin.Header.Set("Origin", "http://attacker.invalid")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, badOrigin)
	if response.Code != http.StatusForbidden {
		t.Fatalf("bad-origin bootstrap = %d, want 403", response.Code)
	}

	bootstrap := jsonRequest(t, http.MethodPost, baseURL+"/session/bootstrap", map[string]string{"token": gateway.bootstrapSecret})
	bootstrap.Header.Set("Origin", baseURL)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, bootstrap)
	if response.Code != http.StatusNoContent {
		t.Fatalf("bootstrap = %d: %s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("unsafe session cookie: %#v", cookies)
	}

	reuse := jsonRequest(t, http.MethodPost, baseURL+"/session/bootstrap", map[string]string{"token": gateway.bootstrapSecret})
	reuse.Header.Set("Origin", baseURL)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, reuse)
	if response.Code != http.StatusForbidden {
		t.Fatalf("reused bootstrap = %d, want 403", response.Code)
	}

	page := httptest.NewRequest(http.MethodGet, baseURL+"/", nil)
	page.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, page)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "SAMPO") {
		t.Fatalf("authenticated page = %d: %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("missing Content-Security-Policy")
	}

	withoutCSRF := jsonRequest(t, http.MethodPost, baseURL+"/api/providers", map[string]string{"displayName": "Test", "root": `D:\Data`})
	withoutCSRF.Header.Set("Origin", baseURL)
	withoutCSRF.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, withoutCSRF)
	if response.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF = %d, want 403", response.Code)
	}

	gateway.mu.Lock()
	csrf := gateway.sessions[cookies[0].Value].csrf
	gateway.mu.Unlock()
	withCSRF := jsonRequest(t, http.MethodPost, baseURL+"/api/providers", map[string]string{"displayName": "Test", "root": `D:\Data`})
	withCSRF.Header.Set("Origin", baseURL)
	withCSRF.Header.Set("X-CSRF-Token", csrf)
	withCSRF.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, withCSRF)
	if response.Code != http.StatusCreated {
		t.Fatalf("valid mutation = %d: %s", response.Code, response.Body.String())
	}

	badHost := httptest.NewRequest(http.MethodGet, baseURL+"/bootstrap", nil)
	badHost.Host = "localhost:43123"
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, badHost)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("bad Host = %d, want 400", response.Code)
	}
}

func jsonRequest(t *testing.T, method, url string, body any) *http.Request {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, url, bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	return request
}
