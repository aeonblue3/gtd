package commands

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"net/http"

	"gtd/internal/api"
	"gtd/internal/api/handlers"
	"gtd/internal/auth"
	"gtd/internal/config"
	"gtd/internal/database"
	"gtd/internal/notify"
	"gtd/internal/storage"
	"github.com/skip2/go-qrcode"
	"golang.org/x/term"
)

// Server starts the next-generation authenticated API skeleton.
func Server(args []string) error {
	cfg, err := config.LoadServerConfig()
	if err != nil {
		return fmt.Errorf("load server config: %w", err)
	}

	store, err := storage.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("initialize sqlite store: %w", err)
	}
	defer store.Close()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open sqlite db: %w", err)
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("migrate sqlite db: %w", err)
	}

	svc := auth.NewService(db, cfg.TOTPIssuer)
	var notifier *notify.Service

	if len(args) > 0 && args[0] == "setup" {
		return runServerSetup(cfg, svc)
	}

	validator := authAdapter{svc: svc}
	count, err := svc.ActiveKeyCount(context.Background())
	if err != nil {
		return fmt.Errorf("count api keys: %w", err)
	}
	if count == 0 {
		_, raw, err := svc.CreateAPIKey("initial-cli-bootstrap")
		if err != nil {
			return fmt.Errorf("create initial api key: %w", err)
		}
		fmt.Printf("Initial API key (shown once): %s\n", raw)
	}

	if cfg.NotificationsEnabled {
		notifier, err = notify.NewService(db, notify.FromServerConfig(cfg), log.Default())
		if err != nil {
			return fmt.Errorf("initialize notification service: %w", err)
		}
		go notifier.Start(context.Background())
		fmt.Printf("Notifications enabled (interval: %ds, dry-run: %t)\n", cfg.NotificationCheckSeconds, cfg.NotificationDryRun)
	}
	notifyAdapter := notificationAdapter{cfg: cfg, service: notifier}
	srv := api.NewServer(store, cfg, validator, sessionAdapter{svc: svc}, notifyAdapter, notifyAdapter)
	fmt.Printf("Serving GTD API skeleton at http://%s\n", cfg.ListenAddr)
	return http.ListenAndServe(cfg.ListenAddr, srv.Handler())
}

func runServerSetup(cfg *config.ServerConfig, svc *auth.Service) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("GTD server setup")
	fmt.Println("--------------")

	email, err := promptLine(reader, "Email")
	if err != nil {
		return err
	}
	if strings.TrimSpace(email) == "" {
		return fmt.Errorf("email is required")
	}

	password, err := promptPassword("Password")
	if err != nil {
		return err
	}
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("password is required")
	}

	mfaSetup, err := svc.SetupMFA(context.Background(), email, password)
	if err != nil {
		return fmt.Errorf("setup mfa: %w", err)
	}

	qrPath := filepath.Join(filepath.Dir(cfg.DBPath), "totp-setup.png")
	if err := qrcode.WriteFile(mfaSetup.OTPAuthURL, qrcode.Medium, 256, qrPath); err != nil {
		return fmt.Errorf("write totp qr file: %w", err)
	}
	fmt.Printf("MFA secret generated.\n")
	fmt.Printf("QR code saved: %s\n", qrPath)
	fmt.Printf("OTP Auth URL: %s\n", mfaSetup.OTPAuthURL)

	code, err := promptLine(reader, "Enter current TOTP code")
	if err != nil {
		return err
	}
	if err := svc.VerifyMFA(context.Background(), email, code); err != nil {
		return fmt.Errorf("verify mfa: %w", err)
	}

	_, key, err := svc.CreateAPIKey("initial-cli-bootstrap")
	if err != nil {
		return fmt.Errorf("create initial api key: %w", err)
	}
	credPath := filepath.Join(filepath.Dir(cfg.DBPath), "credentials")
	content := fmt.Sprintf("gtd_api_key=%s\n", key)
	if err := os.WriteFile(credPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write credentials file: %w", err)
	}
	if err := os.Chmod(credPath, 0o600); err != nil {
		return fmt.Errorf("set credentials permissions: %w", err)
	}

	fmt.Println("Setup complete.")
	fmt.Printf("Credentials saved to %s\n", credPath)
	fmt.Println("Start server with: gtd server")
	return nil
}

