package pdf

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appconfig "github.com/javaknight1/servicepro/backend/config"
)

// =============================================================================
// Errors
// =============================================================================

var (
	ErrInvalidConfig       = errors.New("invalid configuration")
	ErrMissingTemplate     = errors.New("template not found")
	ErrFontNotFound        = errors.New("font not found")
	ErrS3UploadFailed      = errors.New("failed to upload to S3")
	ErrPDFGenerationFailed = errors.New("PDF generation failed")
	ErrInvalidContent      = errors.New("invalid content provided")
	ErrTemplateRender      = errors.New("template rendering failed")
)

// =============================================================================
// Configuration Types
// =============================================================================

// Config contains all configuration for the PDF service
type Config struct {
	// Storage
	Storage StorageConfig `json:"storage"`

	// Fonts
	Fonts FontConfig `json:"fonts"`

	// Branding
	Branding BrandingConfig `json:"branding"`

	// Page defaults
	PageDefaults PageConfig `json:"page_defaults"`

	// Performance
	Performance PerformanceConfig `json:"performance"`

	// Paths
	TempDir     string `json:"temp_dir"`
	FontDir     string `json:"font_dir"`
	TemplateDir string `json:"template_dir"`
	AssetDir    string `json:"asset_dir"`
}

// StorageConfig configures where PDFs are stored
type StorageConfig struct {
	// Type can be "local", "s3", or "both"
	Type string `json:"type"`

	// Local storage settings
	LocalPath string `json:"local_path"`

	// S3 settings
	S3Bucket    string `json:"s3_bucket"`
	S3Region    string `json:"s3_region"`
	S3Prefix    string `json:"s3_prefix"`
	S3Endpoint  string `json:"s3_endpoint"` // For S3-compatible services
	S3PublicURL string `json:"s3_public_url"`

	// Presigned URL expiration
	PresignExpiration time.Duration `json:"presign_expiration"`
}

// FontConfig contains font settings
type FontConfig struct {
	// Default font family
	DefaultFamily string `json:"default_family"`

	// Font sizes (in points)
	SizeTitle      float64 `json:"size_title"`
	SizeHeading    float64 `json:"size_heading"`
	SizeSubheading float64 `json:"size_subheading"`
	SizeBody       float64 `json:"size_body"`
	SizeSmall      float64 `json:"size_small"`
	SizeFooter     float64 `json:"size_footer"`

	// Custom fonts (path -> font name mapping)
	CustomFonts map[string]CustomFont `json:"custom_fonts"`
}

// CustomFont defines a custom font to be loaded
type CustomFont struct {
	Family  string `json:"family"`
	Style   string `json:"style"` // "", "B", "I", "BI"
	Path    string `json:"path"`
	Unicode bool   `json:"unicode"`
}

// BrandingConfig contains company branding settings
type BrandingConfig struct {
	// Company information
	CompanyName    string `json:"company_name"`
	CompanyTagline string `json:"company_tagline"`
	CompanyAddress string `json:"company_address"`
	CompanyPhone   string `json:"company_phone"`
	CompanyEmail   string `json:"company_email"`
	CompanyWebsite string `json:"company_website"`

	// Logo
	LogoPath   string  `json:"logo_path"`
	LogoWidth  float64 `json:"logo_width"`
	LogoHeight float64 `json:"logo_height"`

	// Colors (hex format)
	PrimaryColor    string `json:"primary_color"`
	SecondaryColor  string `json:"secondary_color"`
	AccentColor     string `json:"accent_color"`
	TextColor       string `json:"text_color"`
	MutedColor      string `json:"muted_color"`
	BorderColor     string `json:"border_color"`
	BackgroundColor string `json:"background_color"`

	// Footer
	FooterText string `json:"footer_text"`
}

// PageConfig contains default page settings
type PageConfig struct {
	// Page size: "A4", "Letter", "Legal"
	Size string `json:"size"`

	// Orientation: "P" (portrait) or "L" (landscape)
	Orientation string `json:"orientation"`

	// Units: "mm", "cm", "in", "pt"
	Unit string `json:"unit"`

	// Margins
	MarginTop    float64 `json:"margin_top"`
	MarginBottom float64 `json:"margin_bottom"`
	MarginLeft   float64 `json:"margin_left"`
	MarginRight  float64 `json:"margin_right"`

	// Header/Footer heights
	HeaderHeight float64 `json:"header_height"`
	FooterHeight float64 `json:"footer_height"`
}

// PerformanceConfig contains performance-related settings
type PerformanceConfig struct {
	// Max concurrent PDF generations
	MaxConcurrent int `json:"max_concurrent"`

	// Timeout for PDF generation
	GenerationTimeout time.Duration `json:"generation_timeout"`

	// Enable compression
	EnableCompression bool `json:"enable_compression"`

	// Image quality (1-100)
	ImageQuality int `json:"image_quality"`

	// Cache templates
	CacheTemplates bool `json:"cache_templates"`

	// Worker pool size
	WorkerPoolSize int `json:"worker_pool_size"`
}

