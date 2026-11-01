package ai_services

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"pengi-med-saas/core/config"
	"pengi-med-saas/core/secretbox"
	ai_models "pengi-med-saas/features/ai/models"
	settings_models "pengi-med-saas/features/settings/models"

	"gorm.io/gorm"
)

// System settings (platform-wide, set in backoffice) of each AI function: the
// provider (base URL, API key, model, reasoning effort) and its price per
// million tokens, used to log the cost of each use. Clinics never choose.
// Every provider field left empty falls back to the env (AI_BASE_URL,
// AI_API_KEY, AI_DEFAULT_SUMMARY_MODEL, AI_REASONING_EFFORT), so switching
// provider needs no env change nor restart.
const (
	settingBaseURLPrefix     = "ai.base_url."
	settingAPIKeyPrefix      = "ai.api_key." // sealed with AI_ENCRYPTION_KEY
	settingModelPrefix       = "ai.model."
	settingEffortPrefix      = "ai.reasoning_effort."
	settingPriceInputPrefix  = "ai.price_input."
	settingPriceOutputPrefix = "ai.price_output."
)

// EncryptionKeyEnv seals the API keys saved in backoffice. Without it a key
// can't be saved there (the env key still works).
const EncryptionKeyEnv = "AI_ENCRYPTION_KEY"

// Errors of SaveSettings, mapped by the backoffice handler.
var (
	ErrUnknownFunction       = errors.New("ai settings: unknown function")
	ErrInvalidSettings       = errors.New("ai settings: invalid value")
	ErrInvalidBaseURL        = errors.New("ai settings: base URL must be an http(s) URL")
	ErrEncryptionUnavailable = errors.New("ai settings: " + EncryptionKeyEnv + " is not set")
)

// Key sources reported to backoffice (never the key itself).
const (
	KeySourceDB   = "db"
	KeySourceEnv  = "env"
	KeySourceNone = "none"
)

// FunctionSettings is what backoffice reads for one AI function. Empty
// fields mean "use the env default", shown next to them. The API key is
// write-only: only whether one is stored and where the effective one comes
// from are returned.
type FunctionSettings struct {
	Function               string  `json:"function"`
	BaseURL                string  `json:"base_url"`
	DefaultBaseURL         string  `json:"default_base_url"`
	Model                  string  `json:"model"`
	DefaultModel           string  `json:"default_model"`
	ReasoningEffort        string  `json:"reasoning_effort"`
	DefaultReasoningEffort string  `json:"default_reasoning_effort"`
	HasAPIKey              bool    `json:"has_api_key"`     // a key is stored in backoffice
	EnvHasAPIKey           bool    `json:"env_has_api_key"` // AI_API_KEY is set
	APIKeySource           string  `json:"api_key_source"`  // KeySource*: the key calls will use
	EncryptionAvailable    bool    `json:"encryption_available"`
	PriceInput             float64 `json:"price_input"`
	PriceOutput            float64 `json:"price_output"`
}

// SettingsUpdate is what backoffice saves for one function. APIKey empty
// keeps the stored key; ClearAPIKey removes it (back to the env key).
type SettingsUpdate struct {
	Function        string  `json:"function"`
	BaseURL         string  `json:"base_url"`
	Model           string  `json:"model"`
	ReasoningEffort string  `json:"reasoning_effort"`
	APIKey          string  `json:"api_key"`
	ClearAPIKey     bool    `json:"clear_api_key"`
	PriceInput      float64 `json:"price_input"`
	PriceOutput     float64 `json:"price_output"`
}

// Provider is the resolved provider of one function: what a call uses.
type Provider struct {
	BaseURL         string
	APIKey          string `json:"-"`
	Model           string
	ReasoningEffort string
	KeySource       string
}

// Configured reports whether calls can be made (a key and a base URL).
func (p Provider) Configured() bool { return p.APIKey != "" && p.BaseURL != "" }

// Host is the provider's host, logged with each use (never the key).
func (p Provider) Host() string {
	u, err := url.Parse(p.BaseURL)
	if err != nil {
		return ""
	}
	return u.Host
}

