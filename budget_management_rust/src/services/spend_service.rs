use chrono::{Datelike, Utc};
use bigdecimal::{BigDecimal, Zero};
use std::sync::Arc;
use std::time::Duration;
use uuid::Uuid;

use crate::db::Database;
use crate::models::{SpendRecord, SpendSummary, RecordSpendRequest, ModelError, Result};

#[derive(Clone)]
pub struct SpendService {
    db: Arc<Database>,
    redis_client: redis::Client,
}

impl SpendService {
    pub fn new(db: Arc<Database>, redis_client: redis::Client) -> Self {
        Self { db, redis_client }
    }
    
    pub async fn record_spend(&self, request: RecordSpendRequest) -> Result<SpendRecord> {
        // Acquire distributed lock
        let lock_key = format!("spend:{}", request.campaign_id);
        let mut conn = self.redis_client.get_async_connection().await
            .map_err(|e| ModelError::Validation(format!("Redis error: {}", e)))?;
        
        // Try to acquire lock with 5 minute expiry
        let lock_acquired: bool = redis::cmd("SET")
            .arg(&lock_key)
            .arg("1")
            .arg("NX")
            .arg("EX")
            .arg(300)
            .query_async(&mut conn)
            .await
            .map_err(|e| ModelError::Validation(format!("Lock error: {}", e)))?;
        
        if !lock_acquired {
            return Err(ModelError::LockFailed);
        }
        
        // Ensure lock is released
        let result = self.record_spend_internal(request).await;
        
        // Release lock
        let _: () = redis::cmd("DEL")
            .arg(&lock_key)
            .query_async(&mut conn)
            .await
            .map_err(|e| ModelError::Validation(format!("Unlock error: {}", e)))?;
        
        result
    }
    
    async fn record_spend_internal(&self, request: RecordSpendRequest) -> Result<SpendRecord> {
        // Verify campaign exists
        let campaign = self.db.campaign_repo().find_by_id(request.campaign_id).await?;
        
        let spend_date = request.spend_date.unwrap_or_else(|| Utc::now().date_naive());
        
        // Create spend record
        let record = SpendRecord::new(
            request.campaign_id,
            request.amount,
            spend_date,
        );
        
        // Record in database (includes aggregate updates)
        let saved_record = self.db.spend_repo().record_spend(&record).await?;
        
        // Check budget limits
        self.check_and_enforce_budget_limits(request.campaign_id).await?;
        
        Ok(saved_record)
    }
    
    async fn check_and_enforce_budget_limits(&self, campaign_id: Uuid) -> Result<()> {
        let campaign = self.db.campaign_repo().find_by_id(campaign_id).await?;
        
        if campaign.is_paused_by_budget {
            return Ok(());
        }
        
        let today = Utc::now().date_naive();
        let current_year = today.year();
        let current_month = today.month() as i32;
        
        // Check daily budget
        let daily_spend = self.db.spend_repo().get_daily_spend(campaign_id, today).await?;
        
        if daily_spend >= campaign.daily_budget {
            self.db.campaign_repo().pause_for_budget(campaign_id).await?;
            tracing::warn!(
                "Campaign {} paused - daily budget exceeded: {} >= {}",
                campaign.name, daily_spend, campaign.daily_budget
            );
            return Ok(());
        }
        
        // Check monthly budget
        let monthly_spend = self.db.spend_repo()
            .get_monthly_spend(campaign_id, current_year, current_month)
            .await?;
        
        if monthly_spend >= campaign.monthly_budget {
            self.db.campaign_repo().pause_for_budget(campaign_id).await?;
            tracing::warn!(
                "Campaign {} paused - monthly budget exceeded: {} >= {}",
                campaign.name, monthly_spend, campaign.monthly_budget
            );
        }
        
        Ok(())
    }
    
    pub async fn get_spend_summary(&self, campaign_id: Uuid) -> Result<SpendSummary> {
        let campaign = self.db.campaign_repo().find_by_id(campaign_id).await?;
        
        let today = Utc::now().date_naive();
        let current_year = today.year();
        let current_month = today.month() as i32;
        
        let daily_spend = self.db.spend_repo().get_daily_spend(campaign_id, today).await?;
        let monthly_spend = self.db.spend_repo()
            .get_monthly_spend(campaign_id, current_year, current_month)
            .await?;
        
        let zero = BigDecimal::zero();
        let daily_remaining = (&campaign.daily_budget - &daily_spend).max(zero.clone());
        let monthly_remaining = (&campaign.monthly_budget - &monthly_spend).max(zero.clone());
        
        let daily_utilization_percent = if campaign.daily_budget > zero {
            use std::str::FromStr;
            let hundred = BigDecimal::from_str("100").unwrap();
            ((&daily_spend / &campaign.daily_budget) * &hundred)
                .to_string()
                .parse::<f64>()
                .unwrap_or(0.0)
        } else {
            0.0
        };
        
        let monthly_utilization_percent = if campaign.monthly_budget > zero {
            use std::str::FromStr;
            let hundred = BigDecimal::from_str("100").unwrap();
            ((&monthly_spend / &campaign.monthly_budget) * &hundred)
                .to_string()
                .parse::<f64>()
                .unwrap_or(0.0)
        } else {
            0.0
        };
        
        Ok(SpendSummary {
            daily_spend,
            monthly_spend,
            daily_budget: campaign.daily_budget,
            monthly_budget: campaign.monthly_budget,
            daily_remaining,
            monthly_remaining,
            daily_utilization_percent,
            monthly_utilization_percent,
        })
    }
}