// =============================================================================
// Default Configuration
// =============================================================================

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Storage: StorageConfig{
			Type:              "local",
			LocalPath:         "./generated_pdfs",
			S3Prefix:          "pdfs/",
			PresignExpiration: 24 * time.Hour,
		},
		Fonts: FontConfig{
			DefaultFamily:  "Helvetica",
			SizeTitle:      24,
			SizeHeading:    18,
			SizeSubheading: 14,
			SizeBody:       10,
			SizeSmall:      8,
			SizeFooter:     8,
			CustomFonts:    make(map[string]CustomFont),
		},
		Branding: BrandingConfig{
			CompanyName:     "ServicePro",
			PrimaryColor:    "#2563EB",
			SecondaryColor:  "#1E40AF",
			AccentColor:     "#3B82F6",
			TextColor:       "#1F2937",
			MutedColor:      "#6B7280",
			BorderColor:     "#E5E7EB",
			BackgroundColor: "#F9FAFB",
		},
		PageDefaults: PageConfig{
			Size:         "Letter",
			Orientation:  "P",
			Unit:         "mm",
			MarginTop:    20,
			MarginBottom: 20,
			MarginLeft:   20,
			MarginRight:  20,
			HeaderHeight: 30,
			FooterHeight: 20,
		},
		Performance: PerformanceConfig{
			MaxConcurrent:     10,
			GenerationTimeout: 30 * time.Second,
			EnableCompression: true,
			ImageQuality:      85,
			CacheTemplates:    true,
			WorkerPoolSize:    5,
		},
		TempDir:     os.TempDir(),
		FontDir:     "./fonts",
		TemplateDir: "./templates/pdf",
		AssetDir:    "./assets",
	}
}

// NewConfigFromAppConfig creates a PDF Config from the main application config.
// It applies values from the app config while using defaults for everything else.
func NewConfigFromAppConfig(cfg *appconfig.Config) *Config {
	pdfCfg := DefaultConfig()

	if cfg == nil {
		return pdfCfg
	}

	// Apply storage settings from app config
	if cfg.PDF.StorageType != "" {
		pdfCfg.Storage.Type = cfg.PDF.StorageType
	}
	if cfg.PDF.LocalPath != "" {
		pdfCfg.Storage.LocalPath = cfg.PDF.LocalPath
	}
	if cfg.PDF.S3Bucket != "" {
		pdfCfg.Storage.S3Bucket = cfg.PDF.S3Bucket
	}
	if cfg.PDF.S3Prefix != "" {
		pdfCfg.Storage.S3Prefix = cfg.PDF.S3Prefix
	}
	if cfg.PDF.S3PublicURL != "" {
		pdfCfg.Storage.S3PublicURL = cfg.PDF.S3PublicURL
	}

	// Apply region for S3-compatible storage
	if cfg.AWS.Region != "" {
		pdfCfg.Storage.S3Region = cfg.AWS.Region
	}

	// Apply branding settings from app config
	if cfg.PDF.CompanyName != "" {
		pdfCfg.Branding.CompanyName = cfg.PDF.CompanyName
	}
	if cfg.PDF.CompanyLogo != "" {
		pdfCfg.Branding.LogoPath = cfg.PDF.CompanyLogo
	}

	return pdfCfg
}

// =============================================================================
// Configuration Validation
// =============================================================================

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Storage.Type != "local" && c.Storage.Type != "s3" && c.Storage.Type != "both" {
		return fmt.Errorf("%w: invalid storage type: %s", ErrInvalidConfig, c.Storage.Type)
	}

	if c.Storage.Type == "local" || c.Storage.Type == "both" {
		if c.Storage.LocalPath == "" {
			return fmt.Errorf("%w: local path required for local storage", ErrInvalidConfig)
		}
	}

	if c.Storage.Type == "s3" || c.Storage.Type == "both" {
		if c.Storage.S3Bucket == "" {
			return fmt.Errorf("%w: S3 bucket required for S3 storage", ErrInvalidConfig)
		}
	}

	if c.Fonts.DefaultFamily == "" {
		return fmt.Errorf("%w: default font family required", ErrInvalidConfig)
	}

	if c.PageDefaults.Size == "" {
		return fmt.Errorf("%w: page size required", ErrInvalidConfig)
	}

	if c.Performance.MaxConcurrent < 1 {
		c.Performance.MaxConcurrent = 1
	}

	if c.Performance.ImageQuality < 1 || c.Performance.ImageQuality > 100 {
		c.Performance.ImageQuality = 85
	}

	return nil
}

// =============================================================================
// S3-Compatible Storage Client
// =============================================================================

// S3Client wraps the S3-compatible storage client
type S3Client struct {
	client     *s3.Client
	bucket     string
	prefix     string
	publicURL  string
	presignExp time.Duration
}

// NewS3Client creates a new S3 client
func NewS3Client(ctx context.Context, cfg *StorageConfig) (*S3Client, error) {
	var opts []func(*awsconfig.LoadOptions) error

	if cfg.S3Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.S3Region))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 SDK config: %w", err)
	}

	clientOpts := []func(*s3.Options){}

	// Support for S3-compatible services (MinIO, DigitalOcean Spaces, etc.)
	if cfg.S3Endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, clientOpts...)

	return &S3Client{
		client:     client,
		bucket:     cfg.S3Bucket,
		prefix:     cfg.S3Prefix,
		publicURL:  cfg.S3PublicURL,
		presignExp: cfg.PresignExpiration,
	}, nil
}

// GetClient returns the underlying S3 client
func (c *S3Client) GetClient() *s3.Client {
	return c.client
}

// GetBucket returns the bucket name
func (c *S3Client) GetBucket() string {
	return c.bucket
}

// GetKey returns the full S3 key for a filename
func (c *S3Client) GetKey(filename string) string {
	return filepath.Join(c.prefix, filename)
}

// GetPublicURL returns the public URL for a file
// Note: The fallback URL format assumes AWS S3; configure S3PublicURL for other providers
func (c *S3Client) GetPublicURL(filename string) string {
	if c.publicURL != "" {
		return fmt.Sprintf("%s/%s%s", c.publicURL, c.prefix, filename)
	}
	// Default fallback for AWS S3 - configure S3PublicURL for other S3-compatible providers
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s%s", c.bucket, c.prefix, filename)
}

// =============================================================================
// Logger Interface
// =============================================================================

// Logger interface for logging
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// DefaultLogger implements a simple logger
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
