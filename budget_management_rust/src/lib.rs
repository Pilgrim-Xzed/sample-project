pub mod config;
pub mod db;
pub mod handlers;
pub mod models;
pub mod services;
pub mod tasks;

#[cfg(test)]
mod tests {
    use super::*;
    use bigdecimal::BigDecimal;
    use std::str::FromStr;
    use uuid::Uuid;
    
    #[test]
    fn test_campaign_creation() {
        use crate::models::Campaign;
        
        let campaign = Campaign::new(
            Uuid::new_v4(),
            "Test Campaign".to_string(),
            BigDecimal::from_str("1000.00").unwrap(),
            BigDecimal::from_str("25000.00").unwrap(),
        );
        
        assert_eq!(campaign.name, "Test Campaign");
        assert_eq!(campaign.daily_budget, BigDecimal::from_str("1000.00").unwrap());
        assert_eq!(campaign.monthly_budget, BigDecimal::from_str("25000.00").unwrap());
        assert!(campaign.is_active);
        assert!(!campaign.is_paused_by_budget);
    }
    
    #[test]
    fn test_campaign_pause_for_budget() {
        use crate::models::Campaign;
        
        let mut campaign = Campaign::new(
            Uuid::new_v4(),
            "Test Campaign".to_string(),
            BigDecimal::from_str("1000.00").unwrap(),
            BigDecimal::from_str("25000.00").unwrap(),
        );
        
        campaign.pause_for_budget();
        
        assert!(!campaign.is_active);
        assert!(campaign.is_paused_by_budget);
    }
    
    #[test]
    fn test_brand_creation() {
        use crate::models::Brand;
        
        let brand = Brand::new("Nike".to_string());
        
        assert_eq!(brand.name, "Nike");
    }
    
    #[test]
    fn test_spend_record_creation() {
        use crate::models::SpendRecord;
        use chrono::NaiveDate;
        
        let record = SpendRecord::new(
            Uuid::new_v4(),
            BigDecimal::from_str("50.00").unwrap(),
            NaiveDate::from_ymd_opt(2024, 1, 15).unwrap(),
        );
        
        assert_eq!(record.amount, BigDecimal::from_str("50.00").unwrap());
        assert_eq!(record.spend_date, NaiveDate::from_ymd_opt(2024, 1, 15).unwrap());
    }
    
    #[test]
    fn test_dayparting_schedule() {
        use crate::models::{DaypartingSchedule, DayOfWeek};
        use chrono::NaiveTime;
        
        let schedule = DaypartingSchedule::new(
            Uuid::new_v4(),
            DayOfWeek::Monday,
            NaiveTime::from_hms_opt(9, 0, 0).unwrap(),
            NaiveTime::from_hms_opt(17, 0, 0).unwrap(),
        );
        
        // Test within schedule
        assert!(schedule.is_active_at(NaiveTime::from_hms_opt(12, 0, 0).unwrap()));
        
        // Test outside schedule
        assert!(!schedule.is_active_at(NaiveTime::from_hms_opt(8, 0, 0).unwrap()));
        assert!(!schedule.is_active_at(NaiveTime::from_hms_opt(18, 0, 0).unwrap()));
    }
    
    #[test]
    fn test_daily_spend_aggregate() {
        use crate::models::DailySpendAggregate;
        use chrono::NaiveDate;
        use bigdecimal::Zero;
        
        let mut aggregate = DailySpendAggregate::new(
            Uuid::new_v4(),
            NaiveDate::from_ymd_opt(2024, 1, 15).unwrap(),
        );
        
        assert_eq!(aggregate.total_spend, BigDecimal::zero());
        
        aggregate.add_spend(BigDecimal::from_str("100.00").unwrap());
        assert_eq!(aggregate.total_spend, BigDecimal::from_str("100.00").unwrap());
        
        aggregate.add_spend(BigDecimal::from_str("50.00").unwrap());
        assert_eq!(aggregate.total_spend, BigDecimal::from_str("150.00").unwrap());
    }
    
    #[test]
    fn test_monthly_spend_aggregate() {
        use crate::models::MonthlySpendAggregate;
        use bigdecimal::Zero;
        
        let mut aggregate = MonthlySpendAggregate::new(
            Uuid::new_v4(),
            2024,
            1,
        );
        
        assert_eq!(aggregate.total_spend, BigDecimal::zero());
        
        aggregate.add_spend(BigDecimal::from_str("1000.00").unwrap());
        assert_eq!(aggregate.total_spend, BigDecimal::from_str("1000.00").unwrap());
        
        aggregate.add_spend(BigDecimal::from_str("500.00").unwrap());
        assert_eq!(aggregate.total_spend, BigDecimal::from_str("1500.00").unwrap());
    }
}