package user_handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"pengi-med-saas/core/config"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	auth_middleware "pengi-med-saas/features/users/middleware"
	user_models "pengi-med-saas/features/users/models"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Passkeys (WebAuthn) as a way to sign in without a password.
//
// A passkey is a discoverable credential with user verification (biometric
// or device PIN), so it is both factors at once: signing in with one skips
// the TOTP step. Because of that, registering one asks for the current
// password and, with 2FA on, a second-factor code: a stolen session alone
// can't plant a backdoor that bypasses 2FA.
//
// The relying party is derived from APP_URL (RP ID = its host, origin = its
// scheme + host): the web app is served from one host for every tenant. The
// RP ID is baked into every credential; changing the host of APP_URL
// invalidates all passkeys.

const (
	// passkeyChallengeTTL is how long a ceremony may take (the browser prompt).
	passkeyChallengeTTL = 5 * time.Minute
	maxPasskeysPerUser  = 10
	maxPasskeyNameLen   = 100
)

// newRelyingParty builds the WebAuthn relying party from APP_URL.
func newRelyingParty() (*webauthn.WebAuthn, error) {
	u, err := url.Parse(strings.TrimSpace(config.Get().AppURL))
	if err != nil || u.Scheme == "" || u.Hostname() == "" {
		return nil, errors.New("APP_URL is not a valid URL: cannot derive the passkey relying party")
	}
	return webauthn.New(&webauthn.Config{
		RPID:          u.Hostname(),
		RPDisplayName: "Gentoo",
		RPOrigins:     []string{u.Scheme + "://" + u.Host},
	})
}

// passkeyUser adapts a user and its stored passkeys to webauthn.User.
type passkeyUser struct {
	user  user_models.User
	creds []webauthn.Credential
}

// WebAuthnID is the opaque user handle the authenticator stores with the
// credential (the user ID, 8 bytes): it lets a discoverable login find the
// user without a username.
func (u passkeyUser) WebAuthnID() []byte {
	id := make([]byte, 8)
	binary.BigEndian.PutUint64(id, uint64(u.user.ID))
	return id
}

func (u passkeyUser) WebAuthnName() string                       { return u.user.UserName }
func (u passkeyUser) WebAuthnDisplayName() string                { return u.user.UserName }
func (u passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }

func userIDFromHandle(handle []byte) (uint, bool) {
	if len(handle) != 8 {
		return 0, false
	}
	id := binary.BigEndian.Uint64(handle)
	return uint(id), id != 0
}

// loadPasskeyUser loads the user and decodes its passkeys.
func (h *UserHandler) loadPasskeyUser(user user_models.User) (passkeyUser, error) {
	var rows []user_models.Passkey
	if err := h.db.Where("user_id = ?", user.ID).Order("id").Find(&rows).Error; err != nil {
		return passkeyUser{}, err
	}
	creds := make([]webauthn.Credential, 0, len(rows))
	for _, row := range rows {
		var cred webauthn.Credential
		if err := json.Unmarshal(row.Credential, &cred); err != nil {
			return passkeyUser{}, err
		}
		creds = append(creds, cred)
	}
	return passkeyUser{user: user, creds: creds}, nil
}

// saveChallenge stores session and returns the id the client sends back to
// finish the ceremony. Expired challenges are swept here.
func (h *UserHandler) saveChallenge(kind string, userID uint, session *webauthn.SessionData) (string, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	id := base64.RawURLEncoding.EncodeToString(raw)
	now := h.now()
	if err := h.db.Where("expires_at < ?", now).Delete(&user_models.WebAuthnChallenge{}).Error; err != nil {
		return "", err
	}
	row := user_models.WebAuthnChallenge{ID: id, Kind: kind, UserID: userID, Data: data, ExpiresAt: now.Add(passkeyChallengeTTL)}
	if err := h.db.Create(&row).Error; err != nil {
		return "", err
	}
	return id, nil
}

