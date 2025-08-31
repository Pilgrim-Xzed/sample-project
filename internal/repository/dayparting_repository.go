package repository

import (
	"time"

	"budget-management/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DaypartingRepository handles dayparting schedule database operations
type DaypartingRepository struct {
	db *gorm.DB
}

// NewDaypartingRepository creates a new dayparting repository
func NewDaypartingRepository(db *gorm.DB) *DaypartingRepository {
	return &DaypartingRepository{db: db}
}

// Create creates a new dayparting schedule
func (r *DaypartingRepository) Create(schedule *models.DaypartingSchedule) error {
	return r.db.Create(schedule).Error
}

// GetByID retrieves a dayparting schedule by ID
func (r *DaypartingRepository) GetByID(id uuid.UUID) (*models.DaypartingSchedule, error) {
	var schedule models.DaypartingSchedule
	err := r.db.Preload("Campaign").Where("id = ?", id).First(&schedule).Error
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

// GetByCampaignID retrieves all dayparting schedules for a campaign
func (r *DaypartingRepository) GetByCampaignID(campaignID uuid.UUID) ([]models.DaypartingSchedule, error) {
	var schedules []models.DaypartingSchedule
	err := r.db.Where("campaign_id = ?", campaignID).
		Order("day_of_week, start_time").
		Find(&schedules).Error
	return schedules, err
}

// GetActiveByCampaignID retrieves active dayparting schedules for a campaign
func (r *DaypartingRepository) GetActiveByCampaignID(campaignID uuid.UUID) ([]models.DaypartingSchedule, error) {
	var schedules []models.DaypartingSchedule
	err := r.db.Where("campaign_id = ? AND is_active = ?", campaignID, true).
		Order("day_of_week, start_time").
		Find(&schedules).Error
	return schedules, err
}

// GetActiveSchedulesForTime retrieves active schedules for a specific day and time
func (r *DaypartingRepository) GetActiveSchedulesForTime(campaignID uuid.UUID, dayOfWeek models.DayOfWeek, currentTime time.Time) ([]models.DaypartingSchedule, error) {
	var schedules []models.DaypartingSchedule
	
	// Extract time component for comparison
	timeOnly := currentTime.Format("15:04:05")
	
	err := r.db.Where("campaign_id = ? AND day_of_week = ? AND is_active = ? AND start_time <= ? AND end_time >= ?",
		campaignID, dayOfWeek, true, timeOnly, timeOnly).
		Find(&schedules).Error
	
	return schedules, err
}

// GetSchedulesForDayAndTime retrieves schedules that should be active at a specific day/time
func (r *DaypartingRepository) GetSchedulesForDayAndTime(dayOfWeek models.DayOfWeek, currentTime time.Time) ([]models.DaypartingSchedule, error) {
	var schedules []models.DaypartingSchedule
	
	// Extract time component for comparison
	timeOnly := currentTime.Format("15:04:05")
	
	err := r.db.Preload("Campaign").
		Where("day_of_week = ? AND is_active = ? AND start_time <= ? AND end_time >= ?",
			dayOfWeek, true, timeOnly, timeOnly).
		Find(&schedules).Error
	
	return schedules, err
}

// Update updates a dayparting schedule
func (r *DaypartingRepository) Update(schedule *models.DaypartingSchedule) error {
	return r.db.Save(schedule).Error
}

// Delete deletes a dayparting schedule by ID
func (r *DaypartingRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.DaypartingSchedule{}).Error
}

// DeleteByCampaignID deletes all dayparting schedules for a campaign
func (r *DaypartingRepository) DeleteByCampaignID(campaignID uuid.UUID) error {
	return r.db.Where("campaign_id = ?", campaignID).Delete(&models.DaypartingSchedule{}).Error
}

// Exists checks if a dayparting schedule exists by ID
func (r *DaypartingRepository) Exists(id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.DaypartingSchedule{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

// HasOverlappingSchedule checks if there's an overlapping schedule for the same campaign and day
func (r *DaypartingRepository) HasOverlappingSchedule(campaignID uuid.UUID, dayOfWeek models.DayOfWeek, startTime, endTime time.Time, excludeID *uuid.UUID) (bool, error) {
	query := r.db.Model(&models.DaypartingSchedule{}).
		Where("campaign_id = ? AND day_of_week = ? AND is_active = ?", campaignID, dayOfWeek, true).
		Where("(start_time <= ? AND end_time >= ?) OR (start_time <= ? AND end_time >= ?) OR (start_time >= ? AND end_time <= ?)",
			startTime.Format("15:04:05"), startTime.Format("15:04:05"),
			endTime.Format("15:04:05"), endTime.Format("15:04:05"),
			startTime.Format("15:04:05"), endTime.Format("15:04:05"))
	
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

// CountByCampaignID returns the number of dayparting schedules for a campaign
func (r *DaypartingRepository) CountByCampaignID(campaignID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.DaypartingSchedule{}).Where("campaign_id = ?", campaignID).Count(&count).Error
	return count, err
}

// GetCampaignsWithActiveSchedules retrieves campaigns that have active dayparting schedules
func (r *DaypartingRepository) GetCampaignsWithActiveSchedules() ([]uuid.UUID, error) {
	var campaignIDs []uuid.UUID
	err := r.db.Model(&models.DaypartingSchedule{}).
		Where("is_active = ?", true).
		Distinct("campaign_id").
		Pluck("campaign_id", &campaignIDs).Error
	return campaignIDs, err
}