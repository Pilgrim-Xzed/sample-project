package repository

import (
	"time"

	"budget-management/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// SpendRepository handles spend record database operations
type SpendRepository struct {
	db *gorm.DB
}

// NewSpendRepository creates a new spend repository
func NewSpendRepository(db *gorm.DB) *SpendRepository {
	return &SpendRepository{db: db}
}

// CreateSpendRecord creates a new spend record
func (r *SpendRepository) CreateSpendRecord(record *models.SpendRecord) error {
	return r.db.Create(record).Error
}

// CreateSpendRecordWithTx creates a new spend record within a transaction
func (r *SpendRepository) CreateSpendRecordWithTx(tx *gorm.DB, record *models.SpendRecord) error {
	return tx.Create(record).Error
}

// GetSpendRecords retrieves spend records for a campaign with pagination
func (r *SpendRepository) GetSpendRecords(campaignID uuid.UUID, offset, limit int) ([]models.SpendRecord, error) {
	var records []models.SpendRecord
	err := r.db.Where("campaign_id = ?", campaignID).
		Order("spend_date DESC, created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&records).Error
	return records, err
}

// GetSpendRecordsByDateRange retrieves spend records for a campaign within date range
func (r *SpendRepository) GetSpendRecordsByDateRange(campaignID uuid.UUID, startDate, endDate time.Time) ([]models.SpendRecord, error) {
	var records []models.SpendRecord
	err := r.db.Where("campaign_id = ? AND spend_date >= ? AND spend_date <= ?", 
		campaignID, startDate, endDate).
		Order("spend_date DESC, created_at DESC").
		Find(&records).Error
	return records, err
}

// GetTotalSpendByDate calculates total spend for a campaign on a specific date
func (r *SpendRepository) GetTotalSpendByDate(campaignID uuid.UUID, date time.Time) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := r.db.Model(&models.SpendRecord{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("campaign_id = ? AND spend_date = ?", campaignID, date).
		Scan(&total).Error
	return total, err
}

// GetTotalSpendByMonth calculates total spend for a campaign in a specific month
func (r *SpendRepository) GetTotalSpendByMonth(campaignID uuid.UUID, year, month int) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := r.db.Model(&models.SpendRecord{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("campaign_id = ? AND EXTRACT(YEAR FROM spend_date) = ? AND EXTRACT(MONTH FROM spend_date) = ?", 
			campaignID, year, month).
		Scan(&total).Error
	return total, err
}

// DeleteSpendRecord deletes a spend record by ID
func (r *SpendRepository) DeleteSpendRecord(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.SpendRecord{}).Error
}

// CountSpendRecords returns total number of spend records for a campaign
func (r *SpendRepository) CountSpendRecords(campaignID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.SpendRecord{}).Where("campaign_id = ?", campaignID).Count(&count).Error
	return count, err
}