// DefaultModel is the configured model (AI_DEFAULT_SUMMARY_MODEL) used when
// backoffice set none. Only the summary exists today, so every function
// shares it until scribe and bot get their own variables.
func DefaultModel(string) string {
	return config.Get().AIDefaultSummaryModel
}

// ModelFor returns the model of fn: the backoffice setting, else DefaultModel.
func ModelFor(db *gorm.DB, fn string) string {
	if v := strings.TrimSpace(readSetting(db, settingModelPrefix+fn)); v != "" {
		return v
	}
	return DefaultModel(fn)
}

// ResolveProvider returns the provider of fn: each backoffice field, else the
// env. The env key is used only with the env base URL: it is never sent to a
// host backoffice set without also setting that host's key. A stored key
// that can't be opened (AI_ENCRYPTION_KEY missing or changed) is an error.
func ResolveProvider(db *gorm.DB, fn string) (Provider, error) {
	cfg := config.Get()
	envBase := normalizeBaseURL(cfg.AIBaseURL)
	p := Provider{
		BaseURL:         envBase,
		Model:           ModelFor(db, fn),
		ReasoningEffort: strings.TrimSpace(cfg.AIReasoningEffort),
		KeySource:       KeySourceNone,
	}
	if v := normalizeBaseURL(readSetting(db, settingBaseURLPrefix+fn)); v != "" {
		p.BaseURL = v
	}
	if v := strings.TrimSpace(readSetting(db, settingEffortPrefix+fn)); v != "" {
		p.ReasoningEffort = v
	}
	if sealed := readSetting(db, settingAPIKeyPrefix+fn); sealed != "" {
		box, err := secretbox.FromEnvVar(EncryptionKeyEnv)
		if err != nil {
			return p, fmt.Errorf("ai settings: stored key of %s unusable: %w", fn, err)
		}
		key, err := box.Open(sealed)
		if err != nil {
			return p, fmt.Errorf("ai settings: stored key of %s unreadable: %w", fn, err)
		}
		p.APIKey, p.KeySource = key, KeySourceDB
		return p, nil
	}
	if envKey := strings.TrimSpace(cfg.AIAPIKey); envKey != "" && p.BaseURL == envBase {
		p.APIKey, p.KeySource = envKey, KeySourceEnv
	}
	return p, nil
}

// Settings returns the configuration of every function.
func Settings(db *gorm.DB) []FunctionSettings {
	out := make([]FunctionSettings, 0, len(ai_models.Functions))
	for _, fn := range ai_models.Functions {
		out = append(out, settingsOf(db, fn))
	}
	return out
}

func settingsOf(db *gorm.DB, fn string) FunctionSettings {
	cfg := config.Get()
	s := FunctionSettings{
		Function:               fn,
		BaseURL:                strings.TrimSpace(readSetting(db, settingBaseURLPrefix+fn)),
		DefaultBaseURL:         normalizeBaseURL(cfg.AIBaseURL),
		Model:                  strings.TrimSpace(readSetting(db, settingModelPrefix+fn)),
		DefaultModel:           DefaultModel(fn),
		ReasoningEffort:        strings.TrimSpace(readSetting(db, settingEffortPrefix+fn)),
		DefaultReasoningEffort: strings.TrimSpace(cfg.AIReasoningEffort),
		HasAPIKey:              readSetting(db, settingAPIKeyPrefix+fn) != "",
		EnvHasAPIKey:           strings.TrimSpace(cfg.AIAPIKey) != "",
		APIKeySource:           KeySourceNone,
		EncryptionAvailable:    encryptionAvailable(),
		PriceInput:             readFloat(db, settingPriceInputPrefix+fn),
		PriceOutput:            readFloat(db, settingPriceOutputPrefix+fn),
	}
	if p, err := ResolveProvider(db, fn); err == nil {
		s.APIKeySource = p.KeySource
	}
	return s
}

func encryptionAvailable() bool {
	_, err := secretbox.FromEnvVar(EncryptionKeyEnv)
	return err == nil
}

