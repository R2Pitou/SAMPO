package gateway

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"sampo/internal/diagnostics"
	"sampo/internal/domain"
)

//go:embed templates/*.html static/*
var assets embed.FS

const sessionCookie = "sampo_session"

type Application interface {
	EnrollFilesystem(context.Context, string, string) (domain.Provider, error)
	Scan(context.Context, string) (domain.ScanSummary, error)
	Providers(context.Context) ([]domain.Provider, error)
	Search(context.Context, string, int) ([]domain.SearchResult, error)
	Stats(context.Context) (domain.CatalogueStats, error)
}

type Gateway struct {
	app             Application
	background      context.Context
	logger          *log.Logger
	diagnostics     diagnostics.Controller
	templates       *template.Template
	bootstrapSecret string

	mu            sync.Mutex
	bootstrapUsed bool
	sessions      map[string]session
	scanWG        sync.WaitGroup
}

type session struct {
	csrf     string
	lastSeen time.Time
}

type sessionContextKey struct{}

func New(app Application, background context.Context, logger *log.Logger, diagnostic diagnostics.Controller) (*Gateway, error) {
	if logger == nil {
		logger = log.Default()
	}
	if diagnostic == nil {
		return nil, errors.New("Debug Mode controller is required")
	}
	templates, err := template.New("root").Funcs(template.FuncMap{
		"shortDigest": func(value string) string {
			if len(value) > 16 {
				return value[:16]
			}
			return value
		},
		"formatBytes": formatBytes,
		"formatTime": func(value time.Time) string {
			if value.IsZero() {
				return "never"
			}
			return value.Local().Format("2006-01-02 15:04:05")
		},
	}).ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse gateway templates: %w", err)
	}
	secret, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	return &Gateway{
		app: app, background: background, logger: logger, diagnostics: diagnostic, templates: templates,
		bootstrapSecret: secret, sessions: make(map[string]session),
	}, nil
}

func (g *Gateway) BootstrapURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/bootstrap#" + g.bootstrapSecret
}

func (g *Gateway) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		g.scanWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *Gateway) Handler(baseURL string) (http.Handler, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" {
		return nil, errors.New("invalid Gateway base URL")
	}
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, fmt.Errorf("open embedded static assets: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("GET /bootstrap", g.bootstrapPage)
	mux.HandleFunc("POST /session/bootstrap", g.bootstrapSession)
	mux.HandleFunc("GET /", g.index)
	mux.HandleFunc("GET /api/status", g.status)
	mux.HandleFunc("GET /api/providers", g.providers)
	mux.HandleFunc("POST /api/providers", g.enrollProvider)
	mux.HandleFunc("POST /api/providers/{id}/scan", g.scanProvider)
	mux.HandleFunc("GET /api/search", g.search)
	mux.HandleFunc("GET /api/debug", g.debugStatus)
	mux.HandleFunc("POST /api/debug/start", g.startDebug)
	mux.HandleFunc("POST /api/debug/stop", g.stopDebug)

	expectedOrigin := parsed.Scheme + "://" + parsed.Host
	return g.recoverPanics(g.securityMiddleware(mux, parsed.Host, expectedOrigin)), nil
}

func (g *Gateway) securityMiddleware(next http.Handler, expectedHost, expectedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.diagnostics.Enabled() {
			r = r.WithContext(diagnostics.EnsureCorrelation(r.Context()))
		}
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		if r.Host != expectedHost {
			g.validationFailure(r.Context(), "unexpected_host", r.URL.Path)
			g.logger.Printf("security reject reason=unexpected-host remote=%q host=%q", r.RemoteAddr, r.Host)
			http.Error(w, "invalid host", http.StatusBadRequest)
			return
		}
		if forwarded := r.Header.Get("Forwarded"); forwarded != "" || r.Header.Get("X-Forwarded-Host") != "" {
			g.validationFailure(r.Context(), "proxy_headers", r.URL.Path)
			g.logger.Printf("security reject reason=proxy-headers remote=%q", r.RemoteAddr)
			http.Error(w, "proxy headers are not accepted", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if r.Header.Get("Origin") != expectedOrigin {
				g.validationFailure(r.Context(), "invalid_origin", r.URL.Path)
				g.logger.Printf("security reject reason=origin path=%q remote=%q", r.URL.Path, r.RemoteAddr)
				http.Error(w, "invalid origin", http.StatusForbidden)
				return
			}
		}
		if strings.HasPrefix(r.URL.Path, "/static/") || r.URL.Path == "/bootstrap" || r.URL.Path == "/session/bootstrap" {
			next.ServeHTTP(w, r)
			return
		}
		sess, ok := g.authenticate(r)
		if !ok {
			g.validationFailure(r.Context(), "session_required", r.URL.Path)
			http.Error(w, "session required", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if r.Header.Get("X-CSRF-Token") != sess.csrf {
				g.validationFailure(r.Context(), "invalid_csrf", r.URL.Path)
				g.logger.Printf("security reject reason=csrf path=%q remote=%q", r.URL.Path, r.RemoteAddr)
				http.Error(w, "invalid request token", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionContextKey{}, sess)))
	})
}

