package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gtd/internal/api/handlers"
	"gtd/internal/auth"
	"gtd/internal/config"
	"gtd/internal/database"
	"gtd/internal/storage"
	"github.com/pquerna/otp/totp"
)

type allowAllValidator struct{}

func (allowAllValidator) ValidateAPIKey(_ context.Context, _ string) (bool, error) { return true, nil }

type fakeSessions struct {
	items   []handlers.SessionView
	revoked []string
}

type fakeNotifyRunner struct {
	stats handlers.NotificationRunStats
	err   error
}

func (f fakeNotifyRunner) RunOnce(_ context.Context) (handlers.NotificationRunStats, error) {
	if f.err != nil {
		return handlers.NotificationRunStats{}, f.err
	}
	return f.stats, nil
}

type fakeNotifyStore struct {
	cfg handlers.NotificationConfig
}

func (f *fakeNotifyStore) GetNotificationConfig() handlers.NotificationConfig {
	return f.cfg
}

func (f *fakeNotifyStore) UpdateNotificationConfig(_ context.Context, next handlers.NotificationConfigUpdate) (handlers.NotificationConfig, error) {
	if next.Enabled != nil {
		f.cfg.Enabled = *next.Enabled
	}
	if next.CheckSeconds != nil {
		f.cfg.CheckSeconds = *next.CheckSeconds
	}
	if next.DryRun != nil {
		f.cfg.DryRun = *next.DryRun
	}
	if next.EmailTo != nil {
		f.cfg.EmailTo = *next.EmailTo
	}
	if next.EmailFrom != nil {
		f.cfg.EmailFrom = *next.EmailFrom
	}
	if next.SMTPHost != nil {
		f.cfg.SMTPHost = *next.SMTPHost
	}
	if next.SMTPPort != nil {
		f.cfg.SMTPPort = *next.SMTPPort
	}
	if next.SMTPUsername != nil {
		f.cfg.SMTPUsername = *next.SMTPUsername
	}
	if next.SMTPPassword != nil {
		f.cfg.HasSMTPPassword = *next.SMTPPassword != ""
	}
	return f.cfg, nil
}

type fullAuthAdapter struct {
	svc *auth.Service
}

func (a fullAuthAdapter) ValidateAPIKey(ctx context.Context, token string) (bool, error) {
	return a.svc.ValidateAPIKey(ctx, token)
}

func (a fullAuthAdapter) CreateAPIKey(description string) (string, string, error) {
	return a.svc.CreateAPIKey(description)
}

func (a fullAuthAdapter) RevokeAPIKey(ctx context.Context, id string) error {
	return a.svc.RevokeAPIKey(ctx, id)
}

func (a fullAuthAdapter) RevokeAPIKeyByToken(ctx context.Context, token string) (string, error) {
	return a.svc.RevokeAPIKeyByToken(ctx, token)
}

func (a fullAuthAdapter) RotateAPIKeyByToken(ctx context.Context, token, description string) (string, string, string, error) {
	return a.svc.RotateAPIKeyByToken(ctx, token, description)
}

func (a fullAuthAdapter) SetupMFA(ctx context.Context, email, password string) (*handlers.MFASetupResult, error) {
	item, err := a.svc.SetupMFA(ctx, email, password)
	if err != nil {
		return nil, err
	}
	return &handlers.MFASetupResult{Email: item.Email, Secret: item.Secret, OTPAuthURL: item.OTPAuthURL}, nil
}

func (a fullAuthAdapter) VerifyMFA(ctx context.Context, email, code string) error {
	return a.svc.VerifyMFA(ctx, email, code)
}

func (a fullAuthAdapter) Login(ctx context.Context, email, password, totpCode string) (*handlers.LoginResult, error) {
	item, err := a.svc.Login(ctx, email, password, totpCode)
	if err != nil {
		return nil, err
	}
	return &handlers.LoginResult{
		SessionID:    item.SessionID,
		UserID:       item.UserID,
		AccessToken:  item.AccessToken,
		RefreshToken: item.RefreshToken,
		ExpiresAt:    item.ExpiresAt,
	}, nil
}

