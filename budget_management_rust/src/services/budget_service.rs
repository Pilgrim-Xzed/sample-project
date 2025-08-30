use chrono::{Datelike, Utc};
use bigdecimal::{BigDecimal, Zero};
use std::sync::Arc;
use uuid::Uuid;

use crate::db::Database;
use crate::models::{Campaign, ModelError, Result};

#[derive(Clone)]
pub struct BudgetService {
    db: Arc<Database>,
    redis_client: redis::Client,
}

impl BudgetService {
    pub fn new(db: Arc<Database>, redis_client: redis::Client) -> Self {
        Self { db, redis_client }
    }
    
    /// Reset campaigns that were paused due to daily budget limits
    pub async fn daily_reset(&self) -> Result<usize> {
        tracing::info!("Starting daily budget reset");
        
        let paused_campaigns = self.db.campaign_repo().find_paused_by_budget().await?;
        let mut reactivated_count = 0;
        
        let today = Utc::now().date_naive();
        let yesterday = today.pred_opt().unwrap_or(today);
        let current_year = today.year();
        let current_month = today.month() as i32;
        
        for campaign in paused_campaigns {
            // Check if campaign was paused due to daily budget
            let yesterday_spend = self.db.spend_repo()
                .get_daily_spend(campaign.id, yesterday)
                .await?;
            
            if yesterday_spend >= campaign.daily_budget {
                // Campaign was paused due to daily budget
                // Check if monthly budget still allows activation
                let monthly_spend = self.db.spend_repo()
                    .get_monthly_spend(campaign.id, current_year, current_month)
                    .await?;
                
                if monthly_spend < campaign.monthly_budget {
                    self.db.campaign_repo().reactivate(campaign.id).await?;
                    reactivated_count += 1;
                    tracing::info!("Reactivated campaign {} after daily reset", campaign.name);
                }
            }
        }
        
        tracing::info!("Daily reset completed. Reactivated {} campaigns", reactivated_count);
        Ok(reactivated_count)
    }
    
    /// Reset all budget-paused campaigns at the start of a new month
    pub async fn monthly_reset(&self) -> Result<usize> {
        tracing::info!("Starting monthly budget reset");
        
        let paused_campaigns = self.db.campaign_repo().find_paused_by_budget().await?;
        let mut reactivated_count = 0;
        
        for campaign in paused_campaigns {
            self.db.campaign_repo().reactivate(campaign.id).await?;
            reactivated_count += 1;
            tracing::info!("Reactivated campaign {} after monthly reset", campaign.name);
        }
        
        tracing::info!("Monthly reset completed. Reactivated {} campaigns", reactivated_count);
        Ok(reactivated_count)
    }
    
    /// Check all active campaigns for budget limits
    pub async fn check_all_budgets(&self) -> Result<usize> {
        let active_campaigns = self.db.campaign_repo().find_active().await?;
        let mut checked_count = 0;
        
        for campaign in active_campaigns {
            self.check_campaign_budget(campaign.id).await?;
            checked_count += 1;
        }
        
        tracing::info!("Checked budgets for {} active campaigns", checked_count);
        Ok(checked_count)
    }
    
    async fn check_campaign_budget(&self, campaign_id: Uuid) -> Result<()> {
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
}