package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =============================================================================
// Errors
// =============================================================================

var (
	ErrPlanNotFound        = errors.New("plan not found")
	ErrPlanInactive        = errors.New("plan is not active")
	ErrInvalidPlanType     = errors.New("invalid plan type")
	ErrPlanAlreadyExists   = errors.New("plan already exists")
	ErrCannotDeletePlan    = errors.New("cannot delete plan with active subscriptions")
	ErrInvalidBillingCycle = errors.New("invalid billing cycle")
)

// =============================================================================
// Plan Types and Constants
// =============================================================================

// BillingCycle represents the billing frequency
type BillingCycle string

const (
	BillingCycleMonthly   BillingCycle = "monthly"
	BillingCycleQuarterly BillingCycle = "quarterly"
	BillingCycleAnnual    BillingCycle = "annual"
)

// PlanTier represents the tier level of a plan
type PlanTier string

const (
	PlanTierFree         PlanTier = "free"
	PlanTierStarter      PlanTier = "starter"
	PlanTierProfessional PlanTier = "professional"
	PlanTierBusiness     PlanTier = "business"
	PlanTierEnterprise   PlanTier = "enterprise"
)

// FeatureType represents types of features
type FeatureType string

const (
	FeatureTypeBoolean FeatureType = "boolean"
	FeatureTypeNumeric FeatureType = "numeric"
	FeatureTypeText    FeatureType = "text"
)

// =============================================================================
// Plan Models
// =============================================================================

// Plan represents a subscription plan
type Plan struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Tier        PlanTier  `gorm:"type:varchar(50);not null;index" json:"tier"`

	// Pricing
	MonthlyPrice   int64  `gorm:"not null" json:"monthly_price"` // in cents
	AnnualPrice    int64  `gorm:"not null" json:"annual_price"`  // in cents
	QuarterlyPrice int64  `json:"quarterly_price"`               // in cents
	Currency       string `gorm:"type:varchar(3);default:'USD'" json:"currency"`

	// Stripe integration
	StripePriceIDMonthly   string `gorm:"type:varchar(255)" json:"stripe_price_id_monthly,omitempty"`
	StripePriceIDAnnual    string `gorm:"type:varchar(255)" json:"stripe_price_id_annual,omitempty"`
	StripePriceIDQuarterly string `gorm:"type:varchar(255)" json:"stripe_price_id_quarterly,omitempty"`
	StripeProductID        string `gorm:"type:varchar(255)" json:"stripe_product_id,omitempty"`

	// Trial
	TrialDays int `gorm:"default:0" json:"trial_days"`

	// Status
	IsActive  bool `gorm:"default:true;index" json:"is_active"`
	IsPublic  bool `gorm:"default:true" json:"is_public"` // visible to customers
	SortOrder int  `gorm:"default:0" json:"sort_order"`

	// Features
	Features []PlanFeature `gorm:"foreignKey:PlanID" json:"features,omitempty"`

	// Metadata
	Metadata JSONB `gorm:"type:jsonb" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for Plan
func (Plan) TableName() string {
	return "subscription_plans"
}

// BeforeCreate sets the UUID before creating a new record
func (p *Plan) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// PlanFeature represents a feature included in a plan
type PlanFeature struct {
	ID           uuid.UUID   `gorm:"type:uuid;primary_key" json:"id"`
	PlanID       uuid.UUID   `gorm:"type:uuid;not null;index" json:"plan_id"`
	FeatureKey   string      `gorm:"type:varchar(100);not null" json:"feature_key"`
	FeatureName  string      `gorm:"type:varchar(200);not null" json:"feature_name"`
	FeatureType  FeatureType `gorm:"type:varchar(50);not null" json:"feature_type"`
	Value        string      `gorm:"type:varchar(500)" json:"value"`
	NumericValue *int64      `json:"numeric_value,omitempty"`
	IsEnabled    bool        `gorm:"default:true" json:"is_enabled"`
	Description  string      `gorm:"type:text" json:"description,omitempty"`
	SortOrder    int         `gorm:"default:0" json:"sort_order"`
	CreatedAt    time.Time   `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for PlanFeature
func (PlanFeature) TableName() string {
	return "plan_features"
}

