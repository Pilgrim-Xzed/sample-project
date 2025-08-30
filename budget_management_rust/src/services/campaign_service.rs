use std::sync::Arc;
use uuid::Uuid;

use crate::db::Database;
use crate::models::{Campaign, CreateCampaignRequest, UpdateCampaignRequest, Brand, Result};

#[derive(Clone)]
pub struct CampaignService {
    db: Arc<Database>,
}

impl CampaignService {
    pub fn new(db: Arc<Database>) -> Self {
        Self { db }
    }
    
    pub async fn create_campaign(&self, request: CreateCampaignRequest) -> Result<Campaign> {
        // Verify brand exists
        let _brand = self.db.brand_repo().find_by_id(request.brand_id).await?;
        
        let campaign = Campaign::new(
            request.brand_id,
            request.name,
            request.daily_budget,
            request.monthly_budget,
        );
        
        self.db.campaign_repo().create(&campaign).await
    }
    
    pub async fn get_campaign(&self, id: Uuid) -> Result<Campaign> {
        self.db.campaign_repo().find_by_id(id).await
    }
    
    pub async fn update_campaign(&self, id: Uuid, request: UpdateCampaignRequest) -> Result<Campaign> {
        let mut campaign = self.db.campaign_repo().find_by_id(id).await?;
        
        if let Some(name) = request.name {
            campaign.name = name;
        }
        
        if let Some(daily_budget) = request.daily_budget {
            campaign.daily_budget = daily_budget;
        }
        
        if let Some(monthly_budget) = request.monthly_budget {
            campaign.monthly_budget = monthly_budget;
        }
        
        if let Some(is_active) = request.is_active {
            campaign.is_active = is_active;
        }
        
        self.db.campaign_repo().update(&campaign).await
    }
    
    pub async fn list_campaigns(&self, brand_id: Option<Uuid>) -> Result<Vec<Campaign>> {
        if let Some(brand_id) = brand_id {
            // TODO: Add filter by brand_id in repository
            let all_campaigns = self.db.campaign_repo().find_active().await?;
            Ok(all_campaigns.into_iter()
                .filter(|c| c.brand_id == brand_id)
                .collect())
        } else {
            self.db.campaign_repo().find_active().await
        }
    }
    
    pub async fn pause_campaign(&self, id: Uuid) -> Result<()> {
        let mut campaign = self.db.campaign_repo().find_by_id(id).await?;
        campaign.is_active = false;
        self.db.campaign_repo().update(&campaign).await?;
        Ok(())
    }
    
    pub async fn activate_campaign(&self, id: Uuid) -> Result<()> {
        let mut campaign = self.db.campaign_repo().find_by_id(id).await?;
        
        // Don't activate if paused by budget
        if !campaign.is_paused_by_budget {
            campaign.is_active = true;
            self.db.campaign_repo().update(&campaign).await?;
        }
        
        Ok(())
    }
    
    pub async fn create_brand(&self, name: String) -> Result<Brand> {
        let brand = Brand::new(name);
        self.db.brand_repo().create(&brand).await
    }
    
    pub async fn list_brands(&self) -> Result<Vec<Brand>> {
        self.db.brand_repo().list().await
    }
}