package quickbooks

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/permissions"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	qbo "github.com/javaknight1/servicepro/backend/pkg/clients/quickbooks"
)

// StateTTL is how long a connect link (OAuth state) stays valid.
const StateTTL = 10 * time.Minute

// refreshMargin refreshes an access token this long before it expires.
const refreshMargin = 2 * time.Minute

const statePurpose = "quickbooks-oauth-state"

// oauthState is the signed OAuth state: who started the connect, for which
// tenant, until when, and a single-use nonce whose hash is stored on the
// connection row.
type oauthState struct {
	TenantID uuid.UUID `json:"t"`
	UserID   uuid.UUID `json:"u"`
	Nonce    string    `json:"n"`
	Expires  int64     `json:"e"`
}

func (s *Service) signState(st oauthState) (string, error) {
	payload, err := json.Marshal(st)
	if err != nil {
		return "", err
	}
	mac := s.box.MAC(statePurpose, payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(mac), nil
}

// parseState verifies the signature and expiry of a state value.
func (s *Service) parseState(raw string) (*oauthState, error) {
	payloadPart, macPart, ok := strings.Cut(raw, ".")
	if !ok || len(raw) > 1024 {
		return nil, ErrInvalidState
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return nil, ErrInvalidState
	}
	mac, err := base64.RawURLEncoding.DecodeString(macPart)
	if err != nil || !hmac.Equal(mac, s.box.MAC(statePurpose, payload)) {
		return nil, ErrInvalidState
	}
	var st oauthState
	if err := json.Unmarshal(payload, &st); err != nil || st.TenantID == uuid.Nil || st.UserID == uuid.Nil || st.Nonce == "" {
		return nil, ErrInvalidState
	}
	if !s.now().Before(time.Unix(st.Expires, 0)) {
		return nil, ErrInvalidState
	}
	return &st, nil
}

func hashNonce(nonce string) string {
	sum := sha256.Sum256([]byte(nonce))
	return hex.EncodeToString(sum[:])
}

// StartConnect records a single-use connect request for the tenant and
// returns the Intuit consent URL.
func (s *Service) StartConnect(ctx context.Context, tenantID, userID uuid.UUID) (string, error) {
	if !s.Configured() {
		return "", ErrNotConfigured
	}
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	nonce := base64.RawURLEncoding.EncodeToString(buf)
	expires := s.now().Add(StateTTL)
	state, err := s.signState(oauthState{TenantID: tenantID, UserID: userID, Nonce: nonce, Expires: expires.Unix()})
	if err != nil {
		return "", err
	}
	// One pending connect per tenant: a newer link replaces an older one.
	row := models.QuickBooksConnection{
		TenantID: tenantID, Status: models.QuickBooksDisconnected, Environment: s.environment,
		OAuthStateHash: strPtr(hashNonce(nonce)), OAuthStateExpiresAt: &expires,
	}
	err = s.db.WithContext(tenantCtx(ctx, tenantID)).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"oauth_state_hash", "oauth_state_expires_at", "updated_at"}),
	}).Create(&row).Error
	if err != nil {
		return "", fmt.Errorf("record quickbooks connect request: %w", err)
	}
	return s.client.AuthorizeURL(state), nil
}