// BeforeCreate sets the UUID before creating a new record
func (pf *PlanFeature) BeforeCreate(tx *gorm.DB) error {
	if pf.ID == uuid.Nil {
		pf.ID = uuid.New()
	}
	return nil
}

// JSONB is a custom type for JSONB columns
type JSONB []byte

// =============================================================================
// Plan Service
// =============================================================================

// PlanServiceConfig contains configuration for the plan service
type PlanServiceConfig struct {
	CacheTTL        time.Duration
	EnableCache     bool
	DefaultCurrency string
}

// DefaultPlanServiceConfig returns default configuration
func DefaultPlanServiceConfig() *PlanServiceConfig {
	return &PlanServiceConfig{
		CacheTTL:        10 * time.Minute,
		EnableCache:     true,
		DefaultCurrency: "USD",
	}
}

// PlanService handles plan operations
type PlanService struct {
	config *PlanServiceConfig
	db     *gorm.DB
	logger Logger

	// Cache
	planCache     map[uuid.UUID]*Plan
	allPlansCache []*Plan
	cacheExpiry   time.Time
	cacheMu       sync.RWMutex
}

// Logger interface for logging
type Logger interface {
	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
}

// NewPlanService creates a new plan service
func NewPlanService(config *PlanServiceConfig, db *gorm.DB, logger Logger) *PlanService {
	if config == nil {
		config = DefaultPlanServiceConfig()
	}

	return &PlanService{
		config:    config,
		db:        db,
		logger:    logger,
		planCache: make(map[uuid.UUID]*Plan),
	}
}

// =============================================================================
// Plan CRUD Operations
// =============================================================================

// CreatePlan creates a new subscription plan
func (s *PlanService) CreatePlan(ctx context.Context, plan *Plan) error {
	if plan.Name == "" {
		return errors.New("plan name is required")
	}

	// Check for duplicate names
	var existing Plan
	if err := s.db.WithContext(ctx).Where("name = ?", plan.Name).First(&existing).Error; err == nil {
		return ErrPlanAlreadyExists
	}

	// Set defaults
	if plan.Currency == "" {
		plan.Currency = s.config.DefaultCurrency
	}

	// Calculate quarterly price if not set
	if plan.QuarterlyPrice == 0 && plan.MonthlyPrice > 0 {
		plan.QuarterlyPrice = plan.MonthlyPrice * 3 * 90 / 100 // 10% discount
	}

	if err := s.db.WithContext(ctx).Create(plan).Error; err != nil {
		return fmt.Errorf("failed to create plan: %w", err)
	}

	s.invalidateCache()

	return nil
}

// GetPlan retrieves a plan by ID
func (s *PlanService) GetPlan(ctx context.Context, id uuid.UUID) (*Plan, error) {
	// Check cache first
	if s.config.EnableCache {
		if plan := s.getFromCache(id); plan != nil {
			return plan, nil
		}
	}

	var plan Plan
	if err := s.db.WithContext(ctx).
		Preload("Features", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Where("id = ?", id).
		First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlanNotFound
		}
		return nil, err
	}

	// Update cache
	if s.config.EnableCache {
		s.setCache(id, &plan)
	}

	return &plan, nil
}

// GetPlanByTier retrieves a plan by tier
func (s *PlanService) GetPlanByTier(ctx context.Context, tier PlanTier) (*Plan, error) {
	var plan Plan
	if err := s.db.WithContext(ctx).
		Preload("Features", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Where("tier = ? AND is_active = ?", tier, true).
		First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlanNotFound
		}
		return nil, err
	}

	return &plan, nil
}

