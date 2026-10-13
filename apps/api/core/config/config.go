// Package config is the catalog of the API's environment variables: every
// variable the API reads is a field of Config, with its type and default.
// Read them through Get, never os.Getenv.
//
// Load runs once in cmd/main.go and stops the API when a required variable is
// missing or a value doesn't parse. Get parses the environment on every call
// (microseconds) instead of caching, so tests can change variables with
// t.Setenv.
package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	// Server
	GinMode      string `env:"GIN_MODE"`
	Port         string `env:"PORT" envDefault:"8080"` // read by gin itself (r.Run)
	HTTPSEnabled bool   `env:"HTTPS_ENABLED"`
	// Comma-separated IPs or CIDRs; empty means the defaults, "none" trusts no proxy.
	TrustedProxies string `env:"TRUSTED_PROXIES"`
	TZ             string `env:"TZ"`

	// Database
	DBHost       string `env:"DB_HOST,required,notEmpty"`
	DBPort       string `env:"DB_PORT,required,notEmpty"`
	DBUser       string `env:"DB_USER,required,notEmpty"`
	DBPassword   string `env:"DB_PASSWORD,required,notEmpty"`
	DBName       string `env:"DB_NAME,required,notEmpty"`
	DBSSLMode    string `env:"DB_SSL_MODE" envDefault:"disable"`
	DBRootName   string `env:"DB_ROOT_NAME" envDefault:"postgres"`
	TenantDBMode string `env:"TENANTDB_MODE" envDefault:"warn"`

	// Auth
	AuthKey        string `env:"AUTH_KEY,required,notEmpty"`
	AuthExp        int64  `env:"AUTH_EXP,required,notEmpty"`       // access token, minutes
	AuthRefreshExp int64  `env:"AUTH_REFRESH_EXP" envDefault:"30"` // refresh token, days

	// Encryption keys (32 bytes, base64). core/secretbox reads them by name.
	// SIGNATURE_ENCRYPTION_KEY is required in release; outside it a dev key is used.
	SignatureEncryptionKey  string `env:"SIGNATURE_ENCRYPTION_KEY"`
	AttachmentEncryptionKey string `env:"ATTACHMENT_ENCRYPTION_KEY"` // unset: attachments disabled
	WhatsAppEncryptionKey   string `env:"WHATSAPP_ENCRYPTION_KEY"`   // unset: SIGNATURE_ENCRYPTION_KEY

	// RabbitMQ
	RabbitMQUser     string `env:"RABBITMQ_USER" envDefault:"guest"`
	RabbitMQPassword string `env:"RABBITMQ_PASSWORD" envDefault:"guest"`
	RabbitMQHost     string `env:"RABBITMQ_HOST" envDefault:"rabbitmq"`
	RabbitMQPort     string `env:"RABBITMQ_PORT" envDefault:"5672"`

	// Internal services
	GotenbergURL        string `env:"GOTENBERG_URL" envDefault:"http://gotenberg:3000"`
	SRISignerServiceURL string `env:"SRI_SIGNER_SERVICE_URL" envDefault:"http://sri-xml-signer:9000"`
	ICD11APIURL         string `env:"ICD11_API_URL" envDefault:"http://icd-api:80"`
	ICD11Release        string `env:"ICD11_RELEASE" envDefault:"2026-01"`
	ICD11Language       string `env:"ICD11_LANGUAGE" envDefault:"es"`

	// SRI ambiente: "1"/"test" pruebas, "2"/"prod" producción.
	SRIEnv string `env:"SRI_ENV"`

	// Public URLs of the web app (links in emails, redirects, TV display).
	FrontendURL  string `env:"FRONTEND_URL"`
	PublicAppURL string `env:"PUBLIC_APP_URL" envDefault:"http://localhost:5173"`

	// Google Calendar (unset: integration disabled)
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURL  string `env:"GOOGLE_REDIRECT_URL"`

	// Email
	ResendAPIKey     string `env:"RESEND_API_KEY"`
	SMTPFrom         string `env:"SMTP_FROM" envDefault:"noreply@pengistudio.com"`
	SMTPFromName     string `env:"SMTP_FROM_NAME" envDefault:"Gentoo"`
	ContactEmail     string `env:"CONTACT_EMAIL" envDefault:"pengiservicios@gmail.com"`
	ContactRateLimit int    `env:"CONTACT_RATE_LIMIT" envDefault:"3"` // per IP per hour

	// dLocal Go payments ("production" uses the live API, anything else sandbox)
	DLocalEnv             string `env:"DLOCAL_ENV"`
	DLocalAPIKey          string `env:"DLOCAL_API_KEY"`
	DLocalSecret          string `env:"DLOCAL_SECRET"`
	DLocalNotificationURL string `env:"DLOCAL_NOTIFICATION_URL"`

	// Meta / WhatsApp (unset: WhatsApp disabled)
	MetaAppID                  string `env:"META_APP_ID"`
	MetaAppSecret              string `env:"META_APP_SECRET"`
	MetaESConfigID             string `env:"META_ES_CONFIG_ID"`
	WhatsAppGraphVersion       string `env:"WHATSAPP_GRAPH_VERSION"`
	WhatsAppWebhookVerifyToken string `env:"WHATSAPP_WEBHOOK_VERIFY_TOKEN"`

	// Read only by committed code migrations, which can't be edited
	// (docs/backend/api-code-migration.md): they still call os.Getenv.
	AllowedOrigins          string `env:"ALLOWED_ORIGINS"`
	BackofficeAdminPassword string `env:"BACKOFFICE_ADMIN_PASSWORD"`
}