func (f *fakeSessions) ListActiveSessions(_ context.Context) ([]handlers.SessionView, error) {
	return append([]handlers.SessionView{}, f.items...), nil
}

func (f *fakeSessions) RevokeSession(_ context.Context, id string) error {
	f.revoked = append(f.revoked, id)
	return nil
}

func (f *fakeSessions) RevokeAllSessions(_ context.Context) (int64, error) {
	n := int64(len(f.items))
	f.items = nil
	return n, nil
}

func (f *fakeSessions) RevokeSessionByAccessToken(_ context.Context, _ string) (string, error) {
	return "s-token", nil
}

func (f *fakeSessions) RotateSessionByAccessToken(_ context.Context, _ string) (*handlers.LoginResult, string, error) {
	return &handlers.LoginResult{
		SessionID:    "s-new",
		UserID:       "u-1",
		AccessToken:  "access-new",
		RefreshToken: "refresh-new",
		ExpiresAt:    999,
	}, "s-old", nil
}

func (f *fakeSessions) RotateSessionByRefreshToken(_ context.Context, _ string) (*handlers.LoginResult, string, error) {
	return &handlers.LoginResult{
		SessionID:    "s-new-r",
		UserID:       "u-1",
		AccessToken:  "access-new-r",
		RefreshToken: "refresh-new-r",
		ExpiresAt:    1001,
	}, "s-old-r", nil
}

func TestHealthRoute(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, &config.ServerConfig{APITokenName: "gtd_api_key"}, allowAllValidator{}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestTasksRequireAuth(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, &config.ServerConfig{APITokenName: "gtd_api_key"}, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestCreateTaskWithBearerAuth(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, &config.ServerConfig{APITokenName: "gtd_api_key"}, allowAllValidator{}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"title":"API skeleton task"}`))
	req.Header.Set("Authorization", "Bearer test-key")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateTaskWithBearerAuthV1Prefix(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, &config.ServerConfig{APITokenName: "gtd_api_key"}, allowAllValidator{}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(`{"title":"API v1 task"}`))
	req.Header.Set("Authorization", "Bearer test-key")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTaskValidationInvalidStatus(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, &config.ServerConfig{APITokenName: "gtd_api_key"}, allowAllValidator{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"title":"t","status":"bad-status"}`))
	req.Header.Set("Authorization", "Bearer test-key")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTaskValidationInvalidPriorityFilter(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, &config.ServerConfig{APITokenName: "gtd_api_key"}, allowAllValidator{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/tasks?priority=urgent", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTaskUpdateClearDueDate(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, &config.ServerConfig{APITokenName: "gtd_api_key"}, allowAllValidator{}, nil, nil, nil)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(`{"title":"due task","dueDate":"2026-02-20T12:00:00Z"}`))
	createReq.Header.Set("Authorization", "Bearer test-key")
	createRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createRec.Code, createRec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("missing id in create response: %s", createRec.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/"+id, bytes.NewBufferString(`{"clearDueDate":true}`))
	updateReq.Header.Set("Authorization", "Bearer test-key")
	updateRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", updateRec.Code, updateRec.Body.String())
	}
	var updated map[string]any
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if v, ok := updated["dueDate"]; ok && v != nil {
		t.Fatalf("expected dueDate cleared, got %#v", v)
	}
}

func TestCSRFMissingForCookieAuthenticatedWrite(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, &config.ServerConfig{
		APITokenName: "gtd_api_key",
		CSRFCookie:   "gtd_csrf",
	}, allowAllValidator{}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"title":"cookie write"}`))
	req.AddCookie(&http.Cookie{Name: "gtd_api_key", Value: "cookie-token"})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCSRFAcceptedForCookieAuthenticatedWrite(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, &config.ServerConfig{
		APITokenName: "gtd_api_key",
		CSRFCookie:   "gtd_csrf",
	}, allowAllValidator{}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"title":"cookie write ok"}`))
	req.AddCookie(&http.Cookie{Name: "gtd_api_key", Value: "cookie-token"})
	req.AddCookie(&http.Cookie{Name: "gtd_csrf", Value: "csrf-ok"})
	req.Header.Set("X-CSRF-Token", "csrf-ok")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRefreshCookieRequiresCSRFHeader(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sessions := &fakeSessions{}
	srv := NewServer(store, &config.ServerConfig{
		APITokenName: "gtd_api_key",
		CSRFCookie:   "gtd_csrf",
	}, allowAllValidator{}, sessions, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh-cookie-token"})
	req.AddCookie(&http.Cookie{Name: "gtd_csrf", Value: "csrf-cookie"})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIKeyCreateAndRevoke(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "gtd.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}

	validator := auth.NewAPIKeyService(db)
	_, bootstrap, err := validator.CreateAPIKey("bootstrap")
	if err != nil {
		t.Fatal(err)
	}

	srv := NewServer(store, &config.ServerConfig{APITokenName: "gtd_api_key"}, validator, nil, nil, nil)

	createReq := httptest.NewRequest(http.MethodPost, "/api/auth/apikey/create", bytes.NewBufferString(`{"description":"secondary"}`))
	createReq.Header.Set("Authorization", "Bearer "+bootstrap)
	createRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createRec.Code, createRec.Body.String())
	}
	var created map[string]string
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("parse create response: %v", err)
	}
	if created["id"] == "" || created["api_key"] == "" {
		t.Fatalf("expected id and api_key, got: %#v", created)
	}

	revokeReq := httptest.NewRequest(http.MethodDelete, "/api/auth/apikey/"+created["id"], nil)
	revokeReq.Header.Set("Authorization", "Bearer "+bootstrap)
	revokeRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", revokeRec.Code, revokeRec.Body.String())
	}

	ok, err := validator.ValidateAPIKey(context.Background(), created["api_key"])
	if err != nil {
		t.Fatalf("validate revoked key: %v", err)
	}
	if ok {
		t.Fatal("expected revoked key to be invalid")
	}
}

