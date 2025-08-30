use axum::{
    extract::{Path, Extension},
    http::StatusCode,
    response::Json,
};
use std::sync::Arc;
use uuid::Uuid;

use crate::models::{RecordSpendRequest, SpendRecord, SpendSummary};
use crate::services::Services;
use super::campaign_handlers::ApiResponse;

pub async fn record_spend(
    Extension(services): Extension<Arc<Services>>,
    Json(request): Json<RecordSpendRequest>,
) -> Result<Json<ApiResponse<SpendRecord>>, StatusCode> {
    match services.spend.record_spend(request).await {
        Ok(record) => Ok(Json(ApiResponse::success(record))),
        Err(e) => {
            tracing::error!("Failed to record spend: {}", e);
            Ok(Json(ApiResponse::error(e.to_string())))
        }
    }
}

pub async fn get_spend_summary(
    Extension(services): Extension<Arc<Services>>,
    Path(campaign_id): Path<Uuid>,
) -> Result<Json<ApiResponse<SpendSummary>>, StatusCode> {
    match services.spend.get_spend_summary(campaign_id).await {
        Ok(summary) => Ok(Json(ApiResponse::success(summary))),
        Err(e) => {
            tracing::error!("Failed to get spend summary: {}", e);
            Ok(Json(ApiResponse::error(e.to_string())))
        }
    }
}