// ListPlans lists all active plans
func (s *PlanService) ListPlans(ctx context.Context, includePrivate bool) ([]*Plan, error) {
	// Check cache
	if s.config.EnableCache && !includePrivate {
		if plans := s.getAllFromCache(); plans != nil {
			return plans, nil
		}
	}

	var plans []*Plan
	query := s.db.WithContext(ctx).
		Preload("Features", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Where("is_active = ?", true).
		Order("sort_order ASC, monthly_price ASC")

	if !includePrivate {
		query = query.Where("is_public = ?", true)
	}

	if err := query.Find(&plans).Error; err != nil {
		return nil, err
	}

	// Update cache
	if s.config.EnableCache && !includePrivate {
		s.setAllCache(plans)
	}

	return plans, nil
}

// UpdatePlan updates a plan
func (s *PlanService) UpdatePlan(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (*Plan, error) {
	plan, err := s.GetPlan(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Model(plan).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}

	s.invalidateCache()

	return s.GetPlan(ctx, id)
}

// DeletePlan soft deletes a plan
func (s *PlanService) DeletePlan(ctx context.Context, id uuid.UUID) error {
	// Check for active subscriptions
	var count int64
	if err := s.db.WithContext(ctx).
		Table("subscriptions").
		Where("plan_id = ? AND status IN ?", id, []string{"active", "trialing"}).
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return ErrCannotDeletePlan
	}

	if err := s.db.WithContext(ctx).Delete(&Plan{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete plan: %w", err)
	}

	s.invalidateCache()

	return nil
}

// =============================================================================
// Feature Operations
// =============================================================================

// AddFeature adds a feature to a plan
func (s *PlanService) AddFeature(ctx context.Context, planID uuid.UUID, feature *PlanFeature) error {
	// Verify plan exists
	if _, err := s.GetPlan(ctx, planID); err != nil {
		return err
	}

	feature.PlanID = planID

	if err := s.db.WithContext(ctx).Create(feature).Error; err != nil {
		return fmt.Errorf("failed to add feature: %w", err)
	}

	s.invalidateCache()

	return nil
}

// RemoveFeature removes a feature from a plan
func (s *PlanService) RemoveFeature(ctx context.Context, featureID uuid.UUID) error {
	if err := s.db.WithContext(ctx).Delete(&PlanFeature{}, featureID).Error; err != nil {
		return fmt.Errorf("failed to remove feature: %w", err)
	}

	s.invalidateCache()

	return nil
}

// GetPlanFeatures retrieves all features for a plan
func (s *PlanService) GetPlanFeatures(ctx context.Context, planID uuid.UUID) ([]PlanFeature, error) {
	var features []PlanFeature
	if err := s.db.WithContext(ctx).
		Where("plan_id = ?", planID).
		Order("sort_order ASC").
		Find(&features).Error; err != nil {
		return nil, err
	}

	return features, nil
}

// HasFeature checks if a plan has a specific feature enabled
func (s *PlanService) HasFeature(ctx context.Context, planID uuid.UUID, featureKey string) (bool, error) {
	var feature PlanFeature
	if err := s.db.WithContext(ctx).
		Where("plan_id = ? AND feature_key = ? AND is_enabled = ?", planID, featureKey, true).
		First(&feature).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetFeatureLimit gets the numeric limit for a feature
func (s *PlanService) GetFeatureLimit(ctx context.Context, planID uuid.UUID, featureKey string) (int64, error) {
	var feature PlanFeature
	if err := s.db.WithContext(ctx).
		Where("plan_id = ? AND feature_key = ?", planID, featureKey).
		First(&feature).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}

	if feature.NumericValue != nil {
		return *feature.NumericValue, nil
	}

	return 0, nil
}

// =============================================================================
// Pricing Calculations
// =============================================================================

// GetPriceForCycle returns the price for a specific billing cycle
func (s *PlanService) GetPriceForCycle(plan *Plan, cycle BillingCycle) (int64, error) {
	switch cycle {
	case BillingCycleMonthly:
		return plan.MonthlyPrice, nil
	case BillingCycleQuarterly:
		return plan.QuarterlyPrice, nil
	case BillingCycleAnnual:
		return plan.AnnualPrice, nil
	default:
		return 0, ErrInvalidBillingCycle
	}
}

// GetStripePriceID returns the Stripe price ID for a specific billing cycle
func (s *PlanService) GetStripePriceID(plan *Plan, cycle BillingCycle) (string, error) {
	switch cycle {
	case BillingCycleMonthly:
		return plan.StripePriceIDMonthly, nil
	case BillingCycleQuarterly:
		return plan.StripePriceIDQuarterly, nil
	case BillingCycleAnnual:
		return plan.StripePriceIDAnnual, nil
	default:
		return "", ErrInvalidBillingCycle
	}
}

// CalculateSavings calculates savings for annual billing vs monthly
func (s *PlanService) CalculateSavings(plan *Plan) map[string]int64 {
	monthlyAnnualized := plan.MonthlyPrice * 12
	annualSavings := monthlyAnnualized - plan.AnnualPrice
	annualSavingsPercent := int64(0)
	if monthlyAnnualized > 0 {
		annualSavingsPercent = (annualSavings * 100) / monthlyAnnualized
	}

	monthlyQuarterly := plan.MonthlyPrice * 3
	quarterlySavings := monthlyQuarterly - plan.QuarterlyPrice
	quarterlySavingsPercent := int64(0)
	if monthlyQuarterly > 0 {
		quarterlySavingsPercent = (quarterlySavings * 100) / monthlyQuarterly
	}

	return map[string]int64{
		"annual_savings":            annualSavings,
		"annual_savings_percent":    annualSavingsPercent,
		"quarterly_savings":         quarterlySavings,
		"quarterly_savings_percent": quarterlySavingsPercent,
	}
}

// =============================================================================
// Plan Comparison
// =============================================================================

// ComparePlans compares features between two plans
func (s *PlanService) ComparePlans(ctx context.Context, planID1, planID2 uuid.UUID) (*PlanComparison, error) {
	plan1, err := s.GetPlan(ctx, planID1)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan 1: %w", err)
	}

	plan2, err := s.GetPlan(ctx, planID2)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan 2: %w", err)
	}

	comparison := &PlanComparison{
		Plan1:              plan1,
		Plan2:              plan2,
		FeatureComparisons: make([]FeatureComparison, 0),
	}

	// Build feature map for plan 2
	plan2Features := make(map[string]PlanFeature)
	for _, f := range plan2.Features {
		plan2Features[f.FeatureKey] = f
	}

	// Compare features
	for _, f1 := range plan1.Features {
		fc := FeatureComparison{
			FeatureKey:   f1.FeatureKey,
			FeatureName:  f1.FeatureName,
			Plan1Value:   f1.Value,
			Plan1Enabled: f1.IsEnabled,
		}

		if f2, exists := plan2Features[f1.FeatureKey]; exists {
			fc.Plan2Value = f2.Value
			fc.Plan2Enabled = f2.IsEnabled
			delete(plan2Features, f1.FeatureKey)
		}

		comparison.FeatureComparisons = append(comparison.FeatureComparisons, fc)
	}

	// Add remaining plan 2 features
	for _, f2 := range plan2Features {
		comparison.FeatureComparisons = append(comparison.FeatureComparisons, FeatureComparison{
			FeatureKey:   f2.FeatureKey,
			FeatureName:  f2.FeatureName,
			Plan2Value:   f2.Value,
			Plan2Enabled: f2.IsEnabled,
		})
	}

	return comparison, nil
}