// CompleteConnect handles Intuit's redirect: it verifies the state (signed,
// unexpired, single use, its user still allowed), exchanges the code and
// stores the sealed tokens. It returns the tenant connected.
func (s *Service) CompleteConnect(ctx context.Context, code, rawState, realmID string) (uuid.UUID, error) {
	if !s.Configured() {
		return uuid.Nil, ErrNotConfigured
	}
	st, err := s.parseState(rawState)
	if err != nil {
		return uuid.Nil, err
	}
	tctx := tenantCtx(ctx, st.TenantID)

	// Consume the nonce atomically: a replayed state finds no row.
	res := s.db.WithContext(tctx).Model(&models.QuickBooksConnection{}).
		Where("tenant_id = ? AND oauth_state_hash = ? AND oauth_state_expires_at > ?", st.TenantID, hashNonce(st.Nonce), s.now()).
		Updates(map[string]any{"oauth_state_hash": nil, "oauth_state_expires_at": nil})
	if res.Error != nil {
		return uuid.Nil, res.Error
	}
	if res.RowsAffected != 1 {
		return uuid.Nil, ErrInvalidState
	}
	if s.perms != nil {
		ok, err := s.perms.CheckPermission(ctx, st.UserID, st.TenantID, permissions.IntegrationsManage)
		if err != nil {
			return uuid.Nil, err
		}
		if !ok {
			return uuid.Nil, ErrInvalidState
		}
	}
	realmID = strings.TrimSpace(realmID)
	if code == "" || realmID == "" || len(realmID) > 50 {
		return uuid.Nil, ErrInvalidState
	}

	tokens, err := s.client.ExchangeCode(ctx, code)
	if err != nil {
		if errors.Is(err, qbo.ErrInvalidGrant) {
			return uuid.Nil, ErrInvalidState
		}
		logging.Error(ctx, "[QUICKBOOKS] Code exchange failed", map[string]any{"tenant_id": st.TenantID, "error": err.Error()})
		return uuid.Nil, ErrQuickBooksUnavailable
	}
	company, err := s.client.CompanyName(ctx, qbo.Session{RealmID: realmID, AccessToken: tokens.AccessToken})
	if err != nil {
		logging.Warn(ctx, "[QUICKBOOKS] Company name lookup failed", map[string]any{"tenant_id": st.TenantID, "error": err.Error()})
	}

	access, refresh, err := s.seal(st.TenantID, tokens)
	if err != nil {
		return uuid.Nil, err
	}
	now := s.now()
	err = s.db.WithContext(tctx).Transaction(func(tx *gorm.DB) error {
		var conn models.QuickBooksConnection
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ?", st.TenantID).First(&conn).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"status": models.QuickBooksConnected, "environment": s.environment, "realm_id": realmID,
			"company_name": strPtr(company), "access_token_enc": access, "access_token_expires_at": tokens.AccessExpiresAt,
			"refresh_token_enc": refresh, "refresh_token_expires_at": tokens.RefreshExpiresAt,
			"connected_by": st.UserID, "connected_at": now, "disconnected_at": nil, "last_sync_error": nil,
		}
		if deref(conn.RealmID) != realmID {
			// A different company: what we synced lives in the old one. Start
			// over, from today, with this company's own mapping settings.
			if err := tx.Where("tenant_id = ?", st.TenantID).Delete(&models.QuickBooksSyncRecord{}).Error; err != nil {
				return err
			}
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			updates["sync_from"] = today
			for _, col := range []string{"income_account_id", "income_account_name", "item_id", "item_name",
				"tax_code_id", "tax_code_name", "deposit_account_id", "deposit_account_name"} {
				updates[col] = nil
			}
		}
		return tx.Model(&models.QuickBooksConnection{}).Where("id = ?", conn.ID).Updates(updates).Error
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("store quickbooks connection: %w", err)
	}
	logging.Info(ctx, "[QUICKBOOKS] Connected", map[string]any{"tenant_id": st.TenantID, "user_id": st.UserID})
	return st.TenantID, nil
}

// StateTenant returns the tenant a (valid) state names, without consuming
// it; used to pick the redirect after a failed callback.
func (s *Service) StateTenant(rawState string) (uuid.UUID, bool) {
	if !s.Configured() {
		return uuid.Nil, false
	}
	st, err := s.parseState(rawState)
	if err != nil {
		return uuid.Nil, false
	}
	return st.TenantID, true
}

func (s *Service) seal(tenantID uuid.UUID, t *qbo.Tokens) (access, refresh string, err error) {
	access, err = s.box.Seal(t.AccessToken, "quickbooks:access:"+tenantID.String())
	if err != nil {
		return "", "", err
	}
	refresh, err = s.box.Seal(t.RefreshToken, "quickbooks:refresh:"+tenantID.String())
	return access, refresh, err
}

func (s *Service) openAccess(conn *models.QuickBooksConnection) (string, error) {
	return s.box.Open(deref(conn.AccessTokenEnc), "quickbooks:access:"+conn.TenantID.String())
}

func (s *Service) openRefresh(conn *models.QuickBooksConnection) (string, error) {
	return s.box.Open(deref(conn.RefreshTokenEnc), "quickbooks:refresh:"+conn.TenantID.String())
}

