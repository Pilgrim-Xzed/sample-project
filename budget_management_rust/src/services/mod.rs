pub mod campaign_service;
pub mod spend_service;
pub mod budget_service;
pub mod dayparting_service;

pub use campaign_service::CampaignService;
pub use spend_service::SpendService;
pub use budget_service::BudgetService;
pub use dayparting_service::DaypartingService;

use crate::db::Database;
use std::sync::Arc;

#[derive(Clone)]
pub struct Services {
    pub campaign: CampaignService,
    pub spend: SpendService,
    pub budget: BudgetService,
    pub dayparting: DaypartingService,
}

impl Services {
    pub fn new(db: Arc<Database>, redis_client: redis::Client) -> Self {
        Self {
            campaign: CampaignService::new(db.clone()),
            spend: SpendService::new(db.clone(), redis_client.clone()),
            budget: BudgetService::new(db.clone(), redis_client.clone()),
            dayparting: DaypartingService::new(db.clone()),
        }
    }
}