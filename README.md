# Budget Management System

A Django + Celery backend system for managing advertising campaign budgets with real-time spend tracking, automatic campaign control based on budget limits, and dayparting (time-based scheduling).

## Features

- **Real-time Spend Tracking**: Track daily and monthly ad spend per brand and campaign
- **Automatic Budget Control**: Campaigns are automatically paused when budget limits are reached
- **Smart Budget Resets**: Daily and monthly automatic budget resets with campaign reactivation
- **Dayparting Support**: Configure campaigns to run only during specific hours/days
- **Race Condition Prevention**: Uses Redis locks and database transactions for data integrity
- **Type-Safe Codebase**: Fully typed Python code with MyPy validation
- **Admin Interface**: Rich Django admin panel for managing campaigns and viewing spend data

## System Architecture

### Data Models

1. **Brand**: Represents advertising clients
2. **Campaign**: Marketing campaigns with budget constraints
3. **SpendRecord**: Individual spend transactions
4. **DailySpendAggregate**: Optimized daily spend totals
5. **MonthlySpendAggregate**: Optimized monthly spend totals
6. **DaypartingSchedule**: Time-based campaign scheduling rules

### Background Tasks (Celery)

- **Daily Reset Task**: Runs at midnight to reset daily budgets
- **Monthly Reset Task**: Runs on the 1st of each month to reset monthly budgets
- **Dayparting Check**: Runs every minute to activate/deactivate campaigns based on schedules
- **Budget Check**: Runs every 5 minutes to enforce budget limits

## Installation

### Prerequisites

- Python 3.11+
- Redis (for Celery broker)
- SQLite or PostgreSQL

### Setup Instructions

1. Clone the repository:
```bash
git clone <repository-url>
cd budget_management_system
```

2. Create and activate a virtual environment:
```bash
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
```

3. Install dependencies:
```bash
pip install -r requirements.txt
```

4. Run database migrations:
```bash
python manage.py migrate
```

5. Create a superuser for admin access:
```bash
python manage.py createsuperuser
```

6. (Optional) Load test data:
```bash
python manage.py create_test_data
```

## Running the System

### 1. Start Redis Server
```bash
redis-server
```

### 2. Start Django Development Server
```bash
python manage.py runserver
```

### 3. Start Celery Worker
In a new terminal:
```bash
celery -A budget_management worker -l info
```

### 4. Start Celery Beat Scheduler
In another terminal:
```bash
celery -A budget_management beat -l info
```

### 5. Access the Admin Interface
Navigate to http://localhost:8000/admin and log in with your superuser credentials.

## Usage

### Creating Campaigns

1. Log into the admin interface
2. Create a Brand (e.g., "Nike")
3. Create a Campaign with:
   - Daily budget (e.g., $1000)
   - Monthly budget (e.g., $25000)
   - Optional dayparting schedules

### Recording Spend

You can record spend programmatically:

```python
from campaigns.tasks import record_spend

# Record $50 spend for a campaign
record_spend.delay(campaign_id="campaign-uuid", amount="50.00")
```

### Monitoring Campaigns

The admin interface provides:
- Real-time spend tracking
- Budget utilization percentages
- Campaign status (active/paused)
- Spend history

## Key Workflows

### 1. Spend Recording Flow
- API receives spend data
- Creates SpendRecord entry
- Updates daily and monthly aggregates atomically
- Triggers budget limit check
- Pauses campaign if limits exceeded

### 2. Daily Reset Flow
- Runs at midnight
- Finds campaigns paused by daily budget
- Checks if monthly budget allows reactivation
- Reactivates eligible campaigns

### 3. Monthly Reset Flow
- Runs on the 1st of each month
- Reactivates all budget-paused campaigns
- Fresh start for monthly budgets

### 4. Dayparting Enforcement
- Runs every minute
- Checks current time against schedules
- Activates/deactivates campaigns accordingly
- Respects budget pause status

## Type Safety

The codebase uses Python type hints throughout and is validated with MyPy:

```bash
mypy campaigns/
```

Configuration is in `mypy.ini` with strict type checking enabled.

## Development

### Running Tests
```bash
python manage.py test
```

### Code Style
The project follows PEP 8 and uses type hints. Configuration is in `setup.cfg`.

### Database Schema
Run migrations after model changes:
```bash
python manage.py makemigrations
python manage.py migrate
```

## Assumptions and Simplifications

1. **Time Zone**: System uses UTC by default
2. **Currency**: All amounts are in USD
3. **Spend Recording**: Assumes spend data comes from external sources
4. **Budget Enforcement**: Hard stops at budget limits (no overspend)
5. **Dayparting**: Based on server time, not user timezone

## Performance Considerations

- Uses database indexes for frequent queries
- Aggregated tables reduce calculation overhead
- Redis locks prevent race conditions
- Celery tasks run asynchronously

## Security Notes

- Change `SECRET_KEY` in production
- Use environment variables for sensitive settings
- Configure proper database credentials
- Set up HTTPS in production

## Future Enhancements

- REST API for spend recording
- Real-time notifications for budget alerts
- Multi-currency support
- Timezone-aware dayparting
- Spend forecasting
- Budget pacing algorithms

## License

MIT License

## Support

For issues or questions, please open a GitHub issue.