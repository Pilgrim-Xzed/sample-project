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

// SpendService handles spend record business logic
type SpendService struct {
	spendRepo       *repository.SpendRepository
	aggregateRepo   *repository.AggregateRepository
	campaignService *CampaignService
	lockService     *LockService
	logger          *logrus.Logger
	db              *gorm.DB
}

// NewSpendService creates a new spend service
func NewSpendService(
	spendRepo *repository.SpendRepository,
	aggregateRepo *repository.AggregateRepository,
	campaignService *CampaignService,
	lockService *LockService,
	logger *logrus.Logger,
) *SpendService {
	return &SpendService{
		spendRepo:       spendRepo,
		aggregateRepo:   aggregateRepo,
		campaignService: campaignService,
		lockService:     lockService,
		logger:          logger,
	}
}

// SetDB sets the database instance for transactions
func (s *SpendService) SetDB(db *gorm.DB) {
	s.db = db
}

// RecordSpend records spend for a campaign and updates aggregates
func (s *SpendService) RecordSpend(ctx context.Context, campaignID uuid.UUID, amount decimal.Decimal, spendDate *time.Time) error {
	lockID := fmt.Sprintf("spend:%s", campaignID)
	
	return s.lockService.WithLock(ctx, lockID, 5*time.Minute, func() error {
		return s.recordSpendInternal(campaignID, amount, spendDate)
	})
}

// recordSpendInternal performs the actual spend recording within a lock
func (s *SpendService) recordSpendInternal(campaignID uuid.UUID, amount decimal.Decimal, spendDate *time.Time) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("spend amount must be greater than zero")
	}

	// Default to today if no date provided
	if spendDate == nil {
		today := time.Now().UTC().Truncate(24 * time.Hour)
		spendDate = &today
	} else {
		// Truncate to date only
		truncated := spendDate.Truncate(24 * time.Hour)
		spendDate = &truncated
	}

	// Start database transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Get campaign with lock
		campaign, err := s.campaignService.campaignRepo.GetByIDForUpdate(tx, campaignID)
		if err != nil {
			return fmt.Errorf("campaign not found: %w", err)
		}

		// Create spend record
		spendRecord := &models.SpendRecord{
			CampaignID: campaignID,
			Amount:     amount,
			SpendDate:  *spendDate,
		}

		if err := s.spendRepo.CreateSpendRecordWithTx(tx, spendRecord); err != nil {
			return fmt.Errorf("failed to create spend record: %w", err)
		}

		// Update daily aggregate
		if err := s.updateDailyAggregate(tx, campaignID, *spendDate, amount); err != nil {
			return fmt.Errorf("failed to update daily aggregate: %w", err)
		}

		// Update monthly aggregate
		if err := s.updateMonthlyAggregate(tx, campaignID, *spendDate, amount); err != nil {
			return fmt.Errorf("failed to update monthly aggregate: %w", err)
		}

		s.logger.Infof("Recorded spend of %s for campaign %s on %s", 
			amount, campaign.Name, spendDate.Format("2006-01-02"))

		return nil
	})
}

// updateDailyAggregate updates the daily spend aggregate
func (s *SpendService) updateDailyAggregate(tx *gorm.DB, campaignID uuid.UUID, spendDate time.Time, amount decimal.Decimal) error {
	aggregate, err := s.aggregateRepo.GetOrCreateDailyAggregateWithTx(tx, campaignID, spendDate)
	if err != nil {
		return err
	}

	aggregate.TotalSpend = aggregate.TotalSpend.Add(amount)
	return s.aggregateRepo.UpdateDailyAggregateWithTx(tx, aggregate)
}

// updateMonthlyAggregate updates the monthly spend aggregate
func (s *SpendService) updateMonthlyAggregate(tx *gorm.DB, campaignID uuid.UUID, spendDate time.Time, amount decimal.Decimal) error {
	aggregate, err := s.aggregateRepo.GetOrCreateMonthlyAggregateWithTx(tx, campaignID, spendDate.Year(), int(spendDate.Month()))
	if err != nil {
		return err
	}

	aggregate.TotalSpend = aggregate.TotalSpend.Add(amount)
	return s.aggregateRepo.UpdateMonthlyAggregateWithTx(tx, aggregate)
}

// GetCampaignSpend retrieves spend records for a campaign
func (s *SpendService) GetCampaignSpend(campaignID uuid.UUID, offset, limit int) ([]models.SpendRecord, error) {
	return s.spendRepo.GetSpendRecords(campaignID, offset, limit)
}

// GetCampaignSpendByDateRange retrieves spend records for a campaign within date range
func (s *SpendService) GetCampaignSpendByDateRange(campaignID uuid.UUID, startDate, endDate time.Time) ([]models.SpendRecord, error) {
	return s.spendRepo.GetSpendRecordsByDateRange(campaignID, startDate, endDate)
}

// GetDailySpend retrieves daily spend aggregates for a campaign
func (s *SpendService) GetDailySpend(campaignID uuid.UUID, startDate, endDate time.Time) ([]models.DailySpendAggregate, error) {
	return s.aggregateRepo.GetDailyAggregates(campaignID, startDate, endDate)
}

// GetMonthlySpend retrieves monthly spend aggregates for a campaign
func (s *SpendService) GetMonthlySpend(campaignID uuid.UUID, limit int) ([]models.MonthlySpendAggregate, error) {
	return s.aggregateRepo.GetMonthlyAggregates(campaignID, limit)
}

// GetTotalSpendByDate calculates total spend for a campaign on a specific date
func (s *SpendService) GetTotalSpendByDate(campaignID uuid.UUID, date time.Time) (decimal.Decimal, error) {
	aggregate, err := s.aggregateRepo.GetDailyAggregate(campaignID, date)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return decimal.Zero, nil
		}
		return decimal.Zero, err
	}
	return aggregate.TotalSpend, nil
}