// Disconnect revokes the tokens at Intuit (best effort) and forgets them.
// The sync map is kept: reconnecting the same company carries on.
func (s *Service) Disconnect(ctx context.Context, tenantID uuid.UUID) error {
	if !s.Configured() {
		return ErrNotConfigured
	}
	conn, err := s.loadConnection(ctx, tenantID)
	if err != nil {
		return err
	}
	if conn == nil || conn.Status == models.QuickBooksDisconnected {
		return ErrNotConnected
	}
	if refresh, err := s.openRefresh(conn); err == nil && refresh != "" {
		if err := s.client.Revoke(ctx, refresh); err != nil {
			logging.Warn(ctx, "[QUICKBOOKS] Token revoke failed; forgetting the tokens anyway", map[string]any{"tenant_id": tenantID, "error": err.Error()})
		}
	}
	return s.db.WithContext(tenantCtx(ctx, tenantID)).Model(&models.QuickBooksConnection{}).
		Where("tenant_id = ?", tenantID).
		Updates(map[string]any{
			"status": models.QuickBooksDisconnected, "access_token_enc": nil, "access_token_expires_at": nil,
			"refresh_token_enc": nil, "refresh_token_expires_at": nil, "disconnected_at": s.now(),
			"sync_locked_until": nil,
		}).Error
}

// session returns a usable session for the tenant, refreshing the access
// token when it is (nearly) expired. force refreshes even a token that looks
// valid (after a 401), unless another caller already replaced staleToken.
func (s *Service) session(ctx context.Context, tenantID uuid.UUID, force bool, staleToken string) (qbo.Session, error) {
	conn, err := s.connected(ctx, tenantID)
	if err != nil {
		return qbo.Session{}, err
	}
	if !force && conn.AccessTokenExpiresAt != nil && s.now().Add(refreshMargin).Before(*conn.AccessTokenExpiresAt) {
		access, err := s.openAccess(conn)
		if err == nil {
			return qbo.Session{RealmID: deref(conn.RealmID), AccessToken: access}, nil
		}
	}
	return s.refresh(ctx, tenantID, force, staleToken)
}

// refresh rotates the tokens under SELECT ... FOR UPDATE: a second caller
// waits, then sees the new token and uses it instead of refreshing with a
// refresh token that Intuit has already rotated away.
func (s *Service) refresh(ctx context.Context, tenantID uuid.UUID, force bool, staleToken string) (qbo.Session, error) {
	var sess qbo.Session
	var invalid bool
	err := s.db.WithContext(tenantCtx(ctx, tenantID)).Transaction(func(tx *gorm.DB) error {
		var conn models.QuickBooksConnection
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ?", tenantID).First(&conn).Error; err != nil {
			return err
		}
		if conn.Status != models.QuickBooksConnected {
			return ErrReauthRequired
		}
		sess.RealmID = deref(conn.RealmID)
		if conn.AccessTokenExpiresAt != nil && s.now().Add(refreshMargin).Before(*conn.AccessTokenExpiresAt) {
			if access, err := s.openAccess(&conn); err == nil && (!force || access != staleToken) {
				sess.AccessToken = access // someone refreshed while we waited
				return nil
			}
		}
		refreshToken, err := s.openRefresh(&conn)
		if err != nil {
			invalid = true
			return ErrReauthRequired
		}
		tokens, err := s.client.RefreshTokens(ctx, refreshToken)
		if errors.Is(err, qbo.ErrInvalidGrant) {
			invalid = true
			return ErrReauthRequired
		}
		if err != nil {
			return fmt.Errorf("refresh quickbooks token: %w", err)
		}
		access, refresh, err := s.seal(tenantID, tokens)
		if err != nil {
			return err
		}
		sess.AccessToken = tokens.AccessToken
		return tx.Model(&models.QuickBooksConnection{}).Where("id = ?", conn.ID).Updates(map[string]any{
			"access_token_enc": access, "access_token_expires_at": tokens.AccessExpiresAt,
			"refresh_token_enc": refresh, "refresh_token_expires_at": tokens.RefreshExpiresAt,
		}).Error
	})
	if invalid {
		s.markReauth(ctx, tenantID)
	}
	if err != nil {
		return qbo.Session{}, err
	}
	return sess, nil
}

// markReauth flags the connection: the tenant must reconnect.
func (s *Service) markReauth(ctx context.Context, tenantID uuid.UUID) {
	msg := "QuickBooks authorization expired or was revoked. Reconnect QuickBooks to resume syncing."
	err := s.db.WithContext(tenantCtx(ctx, tenantID)).Model(&models.QuickBooksConnection{}).
		Where("tenant_id = ? AND status = ?", tenantID, models.QuickBooksConnected).
		Updates(map[string]any{"status": models.QuickBooksReauthRequired, "last_sync_error": msg}).Error
	if err != nil {
		logging.Error(ctx, "[QUICKBOOKS] Could not flag connection for reauth", map[string]any{"tenant_id": tenantID, "error": err.Error()})
		return
	}
	logging.Warn(ctx, "[QUICKBOOKS] Refresh token rejected; tenant must reconnect", map[string]any{"tenant_id": tenantID})
}
