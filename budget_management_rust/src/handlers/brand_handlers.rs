use axum::{
    extract::Extension,
    http::StatusCode,
    response::Json,
};
use serde::Deserialize;
use std::sync::Arc;

use crate::models::Brand;
use crate::services::Services;
use super::campaign_handlers::ApiResponse;

#[derive(Debug, Deserialize)]
pub struct CreateBrandRequest {
    pub name: String,
}

pub async fn create_brand(
    Extension(services): Extension<Arc<Services>>,
    Json(request): Json<CreateBrandRequest>,
) -> Result<Json<ApiResponse<Brand>>, StatusCode> {
    match services.campaign.create_brand(request.name).await {
        Ok(brand) => Ok(Json(ApiResponse::success(brand))),
        Err(e) => {
            tracing::error!("Failed to create brand: {}", e);
            Ok(Json(ApiResponse::error(e.to_string())))
        }
    }
}

pub async fn list_brands(
    Extension(services): Extension<Arc<Services>>,
) -> Result<Json<ApiResponse<Vec<Brand>>>, StatusCode> {
    match services.campaign.list_brands().await {
        Ok(brands) => Ok(Json(ApiResponse::success(brands))),
        Err(e) => {
            tracing::error!("Failed to list brands: {}", e);
            Ok(Json(ApiResponse::error(e.to_string())))
        }
    }
}