pub mod brand;
pub mod campaign;
pub mod dayparting;
pub mod spend;

pub use brand::Brand;
pub use campaign::{Campaign, CreateCampaignRequest, UpdateCampaignRequest};
pub use dayparting::{DaypartingSchedule, DayOfWeek, CreateDaypartingScheduleRequest};
pub use spend::{SpendRecord, DailySpendAggregate, MonthlySpendAggregate, RecordSpendRequest, SpendSummary};

// Common traits for all models
pub trait Model {
    fn table_name() -> &'static str;
}

// Common error types
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ModelError {
    #[error("Database error: {0}")]
    Database(#[from] sqlx::Error),
    
    #[error("Not found")]
    NotFound,
    
    #[error("Validation error: {0}")]
    Validation(String),
    
    #[error("Lock acquisition failed")]
    LockFailed,
}

pub type Result<T> = std::result::Result<T, ModelError>;