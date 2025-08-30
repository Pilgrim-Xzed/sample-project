-- Create brands table
CREATE TABLE IF NOT EXISTS brands (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_brands_name ON brands(name);

-- Create campaigns table
CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY,
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    daily_budget DECIMAL(10, 2) NOT NULL CHECK (daily_budget >= 0.01),
    monthly_budget DECIMAL(12, 2) NOT NULL CHECK (monthly_budget >= 0.01),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_paused_by_budget BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(brand_id, name)
);

CREATE INDEX idx_campaigns_brand_id ON campaigns(brand_id);
CREATE INDEX idx_campaigns_active_status ON campaigns(is_active, is_paused_by_budget);

-- Create dayparting_schedules table
CREATE TABLE IF NOT EXISTS dayparting_schedules (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dayparting_campaign_day ON dayparting_schedules(campaign_id, day_of_week, is_active);

-- Create spend_records table
CREATE TABLE IF NOT EXISTS spend_records (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    amount DECIMAL(10, 2) NOT NULL CHECK (amount >= 0.01),
    spend_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_spend_records_campaign_date ON spend_records(campaign_id, spend_date);
CREATE INDEX idx_spend_records_date ON spend_records(spend_date);

-- Create daily_spend_aggregates table
CREATE TABLE IF NOT EXISTS daily_spend_aggregates (
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    total_spend DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (campaign_id, date)
);

CREATE INDEX idx_daily_aggregates_date ON daily_spend_aggregates(date, campaign_id);

-- Create monthly_spend_aggregates table
CREATE TABLE IF NOT EXISTS monthly_spend_aggregates (
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    year INTEGER NOT NULL CHECK (year >= 2020),
    month INTEGER NOT NULL CHECK (month >= 1 AND month <= 12),
    total_spend DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (campaign_id, year, month)
);

CREATE INDEX idx_monthly_aggregates_year_month ON monthly_spend_aggregates(year, month, campaign_id);