use chrono::{DateTime, NaiveTime, Utc, Weekday};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;
use validator::Validate;

use super::Model;

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum DayOfWeek {
    Monday = 0,
    Tuesday = 1,
    Wednesday = 2,
    Thursday = 3,
    Friday = 4,
    Saturday = 5,
    Sunday = 6,
}

impl From<Weekday> for DayOfWeek {
    fn from(weekday: Weekday) -> Self {
        match weekday {
            Weekday::Mon => DayOfWeek::Monday,
            Weekday::Tue => DayOfWeek::Tuesday,
            Weekday::Wed => DayOfWeek::Wednesday,
            Weekday::Thu => DayOfWeek::Thursday,
            Weekday::Fri => DayOfWeek::Friday,
            Weekday::Sat => DayOfWeek::Saturday,
            Weekday::Sun => DayOfWeek::Sunday,
        }
    }
}

impl From<DayOfWeek> for i32 {
    fn from(day: DayOfWeek) -> Self {
        day as i32
    }
}

impl TryFrom<i32> for DayOfWeek {
    type Error = String;
    
    fn try_from(value: i32) -> Result<Self, Self::Error> {
        match value {
            0 => Ok(DayOfWeek::Monday),
            1 => Ok(DayOfWeek::Tuesday),
            2 => Ok(DayOfWeek::Wednesday),
            3 => Ok(DayOfWeek::Thursday),
            4 => Ok(DayOfWeek::Friday),
            5 => Ok(DayOfWeek::Saturday),
            6 => Ok(DayOfWeek::Sunday),
            _ => Err(format!("Invalid day of week: {}", value)),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow, Validate)]
pub struct DaypartingSchedule {
    pub id: Uuid,
    pub campaign_id: Uuid,
    
    pub day_of_week: i32, // 0-6 representing Monday-Sunday
    pub start_time: NaiveTime,
    pub end_time: NaiveTime,
    pub is_active: bool,
    
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl Model for DaypartingSchedule {
    fn table_name() -> &'static str {
        "dayparting_schedules"
    }
}

impl DaypartingSchedule {
    pub fn new(
        campaign_id: Uuid,
        day_of_week: DayOfWeek,
        start_time: NaiveTime,
        end_time: NaiveTime,
    ) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4(),
            campaign_id,
            day_of_week: day_of_week.into(),
            start_time,
            end_time,
            is_active: true,
            created_at: now,
            updated_at: now,
        }
    }
    
    pub fn get_day_of_week(&self) -> Result<DayOfWeek, String> {
        DayOfWeek::try_from(self.day_of_week)
    }
    
    pub fn is_active_at(&self, time: NaiveTime) -> bool {
        self.is_active && time >= self.start_time && time <= self.end_time
    }
}

// DTO for creating dayparting schedules
#[derive(Debug, Deserialize, Validate)]
pub struct CreateDaypartingScheduleRequest {
    pub campaign_id: Uuid,
    pub day_of_week: DayOfWeek,
    pub start_time: NaiveTime,
    pub end_time: NaiveTime,
}