package services

import (
	"context"
	"fmt"
	"time"

	"budget-management/internal/models"
	"budget-management/internal/repository"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// CampaignService handles campaign business logic
type CampaignService struct {
	campaignRepo  *repository.CampaignRepository
	spendRepo     *repository.SpendRepository
	aggregateRepo *repository.AggregateRepository
	lockService   *LockService
	logger        *logrus.Logger
}

// NewCampaignService creates a new campaign service
func NewCampaignService(
	campaignRepo *repository.CampaignRepository,
	spendRepo *repository.SpendRepository,
	aggregateRepo *repository.AggregateRepository,
	lockService *LockService,
	logger *logrus.Logger,
) *CampaignService {
	return &CampaignService{
		campaignRepo:  campaignRepo,
		spendRepo:     spendRepo,
		aggregateRepo: aggregateRepo,
		lockService:   lockService,
		logger:        logger,
	}
}

// CreateCampaign creates a new campaign
func (s *CampaignService) CreateCampaign(campaign *models.Campaign) error {
	// Validate budgets
	if campaign.DailyBudget.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("daily budget must be greater than zero")
	}
	if campaign.MonthlyBudget.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("monthly budget must be greater than zero")
	}
	if campaign.DailyBudget.GreaterThan(campaign.MonthlyBudget) {
		return fmt.Errorf("daily budget cannot exceed monthly budget")
	}

	return s.campaignRepo.Create(campaign)
}

// GetCampaign retrieves a campaign by ID
func (s *CampaignService) GetCampaign(id uuid.UUID) (*models.Campaign, error) {
	return s.campaignRepo.GetByID(id)
}

// UpdateCampaign updates a campaign
func (s *CampaignService) UpdateCampaign(campaign *models.Campaign) error {
	// Validate budgets
	if campaign.DailyBudget.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("daily budget must be greater than zero")
	}
	if campaign.MonthlyBudget.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("monthly budget must be greater than zero")
	}
	if campaign.DailyBudget.GreaterThan(campaign.MonthlyBudget) {
		return fmt.Errorf("daily budget cannot exceed monthly budget")
	}

	return s.campaignRepo.Update(campaign)
}

// DeleteCampaign deletes a campaign
func (s *CampaignService) DeleteCampaign(id uuid.UUID) error {
	return s.campaignRepo.Delete(id)
}

// ListCampaigns retrieves campaigns with pagination
func (s *CampaignService) ListCampaigns(offset, limit int, brandID *uuid.UUID, isActive *bool) ([]models.Campaign, error) {
	return s.campaignRepo.List(offset, limit, brandID, isActive)
}

// PauseCampaign manually pauses a campaign
func (s *CampaignService) PauseCampaign(id uuid.UUID) error {
	campaign, err := s.campaignRepo.GetByID(id)
	if err != nil {
		return err
	}

	campaign.IsActive = false
	return s.campaignRepo.Update(campaign)
}

// ActivateCampaign manually activates a campaign
func (s *CampaignService) ActivateCampaign(id uuid.UUID) error {
	campaign, err := s.campaignRepo.GetByID(id)
	if err != nil {
		return err
	}

	campaign.IsActive = true
	return s.campaignRepo.Update(campaign)
}

// CheckBudgetLimits checks and enforces budget limits for a campaign
func (s *CampaignService) CheckBudgetLimits(ctx context.Context, campaignID uuid.UUID) error {
	lockID := fmt.Sprintf("budget_check:%s", campaignID)
	
	acquired, err := s.lockService.TryWithLock(ctx, lockID, 30*time.Second, func() error {
		return s.checkBudgetLimitsInternal(campaignID)
	})
	
	if err != nil {
		return err
	}
	
	if !acquired {
		s.logger.Debugf("Budget check already in progress for campaign %s", campaignID)
		return nil
	}
	
	return nil
}

// checkBudgetLimitsInternal performs the actual budget limit check
func (s *CampaignService) checkBudgetLimitsInternal(campaignID uuid.UUID) error {
	campaign, err := s.campaignRepo.GetByID(campaignID)
	if err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}

	if campaign.IsPausedByBudget {
		s.logger.Debugf("Campaign %s already paused by budget", campaignID)
		return nil
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	now := time.Now().UTC()

	// Check daily budget
	dailyAggregate, err := s.aggregateRepo.GetDailyAggregate(campaignID, today)
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to get daily aggregate: %w", err)
	}

	var dailySpend decimal.Decimal
	if dailyAggregate != nil {
		dailySpend = dailyAggregate.TotalSpend
	}

	if dailySpend.GreaterThanOrEqual(campaign.DailyBudget) {
		campaign.PauseForBudget()
		if err := s.campaignRepo.Update(campaign); err != nil {
			return fmt.Errorf("failed to pause campaign: %w", err)
		}
		
		s.logger.Warnf("Campaign %s paused - daily budget exceeded: %s >= %s", 
			campaign.Name, dailySpend, campaign.DailyBudget)
		return nil
	}

	// Check monthly budget
	monthlyAggregate, err := s.aggregateRepo.GetMonthlyAggregate(campaignID, now.Year(), int(now.Month()))
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to get monthly aggregate: %w", err)
	}

	var monthlySpend decimal.Decimal
	if monthlyAggregate != nil {
		monthlySpend = monthlyAggregate.TotalSpend
	}

	if monthlySpend.GreaterThanOrEqual(campaign.MonthlyBudget) {
		campaign.PauseForBudget()
		if err := s.campaignRepo.Update(campaign); err != nil {
			return fmt.Errorf("failed to pause campaign: %w", err)
		}
		
		s.logger.Warnf("Campaign %s paused - monthly budget exceeded: %s >= %s", 
			campaign.Name, monthlySpend, campaign.MonthlyBudget)
		return nil
	}

	s.logger.Debugf("Budget limits OK for campaign %s", campaign.Name)
	return nil
}

