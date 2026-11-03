package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // Register PNG decoder
	"io"
	"time"

	"github.com/google/uuid"
	"golang.org/x/image/draw"

	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	"github.com/javaknight1/servicepro/backend/pkg/clients/storage"
)

const (
	MaxUploadSize = 2 * 1024 * 1024 // 2MB
	TargetWidth   = 200
	TargetHeight  = 200
	StoragePrefix = "profile-pictures"
	JPEGQuality   = 85
)

// MaxImageDimension bounds the width/height of uploaded profile pictures.
const MaxImageDimension = 4000

// ErrImageTooLarge is returned for images whose dimensions exceed MaxImageDimension.
var ErrImageTooLarge = errors.New("image dimensions too large (max 4000x4000)")

var (
	ErrFileTooLarge       = errors.New("file size exceeds maximum allowed (2MB)")
	ErrInvalidImageFormat = errors.New("invalid image format, only JPEG and PNG are supported")
	ErrImageProcessing    = errors.New("failed to process image")
	// ErrNoProfilePicture is returned when the user has no picture or the
	// viewer may not see it (no shared tenant); both are a 404.
	ErrNoProfilePicture = errors.New("profile picture not found")
)

// ProfilePictureLinkTTL is how long a profile picture's presigned link lives.
const ProfilePictureLinkTTL = time.Hour

// TenantMembership answers whether two users share an active tenant.
type TenantMembership interface {
	UsersShareTenant(ctx context.Context, a, b uuid.UUID) (bool, error)
}

// ProfilePictureService handles profile picture operations
type ProfilePictureService struct {
	userRepo      repository.UserRepositoryGormInterface
	members       TenantMembership
	storageClient storage.Client
}

// NewProfilePictureService creates a new profile picture service
func NewProfilePictureService(userRepo repository.UserRepositoryGormInterface, members TenantMembership, storageClient storage.Client) *ProfilePictureService {
	return &ProfilePictureService{
		userRepo:      userRepo,
		members:       members,
		storageClient: storageClient,
	}
}

// profilePictureKey is where a user's picture lives in the (private) bucket.
func profilePictureKey(userID uuid.UUID) string {
	return fmt.Sprintf("%s/%s.jpg", StoragePrefix, userID.String())
}

// profilePicturePath is the stored profile_picture_url: an API path that
// redirects to a short-lived presigned link (T160: the bucket is private, so a
// storage URL saved in the database would never load). v busts browser caches
// when the picture changes.
func profilePicturePath(userID uuid.UUID, at time.Time) string {
	return fmt.Sprintf("/api/v1/users/%s/profile-picture?v=%d", userID, at.Unix())
}

// PictureURL returns a presigned link to userID's picture for viewerID, who
// must be that user or share an active tenant with them.
func (s *ProfilePictureService) PictureURL(ctx context.Context, viewerID, userID uuid.UUID) (string, error) {
	if viewerID != userID {
		shared, err := s.members.UsersShareTenant(ctx, viewerID, userID)
		if err != nil {
			return "", fmt.Errorf("check shared tenant: %w", err)
		}
		if !shared {
			return "", ErrNoProfilePicture
		}
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil || user.ProfilePictureURL == nil || *user.ProfilePictureURL == "" {
		return "", ErrNoProfilePicture
	}
	url, err := s.storageClient.GetPresignedURL(ctx, profilePictureKey(userID), ProfilePictureLinkTTL)
	if err != nil {
		return "", fmt.Errorf("presign profile picture: %w", err)
	}
	return url, nil
}

// UploadProfilePicture handles profile picture upload, resize, and storage
func (s *ProfilePictureService) UploadProfilePicture(ctx context.Context, userID uuid.UUID, fileData io.Reader, size int64, contentType string) (string, error) {
	// Validate file size
	if size > MaxUploadSize {
		return "", ErrFileTooLarge
	}

	// Validate content type
	if contentType != "image/jpeg" && contentType != "image/png" {
		return "", ErrInvalidImageFormat
	}

	// Read file data
	data, err := io.ReadAll(io.LimitReader(fileData, MaxUploadSize+1))
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	if int64(len(data)) > MaxUploadSize {
		return "", ErrFileTooLarge
	}

	// Check the declared dimensions BEFORE decoding: a tiny PNG can declare
	// 50k x 50k pixels and exhaust memory when decoded (F-19).
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", ErrInvalidImageFormat
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > MaxImageDimension || cfg.Height > MaxImageDimension {
		return "", ErrImageTooLarge
	}

	// Decode image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", ErrInvalidImageFormat
	}

	// Resize image to target dimensions (square, maintaining aspect ratio)
	resized := thumbnail(img, TargetWidth, TargetHeight)

	// Encode as JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return "", fmt.Errorf("%w: %w", ErrImageProcessing, err)
	}

	// Generate unique filename
	filename := profilePictureKey(userID)

	// Delete existing profile picture if any
	_ = s.storageClient.Delete(ctx, filename)

	// Upload to storage
	uploadInput := storage.NewUploadInput(filename).
		WithContentType("image/jpeg").
		WithCacheControl("private, max-age=3600")

	if _, err := s.storageClient.UploadBytes(ctx, buf.Bytes(), uploadInput); err != nil {
		return "", fmt.Errorf("failed to upload to storage: %w", err)
	}

	// Update user's profile picture URL
	path := profilePicturePath(userID, time.Now())
	if err := s.userRepo.UpdateProfilePicture(userID, &path); err != nil {
		// Try to cleanup the uploaded file
		_ = s.storageClient.Delete(ctx, filename)
		return "", fmt.Errorf("failed to update user profile: %w", err)
	}

	return path, nil
}

// DeleteProfilePicture removes a user's profile picture
func (s *ProfilePictureService) DeleteProfilePicture(ctx context.Context, userID uuid.UUID) error {
	// Get user to check if they have a profile picture
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// If no profile picture, nothing to delete
	if user.ProfilePictureURL == nil || *user.ProfilePictureURL == "" {
		return nil
	}

	// Generate the storage key
	filename := profilePictureKey(userID)

	// Delete from storage
	if err := s.storageClient.Delete(ctx, filename); err != nil {
		// Log but don't fail - the file might not exist
		logging.Warn(ctx, "[PROFILE-PICTURE] Failed to delete profile picture from storage", map[string]any{"error": err})
	}

	// Clear the profile picture URL in database
	if err := s.userRepo.UpdateProfilePicture(userID, nil); err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}

	return nil
}

// thumbnail scales img down to fit within maxW x maxH, keeping its aspect
// ratio; smaller images are returned unchanged (like the archived
// nfnt/resize.Thumbnail it replaces, T264). Catmull-Rom keeps edges sharp.
func thumbnail(img image.Image, maxW, maxH uint) image.Image {
	b := img.Bounds()
	w, h := uint(b.Dx()), uint(b.Dy())
	if w <= maxW && h <= maxH || w == 0 || h == 0 {
		return img
	}
	nw, nh := maxW, h*maxW/w
	if nh > maxH {
		nw, nh = w*maxH/h, maxH
	}
	if nw == 0 {
		nw = 1
	}
	if nh == 0 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, int(nw), int(nh)))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
}
