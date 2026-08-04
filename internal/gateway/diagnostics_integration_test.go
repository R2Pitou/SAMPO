package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sampo/internal/app"
	"sampo/internal/catalog"
	"sampo/internal/diagnostics"
	"sampo/internal/domain"
	"sampo/internal/seshat"
)

func TestDebugSessionReconstructsEnrollAndScanWorkflow(t *testing.T) {
	const baseURL = "http://127.0.0.1:43124"
	ctx := context.Background()
	store, err := catalog.Open(ctx, filepath.Join(t.TempDir(), "catalogue.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	manager := diagnostics.NewManager(filepath.Join(t.TempDir(), "debug-sessions"), diagnostics.BuildEnvironment(map[string]any{"catalogue_backend": "sqlite", "token": "must-not-appear"}))
	service := app.New(seshat.WithDiagnostics(store, manager), manager)
	gateway, err := New(service, ctx, log.New(io.Discard, "", 0), manager)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := gateway.Handler(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	cookie, csrf := bootstrapTestSession(t, handler, gateway, baseURL)

	start := authorizedMutation(http.MethodPost, baseURL+"/api/debug/start", nil, cookie, csrf, baseURL)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, start)
	if response.Code != http.StatusOK {
		t.Fatalf("start Debug Mode = %d: %s", response.Code, response.Body.String())
	}

	providerRoot := t.TempDir()
	const fileContents = "private canary file contents must never be narrated"
	if err := os.WriteFile(filepath.Join(providerRoot, "canary.txt"), []byte(fileContents), 0o600); err != nil {
		t.Fatal(err)
	}
	enroll := jsonRequest(t, http.MethodPost, baseURL+"/api/providers", map[string]string{"displayName": "Canary", "root": providerRoot})
	authorize(enroll, cookie, csrf, baseURL)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, enroll)
	if response.Code != http.StatusCreated {
		t.Fatalf("enroll = %d: %s", response.Code, response.Body.String())
	}
	var provider domain.Provider
	if err := json.NewDecoder(response.Body).Decode(&provider); err != nil {
		t.Fatal(err)
	}

	scan := authorizedMutation(http.MethodPost, baseURL+"/api/providers/"+provider.ID+"/scan", nil, cookie, csrf, baseURL)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, scan)
	if response.Code != http.StatusAccepted {
		t.Fatalf("scan = %d: %s", response.Code, response.Body.String())
	}
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := gateway.Wait(waitCtx); err != nil {
		t.Fatal(err)
	}

	stop := authorizedMutation(http.MethodPost, baseURL+"/api/debug/stop", nil, cookie, csrf, baseURL)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, stop)
	if response.Code != http.StatusOK {
		t.Fatalf("stop Debug Mode = %d: %s", response.Code, response.Body.String())
	}
	var session diagnostics.SessionInfo
	if err := json.NewDecoder(response.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}
	if session.Active || session.Status != "completed" || session.BundlePath == "" {
		t.Fatalf("unexpected completed session: %#v", session)
	}

	events := readRecordedEvents(t, filepath.Join(session.BundlePath, "events.jsonl"))
	assertCorrelatedComponents(t, events, "provider.enroll", "gateway", "application", "seshat")
	assertCorrelatedComponents(t, events, "provider.scan", "gateway", "application", "observer", "seshat")
	bundle, err := os.ReadFile(filepath.Join(session.BundlePath, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bundle), fileContents) || strings.Contains(string(bundle), "must-not-appear") {
		t.Fatal("diagnostic bundle contains redacted configuration or file contents")
	}
}

func bootstrapTestSession(t *testing.T, handler http.Handler, gateway *Gateway, baseURL string) (*http.Cookie, string) {
	t.Helper()
	request := jsonRequest(t, http.MethodPost, baseURL+"/session/bootstrap", map[string]string{"token": gateway.bootstrapSecret})
	request.Header.Set("Origin", baseURL)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("bootstrap = %d: %s", response.Code, response.Body.String())
	}
	cookie := response.Result().Cookies()[0]
	gateway.mu.Lock()
	csrf := gateway.sessions[cookie.Value].csrf
	gateway.mu.Unlock()
	return cookie, csrf
}

func authorizedMutation(method, url string, body io.Reader, cookie *http.Cookie, csrf, baseURL string) *http.Request {
	request := httptest.NewRequest(method, url, body)
	authorize(request, cookie, csrf, baseURL)
	return request
}

func authorize(request *http.Request, cookie *http.Cookie, csrf, baseURL string) {
	request.Header.Set("Origin", baseURL)
	request.Header.Set("X-CSRF-Token", csrf)
	request.AddCookie(cookie)
}

func readRecordedEvents(t *testing.T, path string) []diagnostics.RecordedEvent {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var events []diagnostics.RecordedEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event diagnostics.RecordedEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return events
}

func assertCorrelatedComponents(t *testing.T, events []diagnostics.RecordedEvent, operationPrefix string, wanted ...string) {
	t.Helper()
	byCorrelation := make(map[string]map[string]bool)
	for _, event := range events {
		if strings.HasPrefix(event.Operation, operationPrefix) {
			if byCorrelation[event.CorrelationID] == nil {
				byCorrelation[event.CorrelationID] = make(map[string]bool)
			}
			byCorrelation[event.CorrelationID][event.Component] = true
		}
	}
	for correlation, components := range byCorrelation {
		complete := true
		for _, component := range wanted {
			complete = complete && components[component]
		}
		if complete {
			return
		}
		t.Logf("correlation %s had components %v", correlation, components)
	}
	t.Fatalf("no %s correlation contained components %v", operationPrefix, wanted)
}
