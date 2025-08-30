use chrono::{DateTime, Utc};
use bigdecimal::BigDecimal;
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;
use validator::Validate;

use super::Model;

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct Campaign {
    pub id: Uuid,
    pub brand_id: Uuid,
    pub name: String,
    pub daily_budget: BigDecimal,
    pub monthly_budget: BigDecimal,
    pub is_active: bool,
    pub is_paused_by_budget: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl Model for Campaign {
    fn table_name() -> &'static str {
        "campaigns"
    }
}

impl Campaign {
    pub fn new(
        brand_id: Uuid,
        name: String,
        daily_budget: BigDecimal,
        monthly_budget: BigDecimal,
    ) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4(),
            brand_id,
            name,
            daily_budget,
            monthly_budget,
            is_active: true,
            is_paused_by_budget: false,
            created_at: now,
            updated_at: now,
        }
    }
    
    pub fn pause_for_budget(&mut self) {
        self.is_active = false;
        self.is_paused_by_budget = true;
        self.updated_at = Utc::now();
    }
    
    pub fn reactivate(&mut self) {
        self.is_active = true;
        self.is_paused_by_budget = false;
        self.updated_at = Utc::now();
    }
}

// DTO for creating campaigns
#[derive(Debug, Deserialize, Validate)]
pub struct CreateCampaignRequest {
    pub brand_id: Uuid,
    
    #[validate(length(min = 1, max = 255))]
    pub name: String,
    
    pub daily_budget: BigDecimal,
    pub monthly_budget: BigDecimal,
}

// DTO for updating campaigns
#[derive(Debug, Deserialize, Validate)]
pub struct UpdateCampaignRequest {
    #[validate(length(min = 1, max = 255))]
    pub name: Option<String>,
    
    pub daily_budget: Option<BigDecimal>,
    pub monthly_budget: Option<BigDecimal>,
    
    pub is_active: Option<bool>,
}