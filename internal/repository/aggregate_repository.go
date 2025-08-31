package repository

import (
	"time"

	"budget-management/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AggregateRepository handles spend aggregate database operations
type AggregateRepository struct {
	db *gorm.DB
}

// NewAggregateRepository creates a new aggregate repository
func NewAggregateRepository(db *gorm.DB) *AggregateRepository {
	return &AggregateRepository{db: db}
}

// GetOrCreateDailyAggregate gets or creates a daily spend aggregate
func (r *AggregateRepository) GetOrCreateDailyAggregate(campaignID uuid.UUID, date time.Time) (*models.DailySpendAggregate, error) {
	var aggregate models.DailySpendAggregate
	
	err := r.db.Where("campaign_id = ? AND date = ?", campaignID, date).First(&aggregate).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new aggregate
			aggregate = models.DailySpendAggregate{
				CampaignID:  campaignID,
				Date:        date,
				TotalSpend:  decimal.Zero,
				LastUpdated: time.Now().UTC(),
			}
			if err := r.db.Create(&aggregate).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	
	return &aggregate, nil
}

// GetOrCreateDailyAggregateWithTx gets or creates a daily spend aggregate within a transaction
func (r *AggregateRepository) GetOrCreateDailyAggregateWithTx(tx *gorm.DB, campaignID uuid.UUID, date time.Time) (*models.DailySpendAggregate, error) {
	var aggregate models.DailySpendAggregate
	
	err := tx.Where("campaign_id = ? AND date = ?", campaignID, date).First(&aggregate).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new aggregate
			aggregate = models.DailySpendAggregate{
				CampaignID:  campaignID,
				Date:        date,
				TotalSpend:  decimal.Zero,
				LastUpdated: time.Now().UTC(),
			}
			if err := tx.Create(&aggregate).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	
	return &aggregate, nil
}

// UpdateDailyAggregateWithTx updates a daily aggregate within a transaction
func (r *AggregateRepository) UpdateDailyAggregateWithTx(tx *gorm.DB, aggregate *models.DailySpendAggregate) error {
	aggregate.LastUpdated = time.Now().UTC()
	return tx.Save(aggregate).Error
}

// IncrementDailySpendWithTx atomically increments daily spend
func (r *AggregateRepository) IncrementDailySpendWithTx(tx *gorm.DB, campaignID uuid.UUID, date time.Time, amount decimal.Decimal) error {
	return tx.Model(&models.DailySpendAggregate{}).
		Where("campaign_id = ? AND date = ?", campaignID, date).
		UpdateColumn("total_spend", gorm.Expr("total_spend + ?", amount)).
		UpdateColumn("last_updated", time.Now().UTC()).Error
}

// GetOrCreateMonthlyAggregate gets or creates a monthly spend aggregate
func (r *AggregateRepository) GetOrCreateMonthlyAggregate(campaignID uuid.UUID, year, month int) (*models.MonthlySpendAggregate, error) {
	var aggregate models.MonthlySpendAggregate
	
	err := r.db.Where("campaign_id = ? AND year = ? AND month = ?", campaignID, year, month).First(&aggregate).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new aggregate
			aggregate = models.MonthlySpendAggregate{
				CampaignID:  campaignID,
				Year:        year,
				Month:       month,
				TotalSpend:  decimal.Zero,
				LastUpdated: time.Now().UTC(),
			}
			if err := r.db.Create(&aggregate).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	
	return &aggregate, nil
}

// GetOrCreateMonthlyAggregateWithTx gets or creates a monthly spend aggregate within a transaction
func (r *AggregateRepository) GetOrCreateMonthlyAggregateWithTx(tx *gorm.DB, campaignID uuid.UUID, year, month int) (*models.MonthlySpendAggregate, error) {
	var aggregate models.MonthlySpendAggregate
	
	err := tx.Where("campaign_id = ? AND year = ? AND month = ?", campaignID, year, month).First(&aggregate).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new aggregate
			aggregate = models.MonthlySpendAggregate{
				CampaignID:  campaignID,
				Year:        year,
				Month:       month,
				TotalSpend:  decimal.Zero,
				LastUpdated: time.Now().UTC(),
			}
			if err := tx.Create(&aggregate).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	
	return &aggregate, nil
}

// UpdateMonthlyAggregateWithTx updates a monthly aggregate within a transaction
func (r *AggregateRepository) UpdateMonthlyAggregateWithTx(tx *gorm.DB, aggregate *models.MonthlySpendAggregate) error {
	aggregate.LastUpdated = time.Now().UTC()
	return tx.Save(aggregate).Error
}

// IncrementMonthlySpendWithTx atomically increments monthly spend
func (r *AggregateRepository) IncrementMonthlySpendWithTx(tx *gorm.DB, campaignID uuid.UUID, year, month int, amount decimal.Decimal) error {
	return tx.Model(&models.MonthlySpendAggregate{}).
		Where("campaign_id = ? AND year = ? AND month = ?", campaignID, year, month).
		UpdateColumn("total_spend", gorm.Expr("total_spend + ?", amount)).
		UpdateColumn("last_updated", time.Now().UTC()).Error
}

// GetDailyAggregate retrieves a daily aggregate
func (r *AggregateRepository) GetDailyAggregate(campaignID uuid.UUID, date time.Time) (*models.DailySpendAggregate, error) {
	var aggregate models.DailySpendAggregate
	err := r.db.Where("campaign_id = ? AND date = ?", campaignID, date).First(&aggregate).Error
	if err != nil {
		return nil, err
	}
	return &aggregate, nil
}

// GetMonthlyAggregate retrieves a monthly aggregate
func (r *AggregateRepository) GetMonthlyAggregate(campaignID uuid.UUID, year, month int) (*models.MonthlySpendAggregate, error) {
	var aggregate models.MonthlySpendAggregate
	err := r.db.Where("campaign_id = ? AND year = ? AND month = ?", campaignID, year, month).First(&aggregate).Error
	if err != nil {
		return nil, err
	}
	return &aggregate, nil
}

// GetDailyAggregates retrieves daily aggregates for a campaign within date range
func (r *AggregateRepository) GetDailyAggregates(campaignID uuid.UUID, startDate, endDate time.Time) ([]models.DailySpendAggregate, error) {
	var aggregates []models.DailySpendAggregate
	err := r.db.Where("campaign_id = ? AND date >= ? AND date <= ?", campaignID, startDate, endDate).
		Order("date DESC").
		Find(&aggregates).Error
	return aggregates, err
}

// GetMonthlyAggregates retrieves monthly aggregates for a campaign
func (r *AggregateRepository) GetMonthlyAggregates(campaignID uuid.UUID, limit int) ([]models.MonthlySpendAggregate, error) {
	var aggregates []models.MonthlySpendAggregate
	err := r.db.Where("campaign_id = ?", campaignID).
		Order("year DESC, month DESC").
		Limit(limit).
		Find(&aggregates).Error
	return aggregates, err
}

// UpsertDailyAggregateWithTx upserts a daily aggregate within a transaction
func (r *AggregateRepository) UpsertDailyAggregateWithTx(tx *gorm.DB, aggregate *models.DailySpendAggregate) error {
	aggregate.LastUpdated = time.Now().UTC()
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "campaign_id"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_spend", "last_updated"}),
	}).Create(aggregate).Error
}

// UpsertMonthlyAggregateWithTx upserts a monthly aggregate within a transaction
func (r *AggregateRepository) UpsertMonthlyAggregateWithTx(tx *gorm.DB, aggregate *models.MonthlySpendAggregate) error {
	aggregate.LastUpdated = time.Now().UTC()
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "campaign_id"}, {Name: "year"}, {Name: "month"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_spend", "last_updated"}),
	}).Create(aggregate).Error
}