// takeChallenge consumes a challenge: it works once (the delete is the
// check, so concurrent finishes can't both win), for the same kind and user,
// before it expires. ok is false for anything else.
func (h *UserHandler) takeChallenge(id, kind string, userID uint) (session *webauthn.SessionData, ok bool, err error) {
	var row user_models.WebAuthnChallenge
	if err := h.db.First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	res := h.db.Delete(&user_models.WebAuthnChallenge{}, "id = ?", id)
	if res.Error != nil {
		return nil, false, res.Error
	}
	if res.RowsAffected != 1 || row.Kind != kind || row.UserID != userID || !h.now().Before(row.ExpiresAt) {
		return nil, false, nil
	}
	var s webauthn.SessionData
	if err := json.Unmarshal(row.Data, &s); err != nil {
		return nil, false, err
	}
	return &s, true, nil
}

func passkeyInvalidResponse(status int) envelope.Response {
	return envelope.ErrorResponse(status, "auth.passkey.invalid", core_errors.ErrAuthPasskeyInvalid)
}

func internalResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
}

// ListPasskeys: GET /users/me/passkeys.
func (h *UserHandler) ListPasskeys(c *gin.Context) envelope.Response {
	user, errResp := h.currentUser(c)
	if errResp != nil {
		return *errResp
	}
	rows := []user_models.Passkey{}
	if err := h.db.Where("user_id = ?", user.ID).Order("id").Find(&rows).Error; err != nil {
		h.logger.Error("Passkeys: failed to list", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	return envelope.SuccessResponse(rows, "auth.passkey.list.success")
}

// BeginPasskeyRegistration: POST /users/me/passkeys/register/begin. Needs
// the current password and, with 2FA on, a second-factor code.
func (h *UserHandler) BeginPasskeyRegistration(c *gin.Context) envelope.Response {
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		Code            string `json:"code" binding:"max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}
	user, errResp := h.currentUser(c)
	if errResp != nil {
		return *errResp
	}
	if errResp := h.checkCurrentPassword(user, req.CurrentPassword); errResp != nil {
		return *errResp
	}
	if user.TwoFactorEnabled {
		if errResp := h.checkCode(user, func() (bool, error) {
			return acceptSecondFactor(h.db, user, req.Code, h.now())
		}); errResp != nil {
			return *errResp
		}
	}

	pu, err := h.loadPasskeyUser(user)
	if err != nil {
		h.logger.Error("Passkeys: failed to load credentials", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	if len(pu.creds) >= maxPasskeysPerUser {
		return envelope.ErrorResponse(http.StatusConflict, "auth.passkey.limit", core_errors.ErrAuthPasskeyLimit)
	}
	wa, err := newRelyingParty()
	if err != nil {
		h.logger.Error("Passkeys: relying party", zap.Error(err))
		return internalResponse()
	}

	creation, session, err := wa.BeginRegistration(pu,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			RequireResidentKey: protocol.ResidentKeyRequired(),
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			UserVerification:   protocol.VerificationRequired,
		}),
		webauthn.WithExclusions(webauthn.Credentials(pu.creds).CredentialDescriptors()),
	)
	if err != nil {
		h.logger.Error("Passkeys: failed to begin registration", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	id, err := h.saveChallenge(user_models.ChallengeRegister, user.ID, session)
	if err != nil {
		h.logger.Error("Passkeys: failed to store challenge", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	return envelope.SuccessResponse(gin.H{
		"challenge_id": id,
		"options":      creation.Response,
	}, "auth.passkey.register.begin.success")
}

// FinishPasskeyRegistration: POST /users/me/passkeys/register/finish.
func (h *UserHandler) FinishPasskeyRegistration(c *gin.Context) envelope.Response {
	var req struct {
		ChallengeID string          `json:"challenge_id" binding:"required,max=64"`
		Name        string          `json:"name" binding:"max=200"`
		Credential  json.RawMessage `json:"credential" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}
	user, errResp := h.currentUser(c)
	if errResp != nil {
		return *errResp
	}
	session, ok, err := h.takeChallenge(req.ChallengeID, user_models.ChallengeRegister, user.ID)
	if err != nil {
		h.logger.Error("Passkeys: failed to read challenge", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	if !ok {
		return passkeyInvalidResponse(http.StatusUnprocessableEntity)
	}
	pu, err := h.loadPasskeyUser(user)
	if err != nil {
		h.logger.Error("Passkeys: failed to load credentials", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	if len(pu.creds) >= maxPasskeysPerUser {
		return envelope.ErrorResponse(http.StatusConflict, "auth.passkey.limit", core_errors.ErrAuthPasskeyLimit)
	}
	wa, err := newRelyingParty()
	if err != nil {
		h.logger.Error("Passkeys: relying party", zap.Error(err))
		return internalResponse()
	}

	parsed, err := protocol.ParseCredentialCreationResponseBytes(req.Credential)
	if err != nil {
		h.logger.Warn("Passkeys: malformed registration response", zap.Uint("user_id", user.ID), zap.Error(err))
		return passkeyInvalidResponse(http.StatusUnprocessableEntity)
	}
	cred, err := wa.CreateCredential(pu, *session, parsed)
	if err != nil {
		h.logger.Warn("Passkeys: registration refused", zap.Uint("user_id", user.ID), zap.Error(err))
		return passkeyInvalidResponse(http.StatusUnprocessableEntity)
	}
	stored, err := json.Marshal(cred)
	if err != nil {
		h.logger.Error("Passkeys: failed to encode credential", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Passkey"
	}
	if r := []rune(name); len(r) > maxPasskeyNameLen {
		name = string(r[:maxPasskeyNameLen])
	}
	row := user_models.Passkey{UserID: user.ID, Name: name, CredentialID: cred.ID, Credential: stored}
	if err := h.db.Create(&row).Error; err != nil {
		// A duplicate credential ID (already registered) also lands here.
		h.logger.Warn("Passkeys: failed to store credential", zap.Uint("user_id", user.ID), zap.Error(err))
		return passkeyInvalidResponse(http.StatusUnprocessableEntity)
	}
	h.alertSecurity(user, eventPasskeyAdded, c)
	h.logger.Info("Passkey registered", zap.Uint("user_id", user.ID), zap.Uint("passkey_id", row.ID))
	return envelope.SuccessResponse(row, "auth.passkey.register.finish.success")
}

// DeletePasskey: DELETE /users/me/passkeys/:id.
func (h *UserHandler) DeletePasskey(c *gin.Context) envelope.Response {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}
	userID := uint(c.GetInt64("user_id"))
	res := h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&user_models.Passkey{})
	if res.Error != nil {
		h.logger.Error("Passkeys: failed to delete", zap.Uint("user_id", userID), zap.Error(res.Error))
		return internalResponse()
	}
	if res.RowsAffected == 0 {
		return envelope.ErrorResponse(http.StatusNotFound, "auth.passkey.not_found", core_errors.ErrAuthPasskeyNotFound)
	}
	var owner user_models.User
	if err := h.db.First(&owner, userID).Error; err == nil {
		h.alertSecurity(owner, eventPasskeyDeleted, c)
	}
	h.logger.Info("Passkey deleted", zap.Uint("user_id", userID), zap.Uint64("passkey_id", id))
	return envelope.SuccessResponse(nil, "auth.passkey.delete.success")
}

// BeginPasskeyLogin: POST /auth/passkey/begin. Public: a discoverable
// login, the authenticator picks the account, so no username is sent.
func (h *UserHandler) BeginPasskeyLogin(c *gin.Context) envelope.Response {
	wa, err := newRelyingParty()
	if err != nil {
		h.logger.Error("Passkeys: relying party", zap.Error(err))
		return internalResponse()
	}
	assertion, session, err := wa.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		h.logger.Error("Passkeys: failed to begin login", zap.Error(err))
		return internalResponse()
	}
	id, err := h.saveChallenge(user_models.ChallengeLogin, 0, session)
	if err != nil {
		h.logger.Error("Passkeys: failed to store challenge", zap.Error(err))
		return internalResponse()
	}
	return envelope.SuccessResponse(gin.H{
		"challenge_id": id,
		"options":      assertion.Response,
	}, "auth.passkey.login.begin.success")
}

// FinishPasskeyLogin: POST /auth/passkey/login. A valid assertion opens the
// same session as a password login that passed 2FA. Every failure answers
// the same 401, so it says nothing about which accounts exist.
func (h *UserHandler) FinishPasskeyLogin(c *gin.Context) envelope.Response {
	var req struct {
		ChallengeID string          `json:"challenge_id" binding:"required,max=64"`
		Credential  json.RawMessage `json:"credential" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}
	session, ok, err := h.takeChallenge(req.ChallengeID, user_models.ChallengeLogin, 0)
	if err != nil {
		h.logger.Error("Passkeys: failed to read challenge", zap.Error(err))
		return internalResponse()
	}
	if !ok {
		return passkeyInvalidResponse(http.StatusUnauthorized)
	}
	wa, err := newRelyingParty()
	if err != nil {
		h.logger.Error("Passkeys: relying party", zap.Error(err))
		return internalResponse()
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(req.Credential)
	if err != nil {
		h.logger.Warn("Passkeys: malformed login response", zap.Error(err))
		return passkeyInvalidResponse(http.StatusUnauthorized)
	}

	lookup := func(_, userHandle []byte) (webauthn.User, error) {
		uid, ok := userIDFromHandle(userHandle)
		if !ok {
			return nil, errors.New("invalid user handle")
		}
		var user user_models.User
		if err := h.db.First(&user, uid).Error; err != nil {
			return nil, err
		}
		return h.loadPasskeyUser(user)
	}
	waUser, cred, err := wa.ValidatePasskeyLogin(lookup, *session, parsed)
	if err != nil {
		h.logger.Warn("Passkey login refused", zap.Error(err))
		return passkeyInvalidResponse(http.StatusUnauthorized)
	}
	user := waUser.(passkeyUser).user
	// A counter that went backwards means a cloned authenticator.
	if cred.Authenticator.CloneWarning {
		h.logger.Warn("Passkey login refused: clone warning", zap.Uint("user_id", user.ID))
		return passkeyInvalidResponse(http.StatusUnauthorized)
	}
	if !user.EmailVerified {
		return envelope.ErrorResponse(http.StatusForbidden, "auth.login.email_not_verified", core_errors.ErrAuthEmailNotVerified)
	}
	if user.LockedUntil != nil && user.LockedUntil.After(h.now()) {
		h.logger.Warn("Passkey login refused: account locked", zap.Uint("user_id", user.ID))
		return accountLockedResponse()
	}

	// Keep the sign counter and flags the library just updated.
	stored, err := json.Marshal(cred)
	if err != nil {
		h.logger.Error("Passkeys: failed to encode credential", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	if err := h.db.Model(&user_models.Passkey{}).
		Where("user_id = ? AND credential_id = ?", user.ID, cred.ID).
		Updates(map[string]any{"credential": stored, "last_used_at": h.now()}).Error; err != nil {
		h.logger.Error("Passkeys: failed to update credential", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}

	data, errResp := h.issueSession(c, user)
	if errResp != nil {
		return *errResp
	}
	h.logger.Info("User logged in successfully", zap.String("username", user.UserName), zap.Bool("passkey", true))
	return envelope.SuccessResponse(data, "login.successful")
}

// ResetPasskeys removes every passkey of a user (lost device) together with
// their pending ceremonies and ends all their sessions. Used by the backoffice.
func ResetPasskeys(db *gorm.DB, userID uint) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&user_models.Passkey{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&user_models.WebAuthnChallenge{}).Error; err != nil {
			return err
		}
		return RevokeUserSessions(tx, userID)
	})
	if err == nil {
		// Again after commit, for a concurrent rotation (see EnableTwoFactor).
		err = RevokeUserSessions(db, userID)
	}
	auth_middleware.ForgetUser(userID)
	return err
}