func TestSessionEndpoints(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sessions := &fakeSessions{
		items: []handlers.SessionView{
			{ID: "s-1", UserID: "u-1", ExpiresAt: 100, CreatedAt: 10, Revoked: false},
			{ID: "s-2", UserID: "u-1", ExpiresAt: 200, CreatedAt: 20, Revoked: false},
		},
	}

	srv := NewServer(store, &config.ServerConfig{APITokenName: "gtd_api_key"}, allowAllValidator{}, sessions, nil, nil)

	listReq := httptest.NewRequest(http.MethodGet, "/api/auth/sessions", nil)
	listReq.Header.Set("Authorization", "Bearer any")
	listRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}

	revokeReq := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/s-1", nil)
	revokeReq.Header.Set("Authorization", "Bearer any")
	revokeRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", revokeRec.Code, revokeRec.Body.String())
	}

	revokeAllReq := httptest.NewRequest(http.MethodDelete, "/api/auth/sessions/all", nil)
	revokeAllReq.Header.Set("Authorization", "Bearer any")
	revokeAllRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(revokeAllRec, revokeAllReq)
	if revokeAllRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", revokeAllRec.Code, revokeAllRec.Body.String())
	}

	csrfReq := httptest.NewRequest(http.MethodGet, "/api/auth/csrf", nil)
	csrfReq.Header.Set("Authorization", "Bearer any")
	csrfRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(csrfRec, csrfReq)
	if csrfRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", csrfRec.Code, csrfRec.Body.String())
	}
}

