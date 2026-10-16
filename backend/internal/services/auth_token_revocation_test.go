package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

// versionedRepo adds token-version bumping to MockUserRepository.
type versionedRepo struct {
	*MockUserRepository
	bumped []uuid.UUID
}

func (r *versionedRepo) IncrementTokenVersion(id uuid.UUID) error {
	r.bumped = append(r.bumped, id)
	return nil
}

func TestRefreshToken_RevokedByTokenVersion(t *testing.T) {
	jwtManager := auth.NewJWTManager(getTestJWTConfig())
	userID := uuid.New()
	user := &models.User{ID: userID, Email: "v@example.com", EmailVerified: true, TokenVersion: 3}

	mockRepo := new(MockUserRepository)
	mockRepo.On("GetByEmail", "v@example.com").Return(user, nil)
	svc := NewAuthService(mockRepo, jwtManager, getTestAuthConfig())

	_, current, err := jwtManager.GenerateTokenPairForVersion(userID.String(), user.Email, 3)
	require.NoError(t, err)
	resp, err := svc.RefreshToken(current)
	require.NoError(t, err)
	claims, err := jwtManager.ValidateToken(resp.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, 3, claims.TokenVersion, "rotated tokens keep the current version")

	_, stale, err := jwtManager.GenerateTokenPairForVersion(userID.String(), user.Email, 2)
	require.NoError(t, err)
	_, err = svc.RefreshToken(stale)
	assert.ErrorIs(t, err, ErrInvalidRefreshToken, "token from before logout/reset is revoked")

	access, _, err := jwtManager.GenerateTokenPairForVersion(userID.String(), user.Email, 3)
	require.NoError(t, err)
	_, err = svc.RefreshToken(access)
	assert.ErrorIs(t, err, ErrInvalidRefreshToken, "access token can't be used to refresh")
}

func TestLogout_BumpsTokenVersion(t *testing.T) {
	jwtManager := auth.NewJWTManager(getTestJWTConfig())
	repo := &versionedRepo{MockUserRepository: new(MockUserRepository)}
	svc := NewAuthService(repo, jwtManager, getTestAuthConfig())

	userID := uuid.New()
	_, refresh, err := jwtManager.GenerateTokenPairForVersion(userID.String(), "x@example.com", 0)
	require.NoError(t, err)

	require.NoError(t, svc.Logout(refresh))
	assert.Equal(t, []uuid.UUID{userID}, repo.bumped)

	require.NoError(t, svc.Logout("garbage"), "invalid tokens are ignored")
	assert.Len(t, repo.bumped, 1)
}
