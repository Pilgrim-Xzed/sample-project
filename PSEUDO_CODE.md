# Budget Management System - Pseudo-code Design

## Overview
This system manages advertising campaign budgets with real-time spend tracking, automatic campaign control based on budget limits, and dayparting (time-based scheduling).

## Data Models

```
Brand:
    id: UUID
    name: string
    created_at: datetime
    updated_at: datetime

Campaign:
    id: UUID
    brand: Brand (FK)
    name: string
    daily_budget: decimal
    monthly_budget: decimal
    is_active: boolean
    is_paused_by_budget: boolean
    created_at: datetime
    updated_at: datetime

DaypartingSchedule:
    id: UUID
    campaign: Campaign (FK)
    day_of_week: integer (0-6)
    start_time: time
    end_time: time
    is_active: boolean

SpendRecord:
    id: UUID
    campaign: Campaign (FK)
    amount: decimal
    spend_date: date
    created_at: datetime

DailySpendAggregate:
    campaign: Campaign (FK)
    date: date
    total_spend: decimal
    last_updated: datetime

MonthlySpendAggregate:
    campaign: Campaign (FK)
    year: integer
    month: integer
    total_spend: decimal
    last_updated: datetime
```

## Core Logic

### Spend Tracking
```
function record_spend(campaign_id, amount):
    // Record individual spend
    create SpendRecord(campaign_id, amount, current_date)
    
    // Update daily aggregate
    daily_aggregate = get_or_create DailySpendAggregate(campaign_id, current_date)
    daily_aggregate.total_spend += amount
    daily_aggregate.last_updated = now()
    save(daily_aggregate)
    
    // Update monthly aggregate
    monthly_aggregate = get_or_create MonthlySpendAggregate(campaign_id, current_year, current_month)
    monthly_aggregate.total_spend += amount
    monthly_aggregate.last_updated = now()
    save(monthly_aggregate)
    
    // Check budget limits
    check_budget_limits(campaign_id)
```

### Budget Enforcement
```
function check_budget_limits(campaign_id):
    campaign = get Campaign(campaign_id)
    
    // Check daily budget
    daily_spend = get DailySpendAggregate(campaign_id, current_date).total_spend
    if daily_spend >= campaign.daily_budget:
        pause_campaign_for_budget(campaign)
        return
    
    // Check monthly budget
    monthly_spend = get MonthlySpendAggregate(campaign_id, current_year, current_month).total_spend
    if monthly_spend >= campaign.monthly_budget:
        pause_campaign_for_budget(campaign)
        return

function pause_campaign_for_budget(campaign):
    campaign.is_paused_by_budget = true
    campaign.is_active = false
    save(campaign)
    log("Campaign {campaign.name} paused due to budget limit")
```

### Daily Reset Task (runs at midnight)
```
function daily_reset_task():
    // Get all campaigns paused by daily budget
    campaigns = get all Campaign where is_paused_by_budget = true
    
    for campaign in campaigns:
        daily_spend = get DailySpendAggregate(campaign.id, yesterday).total_spend
        
        // If campaign was paused due to daily budget only
        if daily_spend >= campaign.daily_budget:
            monthly_spend = get MonthlySpendAggregate(campaign.id, current_year, current_month).total_spend
            
            // Check if monthly budget still allows activation
            if monthly_spend < campaign.monthly_budget:
                campaign.is_paused_by_budget = false
                campaign.is_active = true
                save(campaign)
                log("Campaign {campaign.name} reactivated after daily reset")
```

### Monthly Reset Task (runs at start of month)
```
function monthly_reset_task():
    // Get all campaigns
    campaigns = get all Campaign
    
    for campaign in campaigns:
        if campaign.is_paused_by_budget:
            campaign.is_paused_by_budget = false
            campaign.is_active = true
            save(campaign)
            log("Campaign {campaign.name} reactivated after monthly reset")
```

### Dayparting Check Task (runs every minute)
```
function dayparting_check_task():
    current_time = now()
    current_day = current_time.day_of_week
    current_hour_minute = current_time.time()
    
    // Get all campaigns with dayparting schedules
    campaigns = get all Campaign with DaypartingSchedule
    
    for campaign in campaigns:
        schedules = get DaypartingSchedule for campaign where day_of_week = current_day
        
        should_be_active = false
        for schedule in schedules:
            if schedule.is_active and schedule.start_time <= current_hour_minute <= schedule.end_time:
                should_be_active = true
                break
        
        // Update campaign status based on dayparting
        if not campaign.is_paused_by_budget:
            if should_be_active and not campaign.is_active:
                campaign.is_active = true
                save(campaign)
                log("Campaign {campaign.name} activated by dayparting")
            elif not should_be_active and campaign.is_active:
                campaign.is_active = false
                save(campaign)
                log("Campaign {campaign.name} deactivated by dayparting")
```

### Budget Check Task (runs every 5 minutes)
```
function periodic_budget_check_task():
    active_campaigns = get all Campaign where is_active = true
    
    for campaign in active_campaigns:
        check_budget_limits(campaign.id)
```

## Race Condition Prevention

1. **Atomic Updates**: Use database transactions for spend recording and aggregate updates
2. **Distributed Locks**: Use Redis locks for campaign state changes
3. **Idempotent Operations**: Ensure reset tasks can be safely re-run
4. **Task Deduplication**: Use Celery's task deduplication features

## Key Workflows

1. **Spend Recording**:
   - API receives spend data
   - Record individual spend
   - Update aggregates atomically
   - Check budget limits
   - Pause campaign if limits exceeded

2. **Daily Reset**:
   - Run at midnight
   - Find campaigns paused by daily budget
   - Check monthly budget status
   - Reactivate if monthly budget allows

3. **Monthly Reset**:
   - Run at start of month
   - Reactivate all budget-paused campaigns
   - Clear monthly aggregates for new month

4. **Dayparting Enforcement**:
   - Run every minute
   - Check current time against schedules
   - Activate/deactivate campaigns accordingly
   - Respect budget pause status