func (g *Gateway) authenticate(r *http.Request) (session, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return session{}, false
	}
	now := time.Now().UTC()
	g.mu.Lock()
	defer g.mu.Unlock()
	sess, ok := g.sessions[cookie.Value]
	if !ok || now.Sub(sess.lastSeen) > 30*time.Minute {
		delete(g.sessions, cookie.Value)
		return session{}, false
	}
	sess.lastSeen = now
	g.sessions[cookie.Value] = sess
	return sess, true
}

func (g *Gateway) bootstrapPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := g.templates.ExecuteTemplate(w, "bootstrap.html", nil); err != nil {
		g.logger.Printf("gateway bootstrap render result=failed error=%q", err)
	}
}

func (g *Gateway) bootstrapSession(w http.ResponseWriter, r *http.Request) {
	if !isJSON(r.Header.Get("Content-Type")) {
		http.Error(w, "JSON body required", http.StatusUnsupportedMediaType)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var request struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(r.Body, &request); err != nil {
		http.Error(w, "invalid bootstrap request", http.StatusBadRequest)
		return
	}

	g.mu.Lock()
	if g.bootstrapUsed || subtle.ConstantTimeCompare([]byte(request.Token), []byte(g.bootstrapSecret)) != 1 {
		g.mu.Unlock()
		g.logger.Printf("security reject reason=bootstrap-token remote=%q", r.RemoteAddr)
		http.Error(w, "invalid bootstrap token", http.StatusForbidden)
		return
	}
	sessionID, err := randomToken(32)
	if err != nil {
		g.mu.Unlock()
		http.Error(w, "session unavailable", http.StatusInternalServerError)
		return
	}
	csrf, err := randomToken(32)
	if err != nil {
		g.mu.Unlock()
		http.Error(w, "session unavailable", http.StatusInternalServerError)
		return
	}
	g.bootstrapUsed = true
	g.sessions[sessionID] = session{csrf: csrf, lastSeen: time.Now().UTC()}
	g.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: sessionID, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteStrictMode, MaxAge: 30 * 60,
	})
	w.WriteHeader(http.StatusNoContent)
}

type indexData struct {
	CSRF      string
	Query     string
	Stats     domain.CatalogueStats
	Providers []domain.Provider
	Results   []domain.SearchResult
	Error     string
	Debug     diagnostics.SessionInfo
}

func (g *Gateway) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	sess := r.Context().Value(sessionContextKey{}).(session)
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	g.record(r.Context(), diagnostics.Event{Component: "gateway", Operation: "dashboard.view", Phase: "ui_action", Outcome: "requested", Attributes: map[string]any{"search_requested": query != ""}})
	data := indexData{CSRF: sess.csrf, Query: query, Debug: g.diagnostics.Status()}
	var err error
	data.Stats, err = g.app.Stats(r.Context())
	if err == nil {
		data.Providers, err = g.app.Providers(r.Context())
	}
	if err == nil && query != "" {
		data.Results, err = g.app.Search(r.Context(), query, 200)
	}
	if err != nil {
		data.Error = "Catalogue query failed. See the application log for details."
		g.logger.Printf("gateway index result=failed error=%q", err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := g.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		g.logger.Printf("gateway index render result=failed error=%q", err)
	}
}

func (g *Gateway) status(w http.ResponseWriter, r *http.Request) {
	stats, err := g.app.Stats(r.Context())
	writeJSON(w, stats, err)
}

func (g *Gateway) providers(w http.ResponseWriter, r *http.Request) {
	providers, err := g.app.Providers(r.Context())
	writeJSON(w, providers, err)
}

