package repository

import (
	"budget-management/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CampaignRepository handles campaign database operations
type CampaignRepository struct {
	db *gorm.DB
}

// NewCampaignRepository creates a new campaign repository
func NewCampaignRepository(db *gorm.DB) *CampaignRepository {
	return &CampaignRepository{db: db}
}

// Create creates a new campaign
func (r *CampaignRepository) Create(campaign *models.Campaign) error {
	return r.db.Create(campaign).Error
}

// GetByID retrieves a campaign by ID with brand
func (r *CampaignRepository) GetByID(id uuid.UUID) (*models.Campaign, error) {
	var campaign models.Campaign
	err := r.db.Preload("Brand").Where("id = ?", id).First(&campaign).Error
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

// GetByIDForUpdate retrieves a campaign by ID with row lock
func (r *CampaignRepository) GetByIDForUpdate(tx *gorm.DB, id uuid.UUID) (*models.Campaign, error) {
	var campaign models.Campaign
	err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", id).First(&campaign).Error
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

// List retrieves campaigns with pagination and optional filters
func (r *CampaignRepository) List(offset, limit int, brandID *uuid.UUID, isActive *bool) ([]models.Campaign, error) {
	query := r.db.Preload("Brand")

	if brandID != nil {
		query = query.Where("brand_id = ?", *brandID)
	}

	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	var campaigns []models.Campaign
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&campaigns).Error
	return campaigns, err
}

// Update updates a campaign
func (r *CampaignRepository) Update(campaign *models.Campaign) error {
	return r.db.Save(campaign).Error
}

// UpdateWithTx updates a campaign within a transaction
func (r *CampaignRepository) UpdateWithTx(tx *gorm.DB, campaign *models.Campaign) error {
	return tx.Save(campaign).Error
}

// Delete deletes a campaign by ID
func (r *CampaignRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.Campaign{}).Error
}

// GetActiveCampaigns retrieves all active campaigns not paused by budget
func (r *CampaignRepository) GetActiveCampaigns() ([]models.Campaign, error) {
	var campaigns []models.Campaign
	err := r.db.Where("is_active = ? AND is_paused_by_budget = ?", true, false).Find(&campaigns).Error
	return campaigns, err
}

// GetBudgetPausedCampaigns retrieves all campaigns paused by budget
func (r *CampaignRepository) GetBudgetPausedCampaigns() ([]models.Campaign, error) {
	var campaigns []models.Campaign
	err := r.db.Preload("Brand").Where("is_paused_by_budget = ?", true).Find(&campaigns).Error
	return campaigns, err
}

// GetCampaignsWithDayparting retrieves campaigns that have dayparting schedules
func (r *CampaignRepository) GetCampaignsWithDayparting() ([]models.Campaign, error) {
	var campaigns []models.Campaign
	err := r.db.Preload("Brand").
		Joins("JOIN dayparting_schedules ON campaigns.id = dayparting_schedules.campaign_id").
		Distinct().
		Find(&campaigns).Error
	return campaigns, err
}

// Count returns total number of campaigns with optional filters
func (r *CampaignRepository) Count(brandID *uuid.UUID, isActive *bool) (int64, error) {
	query := r.db.Model(&models.Campaign{})

	if brandID != nil {
		query = query.Where("brand_id = ?", *brandID)
	}

	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	var count int64
	err := query.Count(&count).Error
	return count, err
}

// Exists checks if a campaign exists by ID
func (r *CampaignRepository) Exists(id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.Campaign{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

// ExistsByBrandAndName checks if a campaign exists by brand and name
func (r *CampaignRepository) ExistsByBrandAndName(brandID uuid.UUID, name string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Campaign{}).Where("brand_id = ? AND name = ?", brandID, name).Count(&count).Error
	return count > 0, err
}

// GetByBrandID retrieves all campaigns for a brand
func (r *CampaignRepository) GetByBrandID(brandID uuid.UUID) ([]models.Campaign, error) {
	var campaigns []models.Campaign
	err := r.db.Where("brand_id = ?", brandID).Find(&campaigns).Error
	return campaigns, err
}