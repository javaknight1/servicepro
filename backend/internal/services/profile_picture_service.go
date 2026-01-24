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

	"github.com/google/uuid"
	"github.com/nfnt/resize"

	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/clients/storage"
)

const (
	MaxUploadSize = 2 * 1024 * 1024 // 2MB
	TargetWidth   = 200
	TargetHeight  = 200
	StoragePrefix = "profile-pictures"
	JPEGQuality   = 85
)

var (
	ErrFileTooLarge       = errors.New("file size exceeds maximum allowed (2MB)")
	ErrInvalidImageFormat = errors.New("invalid image format, only JPEG and PNG are supported")
	ErrImageProcessing    = errors.New("failed to process image")
)

// ProfilePictureService handles profile picture operations
type ProfilePictureService struct {
	userRepo      repository.UserRepositoryGormInterface
	storageClient storage.Client
}

// NewProfilePictureService creates a new profile picture service
func NewProfilePictureService(userRepo repository.UserRepositoryGormInterface, storageClient storage.Client) *ProfilePictureService {
	return &ProfilePictureService{
		userRepo:      userRepo,
		storageClient: storageClient,
	}
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

	// Decode image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", ErrInvalidImageFormat
	}

	// Resize image to target dimensions (square, maintaining aspect ratio)
	resized := resize.Thumbnail(TargetWidth, TargetHeight, img, resize.Lanczos3)

	// Encode as JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return "", fmt.Errorf("%w: %v", ErrImageProcessing, err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s/%s.jpg", StoragePrefix, userID.String())

	// Delete existing profile picture if any
	_ = s.storageClient.Delete(ctx, filename)

	// Upload to storage
	uploadInput := storage.NewUploadInput(filename).
		WithContentType("image/jpeg").
		WithCacheControl("public, max-age=31536000") // Cache for 1 year

	output, err := s.storageClient.UploadBytes(ctx, buf.Bytes(), uploadInput)
	if err != nil {
		return "", fmt.Errorf("failed to upload to storage: %w", err)
	}

	// Update user's profile picture URL
	if err := s.userRepo.UpdateProfilePicture(userID, &output.URL); err != nil {
		// Try to cleanup the uploaded file
		_ = s.storageClient.Delete(ctx, filename)
		return "", fmt.Errorf("failed to update user profile: %w", err)
	}

	return output.URL, nil
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
	filename := fmt.Sprintf("%s/%s.jpg", StoragePrefix, userID.String())

	// Delete from storage
	if err := s.storageClient.Delete(ctx, filename); err != nil {
		// Log but don't fail - the file might not exist
		fmt.Printf("Warning: failed to delete profile picture from storage: %v\n", err)
	}

	// Clear the profile picture URL in database
	if err := s.userRepo.UpdateProfilePicture(userID, nil); err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}

	return nil
}