// PlanComparison represents a comparison between two plans
type PlanComparison struct {
	Plan1              *Plan               `json:"plan1"`
	Plan2              *Plan               `json:"plan2"`
	FeatureComparisons []FeatureComparison `json:"feature_comparisons"`
}

// FeatureComparison represents a comparison of a single feature
type FeatureComparison struct {
	FeatureKey   string `json:"feature_key"`
	FeatureName  string `json:"feature_name"`
	Plan1Value   string `json:"plan1_value,omitempty"`
	Plan1Enabled bool   `json:"plan1_enabled"`
	Plan2Value   string `json:"plan2_value,omitempty"`
	Plan2Enabled bool   `json:"plan2_enabled"`
}

// =============================================================================
// Cache Management
// =============================================================================

func (s *PlanService) getFromCache(id uuid.UUID) *Plan {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	if time.Now().After(s.cacheExpiry) {
		return nil
	}

	return s.planCache[id]
}

func (s *PlanService) setCache(id uuid.UUID, plan *Plan) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.planCache[id] = plan
	s.cacheExpiry = time.Now().Add(s.config.CacheTTL)
}

func (s *PlanService) getAllFromCache() []*Plan {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	if time.Now().After(s.cacheExpiry) || s.allPlansCache == nil {
		return nil
	}

	return s.allPlansCache
}