var effortRe = regexp.MustCompile(`^[a-z]{1,20}$`)

// ValidateUpdate checks and normalizes u (trimmed fields, base URL without
// trailing slash).
func ValidateUpdate(u *SettingsUpdate) error {
	if !ai_models.ValidFunction(u.Function) {
		return ErrUnknownFunction
	}
	if u.PriceInput < 0 || u.PriceOutput < 0 {
		return ErrInvalidSettings
	}
	u.Model = strings.TrimSpace(u.Model)
	u.ReasoningEffort = strings.TrimSpace(u.ReasoningEffort)
	u.APIKey = strings.TrimSpace(u.APIKey)
	if len(u.Model) > 200 || len(u.APIKey) > 4096 {
		return ErrInvalidSettings
	}
	if u.ReasoningEffort != "" && !effortRe.MatchString(u.ReasoningEffort) {
		return ErrInvalidSettings
	}
	base := strings.TrimSpace(u.BaseURL)
	if base != "" {
		parsed, err := url.Parse(base)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
			parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
			return ErrInvalidBaseURL
		}
		base = normalizeBaseURL(base)
	}
	u.BaseURL = base
	if u.APIKey != "" && !encryptionAvailable() {
		return ErrEncryptionUnavailable
	}
	return nil
}

// SaveSettings validates and stores the configuration of one function. The
// key is sealed with AI_ENCRYPTION_KEY before it reaches the database.
func SaveSettings(db *gorm.DB, u SettingsUpdate) error {
	if err := ValidateUpdate(&u); err != nil {
		return err
	}
	values := map[string]string{
		settingBaseURLPrefix + u.Function:     u.BaseURL,
		settingModelPrefix + u.Function:       u.Model,
		settingEffortPrefix + u.Function:      u.ReasoningEffort,
		settingPriceInputPrefix + u.Function:  strconv.FormatFloat(u.PriceInput, 'f', -1, 64),
		settingPriceOutputPrefix + u.Function: strconv.FormatFloat(u.PriceOutput, 'f', -1, 64),
	}
	switch {
	case u.APIKey != "":
		box, err := secretbox.FromEnvVar(EncryptionKeyEnv)
		if err != nil {
			return ErrEncryptionUnavailable
		}
		sealed, err := box.Seal(u.APIKey)
		if err != nil {
			return fmt.Errorf("ai settings: seal key: %w", err)
		}
		values[settingAPIKeyPrefix+u.Function] = sealed
	case u.ClearAPIKey:
		values[settingAPIKeyPrefix+u.Function] = ""
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			if err := writeSetting(tx, key, value); err != nil {
				return err
			}
		}
		return nil
	})
}

// Cost is the USD cost of a use of fn with the stored prices.
func Cost(db *gorm.DB, fn string, promptTokens, completionTokens int) float64 {
	in := readFloat(db, settingPriceInputPrefix+fn)
	out := readFloat(db, settingPriceOutputPrefix+fn)
	return (float64(promptTokens)*in + float64(completionTokens)*out) / 1_000_000
}

func normalizeBaseURL(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), "/")
}

func readSetting(db *gorm.DB, key string) string {
	var s settings_models.SystemSetting
	if err := db.Where("key = ?", key).Limit(1).Find(&s).Error; err != nil {
		return ""
	}
	return s.Value
}

func readFloat(db *gorm.DB, key string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(readSetting(db, key)), 64)
	if err != nil || f < 0 {
		return 0
	}
	return f
}

// writeSetting upserts key. An empty value is stored as "" (Value is NOT
// NULL), which the readers treat as "use the default".
func writeSetting(db *gorm.DB, key, value string) error {
	var s settings_models.SystemSetting
	if err := db.Where("key = ?", key).Limit(1).Find(&s).Error; err != nil {
		return err
	}
	if s.ID == 0 {
		return db.Create(&settings_models.SystemSetting{Key: key, Value: value}).Error
	}
	return db.Model(&settings_models.SystemSetting{}).Where("id = ?", s.ID).Update("value", value).Error
}
