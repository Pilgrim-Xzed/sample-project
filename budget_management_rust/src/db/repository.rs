use chrono::{Datelike, NaiveDate, Utc};
use bigdecimal::{BigDecimal, Zero};
use sqlx::{Pool, Postgres, Transaction};
use uuid::Uuid;

use crate::models::{
    Brand, Campaign, DaypartingSchedule, SpendRecord,
    DailySpendAggregate, MonthlySpendAggregate, ModelError, Result
};

// Brand Repository
#[derive(Clone)]
pub struct BrandRepository {
    pool: Pool<Postgres>,
}

impl BrandRepository {
    pub fn new(pool: Pool<Postgres>) -> Self {
        Self { pool }
    }
    
    pub async fn create(&self, brand: &Brand) -> Result<Brand> {
        let brand = sqlx::query_as!(
            Brand,
            r#"
            INSERT INTO brands (id, name, created_at, updated_at)
            VALUES ($1, $2, $3, $4)
            RETURNING *
            "#,
            brand.id,
            brand.name,
            brand.created_at,
            brand.updated_at
        )
        .fetch_one(&self.pool)
        .await?;
        
        Ok(brand)
    }
    
    pub async fn find_by_id(&self, id: Uuid) -> Result<Brand> {
        let brand = sqlx::query_as!(
            Brand,
            "SELECT * FROM brands WHERE id = $1",
            id
        )
        .fetch_optional(&self.pool)
        .await?
        .ok_or(ModelError::NotFound)?;
        
        Ok(brand)
    }
    
    pub async fn find_by_name(&self, name: &str) -> Result<Brand> {
        let brand = sqlx::query_as!(
            Brand,
            "SELECT * FROM brands WHERE name = $1",
            name
        )
        .fetch_optional(&self.pool)
        .await?
        .ok_or(ModelError::NotFound)?;
        
        Ok(brand)
    }
    
    pub async fn list(&self) -> Result<Vec<Brand>> {
        let brands = sqlx::query_as!(
            Brand,
            "SELECT * FROM brands ORDER BY name"
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(brands)
    }
}

// Campaign Repository
#[derive(Clone)]
pub struct CampaignRepository {
    pool: Pool<Postgres>,
}

impl CampaignRepository {
    pub fn new(pool: Pool<Postgres>) -> Self {
        Self { pool }
    }
    
    pub async fn create(&self, campaign: &Campaign) -> Result<Campaign> {
        let campaign = sqlx::query_as!(
            Campaign,
            r#"
            INSERT INTO campaigns (
                id, brand_id, name, daily_budget, monthly_budget,
                is_active, is_paused_by_budget, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            RETURNING *
            "#,
            campaign.id,
            campaign.brand_id,
            campaign.name,
            campaign.daily_budget,
            campaign.monthly_budget,
            campaign.is_active,
            campaign.is_paused_by_budget,
            campaign.created_at,
            campaign.updated_at
        )
        .fetch_one(&self.pool)
        .await?;
        
        Ok(campaign)
    }
    
    pub async fn find_by_id(&self, id: Uuid) -> Result<Campaign> {
        let campaign = sqlx::query_as!(
            Campaign,
            "SELECT * FROM campaigns WHERE id = $1",
            id
        )
        .fetch_optional(&self.pool)
        .await?
        .ok_or(ModelError::NotFound)?;
        
        Ok(campaign)
    }
    
    pub async fn update(&self, campaign: &Campaign) -> Result<Campaign> {
        let updated = sqlx::query_as!(
            Campaign,
            r#"
            UPDATE campaigns
            SET name = $2, daily_budget = $3, monthly_budget = $4,
                is_active = $5, is_paused_by_budget = $6, updated_at = $7
            WHERE id = $1
            RETURNING *
            "#,
            campaign.id,
            campaign.name,
            campaign.daily_budget,
            campaign.monthly_budget,
            campaign.is_active,
            campaign.is_paused_by_budget,
            Utc::now()
        )
        .fetch_one(&self.pool)
        .await?;
        
        Ok(updated)
    }
    
    pub async fn find_active(&self) -> Result<Vec<Campaign>> {
        let campaigns = sqlx::query_as!(
            Campaign,
            "SELECT * FROM campaigns WHERE is_active = true AND is_paused_by_budget = false"
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(campaigns)
    }
    
    pub async fn find_paused_by_budget(&self) -> Result<Vec<Campaign>> {
        let campaigns = sqlx::query_as!(
            Campaign,
            "SELECT * FROM campaigns WHERE is_paused_by_budget = true"
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(campaigns)
    }
    
    pub async fn pause_for_budget(&self, id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE campaigns
            SET is_active = false, is_paused_by_budget = true, updated_at = $2
            WHERE id = $1
            "#,
            id,
            Utc::now()
        )
        .execute(&self.pool)
        .await?;
        
        Ok(())
    }
    
    pub async fn reactivate(&self, id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE campaigns
            SET is_active = true, is_paused_by_budget = false, updated_at = $2
            WHERE id = $1
            "#,
            id,
            Utc::now()
        )
        .execute(&self.pool)
        .await?;
        
        Ok(())
    }
}

// Dayparting Repository
#[derive(Clone)]
pub struct DaypartingRepository {
    pool: Pool<Postgres>,
}

impl DaypartingRepository {
    pub fn new(pool: Pool<Postgres>) -> Self {
        Self { pool }
    }
    