func (s *PlanService) setAllCache(plans []*Plan) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.allPlansCache = plans
	for _, p := range plans {
		s.planCache[p.ID] = p
	}
	s.cacheExpiry = time.Now().Add(s.config.CacheTTL)
}

func (s *PlanService) invalidateCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.planCache = make(map[uuid.UUID]*Plan)
	s.allPlansCache = nil
	s.cacheExpiry = time.Time{}
}

// ClearCache clears the plan cache
func (s *PlanService) ClearCache() {
	s.invalidateCache()
}

// =============================================================================
// Default Plans
// =============================================================================

// GetDefaultPlans returns a set of default plans for initialization
func GetDefaultPlans() []*Plan {
	return []*Plan{
		{
			Name:         "Free",
			Description:  "Get started with basic features",
			Tier:         PlanTierFree,
			MonthlyPrice: 0,
			AnnualPrice:  0,
			TrialDays:    0,
			IsActive:     true,
			IsPublic:     true,
			SortOrder:    1,
			Features: []PlanFeature{
				{FeatureKey: "projects", FeatureName: "Projects", FeatureType: FeatureTypeNumeric, NumericValue: ptr(int64(3)), IsEnabled: true, SortOrder: 1},
				{FeatureKey: "team_members", FeatureName: "Team Members", FeatureType: FeatureTypeNumeric, NumericValue: ptr(int64(1)), IsEnabled: true, SortOrder: 2},
				{FeatureKey: "storage_gb", FeatureName: "Storage (GB)", FeatureType: FeatureTypeNumeric, NumericValue: ptr(int64(1)), IsEnabled: true, SortOrder: 3},
				{FeatureKey: "api_access", FeatureName: "API Access", FeatureType: FeatureTypeBoolean, IsEnabled: false, SortOrder: 4},
				{FeatureKey: "support", FeatureName: "Support", FeatureType: FeatureTypeText, Value: "Community", IsEnabled: true, SortOrder: 5},
			},
		},
		{
			Name:         "Starter",
			Description:  "Perfect for small teams and growing businesses",
			Tier:         PlanTierStarter,
			MonthlyPrice: 2900,  // $29/month
			AnnualPrice:  29000, // $290/year (save ~17%)
			TrialDays:    14,
			IsActive:     true,
			IsPublic:     true,
			SortOrder:    2,
			Features: []PlanFeature{
				{FeatureKey: "projects", FeatureName: "Projects", FeatureType: FeatureTypeNumeric, NumericValue: ptr(int64(10)), IsEnabled: true, SortOrder: 1},
				{FeatureKey: "team_members", FeatureName: "Team Members", FeatureType: FeatureTypeNumeric, NumericValue: ptr(int64(5)), IsEnabled: true, SortOrder: 2},
				{FeatureKey: "storage_gb", FeatureName: "Storage (GB)", FeatureType: FeatureTypeNumeric, NumericValue: ptr(int64(10)), IsEnabled: true, SortOrder: 3},
				{FeatureKey: "api_access", FeatureName: "API Access", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 4},
				{FeatureKey: "support", FeatureName: "Support", FeatureType: FeatureTypeText, Value: "Email", IsEnabled: true, SortOrder: 5},
			},
		},
		{
			Name:         "Professional",
			Description:  "Advanced features for professional teams",
			Tier:         PlanTierProfessional,
			MonthlyPrice: 7900,  // $79/month
			AnnualPrice:  79000, // $790/year (save ~17%)
			TrialDays:    14,
			IsActive:     true,
			IsPublic:     true,
			SortOrder:    3,
			Features: []PlanFeature{
				{FeatureKey: "projects", FeatureName: "Projects", FeatureType: FeatureTypeNumeric, NumericValue: ptr(int64(50)), IsEnabled: true, SortOrder: 1},
				{FeatureKey: "team_members", FeatureName: "Team Members", FeatureType: FeatureTypeNumeric, NumericValue: ptr(int64(20)), IsEnabled: true, SortOrder: 2},
				{FeatureKey: "storage_gb", FeatureName: "Storage (GB)", FeatureType: FeatureTypeNumeric, NumericValue: ptr(int64(100)), IsEnabled: true, SortOrder: 3},
				{FeatureKey: "api_access", FeatureName: "API Access", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 4},
				{FeatureKey: "priority_support", FeatureName: "Priority Support", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 5},
				{FeatureKey: "advanced_analytics", FeatureName: "Advanced Analytics", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 6},
				{FeatureKey: "support", FeatureName: "Support", FeatureType: FeatureTypeText, Value: "Priority Email & Chat", IsEnabled: true, SortOrder: 7},
			},
		},
		{
			Name:         "Business",
			Description:  "Full-featured solution for large organizations",
			Tier:         PlanTierBusiness,
			MonthlyPrice: 19900,  // $199/month
			AnnualPrice:  199000, // $1990/year (save ~17%)
			TrialDays:    14,
			IsActive:     true,
			IsPublic:     true,
			SortOrder:    4,
			Features: []PlanFeature{
				{FeatureKey: "projects", FeatureName: "Projects", FeatureType: FeatureTypeText, Value: "Unlimited", IsEnabled: true, SortOrder: 1},
				{FeatureKey: "team_members", FeatureName: "Team Members", FeatureType: FeatureTypeText, Value: "Unlimited", IsEnabled: true, SortOrder: 2},
				{FeatureKey: "storage_gb", FeatureName: "Storage (GB)", FeatureType: FeatureTypeNumeric, NumericValue: ptr(int64(500)), IsEnabled: true, SortOrder: 3},
				{FeatureKey: "api_access", FeatureName: "API Access", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 4},
				{FeatureKey: "priority_support", FeatureName: "Priority Support", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 5},
				{FeatureKey: "advanced_analytics", FeatureName: "Advanced Analytics", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 6},
				{FeatureKey: "sso", FeatureName: "Single Sign-On (SSO)", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 7},
				{FeatureKey: "audit_logs", FeatureName: "Audit Logs", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 8},
				{FeatureKey: "support", FeatureName: "Support", FeatureType: FeatureTypeText, Value: "24/7 Phone & Chat", IsEnabled: true, SortOrder: 9},
			},
		},
		{
			Name:         "Enterprise",
			Description:  "Custom solutions for enterprise needs",
			Tier:         PlanTierEnterprise,
			MonthlyPrice: 0, // Custom pricing
			AnnualPrice:  0,
			TrialDays:    30,
			IsActive:     true,
			IsPublic:     true,
			SortOrder:    5,
			Features: []PlanFeature{
				{FeatureKey: "projects", FeatureName: "Projects", FeatureType: FeatureTypeText, Value: "Unlimited", IsEnabled: true, SortOrder: 1},
				{FeatureKey: "team_members", FeatureName: "Team Members", FeatureType: FeatureTypeText, Value: "Unlimited", IsEnabled: true, SortOrder: 2},
				{FeatureKey: "storage_gb", FeatureName: "Storage (GB)", FeatureType: FeatureTypeText, Value: "Custom", IsEnabled: true, SortOrder: 3},
				{FeatureKey: "api_access", FeatureName: "API Access", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 4},
				{FeatureKey: "priority_support", FeatureName: "Priority Support", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 5},
				{FeatureKey: "advanced_analytics", FeatureName: "Advanced Analytics", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 6},
				{FeatureKey: "sso", FeatureName: "Single Sign-On (SSO)", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 7},
				{FeatureKey: "audit_logs", FeatureName: "Audit Logs", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 8},
				{FeatureKey: "dedicated_support", FeatureName: "Dedicated Account Manager", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 9},
				{FeatureKey: "custom_integrations", FeatureName: "Custom Integrations", FeatureType: FeatureTypeBoolean, IsEnabled: true, SortOrder: 10},
				{FeatureKey: "sla", FeatureName: "SLA Guarantee", FeatureType: FeatureTypeText, Value: "99.99%", IsEnabled: true, SortOrder: 11},
				{FeatureKey: "support", FeatureName: "Support", FeatureType: FeatureTypeText, Value: "Dedicated Account Manager", IsEnabled: true, SortOrder: 12},
			},
		},
	}
}

// ptr is a helper function to create a pointer to a value
func ptr[T any](v T) *T {
	return &v
}
