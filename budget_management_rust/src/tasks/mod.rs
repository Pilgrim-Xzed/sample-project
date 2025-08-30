use std::sync::Arc;
use tokio_cron_scheduler::{Job, JobScheduler};

use crate::services::Services;

pub struct TaskScheduler {
    scheduler: JobScheduler,
    services: Arc<Services>,
}

impl TaskScheduler {
    pub async fn new(services: Arc<Services>) -> Result<Self, Box<dyn std::error::Error>> {
        let scheduler = JobScheduler::new().await?;
        
        Ok(Self {
            scheduler,
            services,
        })
    }
    
    pub async fn setup_tasks(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        // Daily reset task - runs at midnight
        let daily_reset_service = self.services.clone();
        let daily_reset_job = Job::new_async("0 0 0 * * *", move |_uuid, _l| {
            let service = daily_reset_service.clone();
            Box::pin(async move {
                match service.budget.daily_reset().await {
                    Ok(count) => {
                        tracing::info!("Daily reset completed. Reactivated {} campaigns", count);
                    }
                    Err(e) => {
                        tracing::error!("Daily reset failed: {}", e);
                    }
                }
            })
        })?;
        self.scheduler.add(daily_reset_job).await?;
        
        // Monthly reset task - runs at midnight on the 1st of each month
        let monthly_reset_service = self.services.clone();
        let monthly_reset_job = Job::new_async("0 0 0 1 * *", move |_uuid, _l| {
            let service = monthly_reset_service.clone();
            Box::pin(async move {
                match service.budget.monthly_reset().await {
                    Ok(count) => {
                        tracing::info!("Monthly reset completed. Reactivated {} campaigns", count);
                    }
                    Err(e) => {
                        tracing::error!("Monthly reset failed: {}", e);
                    }
                }
            })
        })?;
        self.scheduler.add(monthly_reset_job).await?;
        
        // Dayparting check - runs every minute
        let dayparting_service = self.services.clone();
        let dayparting_job = Job::new_async("0 * * * * *", move |_uuid, _l| {
            let service = dayparting_service.clone();
            Box::pin(async move {
                match service.dayparting.check_and_update_campaigns().await {
                    Ok((activated, deactivated)) => {
                        tracing::debug!(
                            "Dayparting check: activated {}, deactivated {}",
                            activated, deactivated
                        );
                    }
                    Err(e) => {
                        tracing::error!("Dayparting check failed: {}", e);
                    }
                }
            })
        })?;
        self.scheduler.add(dayparting_job).await?;
        
        // Budget check - runs every 5 minutes
        let budget_check_service = self.services.clone();
        let budget_check_job = Job::new_async("0 */5 * * * *", move |_uuid, _l| {
            let service = budget_check_service.clone();
            Box::pin(async move {
                match service.budget.check_all_budgets().await {
                    Ok(count) => {
                        tracing::debug!("Budget check completed for {} campaigns", count);
                    }
                    Err(e) => {
                        tracing::error!("Budget check failed: {}", e);
                    }
                }
            })
        })?;
        self.scheduler.add(budget_check_job).await?;
        
        tracing::info!("All background tasks scheduled successfully");
        Ok(())
    }
    
    pub async fn start(&self) -> Result<(), Box<dyn std::error::Error>> {
        self.scheduler.start().await?;
        tracing::info!("Task scheduler started");
        Ok(())
    }
    
    pub async fn shutdown(&mut self) -> Result<(), Box<dyn std::error::Error>> {
        self.scheduler.shutdown().await?;
        tracing::info!("Task scheduler shutdown");
        Ok(())
    }
}