    pub async fn create(&self, schedule: &DaypartingSchedule) -> Result<DaypartingSchedule> {
        let schedule = sqlx::query_as!(
            DaypartingSchedule,
            r#"
            INSERT INTO dayparting_schedules (
                id, campaign_id, day_of_week, start_time, end_time,
                is_active, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            RETURNING *
            "#,
            schedule.id,
            schedule.campaign_id,
            schedule.day_of_week,
            schedule.start_time,
            schedule.end_time,
            schedule.is_active,
            schedule.created_at,
            schedule.updated_at
        )
        .fetch_one(&self.pool)
        .await?;
        
        Ok(schedule)
    }
    
    pub async fn find_by_campaign(&self, campaign_id: Uuid) -> Result<Vec<DaypartingSchedule>> {
        let schedules = sqlx::query_as!(
            DaypartingSchedule,
            "SELECT * FROM dayparting_schedules WHERE campaign_id = $1 AND is_active = true",
            campaign_id
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(schedules)
    }
    
    pub async fn find_active_for_day(&self, campaign_id: Uuid, day_of_week: i32) -> Result<Vec<DaypartingSchedule>> {
        let schedules = sqlx::query_as!(
            DaypartingSchedule,
            r#"
            SELECT * FROM dayparting_schedules
            WHERE campaign_id = $1 AND day_of_week = $2 AND is_active = true
            "#,
            campaign_id,
            day_of_week
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(schedules)
    }
}

// Spend Repository
#[derive(Clone)]
pub struct SpendRepository {
    pool: Pool<Postgres>,
}

impl SpendRepository {
    pub fn new(pool: Pool<Postgres>) -> Self {
        Self { pool }
    }
    
    pub async fn record_spend(&self, record: &SpendRecord) -> Result<SpendRecord> {
        let mut tx = self.pool.begin().await?;
        
        // Insert spend record
        let spend = sqlx::query_as!(
            SpendRecord,
            r#"
            INSERT INTO spend_records (id, campaign_id, amount, spend_date, created_at)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING *
            "#,
            record.id,
            record.campaign_id,
            record.amount,
            record.spend_date,
            record.created_at
        )
        .fetch_one(&mut *tx)
        .await?;
        
        // Update daily aggregate
        sqlx::query!(
            r#"
            INSERT INTO daily_spend_aggregates (campaign_id, date, total_spend, last_updated)
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (campaign_id, date)
            DO UPDATE SET
                total_spend = daily_spend_aggregates.total_spend + $3,
                last_updated = $4
            "#,
            record.campaign_id,
            record.spend_date,
            record.amount,
            Utc::now()
        )
        .execute(&mut *tx)
        .await?;
        
        // Update monthly aggregate
        let year = record.spend_date.year();
        let month = record.spend_date.month() as i32;
        
        sqlx::query!(
            r#"
            INSERT INTO monthly_spend_aggregates (campaign_id, year, month, total_spend, last_updated)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (campaign_id, year, month)
            DO UPDATE SET
                total_spend = monthly_spend_aggregates.total_spend + $4,
                last_updated = $5
            "#,
            record.campaign_id,
            year,
            month,
            record.amount,
            Utc::now()
        )
        .execute(&mut *tx)
        .await?;
        
        tx.commit().await?;
        
        Ok(spend)
    }
    
    pub async fn get_daily_spend(&self, campaign_id: Uuid, date: NaiveDate) -> Result<BigDecimal> {
        let result = sqlx::query!(
            "SELECT total_spend FROM daily_spend_aggregates WHERE campaign_id = $1 AND date = $2",
            campaign_id,
            date
        )
        .fetch_optional(&self.pool)
        .await?;
        
        Ok(result.map(|r| r.total_spend).unwrap_or_else(|| BigDecimal::zero()))
    }
    
    pub async fn get_monthly_spend(&self, campaign_id: Uuid, year: i32, month: i32) -> Result<BigDecimal> {
        let result = sqlx::query!(
            r#"
            SELECT total_spend FROM monthly_spend_aggregates
            WHERE campaign_id = $1 AND year = $2 AND month = $3
            "#,
            campaign_id,
            year,
            month
        )
        .fetch_optional(&self.pool)
        .await?;
        
        Ok(result.map(|r| r.total_spend).unwrap_or_else(|| BigDecimal::zero()))
    }
    
    pub async fn get_daily_aggregate(&self, campaign_id: Uuid, date: NaiveDate) -> Result<Option<DailySpendAggregate>> {
        let aggregate = sqlx::query_as!(
            DailySpendAggregate,
            "SELECT * FROM daily_spend_aggregates WHERE campaign_id = $1 AND date = $2",
            campaign_id,
            date
        )
        .fetch_optional(&self.pool)
        .await?;
        
        Ok(aggregate)
    }
    
    pub async fn get_monthly_aggregate(&self, campaign_id: Uuid, year: i32, month: i32) -> Result<Option<MonthlySpendAggregate>> {
        let aggregate = sqlx::query_as!(
            MonthlySpendAggregate,
            r#"
            SELECT * FROM monthly_spend_aggregates
            WHERE campaign_id = $1 AND year = $2 AND month = $3
            "#,
            campaign_id,
            year,
            month
        )
        .fetch_optional(&self.pool)
        .await?;
        
        Ok(aggregate)
    }
}