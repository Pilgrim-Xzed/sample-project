use serde::Deserialize;
use config::{Config, ConfigError, Environment, File};

#[derive(Debug, Deserialize, Clone)]
pub struct Settings {
    pub server: ServerSettings,
    pub database: DatabaseSettings,
    pub redis: RedisSettings,
    pub logging: LoggingSettings,
}

#[derive(Debug, Deserialize, Clone)]
pub struct ServerSettings {
    pub host: String,
    pub port: u16,
}

#[derive(Debug, Deserialize, Clone)]
pub struct DatabaseSettings {
    pub url: String,
    pub max_connections: u32,
    pub min_connections: u32,
}

#[derive(Debug, Deserialize, Clone)]
pub struct RedisSettings {
    pub url: String,
}

#[derive(Debug, Deserialize, Clone)]
pub struct LoggingSettings {
    pub level: String,
}

impl Settings {
    pub fn new() -> Result<Self, ConfigError> {
        let config = Config::builder()
            // Start with default values
            .set_default("server.host", "127.0.0.1")?
            .set_default("server.port", 8080)?
            .set_default("database.max_connections", 10)?
            .set_default("database.min_connections", 2)?
            .set_default("redis.url", "redis://127.0.0.1:6379")?
            .set_default("logging.level", "info")?
            
            // Try to load from config file if it exists
            .add_source(File::with_name("config").required(false))
            
            // Override with environment variables
            .add_source(Environment::with_prefix("BUDGET_MGMT").separator("_"))
            
            .build()?;
        
        config.try_deserialize()
    }
    
    pub fn database_url(&self) -> &str {
        &self.database.url
    }
    
    pub fn redis_url(&self) -> &str {
        &self.redis.url
    }
    
    pub fn server_address(&self) -> String {
        format!("{}:{}", self.server.host, self.server.port)
    }
}