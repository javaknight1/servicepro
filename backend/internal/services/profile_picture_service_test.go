package services

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	storagemock "github.com/javaknight1/servicepro/backend/pkg/clients/storage/mock"
)

type fakeMembership map[[2]uuid.UUID]bool

func (f fakeMembership) UsersShareTenant(_ context.Context, a, b uuid.UUID) (bool, error) {
	return f[[2]uuid.UUID{a, b}], nil
}

func smallPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	return buf.Bytes()
}

// T160: the stored profile_picture_url was a storage URL, which never loads
// from a private R2 bucket. It is now an API path that redirects to a
// presigned link, for the user and their teammates only.
func TestProfilePicture_StoredAsAPIPathAndTenantScoped(t *testing.T) {
	owner, teammate, stranger := uuid.New(), uuid.New(), uuid.New()
	users := new(MockUserRepositoryGorm)
	var stored *string
	users.On("UpdateProfilePicture", owner, mock.Anything).Run(func(a mock.Arguments) { stored = a.Get(1).(*string) }).Return(nil)
	store := storagemock.NewClient(nil)
	svc := NewProfilePictureService(users, fakeMembership{{teammate, owner}: true}, store)

	data := smallPNG(t)
	path, err := svc.UploadProfilePicture(context.Background(), owner, bytes.NewReader(data), int64(len(data)), "image/png")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(path, "/api/v1/users/"+owner.String()+"/profile-picture?v="), path)
	require.NotNil(t, stored)
	assert.Equal(t, path, *stored)

	users.On("GetByID", owner).Return(&models.User{ID: owner, ProfilePictureURL: &path}, nil)

	for _, viewer := range []uuid.UUID{owner, teammate} {
		url, err := svc.PictureURL(context.Background(), viewer, owner)
		require.NoError(t, err)
		assert.NotEmpty(t, url)
	}
	_, err = svc.PictureURL(context.Background(), stranger, owner)
	assert.ErrorIs(t, err, ErrNoProfilePicture, "no shared tenant → 404")

	users.On("GetByID", teammate).Return(&models.User{ID: teammate}, nil)
	_, err = svc.PictureURL(context.Background(), teammate, teammate)
	assert.ErrorIs(t, err, ErrNoProfilePicture, "no picture → 404")
}
