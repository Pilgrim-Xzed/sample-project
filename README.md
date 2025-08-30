# Budget Management System

[![Python](https://img.shields.io/badge/Python-3.11%2B-blue)](https://www.python.org/)
[![Django](https://img.shields.io/badge/Django-4.2.8-green)](https://www.djangoproject.com/)
[![Celery](https://img.shields.io/badge/Celery-5.3.4-green)](https://docs.celeryproject.org/)
[![Redis](https://img.shields.io/badge/Redis-Required-red)](https://redis.io/)
[![MyPy](https://img.shields.io/badge/MyPy-Typed-blue)](http://mypy-lang.org/)

A robust Django + Celery backend system for managing advertising campaign budgets with real-time spend tracking, automatic campaign control based on budget limits, and intelligent dayparting (time-based scheduling).

## 📋 Table of Contents

- [Features](#features)
- [System Architecture](#system-architecture)
- [Requirements](#requirements)
- [Installation](#installation)
- [Configuration](#configuration)
- [Running the System](#running-the-system)
- [Usage](#usage)
- [API Reference](#api-reference)
- [Testing](#testing)
- [Deployment](#deployment)
- [Troubleshooting](#troubleshooting)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## ✨ Features

### Core Functionality
- **🔄 Real-time Spend Tracking**: Track daily and monthly ad spend per brand and campaign with millisecond precision
- **🎯 Automatic Budget Control**: Campaigns automatically pause when budget limits are reached
- **📅 Smart Budget Resets**: Daily and monthly automatic budget resets with intelligent campaign reactivation
- **⏰ Dayparting Support**: Configure campaigns to run only during specific hours/days of the week
- **🔒 Race Condition Prevention**: Uses Redis distributed locks and database transactions for data integrity
- **📊 Spend Aggregation**: Optimized daily and monthly spend aggregates for fast reporting

### Technical Features
- **🔍 Type-Safe Codebase**: Fully typed Python code with MyPy validation
- **🎨 Rich Admin Interface**: Comprehensive Django admin panel for managing campaigns and viewing spend data
- **📈 Performance Optimized**: Database indexes and aggregation tables for fast queries
- **🔄 Async Task Processing**: Background tasks with Celery for non-blocking operations
- **📝 Audit Logging**: Complete audit trail of all campaign state changes

## 🏗️ System Architecture

### Data Models

```
┌─────────────┐
│    Brand    │
└─────┬───────┘
      │ 1:N
      ▼
┌─────────────┐     1:N    ┌──────────────────┐
│  Campaign   │────────────►│  SpendRecord     │
└─────┬───────┘             └──────────────────┘
      │                              │
      │ 1:N                          │ Aggregates to
      ▼                              ▼
┌─────────────────────┐     ┌──────────────────────┐
│ DaypartingSchedule  │     │ DailySpendAggregate  │
└─────────────────────┘     └──────────────────────┘
                                     │
                                     ▼
                            ┌──────────────────────┐
                            │MonthlySpendAggregate │
                            └──────────────────────┘
```

#### Model Descriptions

| Model | Description | Key Fields |
|-------|-------------|------------|
| **Brand** | Represents advertising clients | `id`, `name`, `created_at` |
| **Campaign** | Marketing campaigns with budget constraints | `brand`, `name`, `daily_budget`, `monthly_budget`, `is_active` |
| **SpendRecord** | Individual spend transactions | `campaign`, `amount`, `spend_date` |
| **DailySpendAggregate** | Optimized daily spend totals | `campaign`, `date`, `total_spend` |
| **MonthlySpendAggregate** | Optimized monthly spend totals | `campaign`, `year`, `month`, `total_spend` |
| **DaypartingSchedule** | Time-based campaign scheduling rules | `campaign`, `day_of_week`, `start_time`, `end_time` |

### Background Tasks (Celery)

| Task | Schedule | Purpose |
|------|----------|---------|
| **Daily Reset** | Daily at 00:00 UTC | Reset daily budgets and reactivate eligible campaigns |
| **Monthly Reset** | 1st of each month at 00:00 UTC | Reset monthly budgets and reactivate all campaigns |
| **Dayparting Check** | Every minute | Activate/deactivate campaigns based on schedules |
| **Budget Check** | Every 5 minutes | Enforce budget limits on active campaigns |

## 📦 Requirements

### System Requirements
- Python 3.11 or higher
- Redis Server 5.0+
- SQLite3 (development) or PostgreSQL 13+ (production)
- 2GB RAM minimum
- 1GB disk space

### Python Dependencies
```
Django==4.2.8
celery==5.3.4
redis==5.0.1
django-celery-beat==2.5.0
mypy==1.7.1
django-stubs==4.2.7
celery-types==0.21.0
types-redis==4.6.0.11
django-extensions==3.2.3
```

## 🚀 Installation

### Quick Start (Development)

```bash
# 1. Clone the repository
git clone <repository-url>
cd budget-management-system

# 2. Create and activate virtual environment
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# 3. Install dependencies
pip install -r requirements.txt

# 4. Run database migrations
python manage.py migrate

# 5. Create superuser
python manage.py createsuperuser

# 6. Load test data (optional)
python manage.py create_test_data

# 7. Start all services (requires 4 terminal windows)
# Terminal 1: Redis
redis-server

# Terminal 2: Django
python manage.py runserver

# Terminal 3: Celery Worker
celery -A budget_management worker -l info

# Terminal 4: Celery Beat
celery -A budget_management beat -l info
```

### Production Installation

See the [Deployment](#deployment) section for production setup instructions.

## ⚙️ Configuration

### Environment Variables

Create a `.env` file in the project root:

```bash
# Django Settings
SECRET_KEY=your-secret-key-here
DEBUG=False
ALLOWED_HOSTS=localhost,127.0.0.1,your-domain.com

# Database (PostgreSQL for production)
DATABASE_URL=postgresql://user:password@localhost/dbname

# Redis
REDIS_URL=redis://localhost:6379/0

# Celery
CELERY_BROKER_URL=redis://localhost:6379/0
CELERY_RESULT_BACKEND=redis://localhost:6379/0

# Time Zone
TIME_ZONE=UTC
USE_TZ=True
```

### Django Settings

Key settings in `budget_management/settings.py`:

```python
# Celery Configuration
CELERY_BEAT_SCHEDULE = {
    'daily-reset': {
        'task': 'campaigns.tasks.daily_reset_task',
        'schedule': crontab(hour=0, minute=0),  # Midnight UTC
    },
    'monthly-reset': {
        'task': 'campaigns.tasks.monthly_reset_task',
        'schedule': crontab(day_of_month=1, hour=0, minute=0),
    },
    'dayparting-check': {
        'task': 'campaigns.tasks.dayparting_check_task',
        'schedule': crontab(minute='*'),  # Every minute
    },
    'budget-check': {
        'task': 'campaigns.tasks.periodic_budget_check_task',
        'schedule': crontab(minute='*/5'),  # Every 5 minutes
    },
}
```

## 🎮 Running the System

### Development Mode

Use the provided shell script or run services individually:

```bash
# Option 1: All-in-one (requires tmux)
./scripts/start-dev.sh

# Option 2: Individual services
redis-server
python manage.py runserver
celery -A budget_management worker -l info
celery -A budget_management beat -l info
```

### Production Mode

```bash
# Using Supervisor or systemd (recommended)
sudo systemctl start redis
sudo systemctl start budget-management-web
sudo systemctl start budget-management-worker
sudo systemctl start budget-management-beat
```

## 📖 Usage

### Admin Interface

Access the Django admin at http://localhost:8000/admin

#### Creating a Campaign

1. **Create a Brand**:
   - Navigate to Brands → Add Brand
   - Enter brand name (e.g., "Nike", "Adidas")
   - Save

2. **Create a Campaign**:
   - Navigate to Campaigns → Add Campaign
   - Select brand
   - Set campaign name
   - Configure daily budget (e.g., $1,000)
   - Configure monthly budget (e.g., $25,000)
   - Save

3. **Configure Dayparting** (Optional):
   - Navigate to Dayparting Schedules → Add Schedule
   - Select campaign
   - Choose day of week (0=Monday, 6=Sunday)
   - Set start and end times
   - Save

### Programmatic Usage

#### Recording Spend

```python
from campaigns.tasks import record_spend
from decimal import Decimal

# Async task (recommended)
record_spend.delay(
    campaign_id="550e8400-e29b-41d4-a716-446655440000",
    amount=Decimal("50.00")
)

# Synchronous call
record_spend(
    campaign_id="550e8400-e29b-41d4-a716-446655440000",
    amount=Decimal("50.00")
)
```

#### Querying Campaign Status

```python
from campaigns.models import Campaign

# Get campaign
campaign = Campaign.objects.get(id="550e8400-e29b-41d4-a716-446655440000")

# Check status
print(f"Active: {campaign.is_active}")
print(f"Paused by budget: {campaign.is_paused_by_budget}")

# Get current spend
daily_spend = campaign.get_daily_spend()
monthly_spend = campaign.get_monthly_spend()

print(f"Daily spend: ${daily_spend} / ${campaign.daily_budget}")
print(f"Monthly spend: ${monthly_spend} / ${campaign.monthly_budget}")
```

#### Managing Campaigns

```python
from campaigns.models import Campaign
from campaigns.utils import pause_campaign, activate_campaign

# Manually pause campaign
campaign = Campaign.objects.get(name="Summer Sale 2024")
pause_campaign(campaign, reason="Manual pause")

# Manually activate campaign
activate_campaign(campaign)

# Bulk operations
Campaign.objects.filter(brand__name="Nike").update(is_active=False)
```

## 🔌 API Reference

### Management Commands

#### create_test_data

Generate test data for development:

```bash
python manage.py create_test_data [options]

Options:
  --brands INTEGER      Number of brands to create (default: 5)
  --campaigns INTEGER   Campaigns per brand (default: 3)
  --days INTEGER       Days of historical data (default: 30)
```

### Celery Tasks

#### record_spend

```python
@shared_task
def record_spend(campaign_id: str, amount: Decimal) -> dict:
    """
    Record advertising spend for a campaign.
    
    Args:
        campaign_id: UUID of the campaign
        amount: Spend amount in USD
        
    Returns:
        dict: Status and updated spend totals
    """
```

#### check_budget_limits

```python
@shared_task
def check_budget_limits(campaign_id: str) -> dict:
    """
    Check and enforce budget limits for a campaign.
    
    Args:
        campaign_id: UUID of the campaign
        
    Returns:
        dict: Campaign status and any actions taken
    """
```

## 🧪 Testing

### Running Tests

```bash
# Run all tests
python manage.py test

# Run with coverage
coverage run --source='.' manage.py test
coverage report
coverage html  # Generate HTML report

# Run specific test module
python manage.py test campaigns.tests.test_models
python manage.py test campaigns.tests.test_tasks

# Run with verbose output
python manage.py test --verbosity=2
```

### Type Checking

```bash
# Check all files
mypy .

# Check specific module
mypy campaigns/

# Strict mode
mypy --strict campaigns/
```

### Code Quality

```bash
# Linting
flake8 campaigns/

# Format checking
black --check campaigns/

# Import sorting
isort --check-only campaigns/
```

## 🚢 Deployment

### Docker Deployment

```dockerfile
# Dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY . .
RUN python manage.py collectstatic --noinput

CMD ["gunicorn", "budget_management.wsgi:application", "--bind", "0.0.0.0:8000"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  redis:
    image: redis:alpine
    ports:
      - "6379:6379"

  db:
    image: postgres:13
    environment:
      POSTGRES_DB: budget_mgmt
      POSTGRES_USER: budget_user
      POSTGRES_PASSWORD: secure_password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  web:
    build: .
    command: gunicorn budget_management.wsgi:application --bind 0.0.0.0:8000
    volumes:
      - .:/app
    ports:
      - "8000:8000"
    depends_on:
      - db
      - redis
    environment:
      - DATABASE_URL=postgresql://budget_user:secure_password@db/budget_mgmt
      - REDIS_URL=redis://redis:6379/0

  worker:
    build: .
    command: celery -A budget_management worker -l info
    volumes:
      - .:/app
    depends_on:
      - db
      - redis

  beat:
    build: .
    command: celery -A budget_management beat -l info
    volumes:
      - .:/app
    depends_on:
      - db
      - redis

volumes:
  postgres_data:
```

### Production Checklist

- [ ] Set `DEBUG = False`
- [ ] Configure `ALLOWED_HOSTS`
- [ ] Use PostgreSQL instead of SQLite
- [ ] Set up proper `SECRET_KEY`
- [ ] Configure HTTPS/SSL
- [ ] Set up monitoring (Sentry, New Relic)
- [ ] Configure log aggregation
- [ ] Set up backup strategy
- [ ] Configure rate limiting
- [ ] Set up CDN for static files
- [ ] Configure email backend
- [ ] Set up health checks

## 🔧 Troubleshooting

### Common Issues

#### Redis Connection Error

```
Error: Cannot connect to redis://localhost:6379/0
```

**Solution:**
```bash
# Check if Redis is running
redis-cli ping

# Start Redis
redis-server

# Or with Docker
docker run -d -p 6379:6379 redis:alpine
```

#### Celery Worker Not Processing Tasks

**Solution:**
```bash
# Check worker logs
celery -A budget_management worker -l debug

# Purge task queue
celery -A budget_management purge

# Restart worker
pkill -f 'celery worker'
celery -A budget_management worker -l info
```

#### Database Migration Errors

**Solution:**
```bash
# Reset migrations
python manage.py migrate campaigns zero
python manage.py migrate

# Or create fresh database
python manage.py flush --noinput
python manage.py migrate
python manage.py createsuperuser
```

#### Campaign Not Pausing at Budget Limit

**Check:**
1. Celery Beat is running
2. Budget check task is scheduled
3. Redis locks are working
4. Database transactions are committing

```python
# Debug in Django shell
python manage.py shell
>>> from campaigns.tasks import check_budget_limits
>>> check_budget_limits("campaign-id")
```

## 🛠️ Development

### Project Structure

```
budget-management-system/
├── budget_management/       # Django project settings
│   ├── __init__.py
│   ├── settings.py         # Main settings file
│   ├── urls.py            # URL configuration
│   ├── celery.py          # Celery configuration
│   └── wsgi.py           # WSGI application
├── campaigns/             # Main application
│   ├── models.py         # Data models
│   ├── admin.py          # Admin interface
│   ├── tasks.py          # Celery tasks
│   ├── utils.py          # Helper functions
│   ├── tests.py          # Unit tests
│   └── migrations/       # Database migrations
├── requirements.txt       # Python dependencies
├── manage.py             # Django management script
├── mypy.ini             # MyPy configuration
└── setup.cfg            # Tool configurations
```

### Code Style Guide

- Follow PEP 8
- Use type hints for all functions
- Write docstrings for all public methods
- Keep functions under 20 lines
- Use meaningful variable names
- Add comments for complex logic

### Git Workflow

```bash
# Create feature branch
git checkout -b feature/your-feature-name

# Make changes and commit
git add .
git commit -m "feat: add new feature"

# Push and create PR
git push origin feature/your-feature-name
```

### Commit Message Format

```
<type>: <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

## 🔮 Future Enhancements

### Planned Features

- [ ] **REST API**: Full REST API with DRF for external integrations
- [ ] **GraphQL API**: Alternative GraphQL endpoint
- [ ] **Real-time Updates**: WebSocket support for live spend updates
- [ ] **Budget Alerts**: Email/SMS notifications for budget thresholds
- [ ] **Multi-currency**: Support for multiple currencies with conversion
- [ ] **Timezone Support**: User-specific timezone handling
- [ ] **Budget Pacing**: Intelligent spend distribution algorithms
- [ ] **Forecasting**: ML-based spend prediction
- [ ] **A/B Testing**: Built-in campaign A/B testing support
- [ ] **Reporting Dashboard**: React-based analytics dashboard
- [ ] **Audit Trail**: Comprehensive audit logging with UI
- [ ] **Rate Limiting**: API rate limiting per client
- [ ] **Webhook Support**: Configurable webhooks for events
- [ ] **Budget Templates**: Reusable budget configurations
- [ ] **Campaign Cloning**: Duplicate successful campaigns

### Performance Optimizations

- [ ] Implement caching layer (Redis/Memcached)
- [ ] Add database read replicas
- [ ] Optimize aggregate calculations
- [ ] Implement query result pagination
- [ ] Add CDN for static assets

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### How to Contribute

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### Development Setup

```bash
# Install development dependencies
pip install -r requirements-dev.txt

# Run pre-commit hooks
pre-commit install
pre-commit run --all-files
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Django Software Foundation
- Celery Project
- Redis Labs
- All contributors

## 📞 Support

For issues, questions, or suggestions:

- **GitHub Issues**: [Create an issue](https://github.com/your-repo/issues)
- **Email**: support@example.com
- **Documentation**: [Wiki](https://github.com/your-repo/wiki)

---

**Built with ❤️ by the Budget Management Team**