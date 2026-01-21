package storage

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/robfig/cron/v3"
)

// =============================================================================
// Backup Service
// =============================================================================

// BackupService manages document backups
type BackupService struct {
	s3Client   *S3Client
	metadata   *MetadataManager
	config     *Config
	scheduler  *cron.Cron
	mu         sync.RWMutex
	lastBackup *BackupInfo
	isRunning  bool
	stopChan   chan struct{}
	logger     Logger
}

// Logger interface for logging
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// DefaultLogger is a simple logger implementation
type DefaultLogger struct{}

func (l *DefaultLogger) Debug(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[DEBUG] %s %v\n", msg, keysAndValues)
}

func (l *DefaultLogger) Info(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[INFO] %s %v\n", msg, keysAndValues)
}

func (l *DefaultLogger) Warn(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[WARN] %s %v\n", msg, keysAndValues)
}

func (l *DefaultLogger) Error(msg string, keysAndValues ...interface{}) {
	fmt.Printf("[ERROR] %s %v\n", msg, keysAndValues)
}

// BackupInfo contains information about a backup
type BackupInfo struct {
	ID             string          `json:"id"`
	Type           BackupType      `json:"type"`
	Status         BackupStatus    `json:"status"`
	StartedAt      time.Time       `json:"started_at"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	ObjectCount    int64           `json:"object_count"`
	TotalSize      int64           `json:"total_size"`
	CompressedSize int64           `json:"compressed_size,omitempty"`
	SourceBucket   string          `json:"source_bucket"`
	DestBucket     string          `json:"dest_bucket"`
	Errors         []string        `json:"errors,omitempty"`
	Manifest       *BackupManifest `json:"manifest,omitempty"`
}

// BackupType represents the type of backup
type BackupType string

const (
	BackupTypeFull         BackupType = "full"
	BackupTypeIncremental  BackupType = "incremental"
	BackupTypeDifferential BackupType = "differential"
)

// BackupStatus represents the status of a backup
type BackupStatus string

const (
	BackupStatusPending   BackupStatus = "pending"
	BackupStatusRunning   BackupStatus = "running"
	BackupStatusCompleted BackupStatus = "completed"
	BackupStatusFailed    BackupStatus = "failed"
	BackupStatusCancelled BackupStatus = "cancelled"
)

// BackupManifest contains the list of objects in a backup
type BackupManifest struct {
	Version      string             `json:"version"`
	CreatedAt    time.Time          `json:"created_at"`
	BaseBackupID string             `json:"base_backup_id,omitempty"`
	Objects      []BackupObjectInfo `json:"objects"`
}

// BackupObjectInfo contains information about a backed-up object
type BackupObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	ETag         string    `json:"etag"`
	LastModified time.Time `json:"last_modified"`
	StorageClass string    `json:"storage_class"`
	BackupKey    string    `json:"backup_key"`
	Checksum     string    `json:"checksum,omitempty"`
}

// NewBackupService creates a new backup service
func NewBackupService(s3Client *S3Client, metadata *MetadataManager, logger Logger) *BackupService {
	if logger == nil {
		logger = &DefaultLogger{}
	}

	return &BackupService{
		s3Client:  s3Client,
		metadata:  metadata,
		config:    s3Client.config,
		scheduler: cron.New(),
		stopChan:  make(chan struct{}),
		logger:    logger,
	}
}

// Start starts the backup scheduler
func (b *BackupService) Start() error {
	if !b.config.Backup.Enabled {
		b.logger.Info("Backup service disabled")
		return nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.isRunning {
		return nil
	}

	// Schedule backups
	_, err := b.scheduler.AddFunc(b.config.Backup.Schedule, func() {
		ctx := context.Background()
		backupType := BackupTypeIncremental
		if b.shouldRunFullBackup() {
			backupType = BackupTypeFull
		}

		info, err := b.RunBackup(ctx, backupType)
		if err != nil {
			b.logger.Error("Backup failed", "error", err)
			if b.config.Backup.NotifyOnFailure {
				b.sendNotification("Backup Failed", err.Error())
			}
		} else {
			b.logger.Info("Backup completed", "id", info.ID, "objects", info.ObjectCount)
			if b.config.Backup.NotifyOnComplete {
				b.sendNotification("Backup Completed", fmt.Sprintf("Backed up %d objects", info.ObjectCount))
			}
		}
	})

	if err != nil {
		return fmt.Errorf("failed to schedule backup: %w", err)
	}

	b.scheduler.Start()
	b.isRunning = true
	b.logger.Info("Backup service started", "schedule", b.config.Backup.Schedule)

	return nil
}

// Stop stops the backup scheduler
func (b *BackupService) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.isRunning {
		return
	}

	b.scheduler.Stop()
	close(b.stopChan)
	b.isRunning = false
	b.logger.Info("Backup service stopped")
}

// RunBackup runs a backup immediately
func (b *BackupService) RunBackup(ctx context.Context, backupType BackupType) (*BackupInfo, error) {
	info := &BackupInfo{
		ID:           generateBackupID(),
		Type:         backupType,
		Status:       BackupStatusRunning,
		StartedAt:    time.Now().UTC(),
		SourceBucket: b.config.Buckets.Primary,
		DestBucket:   b.config.Buckets.Backup,
	}

	b.logger.Info("Starting backup", "id", info.ID, "type", backupType)

	// Get objects to backup
	var objects []ObjectInfo
	var err error

	if backupType == BackupTypeFull {
		objects, err = b.s3Client.ListAll(ctx, "")
	} else {
		objects, err = b.getIncrementalObjects(ctx)
	}

	if err != nil {
		info.Status = BackupStatusFailed
		info.Errors = append(info.Errors, err.Error())
		return info, fmt.Errorf("%w: failed to list objects: %v", ErrBackupFailed, err)
	}

	// Create manifest
	manifest := &BackupManifest{
		Version:   "1.0",
		CreatedAt: time.Now().UTC(),
		Objects:   make([]BackupObjectInfo, 0, len(objects)),
	}

	// Copy objects to backup bucket
	var totalSize int64
	var compressedSize int64
	var errors []string

	for _, obj := range objects {
		select {
		case <-ctx.Done():
			info.Status = BackupStatusCancelled
			return info, ctx.Err()
		default:
		}

		backupKey := b.generateBackupKey(info.ID, obj.Key)

		// Copy with optional compression
		objSize, err := b.copyToBackup(ctx, obj.Key, backupKey)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to backup %s: %v", obj.Key, err))
			continue
		}

		totalSize += obj.Size
		compressedSize += objSize

		manifest.Objects = append(manifest.Objects, BackupObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			ETag:         obj.ETag,
			LastModified: obj.LastModified,
			StorageClass: obj.StorageClass,
			BackupKey:    backupKey,
		})
	}

	// Save manifest
	if err := b.saveManifest(ctx, info.ID, manifest); err != nil {
		errors = append(errors, fmt.Sprintf("failed to save manifest: %v", err))
	}

	// Update backup info
	completedAt := time.Now().UTC()
	info.CompletedAt = &completedAt
	info.ObjectCount = int64(len(manifest.Objects))
	info.TotalSize = totalSize
	info.CompressedSize = compressedSize
	info.Manifest = manifest
	info.Errors = errors

	if len(errors) > 0 {
		info.Status = BackupStatusFailed
	} else {
		info.Status = BackupStatusCompleted
	}

	// Save backup info
	if err := b.saveBackupInfo(ctx, info); err != nil {
		b.logger.Error("Failed to save backup info", "error", err)
	}

	b.mu.Lock()
	b.lastBackup = info
	b.mu.Unlock()

	// Clean up old backups
	if err := b.cleanupOldBackups(ctx); err != nil {
		b.logger.Warn("Failed to cleanup old backups", "error", err)
	}

	return info, nil
}

// copyToBackup copies an object to the backup bucket
func (b *BackupService) copyToBackup(ctx context.Context, sourceKey, destKey string) (int64, error) {
	if b.config.Backup.EnableCompression {
		return b.copyWithCompression(ctx, sourceKey, destKey)
	}

	// Direct copy
	_, err := b.s3Client.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(b.config.Buckets.Backup),
		CopySource: aws.String(fmt.Sprintf("%s/%s", b.config.Buckets.Primary, sourceKey)),
		Key:        aws.String(destKey),
	})

	if err != nil {
		return 0, err
	}

	// Get object size
	info, err := b.s3Client.GetObjectInfo(ctx, sourceKey)
	if err != nil {
		return 0, err
	}

	return info.Size, nil
}

// copyWithCompression copies an object with gzip compression
func (b *BackupService) copyWithCompression(ctx context.Context, sourceKey, destKey string) (int64, error) {
	// Download source object
	data, err := b.s3Client.DownloadBytes(ctx, sourceKey)
	if err != nil {
		return 0, err
	}

	// Compress
	var compressed bytes.Buffer
	gzWriter, err := gzip.NewWriterLevel(&compressed, b.config.Backup.CompressionLevel)
	if err != nil {
		return 0, err
	}

	if _, err := gzWriter.Write(data); err != nil {
		return 0, err
	}

	if err := gzWriter.Close(); err != nil {
		return 0, err
	}

	// Upload compressed data
	compressedData := compressed.Bytes()

	_, err = b.s3Client.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:          aws.String(b.config.Buckets.Backup),
		Key:             aws.String(destKey + ".gz"),
		Body:            bytes.NewReader(compressedData),
		ContentType:     aws.String("application/gzip"),
		ContentEncoding: aws.String("gzip"),
	})

	if err != nil {
		return 0, err
	}

	return int64(len(compressedData)), nil
}

// getIncrementalObjects gets objects modified since last backup
func (b *BackupService) getIncrementalObjects(ctx context.Context) ([]ObjectInfo, error) {
	allObjects, err := b.s3Client.ListAll(ctx, "")
	if err != nil {
		return nil, err
	}

	b.mu.RLock()
	lastBackup := b.lastBackup
	b.mu.RUnlock()

	if lastBackup == nil {
		return allObjects, nil
	}

	var modifiedObjects []ObjectInfo
	for _, obj := range allObjects {
		if obj.LastModified.After(lastBackup.StartedAt) {
			modifiedObjects = append(modifiedObjects, obj)
		}
	}

	return modifiedObjects, nil
}

// generateBackupKey generates a key for the backup object
func (b *BackupService) generateBackupKey(backupID, originalKey string) string {
	return fmt.Sprintf("backups/%s/%s", backupID, originalKey)
}

// saveManifest saves the backup manifest
func (b *BackupService) saveManifest(ctx context.Context, backupID string, manifest *BackupManifest) error {
	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	_, err = b.s3Client.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(b.config.Buckets.Backup),
		Key:         aws.String(fmt.Sprintf("backups/%s/manifest.json", backupID)),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})

	return err
}

// saveBackupInfo saves the backup info
func (b *BackupService) saveBackupInfo(ctx context.Context, info *BackupInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	_, err = b.s3Client.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(b.config.Buckets.Backup),
		Key:         aws.String(fmt.Sprintf("backups/%s/info.json", info.ID)),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})

	return err
}

// shouldRunFullBackup determines if a full backup should be run
func (b *BackupService) shouldRunFullBackup() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.lastBackup == nil {
		return true
	}

	if b.lastBackup.Type == BackupTypeFull {
		return time.Since(b.lastBackup.StartedAt) >= b.config.Backup.FullBackupInterval
	}

	return false
}

// cleanupOldBackups removes old backups beyond the retention limit
func (b *BackupService) cleanupOldBackups(ctx context.Context) error {
	// List all backup directories
	result, err := b.s3Client.List(ctx, &ListInput{
		Prefix:    "backups/",
		Delimiter: "/",
	})
	if err != nil {
		return err
	}

	if len(result.CommonPrefixes) <= b.config.Backup.MaxCopies {
		return nil
	}

	// Sort by date (backup ID contains timestamp)
	sort.Strings(result.CommonPrefixes)

	// Delete oldest backups
	toDelete := result.CommonPrefixes[:len(result.CommonPrefixes)-b.config.Backup.MaxCopies]
	for _, prefix := range toDelete {
		if err := b.deleteBackup(ctx, prefix); err != nil {
			b.logger.Warn("Failed to delete old backup", "prefix", prefix, "error", err)
		}
	}

	return nil
}

// deleteBackup deletes a backup and all its objects
func (b *BackupService) deleteBackup(ctx context.Context, prefix string) error {
	objects, err := b.s3Client.ListAll(ctx, prefix)
	if err != nil {
		return err
	}

	var keys []string
	for _, obj := range objects {
		keys = append(keys, obj.Key)
	}

	// Delete in batches of 1000
	for i := 0; i < len(keys); i += 1000 {
		end := i + 1000
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		objectIDs := make([]types.ObjectIdentifier, len(batch))
		for j, key := range batch {
			objectIDs[j] = types.ObjectIdentifier{Key: aws.String(key)}
		}

		_, err := b.s3Client.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(b.config.Buckets.Backup),
			Delete: &types.Delete{
				Objects: objectIDs,
				Quiet:   aws.Bool(true),
			},
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// sendNotification sends a backup notification
func (b *BackupService) sendNotification(subject, message string) {
	// Implementation depends on notification service (SNS, email, etc.)
	b.logger.Info("Backup notification", "subject", subject, "message", message)
}

// GetLastBackup returns the last backup info
func (b *BackupService) GetLastBackup() *BackupInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastBackup
}

// ListBackups lists all backups
func (b *BackupService) ListBackups(ctx context.Context) ([]*BackupInfo, error) {
	result, err := b.s3Client.List(ctx, &ListInput{
		Prefix:    "backups/",
		Delimiter: "/",
	})
	if err != nil {
		return nil, err
	}

	var backups []*BackupInfo
	for _, prefix := range result.CommonPrefixes {
		infoKey := prefix + "info.json"
		data, err := b.s3Client.DownloadBytes(ctx, infoKey)
		if err != nil {
			continue
		}

		var info BackupInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}

		backups = append(backups, &info)
	}

	// Sort by date descending
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].StartedAt.After(backups[j].StartedAt)
	})

	return backups, nil
}

// RestoreBackup restores objects from a backup
func (b *BackupService) RestoreBackup(ctx context.Context, backupID string, options *RestoreOptions) error {
	// Load manifest
	manifestKey := fmt.Sprintf("backups/%s/manifest.json", backupID)
	data, err := b.s3Client.DownloadBytes(ctx, manifestKey)
	if err != nil {
		return fmt.Errorf("failed to load backup manifest: %w", err)
	}

	var manifest BackupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	b.logger.Info("Restoring backup", "id", backupID, "objects", len(manifest.Objects))

	// Restore objects
	for _, obj := range manifest.Objects {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if object already exists and options
		if options != nil && !options.Overwrite {
			exists, _ := b.s3Client.Exists(ctx, obj.Key)
			if exists {
				continue
			}
		}

		if err := b.restoreObject(ctx, obj, options); err != nil {
			b.logger.Warn("Failed to restore object", "key", obj.Key, "error", err)
			if options == nil || !options.ContinueOnError {
				return fmt.Errorf("failed to restore %s: %w", obj.Key, err)
			}
		}
	}

	b.logger.Info("Backup restored successfully", "id", backupID)

	return nil
}

// RestoreOptions contains options for restore operations
type RestoreOptions struct {
	Overwrite       bool
	ContinueOnError bool
	TargetPrefix    string
}

// restoreObject restores a single object from backup
func (b *BackupService) restoreObject(ctx context.Context, obj BackupObjectInfo, options *RestoreOptions) error {
	backupKey := obj.BackupKey
	destKey := obj.Key

	if options != nil && options.TargetPrefix != "" {
		destKey = options.TargetPrefix + "/" + obj.Key
	}

	// Check if compressed
	if b.config.Backup.EnableCompression {
		backupKey += ".gz"
		return b.restoreCompressed(ctx, backupKey, destKey)
	}

	// Direct copy
	_, err := b.s3Client.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(b.config.Buckets.Primary),
		CopySource: aws.String(fmt.Sprintf("%s/%s", b.config.Buckets.Backup, backupKey)),
		Key:        aws.String(destKey),
	})

	return err
}

// restoreCompressed restores a compressed object
func (b *BackupService) restoreCompressed(ctx context.Context, backupKey, destKey string) error {
	// Download compressed data
	result, err := b.s3Client.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.config.Buckets.Backup),
		Key:    aws.String(backupKey),
	})
	if err != nil {
		return err
	}
	defer result.Body.Close()

	// Decompress
	gzReader, err := gzip.NewReader(result.Body)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	data, err := io.ReadAll(gzReader)
	if err != nil {
		return err
	}

	// Upload decompressed data
	_, err = b.s3Client.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.config.Buckets.Primary),
		Key:    aws.String(destKey),
		Body:   bytes.NewReader(data),
	})

	return err
}

// generateBackupID generates a unique backup ID
func generateBackupID() string {
	return time.Now().UTC().Format("20060102-150405")
}

// =============================================================================
// Retention Policy Service
// =============================================================================

// RetentionService manages document retention policies
type RetentionService struct {
	s3Client  *S3Client
	metadata  *MetadataManager
	config    *Config
	scheduler *cron.Cron
	logger    Logger
	mu        sync.Mutex
	isRunning bool
}

// NewRetentionService creates a new retention service
func NewRetentionService(s3Client *S3Client, metadata *MetadataManager, logger Logger) *RetentionService {
	if logger == nil {
		logger = &DefaultLogger{}
	}

	return &RetentionService{
		s3Client:  s3Client,
		metadata:  metadata,
		config:    s3Client.config,
		scheduler: cron.New(),
		logger:    logger,
	}
}

// Start starts the retention policy enforcement
func (r *RetentionService) Start() error {
	if !r.config.Retention.Enabled {
		r.logger.Info("Retention service disabled")
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRunning {
		return nil
	}

	// Run daily at 3 AM
	_, err := r.scheduler.AddFunc("0 3 * * *", func() {
		ctx := context.Background()
		if err := r.EnforceRetention(ctx); err != nil {
			r.logger.Error("Retention enforcement failed", "error", err)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to schedule retention: %w", err)
	}

	r.scheduler.Start()
	r.isRunning = true
	r.logger.Info("Retention service started")

	return nil
}

// Stop stops the retention service
func (r *RetentionService) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRunning {
		return
	}

	r.scheduler.Stop()
	r.isRunning = false
	r.logger.Info("Retention service stopped")
}

// EnforceRetention enforces retention policies
func (r *RetentionService) EnforceRetention(ctx context.Context) error {
	r.logger.Info("Starting retention enforcement")

	// Get all metadata
	result, err := r.metadata.GetStore().List(ctx, &MetadataFilter{
		IncludeArchived: true,
		Limit:           10000,
	})
	if err != nil {
		return fmt.Errorf("failed to list metadata: %w", err)
	}

	now := time.Now().UTC()
	var archived, deleted int

	for _, meta := range result.Items {
		// Skip if legal hold
		if meta.LegalHold {
			continue
		}

		// Skip exempt types
		if r.isExemptType(meta.DocumentType) {
			continue
		}

		// Check minimum retention
		if now.Sub(meta.CreatedAt) < r.config.Retention.MinRetention {
			continue
		}

		// Check if should be archived
		if meta.Status == "active" && now.Sub(meta.CreatedAt) >= r.config.Retention.ArchiveAfter {
			if err := r.archiveDocument(ctx, meta); err != nil {
				r.logger.Warn("Failed to archive document", "id", meta.ID, "error", err)
			} else {
				archived++
			}
			continue
		}

		// Check if should be deleted
		if meta.Status == "archived" && meta.ArchivedAt != nil {
			if now.Sub(*meta.ArchivedAt) >= r.config.Retention.DeleteAfterArchive {
				if err := r.deleteDocument(ctx, meta); err != nil {
					r.logger.Warn("Failed to delete document", "id", meta.ID, "error", err)
				} else {
					deleted++
				}
			}
		}

		// Check expiration
		if meta.ExpiresAt != nil && now.After(*meta.ExpiresAt) {
			if err := r.deleteDocument(ctx, meta); err != nil {
				r.logger.Warn("Failed to delete expired document", "id", meta.ID, "error", err)
			} else {
				deleted++
			}
		}
	}

	r.logger.Info("Retention enforcement complete", "archived", archived, "deleted", deleted)

	return nil
}

// archiveDocument archives a document
func (r *RetentionService) archiveDocument(ctx context.Context, meta *DocumentMetadata) error {
	// Change storage class to Glacier
	if r.config.Retention.GlacierTransition > 0 {
		_, err := r.s3Client.client.CopyObject(ctx, &s3.CopyObjectInput{
			Bucket:            aws.String(r.config.Buckets.Primary),
			CopySource:        aws.String(fmt.Sprintf("%s/%s", r.config.Buckets.Primary, meta.Key)),
			Key:               aws.String(meta.Key),
			StorageClass:      types.StorageClassGlacier,
			MetadataDirective: types.MetadataDirectiveCopy,
		})
		if err != nil {
			return err
		}
	}

	// Optionally move to archive bucket
	if r.config.Buckets.Archive != "" {
		_, err := r.s3Client.client.CopyObject(ctx, &s3.CopyObjectInput{
			Bucket:     aws.String(r.config.Buckets.Archive),
			CopySource: aws.String(fmt.Sprintf("%s/%s", r.config.Buckets.Primary, meta.Key)),
			Key:        aws.String(meta.Key),
		})
		if err != nil {
			return err
		}

		// Delete from primary
		if err := r.s3Client.Delete(ctx, meta.Key); err != nil {
			r.logger.Warn("Failed to delete from primary after archive", "key", meta.Key)
		}
	}

	// Update metadata
	now := time.Now().UTC()
	meta.Status = "archived"
	meta.ArchivedAt = &now
	meta.StorageClass = string(types.StorageClassGlacier)

	return r.metadata.GetStore().Update(ctx, meta)
}

// deleteDocument permanently deletes a document
func (r *RetentionService) deleteDocument(ctx context.Context, meta *DocumentMetadata) error {
	// Delete from storage
	if err := r.s3Client.Delete(ctx, meta.Key); err != nil {
		// Ignore if already deleted
		if !strings.Contains(err.Error(), "NotFound") {
			return err
		}
	}

	// Delete from archive bucket if exists
	if r.config.Buckets.Archive != "" {
		r.s3Client.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(r.config.Buckets.Archive),
			Key:    aws.String(meta.Key),
		})
	}

	// Delete metadata
	return r.metadata.GetStore().Delete(ctx, meta.ID)
}

// isExemptType checks if a document type is exempt from deletion
func (r *RetentionService) isExemptType(docType DocumentType) bool {
	for _, exempt := range r.config.Retention.ExemptTypes {
		if string(docType) == exempt {
			return true
		}
	}
	return false
}

// GetRetentionPolicy gets the retention period for a document type
func (r *RetentionService) GetRetentionPolicy(docType DocumentType) time.Duration {
	if retention, ok := r.config.Retention.DocumentRetention[string(docType)]; ok {
		return retention
	}
	return r.config.Retention.DefaultRetention
}

// SetRetentionPolicy sets a custom retention policy for a document
func (r *RetentionService) SetRetentionPolicy(ctx context.Context, id string, retention time.Duration) error {
	meta, err := r.metadata.GetStore().Get(ctx, id)
	if err != nil {
		return err
	}

	if retention < r.config.Retention.MinRetention {
		return fmt.Errorf("%w: retention period cannot be less than %v",
			ErrRetentionViolation, r.config.Retention.MinRetention)
	}

	expiresAt := meta.CreatedAt.Add(retention)
	meta.ExpiresAt = &expiresAt

	return r.metadata.GetStore().Update(ctx, meta)
}

// ExtendRetention extends the retention period for a document
func (r *RetentionService) ExtendRetention(ctx context.Context, id string, extension time.Duration) error {
	meta, err := r.metadata.GetStore().Get(ctx, id)
	if err != nil {
		return err
	}

	var currentExpiry time.Time
	if meta.ExpiresAt != nil {
		currentExpiry = *meta.ExpiresAt
	} else {
		currentExpiry = meta.CreatedAt.Add(r.GetRetentionPolicy(meta.DocumentType))
	}

	newExpiry := currentExpiry.Add(extension)
	meta.ExpiresAt = &newExpiry

	return r.metadata.GetStore().Update(ctx, meta)
}

// =============================================================================
// Lifecycle Rules
// =============================================================================

// SetupLifecycleRules configures S3 lifecycle rules
func (r *RetentionService) SetupLifecycleRules(ctx context.Context) error {
	if !r.config.Retention.EnableLifecycle {
		return nil
	}

	var rules []types.LifecycleRule

	// Transition to Glacier
	if r.config.Retention.GlacierTransition > 0 {
		days := int32(r.config.Retention.GlacierTransition / (24 * time.Hour))
		rules = append(rules, types.LifecycleRule{
			ID:     aws.String("TransitionToGlacier"),
			Status: types.ExpirationStatusEnabled,
			Filter: &types.LifecycleRuleFilter{Prefix: aws.String("")},
			Transitions: []types.Transition{
				{
					Days:         aws.Int32(days),
					StorageClass: types.TransitionStorageClassGlacier,
				},
			},
		})
	}

	// Transition to Deep Archive
	if r.config.Retention.DeepArchiveTransition > 0 {
		days := int32(r.config.Retention.DeepArchiveTransition / (24 * time.Hour))
		rules = append(rules, types.LifecycleRule{
			ID:     aws.String("TransitionToDeepArchive"),
			Status: types.ExpirationStatusEnabled,
			Filter: &types.LifecycleRuleFilter{Prefix: aws.String("")},
			Transitions: []types.Transition{
				{
					Days:         aws.Int32(days),
					StorageClass: types.TransitionStorageClassDeepArchive,
				},
			},
		})
	}

	// Expire temporary files
	rules = append(rules, types.LifecycleRule{
		ID:     aws.String("ExpireTemporary"),
		Status: types.ExpirationStatusEnabled,
		Filter: &types.LifecycleRuleFilter{Prefix: aws.String("temporary/")},
		Expiration: &types.LifecycleExpiration{
			Days: aws.Int32(30),
		},
	})

	// Clean up incomplete multipart uploads
	rules = append(rules, types.LifecycleRule{
		ID:     aws.String("AbortIncompleteMultipart"),
		Status: types.ExpirationStatusEnabled,
		Filter: &types.LifecycleRuleFilter{Prefix: aws.String("")},
		AbortIncompleteMultipartUpload: &types.AbortIncompleteMultipartUpload{
			DaysAfterInitiation: aws.Int32(7),
		},
	})

	if len(rules) == 0 {
		return nil
	}

	_, err := r.s3Client.client.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(r.config.Buckets.Primary),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: rules,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to configure lifecycle rules: %w", err)
	}

	r.logger.Info("Lifecycle rules configured", "rules", len(rules))

	return nil
}
