mod config;
mod db;
mod handlers;
mod models;
mod services;
mod tasks;

use std::sync::Arc;
use axum::Server;
use clap::Parser;
use tower_http::cors::CorsLayer;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use crate::config::Settings;
use crate::db::Database;
use crate::services::Services;
use crate::tasks::TaskScheduler;

#[derive(Parser, Debug)]
#[clap(name = "budget_management")]
#[clap(about = "Budget Management System - Rust Implementation", long_about = None)]
struct Args {
    /// Run database migrations
    #[clap(long)]
    migrate: bool,
    
    /// Skip starting background tasks
    #[clap(long)]
    no_tasks: bool,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Parse command line arguments
    let args = Args::parse();
    
    // Load configuration
    let settings = Settings::new()?;
    
    // Initialize logging
    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| format!("budget_management={},tower_http=debug", settings.logging.level).into()),
        )
        .with(tracing_subscriber::fmt::layer())
        .init();
    
    tracing::info!("Starting Budget Management System");
    
    // Set up database connection
    let database_url = std::env::var("DATABASE_URL")
        .unwrap_or_else(|_| settings.database_url().to_string());
    
    if database_url.is_empty() {
        tracing::error!("DATABASE_URL is not set. Please set it to a valid PostgreSQL connection string.");
        std::process::exit(1);
    }
    
    let db = Arc::new(Database::new(&database_url).await?);
    tracing::info!("Database connection established");
    
    // Run migrations if requested
    if args.migrate {
        tracing::info!("Running database migrations...");
        db.migrate().await?;
        tracing::info!("Migrations completed successfully");
        
        if !args.no_tasks {
            return Ok(());
        }
    }
    
    // Set up Redis connection
    let redis_url = std::env::var("REDIS_URL")
        .unwrap_or_else(|_| settings.redis_url().to_string());
    
    let redis_client = redis::Client::open(redis_url)?;
    tracing::info!("Redis connection established");
    
    // Initialize services
    let services = Arc::new(Services::new(db.clone(), redis_client));
    
    // Set up background task scheduler
    let mut task_scheduler = if !args.no_tasks {
        let mut scheduler = TaskScheduler::new(services.clone()).await?;
        scheduler.setup_tasks().await?;
        scheduler.start().await?;
        tracing::info!("Background task scheduler started");
        Some(scheduler)
    } else {
        tracing::info!("Background tasks disabled");
        None
    };
    
    // Create the application router
    let app = handlers::create_router(services.clone())
        .layer(CorsLayer::permissive())
        .layer(tower_http::trace::TraceLayer::new_for_http());
    
    // Start the server
    let addr = settings.server_address().parse()?;
    tracing::info!("Server listening on {}", addr);
    
    // Create shutdown signal handler
    let shutdown_signal = async {
        tokio::signal::ctrl_c()
            .await
            .expect("Failed to install CTRL+C signal handler");
        tracing::info!("Shutdown signal received");
    };
    
    // Run the server
    Server::bind(&addr)
        .serve(app.into_make_service())
        .with_graceful_shutdown(shutdown_signal)
        .await?;
    
    // Cleanup
    if let Some(mut scheduler) = task_scheduler {
        scheduler.shutdown().await?;
    }
    
    tracing::info!("Server shutdown complete");
    Ok(())
}