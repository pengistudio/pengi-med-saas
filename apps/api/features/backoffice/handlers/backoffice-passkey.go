package backoffice_handlers

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

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/config"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	backoffice_models "pengi-med-saas/features/backoffice/models"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Backoffice passkeys (WebAuthn), mirroring features/users/handlers/passkey.go.
//
// A passkey is discoverable and user verified (biometric or PIN), so it is
// both factors at once: it counts as the admin's second factor, and signing
// in with one skips the TOTP step. The relying party is BACKOFFICE_URL (RP ID
// = its host, origin = scheme + host), not APP_URL: the backoffice is served
// from its own host. Changing that host voids every admin passkey.

const (
	adminPasskeyChallengeTTL = 5 * time.Minute
	maxAdminPasskeys         = 10
	maxAdminPasskeyNameLen   = 100
	// adminFreshSessionWindow: an admin without TOTP (passkey only) adds
	// another passkey with the password plus a session that signed in less
	// than this ago (its login just proved a passkey).
	adminFreshSessionWindow = 5 * time.Minute
	// adminHandlePrefix sets admin user handles apart from clinic users'
	// (8 bytes, the user ID): in dev both relying parties are "localhost".
	adminHandlePrefix = 0xB0
)

var errPasskeysUnavailable = errors.New("BACKOFFICE_URL is not set: backoffice passkeys are unavailable")

