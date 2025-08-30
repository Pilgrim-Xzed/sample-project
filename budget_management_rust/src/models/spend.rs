use chrono::{DateTime, NaiveDate, Utc};
use bigdecimal::{BigDecimal, Zero};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;
use validator::Validate;
use std::str::FromStr;

use super::Model;

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct SpendRecord {
    pub id: Uuid,
    pub campaign_id: Uuid,
    pub amount: BigDecimal,
    pub spend_date: NaiveDate,
    pub created_at: DateTime<Utc>,
}

impl Model for SpendRecord {
    fn table_name() -> &'static str {
        "spend_records"
    }
}

impl SpendRecord {
    pub fn new(campaign_id: Uuid, amount: BigDecimal, spend_date: NaiveDate) -> Self {
        Self {
            id: Uuid::new_v4(),
            campaign_id,
            amount,
            spend_date,
            created_at: Utc::now(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct DailySpendAggregate {
    pub campaign_id: Uuid,
    pub date: NaiveDate,
    pub total_spend: BigDecimal,
    pub last_updated: DateTime<Utc>,
}

impl Model for DailySpendAggregate {
    fn table_name() -> &'static str {
        "daily_spend_aggregates"
    }
}

impl DailySpendAggregate {
    pub fn new(campaign_id: Uuid, date: NaiveDate) -> Self {
        Self {
            campaign_id,
            date,
            total_spend: BigDecimal::zero(),
            last_updated: Utc::now(),
        }
    }
    
    pub fn add_spend(&mut self, amount: BigDecimal) {
        self.total_spend = &self.total_spend + &amount;
        self.last_updated = Utc::now();
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow, Validate)]
pub struct MonthlySpendAggregate {
    pub campaign_id: Uuid,
    
    #[validate(range(min = 2020))]
    pub year: i32,
    
    #[validate(range(min = 1, max = 12))]
    pub month: i32,
    
    pub total_spend: BigDecimal,
    pub last_updated: DateTime<Utc>,
}

impl Model for MonthlySpendAggregate {
    fn table_name() -> &'static str {
        "monthly_spend_aggregates"
    }
}

impl MonthlySpendAggregate {
    pub fn new(campaign_id: Uuid, year: i32, month: i32) -> Self {
        Self {
            campaign_id,
            year,
            month,
            total_spend: BigDecimal::zero(),
            last_updated: Utc::now(),
        }
    }
    
    pub fn add_spend(&mut self, amount: BigDecimal) {
        self.total_spend = &self.total_spend + &amount;
        self.last_updated = Utc::now();
    }
}

// DTO for recording spend
#[derive(Debug, Deserialize)]
pub struct RecordSpendRequest {
    pub campaign_id: Uuid,
    pub amount: BigDecimal,
    pub spend_date: Option<NaiveDate>,
}

// Response DTOs
#[derive(Debug, Serialize)]
pub struct SpendSummary {
    pub daily_spend: BigDecimal,
    pub monthly_spend: BigDecimal,
    pub daily_budget: BigDecimal,
    pub monthly_budget: BigDecimal,
    pub daily_remaining: BigDecimal,
    pub monthly_remaining: BigDecimal,
    pub daily_utilization_percent: f64,
    pub monthly_utilization_percent: f64,
}