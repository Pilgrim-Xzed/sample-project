use sqlx::{postgres::PgPoolOptions, Pool, Postgres};
use std::time::Duration;

pub mod repository;
pub mod migrations;

pub use repository::{
    BrandRepository, CampaignRepository, DaypartingRepository, SpendRepository
};

pub type DbPool = Pool<Postgres>;

#[derive(Debug, Clone)]
pub struct Database {
    pub pool: DbPool,
}

impl Database {
    pub async fn new(database_url: &str) -> Result<Self, sqlx::Error> {
        let pool = PgPoolOptions::new()
            .max_connections(10)
            .min_connections(2)
            .acquire_timeout(Duration::from_secs(3))
            .connect(database_url)
            .await?;
        
        Ok(Self { pool })
    }
    
    pub async fn migrate(&self) -> Result<(), sqlx::migrate::MigrateError> {
        sqlx::migrate!("./migrations")
            .run(&self.pool)
            .await
    }
    
    pub fn brand_repo(&self) -> BrandRepository {
        BrandRepository::new(self.pool.clone())
    }
    
    pub fn campaign_repo(&self) -> CampaignRepository {
        CampaignRepository::new(self.pool.clone())
    }
    
    pub fn dayparting_repo(&self) -> DaypartingRepository {
        DaypartingRepository::new(self.pool.clone())
    }
    
    pub fn spend_repo(&self) -> SpendRepository {
        SpendRepository::new(self.pool.clone())
    }
}