func (g *Gateway) enrollProvider(w http.ResponseWriter, r *http.Request) {
	ctx := diagnostics.EnsureCorrelation(r.Context())
	if !isJSON(r.Header.Get("Content-Type")) {
		g.validationFailure(ctx, "json_required", r.URL.Path)
		http.Error(w, "JSON body required", http.StatusUnsupportedMediaType)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var request struct {
		DisplayName string `json:"displayName"`
		Root        string `json:"root"`
	}
	if err := decodeJSON(r.Body, &request); err != nil {
		g.validationFailure(ctx, "invalid_provider_request", r.URL.Path)
		http.Error(w, "invalid provider request", http.StatusBadRequest)
		return
	}
	g.record(ctx, diagnostics.Event{Component: "gateway", Operation: "provider.enroll", Phase: "ui_action", Outcome: "submitted", Attributes: map[string]any{"display_name": request.DisplayName, "submitted_locator": request.Root}})
	started := time.Now()
	provider, err := g.app.EnrollFilesystem(ctx, request.DisplayName, request.Root)
	if err != nil {
		g.record(ctx, diagnostics.Event{Severity: diagnostics.SeverityWarn, Component: "gateway", Operation: "provider.enroll", Phase: "response", Outcome: "refused", Message: err.Error(), Duration: time.Since(started)})
		g.logger.Printf("provider enroll result=rejected reason=%q", err)
		writeAPIError(w, err, http.StatusBadRequest)
		return
	}
	g.record(ctx, diagnostics.Event{Component: "gateway", Operation: "provider.enroll", Phase: "response", Outcome: "accepted", Duration: time.Since(started), Attributes: map[string]any{"provider_id": provider.ID}})
	g.logger.Printf("provider enroll result=accepted provider=%q root=%q next=scan", provider.ID, provider.RootLocator)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(provider)
}

func (g *Gateway) scanProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" || id == "scan" {
		g.validationFailure(r.Context(), "provider_id_required", r.URL.Path)
		http.Error(w, "provider id required", http.StatusBadRequest)
		return
	}
	ctx := diagnostics.EnsureCorrelation(r.Context())
	g.record(ctx, diagnostics.Event{Component: "gateway", Operation: "provider.scan", Phase: "ui_action", Outcome: "queued", Attributes: map[string]any{"provider_id": id}})
	correlationID := diagnostics.CorrelationID(ctx)
	g.logger.Printf("provider scan action=queued provider=%q", id)
	g.scanWG.Add(1)
	go func() {
		defer g.scanWG.Done()
		scanContext := diagnostics.WithCorrelation(g.background, correlationID)
		started := time.Now()
		summary, err := g.app.Scan(scanContext, id)
		if err != nil {
			g.record(scanContext, diagnostics.Event{Severity: diagnostics.SeverityError, Component: "gateway", Operation: "provider.scan", Phase: "background_result", Outcome: "failed", Message: err.Error(), Duration: time.Since(started), Attributes: map[string]any{"provider_id": id}})
			g.logger.Printf("provider scan result=failed provider=%q error=%q", id, err)
			return
		}
		g.record(scanContext, diagnostics.Event{Component: "gateway", Operation: "provider.scan", Phase: "background_result", Outcome: "completed", Duration: time.Since(started), Attributes: map[string]any{"provider_id": id, "observed": summary.Observed, "unstable": summary.Unstable, "issues": len(summary.Issues)}})
		g.logger.Printf("provider scan result=complete provider=%q observed=%d unstable=%d issues=%d next=reconcile-complete",
			id, summary.Observed, summary.Unstable, len(summary.Issues))
	}()
	w.WriteHeader(http.StatusAccepted)
}

func (g *Gateway) debugStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, g.diagnostics.Status(), nil)
}

func (g *Gateway) startDebug(w http.ResponseWriter, r *http.Request) {
	ctx := diagnostics.EnsureCorrelation(r.Context())
	info, err := g.diagnostics.Start(ctx)
	if err != nil {
		writeAPIError(w, err, http.StatusConflict)
		return
	}
	g.record(ctx, diagnostics.Event{Component: "gateway", Operation: "debug.session", Phase: "ui_action", Outcome: "started"})
	writeJSON(w, info, nil)
}

func (g *Gateway) stopDebug(w http.ResponseWriter, r *http.Request) {
	ctx := diagnostics.EnsureCorrelation(r.Context())
	g.record(ctx, diagnostics.Event{Component: "gateway", Operation: "debug.session", Phase: "ui_action", Outcome: "stop_requested"})
	info, err := g.diagnostics.Stop(ctx)
	if err != nil {
		writeAPIError(w, err, http.StatusConflict)
		return
	}
	writeJSON(w, info, nil)
}

func (g *Gateway) record(ctx context.Context, event diagnostics.Event) {
	if g.diagnostics.Enabled() {
		g.diagnostics.Record(ctx, event)
	}
}

func (g *Gateway) validationFailure(ctx context.Context, reason, path string) {
	g.record(ctx, diagnostics.Event{Severity: diagnostics.SeverityWarn, Component: "gateway", Operation: "request.validate", Phase: "validation", Outcome: "refused", Message: reason, Attributes: map[string]any{"path": path}})
}

func (g *Gateway) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				g.diagnostics.CapturePanic(r.Context(), "gateway", "http.request", recovered, debug.Stack())
				g.logger.Printf("gateway panic path=%q panic=%q", r.URL.Path, recovered)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (g *Gateway) search(w http.ResponseWriter, r *http.Request) {
	results, err := g.app.Search(r.Context(), r.URL.Query().Get("q"), 200)
	writeJSON(w, results, err)
}

func writeJSON(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeAPIError(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeAPIError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func decodeJSON(reader io.Reader, value any) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON value")
	}
	return nil
}

func isJSON(value string) bool {
	value = strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	return value == "application/json"
}

func randomToken(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func formatBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	divisor, exponent := int64(unit), 0
	for n := value / unit; n >= unit; n /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(value)/float64(divisor), "KMGTPE"[exponent])
}