// GetSpendSummary returns spend summary for a campaign
func (s *CampaignService) GetSpendSummary(campaignID uuid.UUID) (*models.SpendSummary, error) {
	campaign, err := s.campaignRepo.GetByID(campaignID)
	if err != nil {
		return nil, err
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	now := time.Now().UTC()

	// Get daily spend
	var dailySpend decimal.Decimal
	dailyAggregate, err := s.aggregateRepo.GetDailyAggregate(campaignID, today)
	if err == nil {
		dailySpend = dailyAggregate.TotalSpend
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to get daily spend: %w", err)
	}

	// Get monthly spend
	var monthlySpend decimal.Decimal
	monthlyAggregate, err := s.aggregateRepo.GetMonthlyAggregate(campaignID, now.Year(), int(now.Month()))
	if err == nil {
		monthlySpend = monthlyAggregate.TotalSpend
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to get monthly spend: %w", err)
	}

	// Calculate remaining budgets
	dailyRemaining := campaign.DailyBudget.Sub(dailySpend)
	if dailyRemaining.LessThan(decimal.Zero) {
		dailyRemaining = decimal.Zero
	}

	monthlyRemaining := campaign.MonthlyBudget.Sub(monthlySpend)
	if monthlyRemaining.LessThan(decimal.Zero) {
		monthlyRemaining = decimal.Zero
	}

	return &models.SpendSummary{
		CampaignID:       campaignID,
		DailySpend:       dailySpend,
		MonthlySpend:     monthlySpend,
		DailyBudget:      campaign.DailyBudget,
		MonthlyBudget:    campaign.MonthlyBudget,
		DailyRemaining:   dailyRemaining,
		MonthlyRemaining: monthlyRemaining,
	}, nil
}

// DailyReset reactivates campaigns that were paused due to daily budget limits
func (s *CampaignService) DailyReset(ctx context.Context) (int, error) {
	s.logger.Info("Starting daily reset task")

	campaigns, err := s.campaignRepo.GetBudgetPausedCampaigns()
	if err != nil {
		return 0, fmt.Errorf("failed to get budget paused campaigns: %w", err)
	}

	reactivatedCount := 0
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	now := time.Now().UTC()

	for _, campaign := range campaigns {
		// Check if campaign was paused due to daily budget
		dailyAggregate, err := s.aggregateRepo.GetDailyAggregate(campaign.ID, yesterday)
		if err != nil && err != gorm.ErrRecordNotFound {
			s.logger.Errorf("Failed to get daily aggregate for campaign %s: %v", campaign.ID, err)
			continue
		}

		if dailyAggregate != nil && dailyAggregate.TotalSpend.GreaterThanOrEqual(campaign.DailyBudget) {
			// Check if monthly budget still allows activation
			monthlyAggregate, err := s.aggregateRepo.GetMonthlyAggregate(campaign.ID, now.Year(), int(now.Month()))
			if err != nil && err != gorm.ErrRecordNotFound {
				s.logger.Errorf("Failed to get monthly aggregate for campaign %s: %v", campaign.ID, err)
				continue
			}

			var monthlySpend decimal.Decimal
			if monthlyAggregate != nil {
				monthlySpend = monthlyAggregate.TotalSpend
			}

			if monthlySpend.LessThan(campaign.MonthlyBudget) {
				campaign.Reactivate()
				if err := s.campaignRepo.Update(&campaign); err != nil {
					s.logger.Errorf("Failed to reactivate campaign %s: %v", campaign.ID, err)
					continue
				}
				
				reactivatedCount++
				s.logger.Infof("Reactivated campaign %s after daily reset", campaign.Name)
			}
		}
	}

	s.logger.Infof("Daily reset completed. Reactivated %d campaigns", reactivatedCount)
	return reactivatedCount, nil
}

// MonthlyReset reactivates all budget-paused campaigns at the start of a new month
func (s *CampaignService) MonthlyReset(ctx context.Context) (int, error) {
	s.logger.Info("Starting monthly reset task")

	campaigns, err := s.campaignRepo.GetBudgetPausedCampaigns()
	if err != nil {
		return 0, fmt.Errorf("failed to get budget paused campaigns: %w", err)
	}

	reactivatedCount := 0

	for _, campaign := range campaigns {
		campaign.Reactivate()
		if err := s.campaignRepo.Update(&campaign); err != nil {
			s.logger.Errorf("Failed to reactivate campaign %s: %v", campaign.ID, err)
			continue
		}
		
		reactivatedCount++
		s.logger.Infof("Reactivated campaign %s after monthly reset", campaign.Name)
	}

	s.logger.Infof("Monthly reset completed. Reactivated %d campaigns", reactivatedCount)
	return reactivatedCount, nil
}

// PeriodicBudgetCheck checks budget limits for all active campaigns
func (s *CampaignService) PeriodicBudgetCheck(ctx context.Context) (int, error) {
	campaigns, err := s.campaignRepo.GetActiveCampaigns()
	if err != nil {
		return 0, fmt.Errorf("failed to get active campaigns: %w", err)
	}

	checkedCount := 0

	for _, campaign := range campaigns {
		if err := s.CheckBudgetLimits(ctx, campaign.ID); err != nil {
			s.logger.Errorf("Failed to check budget limits for campaign %s: %v", campaign.ID, err)
			continue
		}
		checkedCount++
	}

	s.logger.Infof("Checked budget limits for %d active campaigns", checkedCount)
	return checkedCount, nil
}