pub mod campaign_handlers;
pub mod spend_handlers;
pub mod brand_handlers;

use axum::{
    Router,
    routing::{get, post, put},
};
use std::sync::Arc;

use crate::services::Services;

pub fn create_router(services: Arc<Services>) -> Router {
    Router::new()
        // Brand routes
        .route("/api/brands", post(brand_handlers::create_brand))
        .route("/api/brands", get(brand_handlers::list_brands))
        
        // Campaign routes
        .route("/api/campaigns", post(campaign_handlers::create_campaign))
        .route("/api/campaigns", get(campaign_handlers::list_campaigns))
        .route("/api/campaigns/:id", get(campaign_handlers::get_campaign))
        .route("/api/campaigns/:id", put(campaign_handlers::update_campaign))
        .route("/api/campaigns/:id/pause", post(campaign_handlers::pause_campaign))
        .route("/api/campaigns/:id/activate", post(campaign_handlers::activate_campaign))
        
        // Spend routes
        .route("/api/spend", post(spend_handlers::record_spend))
        .route("/api/campaigns/:id/spend-summary", get(spend_handlers::get_spend_summary))
        
        // Dayparting routes
        .route("/api/campaigns/:id/dayparting", post(campaign_handlers::create_dayparting_schedule))
        .route("/api/campaigns/:id/dayparting", get(campaign_handlers::get_dayparting_schedules))
        
        // Health check
        .route("/health", get(health_check))
        
        .layer(axum::Extension(services))
}

async fn health_check() -> &'static str {
    "OK"
}