func TestRefreshAndLogout(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "gtd.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}

	validator := auth.NewAPIKeyService(db)
	_, bootstrap, err := validator.CreateAPIKey("bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(store, &config.ServerConfig{APITokenName: "gtd_api_key"}, validator, nil, nil, nil)

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewBufferString(`{"description":"rotated"}`))
	refreshReq.Header.Set("Authorization", "Bearer "+bootstrap)
	refreshRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(refreshRec, refreshReq)
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", refreshRec.Code, refreshRec.Body.String())
	}
	var refreshed map[string]string
	if err := json.Unmarshal(refreshRec.Body.Bytes(), &refreshed); err != nil {
		t.Fatalf("parse refresh response: %v", err)
	}
	if refreshed["api_key"] == "" || refreshed["id"] == "" || refreshed["revoked_api_key_id"] == "" {
		t.Fatalf("unexpected refresh payload: %#v", refreshed)
	}
	ok, err := validator.ValidateAPIKey(context.Background(), bootstrap)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected old api key to be revoked after refresh")
	}
	ok, err = validator.ValidateAPIKey(context.Background(), refreshed["api_key"])
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected rotated api key to be valid")
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+refreshed["api_key"])
	logoutRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", logoutRec.Code, logoutRec.Body.String())
	}
	ok, err = validator.ValidateAPIKey(context.Background(), refreshed["api_key"])
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected api key to be revoked after logout")
	}
}