func backofficeRelyingParty() (*webauthn.WebAuthn, error) {
	raw := strings.TrimSpace(config.Get().BackofficeURL)
	if raw == "" {
		return nil, errPasskeysUnavailable
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Hostname() == "" {
		return nil, errors.New("BACKOFFICE_URL is not a valid URL: cannot derive the passkey relying party")
	}
	return webauthn.New(&webauthn.Config{
		RPID:          u.Hostname(),
		RPDisplayName: "Gentoo Backoffice",
		RPOrigins:     []string{u.Scheme + "://" + u.Host},
	})
}

// passkeysAvailable reports whether the backoffice relying party is set up.
func passkeysAvailable() bool {
	_, err := backofficeRelyingParty()
	return err == nil
}

// relyingParty returns the relying party or the response to give without one.
func (h *BackofficeUserHandler) relyingParty() (*webauthn.WebAuthn, *envelope.Response) {
	wa, err := backofficeRelyingParty()
	if err == nil {
		return wa, nil
	}
	var resp envelope.Response
	if errors.Is(err, errPasskeysUnavailable) {
		resp = envelope.ErrorResponse(http.StatusServiceUnavailable, "backoffice.passkey.unavailable", core_errors.ErrBackofficePasskeysUnavailable)
	} else {
		h.logger.Error("Backoffice passkeys: relying party", zap.Error(err))
		resp = internalResponse()
	}
	return nil, &resp
}

type adminPasskeyUser struct {
	user  backoffice_models.BackofficeUser
	creds []webauthn.Credential
}

func (u adminPasskeyUser) WebAuthnID() []byte {
	id := make([]byte, 9)
	id[0] = adminHandlePrefix
	binary.BigEndian.PutUint64(id[1:], uint64(u.user.ID))
	return id
}

func (u adminPasskeyUser) WebAuthnName() string                       { return u.user.UserName }
func (u adminPasskeyUser) WebAuthnDisplayName() string                { return u.user.UserName }
func (u adminPasskeyUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }

func adminIDFromHandle(handle []byte) (uint, bool) {
	if len(handle) != 9 || handle[0] != adminHandlePrefix {
		return 0, false
	}
	id := binary.BigEndian.Uint64(handle[1:])
	return uint(id), id != 0
}

func (h *BackofficeUserHandler) loadAdminPasskeyUser(user backoffice_models.BackofficeUser) (adminPasskeyUser, error) {
	var rows []backoffice_models.BackofficePasskey
	if err := h.db.Where("backoffice_user_id = ?", user.ID).Order("id").Find(&rows).Error; err != nil {
		return adminPasskeyUser{}, err
	}
	creds := make([]webauthn.Credential, 0, len(rows))
	for _, row := range rows {
		var cred webauthn.Credential
		if err := json.Unmarshal(row.Credential, &cred); err != nil {
			return adminPasskeyUser{}, err
		}
		creds = append(creds, cred)
	}
	return adminPasskeyUser{user: user, creds: creds}, nil
}

func (h *BackofficeUserHandler) saveChallenge(kind string, userID uint, session *webauthn.SessionData) (string, error) {
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
	if err := h.db.Where("expires_at < ?", now).Delete(&backoffice_models.BackofficeWebAuthnChallenge{}).Error; err != nil {
		return "", err
	}
	row := backoffice_models.BackofficeWebAuthnChallenge{ID: id, Kind: kind, BackofficeUserID: userID, Data: data, ExpiresAt: now.Add(adminPasskeyChallengeTTL)}
	if err := h.db.Create(&row).Error; err != nil {
		return "", err
	}
	return id, nil
}

// takeChallenge consumes a challenge: once, same kind and admin, unexpired.
func (h *BackofficeUserHandler) takeChallenge(id, kind string, userID uint) (*webauthn.SessionData, bool, error) {
	var row backoffice_models.BackofficeWebAuthnChallenge
	if err := h.db.First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	res := h.db.Delete(&backoffice_models.BackofficeWebAuthnChallenge{}, "id = ?", id)
	if res.Error != nil {
		return nil, false, res.Error
	}
	if res.RowsAffected != 1 || row.Kind != kind || row.BackofficeUserID != userID || !h.now().Before(row.ExpiresAt) {
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

func passkeyLimitResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "auth.passkey.limit", core_errors.ErrAuthPasskeyLimit)
}

// beginRegistration starts a passkey registration ceremony for user.
func (h *BackofficeUserHandler) beginRegistration(user backoffice_models.BackofficeUser) envelope.Response {
	wa, errResp := h.relyingParty()
	if errResp != nil {
		return *errResp
	}
	pu, err := h.loadAdminPasskeyUser(user)
	if err != nil {
		h.logger.Error("Backoffice passkeys: failed to load credentials", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	if len(pu.creds) >= maxAdminPasskeys {
		return passkeyLimitResponse()
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
		h.logger.Error("Backoffice passkeys: failed to begin registration", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	id, err := h.saveChallenge(backoffice_models.BackofficeChallengeRegister, user.ID, session)
	if err != nil {
		h.logger.Error("Backoffice passkeys: failed to store challenge", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	return envelope.SuccessResponse(gin.H{"challenge_id": id, "options": creation.Response}, "auth.passkey.register.begin.success")
}

type finishRegistrationRequest struct {
	ChallengeID string          `json:"challenge_id" binding:"required,max=64"`
	Name        string          `json:"name" binding:"max=200"`
	Credential  json.RawMessage `json:"credential" binding:"required"`
}

// finishRegistration verifies the authenticator's answer and stores the
// passkey of user.
func (h *BackofficeUserHandler) finishRegistration(user backoffice_models.BackofficeUser, req finishRegistrationRequest) (*backoffice_models.BackofficePasskey, *envelope.Response) {
	fail := func(resp envelope.Response) (*backoffice_models.BackofficePasskey, *envelope.Response) {
		return nil, &resp
	}
	session, ok, err := h.takeChallenge(req.ChallengeID, backoffice_models.BackofficeChallengeRegister, user.ID)
	if err != nil {
		h.logger.Error("Backoffice passkeys: failed to read challenge", zap.Uint("user_id", user.ID), zap.Error(err))
		return fail(internalResponse())
	}
	if !ok {
		return fail(passkeyInvalidResponse(http.StatusUnprocessableEntity))
	}
	wa, errResp := h.relyingParty()
	if errResp != nil {
		return nil, errResp
	}
	pu, err := h.loadAdminPasskeyUser(user)
	if err != nil {
		h.logger.Error("Backoffice passkeys: failed to load credentials", zap.Uint("user_id", user.ID), zap.Error(err))
		return fail(internalResponse())
	}
	if len(pu.creds) >= maxAdminPasskeys {
		return fail(passkeyLimitResponse())
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(req.Credential)
	if err != nil {
		h.logger.Warn("Backoffice passkeys: malformed registration response", zap.Uint("user_id", user.ID), zap.Error(err))
		return fail(passkeyInvalidResponse(http.StatusUnprocessableEntity))
	}
	cred, err := wa.CreateCredential(pu, *session, parsed)
	if err != nil {
		h.logger.Warn("Backoffice passkeys: registration refused", zap.Uint("user_id", user.ID), zap.Error(err))
		return fail(passkeyInvalidResponse(http.StatusUnprocessableEntity))
	}
	stored, err := json.Marshal(cred)
	if err != nil {
		h.logger.Error("Backoffice passkeys: failed to encode credential", zap.Uint("user_id", user.ID), zap.Error(err))
		return fail(internalResponse())
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Passkey"
	}
	if r := []rune(name); len(r) > maxAdminPasskeyNameLen {
		name = string(r[:maxAdminPasskeyNameLen])
	}
	row := backoffice_models.BackofficePasskey{BackofficeUserID: user.ID, Name: name, CredentialID: cred.ID, Credential: stored}
	if err := h.db.Create(&row).Error; err != nil {
		h.logger.Warn("Backoffice passkeys: failed to store credential", zap.Uint("user_id", user.ID), zap.Error(err))
		return fail(passkeyInvalidResponse(http.StatusUnprocessableEntity))
	}
	h.logger.Info("Backoffice passkey registered", zap.Uint("user_id", user.ID), zap.Uint("passkey_id", row.ID))
	return &row, nil
}

// EnrollPasskeyBegin: POST /backoffice/auth/enroll/passkey/begin.
func (h *BackofficeUserHandler) EnrollPasskeyBegin(c *gin.Context) envelope.Response {
	var req enrollmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequestResponse()
	}
	user, errResp := h.enrollee(req.EnrollmentToken)
	if errResp != nil {
		return *errResp
	}
	return h.beginRegistration(user)
}

// EnrollPasskeyFinish: POST /backoffice/auth/enroll/passkey/finish. A valid
// registration enrolls the passkey and opens the session.
func (h *BackofficeUserHandler) EnrollPasskeyFinish(c *gin.Context) envelope.Response {
	var req struct {
		EnrollmentToken string `json:"enrollment_token" binding:"required"`
		finishRegistrationRequest
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequestResponse()
	}
	user, errResp := h.enrollee(req.EnrollmentToken)
	if errResp != nil {
		return *errResp
	}
	row, errResp := h.finishRegistration(user, req.finishRegistrationRequest)
	if errResp != nil {
		return *errResp
	}
	data, errResp := h.issueSession(c, user)
	if errResp != nil {
		return *errResp
	}
	data["passkey"] = row
	h.logger.Info("Backoffice user enrolled a passkey and logged in", zap.String("username", user.UserName))
	return envelope.SuccessResponse(data, "backoffice.auth.enroll.success")
}

// sessionIsFresh reports whether the request's access token comes from a
// login less than adminFreshSessionWindow ago (auth_time; refreshed tokens
// carry none and are never fresh).
func (h *BackofficeUserHandler) sessionIsFresh(c *gin.Context) bool {
	token := c.GetString("auth_token")
	if token == "" {
		return false
	}
	claims, err := auth.ParseBackofficeAccessToken(token)
	if err != nil || claims.AuthTime.IsZero() {
		return false
	}
	age := h.now().Sub(claims.AuthTime)
	return age >= 0 && age < adminFreshSessionWindow
}

// ListPasskeys: GET /backoffice/me/passkeys.
func (h *BackofficeUserHandler) ListPasskeys(c *gin.Context) envelope.Response {
	userID := uint(c.GetInt64("user_id"))
	rows := []backoffice_models.BackofficePasskey{}
	if err := h.db.Where("backoffice_user_id = ?", userID).Order("id").Find(&rows).Error; err != nil {
		h.logger.Error("Backoffice passkeys: failed to list", zap.Uint("user_id", userID), zap.Error(err))
		return internalResponse()
	}
	return envelope.SuccessResponse(rows, "auth.passkey.list.success")
}

// BeginPasskeyRegistration: POST /backoffice/me/passkeys/register/begin.
// Needs the current password plus the current second factor: a TOTP or
// recovery code when TOTP is on; for a passkey-only admin, a session that
// signed in (with a passkey) less than 5 minutes ago.
func (h *BackofficeUserHandler) BeginPasskeyRegistration(c *gin.Context) envelope.Response {
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		Code            string `json:"code" binding:"max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequestResponse()
	}
	user, errResp := h.currentAdmin(c)
	if errResp != nil {
		return *errResp
	}
	if errResp := h.checkCurrentPassword(user, req.CurrentPassword); errResp != nil {
		return *errResp
	}
	if user.TwoFactorEnabled {
		if errResp := h.checkCode(user, wrongCodeResponse(), func() (bool, error) {
			return acceptAdminSecondFactor(h.db, user, req.Code, h.now())
		}); errResp != nil {
			return *errResp
		}
	} else if !h.sessionIsFresh(c) {
		return envelope.ErrorResponse(http.StatusUnprocessableEntity, "backoffice.passkey.reauth_required", core_errors.ErrAuthPasskeyReauthRequired)
	}
	return h.beginRegistration(user)
}

// FinishPasskeyRegistration: POST /backoffice/me/passkeys/register/finish.
func (h *BackofficeUserHandler) FinishPasskeyRegistration(c *gin.Context) envelope.Response {
	var req finishRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequestResponse()
	}
	user, errResp := h.currentAdmin(c)
	if errResp != nil {
		return *errResp
	}
	row, errResp := h.finishRegistration(user, req)
	if errResp != nil {
		return *errResp
	}
	return envelope.SuccessResponse(row, "auth.passkey.register.finish.success")
}

// DeletePasskey: DELETE /backoffice/me/passkeys/:id. Not the last factor: an
// admin without TOTP keeps at least one passkey.
func (h *BackofficeUserHandler) DeletePasskey(c *gin.Context) envelope.Response {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return invalidRequestResponse()
	}
	userID := uint(c.GetInt64("user_id"))
	var row backoffice_models.BackofficePasskey
	if err := h.db.Where("id = ? AND backoffice_user_id = ?", id, userID).First(&row).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "auth.passkey.not_found", core_errors.ErrAuthPasskeyNotFound)
	}
	res := h.db.Where("id = ? AND backoffice_user_id = ?", id, userID).
		Where("(EXISTS (SELECT 1 FROM backoffice_users u WHERE u.id = ? AND u.two_factor_enabled = ?) OR "+
			"(SELECT COUNT(*) FROM backoffice_user_passkeys p WHERE p.backoffice_user_id = ?) > 1)", userID, true, userID).
		Delete(&backoffice_models.BackofficePasskey{})
	if res.Error != nil {
		h.logger.Error("Backoffice passkeys: failed to delete", zap.Uint("user_id", userID), zap.Error(res.Error))
		return internalResponse()
	}
	if res.RowsAffected == 0 {
		return lastFactorResponse()
	}
	h.logger.Info("Backoffice passkey deleted", zap.Uint("user_id", userID), zap.Uint64("passkey_id", id))
	return envelope.SuccessResponse(nil, "auth.passkey.delete.success")
}

// BeginPasskeyLogin: POST /backoffice/auth/passkey/begin. Discoverable: no
// user name is sent.
func (h *BackofficeUserHandler) BeginPasskeyLogin(c *gin.Context) envelope.Response {
	wa, errResp := h.relyingParty()
	if errResp != nil {
		return *errResp
	}
	assertion, session, err := wa.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		h.logger.Error("Backoffice passkeys: failed to begin login", zap.Error(err))
		return internalResponse()
	}
	id, err := h.saveChallenge(backoffice_models.BackofficeChallengeLogin, 0, session)
	if err != nil {
		h.logger.Error("Backoffice passkeys: failed to store challenge", zap.Error(err))
		return internalResponse()
	}
	return envelope.SuccessResponse(gin.H{"challenge_id": id, "options": assertion.Response}, "auth.passkey.login.begin.success")
}

// FinishPasskeyLogin: POST /backoffice/auth/passkey/login. A valid assertion
// (user verified) opens the session with no password or TOTP step. Every
// failure answers the same 401.
func (h *BackofficeUserHandler) FinishPasskeyLogin(c *gin.Context) envelope.Response {
	var req struct {
		ChallengeID string          `json:"challenge_id" binding:"required,max=64"`
		Credential  json.RawMessage `json:"credential" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidRequestResponse()
	}
	session, ok, err := h.takeChallenge(req.ChallengeID, backoffice_models.BackofficeChallengeLogin, 0)
	if err != nil {
		h.logger.Error("Backoffice passkeys: failed to read challenge", zap.Error(err))
		return internalResponse()
	}
	if !ok {
		return passkeyInvalidResponse(http.StatusUnauthorized)
	}
	wa, errResp := h.relyingParty()
	if errResp != nil {
		return *errResp
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(req.Credential)
	if err != nil {
		h.logger.Warn("Backoffice passkeys: malformed login response", zap.Error(err))
		return passkeyInvalidResponse(http.StatusUnauthorized)
	}
	lookup := func(_, userHandle []byte) (webauthn.User, error) {
		uid, ok := adminIDFromHandle(userHandle)
		if !ok {
			return nil, errors.New("invalid user handle")
		}
		var user backoffice_models.BackofficeUser
		if err := h.db.First(&user, uid).Error; err != nil {
			return nil, err
		}
		return h.loadAdminPasskeyUser(user)
	}
	waUser, cred, err := wa.ValidatePasskeyLogin(lookup, *session, parsed)
	if err != nil {
		h.logger.Warn("Backoffice passkey login refused", zap.Error(err))
		return passkeyInvalidResponse(http.StatusUnauthorized)
	}
	user := waUser.(adminPasskeyUser).user
	if cred.Authenticator.CloneWarning {
		h.logger.Warn("Backoffice passkey login refused: clone warning", zap.Uint("user_id", user.ID))
		return passkeyInvalidResponse(http.StatusUnauthorized)
	}
	if user.LockedUntil != nil && user.LockedUntil.After(h.now()) {
		h.logger.Warn("Backoffice passkey login refused: account locked", zap.Uint("user_id", user.ID))
		return accountLockedResponse()
	}
	stored, err := json.Marshal(cred)
	if err != nil {
		h.logger.Error("Backoffice passkeys: failed to encode credential", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	if err := h.db.Model(&backoffice_models.BackofficePasskey{}).
		Where("backoffice_user_id = ? AND credential_id = ?", user.ID, cred.ID).
		Updates(map[string]any{"credential": stored, "last_used_at": h.now()}).Error; err != nil {
		h.logger.Error("Backoffice passkeys: failed to update credential", zap.Uint("user_id", user.ID), zap.Error(err))
		return internalResponse()
	}
	data, errResp := h.issueSession(c, user)
	if errResp != nil {
		return *errResp
	}
	h.logger.Info("Backoffice user logged in successfully", zap.String("username", user.UserName), zap.Bool("passkey", true))
	return envelope.SuccessResponse(data, "login.successful")
}