func promptLine(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func promptPassword(label string) (string, error) {
	fmt.Printf("%s: ", label)
	bytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}

type sessionAdapter struct {
	svc *auth.Service
}

func (a sessionAdapter) ListActiveSessions(ctx context.Context) ([]handlers.SessionView, error) {
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

func (a sessionAdapter) RevokeSession(ctx context.Context, id string) error {
	return a.svc.RevokeSession(ctx, id)
}

func (a sessionAdapter) RevokeAllSessions(ctx context.Context) (int64, error) {
	return a.svc.RevokeAllSessions(ctx)
}

func (a sessionAdapter) RevokeSessionByAccessToken(ctx context.Context, token string) (string, error) {
	return a.svc.RevokeSessionByAccessToken(ctx, token)
}

func (a sessionAdapter) RotateSessionByAccessToken(ctx context.Context, token string) (*handlers.LoginResult, string, error) {
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

func (a sessionAdapter) RotateSessionByRefreshToken(ctx context.Context, refreshToken string) (*handlers.LoginResult, string, error) {
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

type authAdapter struct {
	svc *auth.Service
}

func (a authAdapter) ValidateAPIKey(ctx context.Context, token string) (bool, error) {
	return a.svc.ValidateAPIKey(ctx, token)
}

func (a authAdapter) CreateAPIKey(description string) (string, string, error) {
	return a.svc.CreateAPIKey(description)
}

func (a authAdapter) RevokeAPIKey(ctx context.Context, id string) error {
	return a.svc.RevokeAPIKey(ctx, id)
}

func (a authAdapter) RevokeAPIKeyByToken(ctx context.Context, token string) (string, error) {
	return a.svc.RevokeAPIKeyByToken(ctx, token)
}

func (a authAdapter) RotateAPIKeyByToken(ctx context.Context, token, description string) (string, string, string, error) {
	return a.svc.RotateAPIKeyByToken(ctx, token, description)
}

func (a authAdapter) SetupMFA(ctx context.Context, email, password string) (*handlers.MFASetupResult, error) {
	item, err := a.svc.SetupMFA(ctx, email, password)
	if err != nil {
		return nil, err
	}
	return &handlers.MFASetupResult{
		Email:      item.Email,
		Secret:     item.Secret,
		OTPAuthURL: item.OTPAuthURL,
	}, nil
}

func (a authAdapter) VerifyMFA(ctx context.Context, email, code string) error {
	return a.svc.VerifyMFA(ctx, email, code)
}

func (a authAdapter) Login(ctx context.Context, email, password, totpCode string) (*handlers.LoginResult, error) {
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

func (a authAdapter) Record(ctx context.Context, eventType, ipAddress, userAgent string, metadata map[string]any) error {
	return a.svc.Record(ctx, eventType, ipAddress, userAgent, metadata)
}

type notificationAdapter struct {
	cfg     *config.ServerConfig
	service *notify.Service
}

func (a notificationAdapter) RunOnce(ctx context.Context) (handlers.NotificationRunStats, error) {
	if a.service == nil {
		return handlers.NotificationRunStats{}, fmt.Errorf("notification service is not enabled")
	}
	stats, err := a.service.RunOnce(ctx)
	if err != nil {
		return handlers.NotificationRunStats{}, err
	}
	return handlers.NotificationRunStats{
		Scanned:   stats.Scanned,
		Attempted: stats.Attempted,
		Sent:      stats.Sent,
		DryRun:    stats.DryRun,
		Failed:    stats.Failed,
		Skipped:   stats.Skipped,
	}, nil
}

func (a notificationAdapter) GetNotificationConfig() handlers.NotificationConfig {
	restartRequired := a.service == nil && a.cfg.NotificationsEnabled
	return handlers.NotificationConfig{
		Enabled:         a.cfg.NotificationsEnabled,
		CheckSeconds:    a.cfg.NotificationCheckSeconds,
		DryRun:          a.cfg.NotificationDryRun,
		EmailTo:         a.cfg.NotificationEmailTo,
		EmailFrom:       a.cfg.NotificationEmailFrom,
		SMTPHost:        a.cfg.NotificationSMTPHost,
		SMTPPort:        a.cfg.NotificationSMTPPort,
		SMTPUsername:    a.cfg.NotificationSMTPUsername,
		HasSMTPPassword: strings.TrimSpace(a.cfg.NotificationSMTPPassword) != "",
		RestartRequired: restartRequired,
	}
}

func (a notificationAdapter) UpdateNotificationConfig(ctx context.Context, next handlers.NotificationConfigUpdate) (handlers.NotificationConfig, error) {
	_ = ctx
	if next.Enabled != nil {
		a.cfg.NotificationsEnabled = *next.Enabled
	}
	if next.CheckSeconds != nil {
		a.cfg.NotificationCheckSeconds = *next.CheckSeconds
	}
	if next.DryRun != nil {
		a.cfg.NotificationDryRun = *next.DryRun
	}
	if next.EmailTo != nil {
		a.cfg.NotificationEmailTo = *next.EmailTo
	}
	if next.EmailFrom != nil {
		a.cfg.NotificationEmailFrom = *next.EmailFrom
	}
	if next.SMTPHost != nil {
		a.cfg.NotificationSMTPHost = *next.SMTPHost
	}
	if next.SMTPPort != nil {
		a.cfg.NotificationSMTPPort = *next.SMTPPort
	}
	if next.SMTPUsername != nil {
		a.cfg.NotificationSMTPUsername = *next.SMTPUsername
	}
	if next.SMTPPassword != nil {
		a.cfg.NotificationSMTPPassword = *next.SMTPPassword
	}

	view := a.GetNotificationConfig()
	if err := validateNotificationUpdate(view); err != nil {
		return handlers.NotificationConfig{}, err
	}
	if err := config.SaveServerConfig(a.cfg); err != nil {
		return handlers.NotificationConfig{}, err
	}
	if a.service != nil {
		if err := a.service.UpdateConfig(notify.FromServerConfig(a.cfg)); err != nil {
			return handlers.NotificationConfig{}, err
		}
	}
	cfg := a.GetNotificationConfig()
	cfg.RestartRequired = a.service == nil && a.cfg.NotificationsEnabled
	return cfg, nil
}

func validateNotificationUpdate(cfg handlers.NotificationConfig) error {
	if cfg.CheckSeconds <= 0 {
		return fmt.Errorf("check_seconds must be > 0")
	}
	if cfg.SMTPPort < 0 {
		return fmt.Errorf("smtp_port must be >= 0")
	}
	if !cfg.DryRun {
		if strings.TrimSpace(cfg.EmailTo) == "" || strings.TrimSpace(cfg.EmailFrom) == "" || strings.TrimSpace(cfg.SMTPHost) == "" || cfg.SMTPPort == 0 {
			return fmt.Errorf("smtp configuration is required when dry_run is false")
		}
	}
	return nil
}