// IsRelease reports whether the API runs in production mode (GIN_MODE=release).
func (c Config) IsRelease() bool { return c.GinMode == "release" }

// Get returns the current configuration. Values that fail to parse are left
// at their zero value; Load has already rejected them at startup.
func Get() Config {
	cfg, _ := env.ParseAs[Config]()
	return cfg
}

// Load parses and validates the environment: required variables, values that
// don't parse, and the variables only release requires.
func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	errs, reported := envErrors(err)
	if cfg.AuthExp <= 0 && !reported["AUTH_EXP"] {
		errs = append(errs, errors.New(`env: "AUTH_EXP" must be a positive number of minutes`))
	}
	if cfg.IsRelease() && strings.TrimSpace(cfg.SignatureEncryptionKey) == "" {
		errs = append(errs, errors.New(`env: "SIGNATURE_ENCRYPTION_KEY" is required with GIN_MODE=release`))
	}
	return cfg, errors.Join(errs...)
}

// envErrors splits a parse error into one error per variable, named by the
// variable rather than the struct field, and reports which variables failed.
func envErrors(err error) ([]error, map[string]bool) {
	reported := map[string]bool{}
	var agg env.AggregateError
	if !errors.As(err, &agg) {
		if err != nil {
			return []error{err}, reported
		}
		return nil, reported
	}
	fields := reflect.TypeOf(Config{})
	errs := make([]error, 0, len(agg.Errors))
	for _, e := range agg.Errors {
		var name string
		var parseErr env.ParseError
		var requiredErr env.VarIsNotSetError
		var emptyErr env.EmptyVarError
		switch {
		case errors.As(e, &parseErr):
			if f, ok := fields.FieldByName(parseErr.Name); ok {
				name = strings.Split(f.Tag.Get("env"), ",")[0]
				e = fmt.Errorf("env: %q is not a valid %s: %w", name, parseErr.Type, parseErr.Err)
			}
		case errors.As(e, &requiredErr):
			name = requiredErr.Key
		case errors.As(e, &emptyErr):
			name = emptyErr.Key
		}
		reported[name] = true
		errs = append(errs, e)
	}
	return errs, reported
}