// GetTotalSpendByMonth calculates total spend for a campaign in a specific month
func (s *SpendService) GetTotalSpendByMonth(campaignID uuid.UUID, year, month int) (decimal.Decimal, error) {
	aggregate, err := s.aggregateRepo.GetMonthlyAggregate(campaignID, year, month)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return decimal.Zero, nil
		}
		return decimal.Zero, err
	}
	return aggregate.TotalSpend, nil
}

// RecalculateAggregates recalculates spend aggregates for a campaign (maintenance function)
func (s *SpendService) RecalculateAggregates(ctx context.Context, campaignID uuid.UUID) error {
	s.logger.Infof("Recalculating aggregates for campaign %s", campaignID)

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Get all spend records for the campaign
		var spendRecords []models.SpendRecord
		if err := tx.Where("campaign_id = ?", campaignID).Find(&spendRecords).Error; err != nil {
			return fmt.Errorf("failed to get spend records: %w", err)
		}

		// Group by date for daily aggregates
		dailyTotals := make(map[time.Time]decimal.Decimal)
		monthlyTotals := make(map[string]decimal.Decimal)

		for _, record := range spendRecords {
			date := record.SpendDate.Truncate(24 * time.Hour)
			monthKey := fmt.Sprintf("%d-%02d", date.Year(), date.Month())

			dailyTotals[date] = dailyTotals[date].Add(record.Amount)
			monthlyTotals[monthKey] = monthlyTotals[monthKey].Add(record.Amount)
		}

		// Update daily aggregates
		for date, total := range dailyTotals {
			aggregate := &models.DailySpendAggregate{
				CampaignID:  campaignID,
				Date:        date,
				TotalSpend:  total,
				LastUpdated: time.Now().UTC(),
			}
			if err := s.aggregateRepo.UpsertDailyAggregateWithTx(tx, aggregate); err != nil {
				return fmt.Errorf("failed to upsert daily aggregate: %w", err)
			}
		}

		// Update monthly aggregates
		for monthKey, total := range monthlyTotals {
			var year, month int
			fmt.Sscanf(monthKey, "%d-%d", &year, &month)
			
			aggregate := &models.MonthlySpendAggregate{
				CampaignID:  campaignID,
				Year:        year,
				Month:       month,
				TotalSpend:  total,
				LastUpdated: time.Now().UTC(),
			}
			if err := s.aggregateRepo.UpsertMonthlyAggregateWithTx(tx, aggregate); err != nil {
				return fmt.Errorf("failed to upsert monthly aggregate: %w", err)
			}
		}

		return nil
	})
}

// GetSpendSummary returns a comprehensive spend summary for a campaign
func (s *SpendService) GetSpendSummary(campaignID uuid.UUID) (*models.SpendSummary, error) {
	return s.campaignService.GetSpendSummary(campaignID)
}

// ValidateSpendAmount validates that a spend amount is valid
func (s *SpendService) ValidateSpendAmount(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("spend amount must be greater than zero")
	}
	
	// Check for reasonable maximum (e.g., $1 million per spend record)
	maxAmount := decimal.NewFromInt(1000000)
	if amount.GreaterThan(maxAmount) {
		return fmt.Errorf("spend amount exceeds maximum allowed: %s", maxAmount)
	}
	
	return nil
}

// BulkRecordSpend records multiple spend entries efficiently
func (s *SpendService) BulkRecordSpend(ctx context.Context, spendRecords []models.SpendRecord) error {
	if len(spendRecords) == 0 {
		return nil
	}

	// Group by campaign for locking
	campaignGroups := make(map[uuid.UUID][]models.SpendRecord)
	for _, record := range spendRecords {
		campaignGroups[record.CampaignID] = append(campaignGroups[record.CampaignID], record)
	}

	// Process each campaign group
	for campaignID, records := range campaignGroups {
		lockID := fmt.Sprintf("spend:%s", campaignID)
		
		err := s.lockService.WithLock(ctx, lockID, 5*time.Minute, func() error {
			return s.bulkRecordSpendForCampaign(records)
		})
		
		if err != nil {
			return fmt.Errorf("failed to bulk record spend for campaign %s: %w", campaignID, err)
		}
	}

	return nil
}

// bulkRecordSpendForCampaign records spend for a single campaign in bulk
func (s *SpendService) bulkRecordSpendForCampaign(records []models.SpendRecord) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Create all spend records
		if err := tx.CreateInBatches(records, 100).Error; err != nil {
			return fmt.Errorf("failed to create spend records: %w", err)
		}

		// Group by date and month for aggregation
		dailyTotals := make(map[time.Time]decimal.Decimal)
		monthlyTotals := make(map[string]decimal.Decimal)

		for _, record := range records {
			date := record.SpendDate.Truncate(24 * time.Hour)
			monthKey := fmt.Sprintf("%s-%d-%02d", record.CampaignID, date.Year(), date.Month())

			dailyTotals[date] = dailyTotals[date].Add(record.Amount)
			monthlyTotals[monthKey] = monthlyTotals[monthKey].Add(record.Amount)
		}

		// Update daily aggregates
		for date, amount := range dailyTotals {
			if err := s.updateDailyAggregate(tx, records[0].CampaignID, date, amount); err != nil {
				return err
			}
		}

		// Update monthly aggregates
		for monthKey, amount := range monthlyTotals {
			var campaignIDStr string
			var year, month int
			fmt.Sscanf(monthKey, "%s-%d-%d", &campaignIDStr, &year, &month)
			
			date := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
			if err := s.updateMonthlyAggregate(tx, records[0].CampaignID, date, amount); err != nil {
				return err
			}
		}

		return nil
	})
}