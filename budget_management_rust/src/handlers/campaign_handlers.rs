use axum::{
    extract::{Path, Query, Extension},
    http::StatusCode,
    response::Json,
};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use uuid::Uuid;

use crate::models::{Campaign, CreateCampaignRequest, UpdateCampaignRequest, CreateDaypartingScheduleRequest, DaypartingSchedule};
use crate::services::Services;

#[derive(Debug, Deserialize)]
pub struct ListCampaignsQuery {
    brand_id: Option<Uuid>,
}

#[derive(Debug, Serialize)]
pub struct ApiResponse<T> {
    success: bool,
    data: Option<T>,
    error: Option<String>,
}

impl<T> ApiResponse<T> {
    pub fn success(data: T) -> Self {
        Self {
            success: true,
            data: Some(data),
            error: None,
        }
    }
    
    pub fn error(message: String) -> Self {
        Self {
            success: false,
            data: None,
            error: Some(message),
        }
    }
}

pub async fn create_campaign(
    Extension(services): Extension<Arc<Services>>,
    Json(request): Json<CreateCampaignRequest>,
) -> Result<Json<ApiResponse<Campaign>>, StatusCode> {
    match services.campaign.create_campaign(request).await {
        Ok(campaign) => Ok(Json(ApiResponse::success(campaign))),
        Err(e) => {
            tracing::error!("Failed to create campaign: {}", e);
            Ok(Json(ApiResponse::error(e.to_string())))
        }
    }
}

pub async fn get_campaign(
    Extension(services): Extension<Arc<Services>>,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<Campaign>>, StatusCode> {
    match services.campaign.get_campaign(id).await {
        Ok(campaign) => Ok(Json(ApiResponse::success(campaign))),
        Err(e) => {
            tracing::error!("Failed to get campaign: {}", e);
            Ok(Json(ApiResponse::error(e.to_string())))
        }
    }
}

pub async fn update_campaign(
    Extension(services): Extension<Arc<Services>>,
    Path(id): Path<Uuid>,
    Json(request): Json<UpdateCampaignRequest>,
) -> Result<Json<ApiResponse<Campaign>>, StatusCode> {
    match services.campaign.update_campaign(id, request).await {
        Ok(campaign) => Ok(Json(ApiResponse::success(campaign))),
        Err(e) => {
            tracing::error!("Failed to update campaign: {}", e);
            Ok(Json(ApiResponse::error(e.to_string())))
        }
    }
}

pub async fn list_campaigns(
    Extension(services): Extension<Arc<Services>>,
    Query(query): Query<ListCampaignsQuery>,
) -> Result<Json<ApiResponse<Vec<Campaign>>>, StatusCode> {
    match services.campaign.list_campaigns(query.brand_id).await {
        Ok(campaigns) => Ok(Json(ApiResponse::success(campaigns))),
        Err(e) => {
            tracing::error!("Failed to list campaigns: {}", e);
            Ok(Json(ApiResponse::error(e.to_string())))
        }
    }
}

pub async fn pause_campaign(
    Extension(services): Extension<Arc<Services>>,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<String>>, StatusCode> {
    match services.campaign.pause_campaign(id).await {
        Ok(_) => Ok(Json(ApiResponse::success("Campaign paused".to_string()))),
        Err(e) => {
            tracing::error!("Failed to pause campaign: {}", e);
            Ok(Json(ApiResponse::error(e.to_string())))
        }
    }
}

pub async fn activate_campaign(
    Extension(services): Extension<Arc<Services>>,
    Path(id): Path<Uuid>,
) -> Result<Json<ApiResponse<String>>, StatusCode> {
    match services.campaign.activate_campaign(id).await {
        Ok(_) => Ok(Json(ApiResponse::success("Campaign activated".to_string()))),
        Err(e) => {
            tracing::error!("Failed to activate campaign: {}", e);
            Ok(Json(ApiResponse::error(e.to_string())))
        }
    }
}

pub async fn create_dayparting_schedule(
    Extension(services): Extension<Arc<Services>>,
    Path(campaign_id): Path<Uuid>,
    Json(mut request): Json<CreateDaypartingScheduleRequest>,
) -> Result<Json<ApiResponse<DaypartingSchedule>>, StatusCode> {
    request.campaign_id = campaign_id;
    
    match services.dayparting.create_schedule(request).await {
        Ok(schedule) => Ok(Json(ApiResponse::success(schedule))),
        Err(e) => {
            tracing::error!("Failed to create dayparting schedule: {}", e);
            Ok(Json(ApiResponse::error(e.to_string())))
        }
    }
}

pub async fn get_dayparting_schedules(
    Extension(services): Extension<Arc<Services>>,
    Path(campaign_id): Path<Uuid>,
) -> Result<Json<ApiResponse<Vec<DaypartingSchedule>>>, StatusCode> {
    match services.dayparting.get_campaign_schedules(campaign_id).await {
        Ok(schedules) => Ok(Json(ApiResponse::success(schedules))),
        Err(e) => {
            tracing::error!("Failed to get dayparting schedules: {}", e);
            Ok(Json(ApiResponse::error(e.to_string())))
        }
    }
}