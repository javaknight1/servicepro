package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// T255: the accepted Terms/Privacy version and time are stored with the user.
func TestUserRepository_CreateRecordsTermsAcceptance(t *testing.T) {
	db := openTestDB(t)
	repo := &UserRepositoryGorm{db: db}

	before := time.Now().Add(-time.Second)
	user, err := repo.Create("terms-"+uuid.NewString()+"@example.test", "hash", models.CurrentTermsVersion)
	require.NoError(t, err)

	var stored models.User
	require.NoError(t, db.First(&stored, "id = ?", user.ID).Error)
	require.NotNil(t, stored.TermsVersion)
	assert.Equal(t, models.CurrentTermsVersion, *stored.TermsVersion)
	require.NotNil(t, stored.TermsAcceptedAt)
	assert.True(t, stored.TermsAcceptedAt.After(before))

	// No version supplied: nothing recorded.
	legacy, err := repo.Create("legacy-"+uuid.NewString()+"@example.test", "hash", "")
	require.NoError(t, err)
	var storedLegacy models.User
	require.NoError(t, db.First(&storedLegacy, "id = ?", legacy.ID).Error)
	assert.Nil(t, storedLegacy.TermsVersion)
	assert.Nil(t, storedLegacy.TermsAcceptedAt)
}
