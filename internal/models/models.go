package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Brand represents an advertising brand/client
type Brand struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null;size:255"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Campaigns []Campaign `json:"campaigns,omitempty" gorm:"foreignKey:BrandID"`
}

// BeforeCreate sets UUID before creating
func (b *Brand) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// TableName returns the table name for Brand
func (Brand) TableName() string {
	return "brands"
}

// Campaign represents an advertising campaign with budget constraints
type Campaign struct {
	ID                 uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	BrandID            uuid.UUID       `json:"brand_id" gorm:"type:uuid;not null;index"`
	Name               string          `json:"name" gorm:"not null;size:255;index"`
	DailyBudget        decimal.Decimal `json:"daily_budget" gorm:"type:decimal(10,2);not null"`
	MonthlyBudget      decimal.Decimal `json:"monthly_budget" gorm:"type:decimal(12,2);not null"`
	IsActive           bool            `json:"is_active" gorm:"default:true;index"`
	IsPausedByBudget   bool            `json:"is_paused_by_budget" gorm:"default:false;index"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`

	// Relationships
	Brand                Brand                   `json:"brand,omitempty" gorm:"foreignKey:BrandID"`
	SpendRecords         []SpendRecord           `json:"spend_records,omitempty" gorm:"foreignKey:CampaignID"`
	DailyAggregates      []DailySpendAggregate   `json:"daily_aggregates,omitempty" gorm:"foreignKey:CampaignID"`
	MonthlyAggregates    []MonthlySpendAggregate `json:"monthly_aggregates,omitempty" gorm:"foreignKey:CampaignID"`
	DaypartingSchedules  []DaypartingSchedule    `json:"dayparting_schedules,omitempty" gorm:"foreignKey:CampaignID"`
}

// BeforeCreate sets UUID before creating
func (c *Campaign) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// TableName returns the table name for Campaign
func (Campaign) TableName() string {
	return "campaigns"
}

// PauseForBudget pauses the campaign due to budget constraints
func (c *Campaign) PauseForBudget() {
	c.IsActive = false
	c.IsPausedByBudget = true
}

// Reactivate reactivates the campaign after budget reset
func (c *Campaign) Reactivate() {
	c.IsActive = true
	c.IsPausedByBudget = false
}

// DayOfWeek represents days of the week
type DayOfWeek int

const (
	Monday DayOfWeek = iota
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
	Sunday
)

// DaypartingSchedule defines when a campaign should be active during the week
type DaypartingSchedule struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CampaignID uuid.UUID `json:"campaign_id" gorm:"type:uuid;not null;index"`
	DayOfWeek  DayOfWeek `json:"day_of_week" gorm:"not null"`
	StartTime  time.Time `json:"start_time" gorm:"type:time;not null"`
	EndTime    time.Time `json:"end_time" gorm:"type:time;not null"`
	IsActive   bool      `json:"is_active" gorm:"default:true"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Relationships
	Campaign Campaign `json:"campaign,omitempty" gorm:"foreignKey:CampaignID"`
}

// BeforeCreate sets UUID before creating
func (d *DaypartingSchedule) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

// TableName returns the table name for DaypartingSchedule
func (DaypartingSchedule) TableName() string {
	return "dayparting_schedules"
}

// SpendRecord represents individual spend record for a campaign
type SpendRecord struct {
	ID         uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CampaignID uuid.UUID       `json:"campaign_id" gorm:"type:uuid;not null;index"`
	Amount     decimal.Decimal `json:"amount" gorm:"type:decimal(10,2);not null"`
	SpendDate  time.Time       `json:"spend_date" gorm:"type:date;not null;index"`
	CreatedAt  time.Time       `json:"created_at"`

	// Relationships
	Campaign Campaign `json:"campaign,omitempty" gorm:"foreignKey:CampaignID"`
}

// BeforeCreate sets UUID before creating
func (s *SpendRecord) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// TableName returns the table name for SpendRecord
func (SpendRecord) TableName() string {
	return "spend_records"
}

// DailySpendAggregate represents aggregated daily spend for efficient budget checking
type DailySpendAggregate struct {
	CampaignID   uuid.UUID       `json:"campaign_id" gorm:"type:uuid;not null;primaryKey"`
	Date         time.Time       `json:"date" gorm:"type:date;not null;primaryKey;index"`
	TotalSpend   decimal.Decimal `json:"total_spend" gorm:"type:decimal(10,2);default:0.00"`
	LastUpdated  time.Time       `json:"last_updated"`

	// Relationships
	Campaign Campaign `json:"campaign,omitempty" gorm:"foreignKey:CampaignID"`
}

// TableName returns the table name for DailySpendAggregate
func (DailySpendAggregate) TableName() string {
	return "daily_spend_aggregates"
}

// MonthlySpendAggregate represents aggregated monthly spend for efficient budget checking
type MonthlySpendAggregate struct {
	CampaignID  uuid.UUID       `json:"campaign_id" gorm:"type:uuid;not null;primaryKey"`
	Year        int             `json:"year" gorm:"not null;primaryKey"`
	Month       int             `json:"month" gorm:"not null;primaryKey"`
	TotalSpend  decimal.Decimal `json:"total_spend" gorm:"type:decimal(12,2);default:0.00"`
	LastUpdated time.Time       `json:"last_updated"`

	// Relationships
	Campaign Campaign `json:"campaign,omitempty" gorm:"foreignKey:CampaignID"`
}

// TableName returns the table name for MonthlySpendAggregate
func (MonthlySpendAggregate) TableName() string {
	return "monthly_spend_aggregates"
}

// SpendSummary represents spend summary for a campaign
type SpendSummary struct {
	CampaignID     uuid.UUID       `json:"campaign_id"`
	DailySpend     decimal.Decimal `json:"daily_spend"`
	MonthlySpend   decimal.Decimal `json:"monthly_spend"`
	DailyBudget    decimal.Decimal `json:"daily_budget"`
	MonthlyBudget  decimal.Decimal `json:"monthly_budget"`
	DailyRemaining decimal.Decimal `json:"daily_remaining"`
	MonthlyRemaining decimal.Decimal `json:"monthly_remaining"`
}

// TaskPayload represents the payload for background tasks
type TaskPayload struct {
	CampaignID    string    `json:"campaign_id"`
	Amount        string    `json:"amount,omitempty"`
	SpendDate     string    `json:"spend_date,omitempty"`
	TaskType      string    `json:"task_type"`
	ScheduledTime time.Time `json:"scheduled_time,omitempty"`
}

// TaskResult represents the result of a background task
type TaskResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}