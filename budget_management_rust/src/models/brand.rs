use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;
use validator::Validate;

use super::Model;

#[derive(Debug, Clone, Serialize, Deserialize, FromRow, Validate)]
pub struct Brand {
    pub id: Uuid,
    
    #[validate(length(min = 1, max = 255))]
    pub name: String,
    
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl Model for Brand {
    fn table_name() -> &'static str {
        "brands"
    }
}

impl Brand {
    pub fn new(name: String) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4(),
            name,
            created_at: now,
            updated_at: now,
        }
    }
}