use chrono::{Datelike, Timelike, Utc};
use std::sync::Arc;
use uuid::Uuid;

use crate::db::Database;
use crate::models::{DaypartingSchedule, CreateDaypartingScheduleRequest, DayOfWeek, Result};

#[derive(Clone)]
pub struct DaypartingService {
    db: Arc<Database>,
}

impl DaypartingService {
    pub fn new(db: Arc<Database>) -> Self {
        Self { db }
    }
    
    pub async fn create_schedule(&self, request: CreateDaypartingScheduleRequest) -> Result<DaypartingSchedule> {
        // Verify campaign exists
        let _campaign = self.db.campaign_repo().find_by_id(request.campaign_id).await?;
        
        let schedule = DaypartingSchedule::new(
            request.campaign_id,
            request.day_of_week,
            request.start_time,
            request.end_time,
        );
        
        self.db.dayparting_repo().create(&schedule).await
    }
    
    pub async fn get_campaign_schedules(&self, campaign_id: Uuid) -> Result<Vec<DaypartingSchedule>> {
        self.db.dayparting_repo().find_by_campaign(campaign_id).await
    }
    
    pub async fn check_and_update_campaigns(&self) -> Result<(usize, usize)> {
        let current_time = Utc::now();
        let current_day = match current_time.weekday() {
            chrono::Weekday::Mon => 0,
            chrono::Weekday::Tue => 1,
            chrono::Weekday::Wed => 2,
            chrono::Weekday::Thu => 3,
            chrono::Weekday::Fri => 4,
            chrono::Weekday::Sat => 5,
            chrono::Weekday::Sun => 6,
        };
        let current_time_of_day = chrono::NaiveTime::from_hms_opt(
            current_time.hour(),
            current_time.minute(),
            current_time.second(),
        ).unwrap();
        
        // Get all campaigns
        let all_campaigns = self.db.campaign_repo().find_active().await?;
        let mut activated_count = 0;
        let mut deactivated_count = 0;
        
        for campaign in all_campaigns {
            // Skip if campaign is paused by budget
            if campaign.is_paused_by_budget {
                continue;
            }
            
            // Get schedules for this campaign
            let schedules = self.db.dayparting_repo()
                .find_active_for_day(campaign.id, current_day)
                .await?;
            
            // Check if any schedule is active now
            let should_be_active = schedules.iter().any(|schedule| {
                schedule.is_active_at(current_time_of_day)
            });
            
            // If no schedules exist, campaign should be active
            let should_be_active = if schedules.is_empty() {
                true
            } else {
                should_be_active
            };
            
            // Update campaign status if needed
            if should_be_active && !campaign.is_active {
                self.db.campaign_repo().reactivate(campaign.id).await?;
                activated_count += 1;
                tracing::info!("Activated campaign {} based on dayparting schedule", campaign.name);
            } else if !should_be_active && campaign.is_active {
                let mut updated_campaign = campaign.clone();
                updated_campaign.is_active = false;
                self.db.campaign_repo().update(&updated_campaign).await?;
                deactivated_count += 1;
                tracing::info!("Deactivated campaign {} based on dayparting schedule", campaign.name);
            }
        }
        
        Ok((activated_count, deactivated_count))
    }
}