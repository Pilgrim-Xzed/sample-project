package services

import (
	"context"
	"fmt"
	"time"

	"budget-management/internal/models"
	"budget-management/internal/repository"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DaypartingService handles dayparting schedule business logic
type DaypartingService struct {
	daypartingRepo *repository.DaypartingRepository
	campaignRepo   *repository.CampaignRepository
	logger         *logrus.Logger
}

// NewDaypartingService creates a new dayparting service
func NewDaypartingService(
	daypartingRepo *repository.DaypartingRepository,
	campaignRepo *repository.CampaignRepository,
	logger *logrus.Logger,
) *DaypartingService {
	return &DaypartingService{
		daypartingRepo: daypartingRepo,
		campaignRepo:   campaignRepo,
		logger:         logger,
	}
}

// CreateSchedule creates a new dayparting schedule
func (s *DaypartingService) CreateSchedule(schedule *models.DaypartingSchedule) error {
	// Validate schedule
	if err := s.validateSchedule(schedule); err != nil {
		return err
	}

	// Check for overlapping schedules
	hasOverlap, err := s.daypartingRepo.HasOverlappingSchedule(
		schedule.CampaignID,
		schedule.DayOfWeek,
		schedule.StartTime,
		schedule.EndTime,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to check for overlapping schedules: %w", err)
	}

	if hasOverlap {
		return fmt.Errorf("schedule overlaps with existing schedule for the same day")
	}

	return s.daypartingRepo.Create(schedule)
}

// GetSchedule retrieves a dayparting schedule by ID
func (s *DaypartingService) GetSchedule(id uuid.UUID) (*models.DaypartingSchedule, error) {
	return s.daypartingRepo.GetByID(id)
}

// UpdateSchedule updates a dayparting schedule
func (s *DaypartingService) UpdateSchedule(schedule *models.DaypartingSchedule) error {
	// Validate schedule
	if err := s.validateSchedule(schedule); err != nil {
		return err
	}

	// Check for overlapping schedules (excluding current schedule)
	hasOverlap, err := s.daypartingRepo.HasOverlappingSchedule(
		schedule.CampaignID,
		schedule.DayOfWeek,
		schedule.StartTime,
		schedule.EndTime,
		&schedule.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to check for overlapping schedules: %w", err)
	}

	if hasOverlap {
		return fmt.Errorf("schedule overlaps with existing schedule for the same day")
	}

	return s.daypartingRepo.Update(schedule)
}

// DeleteSchedule deletes a dayparting schedule
func (s *DaypartingService) DeleteSchedule(id uuid.UUID) error {
	return s.daypartingRepo.Delete(id)
}

// ListSchedules retrieves all dayparting schedules for a campaign
func (s *DaypartingService) ListSchedules(campaignID uuid.UUID) ([]models.DaypartingSchedule, error) {
	return s.daypartingRepo.GetByCampaignID(campaignID)
}

// GetActiveSchedules retrieves active dayparting schedules for a campaign
func (s *DaypartingService) GetActiveSchedules(campaignID uuid.UUID) ([]models.DaypartingSchedule, error) {
	return s.daypartingRepo.GetActiveByCampaignID(campaignID)
}

// validateSchedule validates a dayparting schedule
func (s *DaypartingService) validateSchedule(schedule *models.DaypartingSchedule) error {
	// Validate day of week
	if schedule.DayOfWeek < models.Monday || schedule.DayOfWeek > models.Sunday {
		return fmt.Errorf("invalid day of week: %d", schedule.DayOfWeek)
	}

	// Validate time range
	if schedule.EndTime.Before(schedule.StartTime) {
		return fmt.Errorf("end time must be after start time")
	}

	// Check if campaign exists
	exists, err := s.campaignRepo.Exists(schedule.CampaignID)
	if err != nil {
		return fmt.Errorf("failed to check campaign existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("campaign does not exist")
	}

	return nil
}

// CheckDayparting checks and enforces dayparting schedules for all campaigns
func (s *DaypartingService) CheckDayparting(ctx context.Context) (int, int, error) {
	currentTime := time.Now().UTC()
	currentDay := models.DayOfWeek(currentTime.Weekday()) // Convert to our DayOfWeek type

	// Get campaigns with dayparting schedules
	campaigns, err := s.campaignRepo.GetCampaignsWithDayparting()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get campaigns with dayparting: %w", err)
	}

	activatedCount := 0
	deactivatedCount := 0

	for _, campaign := range campaigns {
		// Skip if campaign is paused by budget
		if campaign.IsPausedByBudget {
			continue
		}

		// Check if campaign should be active based on dayparting
		shouldBeActive, err := s.shouldCampaignBeActive(campaign.ID, currentDay, currentTime)
		if err != nil {
			s.logger.Errorf("Failed to check if campaign %s should be active: %v", campaign.ID, err)
			continue
		}

		// Update campaign status if needed
		if shouldBeActive && !campaign.IsActive {
			campaign.IsActive = true
			if err := s.campaignRepo.Update(&campaign); err != nil {
				s.logger.Errorf("Failed to activate campaign %s: %v", campaign.ID, err)
				continue
			}
			activatedCount++
			s.logger.Infof("Activated campaign %s based on dayparting schedule", campaign.Name)
		} else if !shouldBeActive && campaign.IsActive {
			campaign.IsActive = false
			if err := s.campaignRepo.Update(&campaign); err != nil {
				s.logger.Errorf("Failed to deactivate campaign %s: %v", campaign.ID, err)
				continue
			}
			deactivatedCount++
			s.logger.Infof("Deactivated campaign %s based on dayparting schedule", campaign.Name)
		}
	}

	s.logger.Infof("Dayparting check completed: %d activated, %d deactivated", activatedCount, deactivatedCount)
	return activatedCount, deactivatedCount, nil
}

// shouldCampaignBeActive determines if a campaign should be active based on dayparting schedules
func (s *DaypartingService) shouldCampaignBeActive(campaignID uuid.UUID, dayOfWeek models.DayOfWeek, currentTime time.Time) (bool, error) {
	schedules, err := s.daypartingRepo.GetActiveSchedulesForTime(campaignID, dayOfWeek, currentTime)
	if err != nil {
		return false, fmt.Errorf("failed to get active schedules: %w", err)
	}

	return len(schedules) > 0, nil
}

// IsCampaignActiveNow checks if a campaign should be active right now based on dayparting
func (s *DaypartingService) IsCampaignActiveNow(campaignID uuid.UUID) (bool, error) {
	currentTime := time.Now().UTC()
	currentDay := models.DayOfWeek(currentTime.Weekday())

	return s.shouldCampaignBeActive(campaignID, currentDay, currentTime)
}

// GetSchedulesByDay retrieves schedules for a specific day of the week
func (s *DaypartingService) GetSchedulesByDay(campaignID uuid.UUID, dayOfWeek models.DayOfWeek) ([]models.DaypartingSchedule, error) {
	schedules, err := s.daypartingRepo.GetByCampaignID(campaignID)
	if err != nil {
		return nil, err
	}

	var daySchedules []models.DaypartingSchedule
	for _, schedule := range schedules {
		if schedule.DayOfWeek == dayOfWeek {
			daySchedules = append(daySchedules, schedule)
		}
	}

	return daySchedules, nil
}

// GetWeeklySchedule retrieves the complete weekly schedule for a campaign
func (s *DaypartingService) GetWeeklySchedule(campaignID uuid.UUID) (map[models.DayOfWeek][]models.DaypartingSchedule, error) {
	schedules, err := s.daypartingRepo.GetByCampaignID(campaignID)
	if err != nil {
		return nil, err
	}

	weeklySchedule := make(map[models.DayOfWeek][]models.DaypartingSchedule)
	
	// Initialize all days
	for day := models.Monday; day <= models.Sunday; day++ {
		weeklySchedule[day] = []models.DaypartingSchedule{}
	}

	// Group schedules by day
	for _, schedule := range schedules {
		weeklySchedule[schedule.DayOfWeek] = append(weeklySchedule[schedule.DayOfWeek], schedule)
	}

	return weeklySchedule, nil
}

// BulkCreateSchedules creates multiple schedules for a campaign
func (s *DaypartingService) BulkCreateSchedules(schedules []models.DaypartingSchedule) error {
	// Validate all schedules first
	for i, schedule := range schedules {
		if err := s.validateSchedule(&schedule); err != nil {
			return fmt.Errorf("invalid schedule at index %d: %w", i, err)
		}
	}

	// Check for overlaps within the batch and with existing schedules
	for i, schedule := range schedules {
		// Check for overlaps with existing schedules
		hasOverlap, err := s.daypartingRepo.HasOverlappingSchedule(
			schedule.CampaignID,
			schedule.DayOfWeek,
			schedule.StartTime,
			schedule.EndTime,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to check for overlapping schedules at index %d: %w", i, err)
		}
		if hasOverlap {
			return fmt.Errorf("schedule at index %d overlaps with existing schedule", i)
		}

		// Check for overlaps within the batch
		for j, otherSchedule := range schedules {
			if i != j && 
				schedule.CampaignID == otherSchedule.CampaignID &&
				schedule.DayOfWeek == otherSchedule.DayOfWeek &&
				s.timesOverlap(schedule.StartTime, schedule.EndTime, otherSchedule.StartTime, otherSchedule.EndTime) {
				return fmt.Errorf("schedule at index %d overlaps with schedule at index %d", i, j)
			}
		}
	}

	// Create all schedules
	for _, schedule := range schedules {
		if err := s.daypartingRepo.Create(&schedule); err != nil {
			return fmt.Errorf("failed to create schedule: %w", err)
		}
	}

	return nil
}

// timesOverlap checks if two time ranges overlap
func (s *DaypartingService) timesOverlap(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && start2.Before(end1)
}

// DeleteAllSchedules deletes all dayparting schedules for a campaign
func (s *DaypartingService) DeleteAllSchedules(campaignID uuid.UUID) error {
	return s.daypartingRepo.DeleteByCampaignID(campaignID)
}

// GetActiveHours returns the total number of active hours per week for a campaign
func (s *DaypartingService) GetActiveHours(campaignID uuid.UUID) (float64, error) {
	schedules, err := s.daypartingRepo.GetActiveByCampaignID(campaignID)
	if err != nil {
		return 0, err
	}

	totalHours := 0.0
	for _, schedule := range schedules {
		duration := schedule.EndTime.Sub(schedule.StartTime)
		totalHours += duration.Hours()
	}

	return totalHours, nil
}