func TestPasswordTotpSetupAndLogin(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "gtd.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}

	adapter := fullAuthAdapter{svc: auth.NewService(db, "GTD-Test")}
	sessions := sessionAdapterForTest{svc: adapter.svc}
	srv := NewServer(store, &config.ServerConfig{APITokenName: "gtd_api_key"}, adapter, sessions, nil, nil)

	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup-mfa", bytes.NewBufferString(`{"email":"me@example.com","password":"strong-pass"}`))
	setupRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", setupRec.Code, setupRec.Body.String())
	}
	var setup map[string]string
	if err := json.Unmarshal(setupRec.Body.Bytes(), &setup); err != nil {
		t.Fatal(err)
	}
	secret := setup["secret"]
	if secret == "" {
		t.Fatalf("missing secret in response: %s", setupRec.Body.String())
	}

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/auth/verify-mfa", bytes.NewBufferString(`{"email":"me@example.com","code":"`+code+`"}`))
	verifyRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(verifyRec, verifyReq)
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", verifyRec.Code, verifyRec.Body.String())
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"email":"me@example.com","password":"strong-pass","totp_code":"`+code+`"}`))
	loginRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", loginRec.Code, loginRec.Body.String())
	}
	var login map[string]any
	if err := json.Unmarshal(loginRec.Body.Bytes(), &login); err != nil {
		t.Fatal(err)
	}
	access, _ := login["access_token"].(string)
	refresh, _ := login["refresh_token"].(string)
	if access == "" {
		t.Fatalf("missing access token in login response: %s", loginRec.Body.String())
	}
	if refresh == "" {
		t.Fatalf("missing refresh token in login response: %s", loginRec.Body.String())
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewBufferString(`{"refresh_token":"`+refresh+`"}`))
	refreshRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(refreshRec, refreshReq)
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from refresh token flow, got %d body=%s", refreshRec.Code, refreshRec.Body.String())
	}
	var refreshed map[string]any
	if err := json.Unmarshal(refreshRec.Body.Bytes(), &refreshed); err != nil {
		t.Fatal(err)
	}
	rotatedAccess, _ := refreshed["access_token"].(string)
	if rotatedAccess == "" {
		t.Fatalf("missing rotated access token in refresh response: %s", refreshRec.Body.String())
	}

	taskReq := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	taskReq.Header.Set("Authorization", "Bearer "+rotatedAccess)
	taskRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(taskRec, taskReq)
	if taskRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", taskRec.Code, taskRec.Body.String())
	}
}

func TestNotificationRunNowAndConfigEndpoints(t *testing.T) {
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "gtd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	runner := fakeNotifyRunner{
		stats: handlers.NotificationRunStats{Scanned: 3, Attempted: 2, DryRun: 2, Failed: 0, Skipped: 1},
	}
	cfgStore := &fakeNotifyStore{
		cfg: handlers.NotificationConfig{
			Enabled:      false,
			CheckSeconds: 300,
			DryRun:       true,
			EmailTo:      "",
			EmailFrom:    "gtd@localhost",
			SMTPHost:     "",
			SMTPPort:     587,
			SMTPUsername: "",
		},
	}
	srv := NewServer(store, &config.ServerConfig{APITokenName: "gtd_api_key"}, allowAllValidator{}, nil, runner, cfgStore)

	runReq := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/run-now", nil)
	runReq.Header.Set("Authorization", "Bearer any")
	runRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", runRec.Code, runRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/config", nil)
	getReq.Header.Set("Authorization", "Bearer any")
	getRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getRec.Code, getRec.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/config", bytes.NewBufferString(`{"enabled":true,"check_seconds":60,"email_to":"me@example.com"}`))
	updateReq.Header.Set("Authorization", "Bearer any")
	updateReq.Header.Set("X-CSRF-Token", "csrf-ok")
	updateReq.AddCookie(&http.Cookie{Name: "gtd_csrf", Value: "csrf-ok"})
	updateRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", updateRec.Code, updateRec.Body.String())
	}
	var updated handlers.NotificationConfig
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("parse update response: %v", err)
	}
	if !updated.Enabled {
		t.Fatalf("expected enabled=true after update, got %#v", updated)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	statusReq.Header.Set("Authorization", "Bearer any")
	statusRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", statusRec.Code, statusRec.Body.String())
	}
}

type sessionAdapterForTest struct {
	svc *auth.Service
}

func (a sessionAdapterForTest) ListActiveSessions(ctx context.Context) ([]handlers.SessionView, error) {
	items, err := a.svc.ListActiveSessions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]handlers.SessionView, 0, len(items))
	for _, item := range items {
		out = append(out, handlers.SessionView{
			ID:        item.ID,
			UserID:    item.UserID,
			ExpiresAt: item.ExpiresAt,
			CreatedAt: item.CreatedAt,
			Revoked:   item.Revoked,
		})
	}
	return out, nil
}

func (a sessionAdapterForTest) RevokeSession(ctx context.Context, id string) error {
	return a.svc.RevokeSession(ctx, id)
}

func (a sessionAdapterForTest) RevokeAllSessions(ctx context.Context) (int64, error) {
	return a.svc.RevokeAllSessions(ctx)
}

func (a sessionAdapterForTest) RevokeSessionByAccessToken(ctx context.Context, token string) (string, error) {
	return a.svc.RevokeSessionByAccessToken(ctx, token)
}

func (a sessionAdapterForTest) RotateSessionByAccessToken(ctx context.Context, token string) (*handlers.LoginResult, string, error) {
	next, oldID, err := a.svc.RotateSessionByAccessToken(ctx, token)
	if err != nil {
		return nil, "", err
	}
	return &handlers.LoginResult{
		SessionID:    next.ID,
		UserID:       next.UserID,
		AccessToken:  next.AccessToken,
		RefreshToken: next.RefreshToken,
		ExpiresAt:    next.ExpiresAt,
	}, oldID, nil
}

func (a sessionAdapterForTest) RotateSessionByRefreshToken(ctx context.Context, refreshToken string) (*handlers.LoginResult, string, error) {
	next, oldID, err := a.svc.RotateSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, "", err
	}
	return &handlers.LoginResult{
		SessionID:    next.ID,
		UserID:       next.UserID,
		AccessToken:  next.AccessToken,
		RefreshToken: next.RefreshToken,
		ExpiresAt:    next.ExpiresAt,
	}, oldID, nil
}

