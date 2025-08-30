# Contributing to Budget Management System

Thank you for your interest in contributing to the Budget Management System! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct:
- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on constructive criticism
- Accept feedback gracefully

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When creating a bug report, include:

- **Clear title and description**
- **Steps to reproduce**
- **Expected behavior**
- **Actual behavior**
- **System information** (OS, Python version, etc.)
- **Relevant logs or error messages**

### Suggesting Enhancements

Enhancement suggestions are welcome! Please include:

- **Use case** - Why is this enhancement needed?
- **Proposed solution** - How should it work?
- **Alternatives considered** - What other solutions did you consider?
- **Additional context** - Any mockups, examples, etc.

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Follow the code style** guidelines below
3. **Add tests** for new functionality
4. **Update documentation** as needed
5. **Ensure all tests pass**
6. **Submit a pull request**

## Development Setup

### Prerequisites

```bash
# Install Python 3.11+
python --version

# Install Redis
redis-server --version

# Install development dependencies
pip install -r requirements.txt
pip install -r requirements-dev.txt
```

### Local Development

```bash
# 1. Clone your fork
git clone https://github.com/your-username/budget-management-system.git
cd budget-management-system

# 2. Create a virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# 3. Install dependencies
pip install -r requirements.txt
pip install -r requirements-dev.txt

# 4. Set up pre-commit hooks
pre-commit install

# 5. Create a feature branch
git checkout -b feature/your-feature-name

# 6. Make your changes
# ... edit files ...

# 7. Run tests
python manage.py test

# 8. Run type checking
mypy .

# 9. Commit your changes
git add .
git commit -m "feat: add your feature"

# 10. Push to your fork
git push origin feature/your-feature-name
```

## Code Style Guidelines

### Python Style

We follow PEP 8 with some modifications:

```python
# Good - Use type hints
def calculate_spend(amount: Decimal, tax_rate: float) -> Decimal:
    """Calculate total spend including tax."""
    return amount * Decimal(1 + tax_rate)

# Good - Clear variable names
campaign_daily_spend = calculate_daily_spend(campaign_id)

# Bad - Unclear names
cds = calc_spend(cid)
```

### Type Hints

All new code must include type hints:

```python
from typing import Optional, List, Dict
from decimal import Decimal
from datetime import date

def get_campaign_spend(
    campaign_id: str,
    start_date: Optional[date] = None,
    end_date: Optional[date] = None
) -> Dict[str, Decimal]:
    """Get campaign spend for date range."""
    ...
```

### Docstrings

Use Google-style docstrings:

```python
def record_spend(campaign_id: str, amount: Decimal) -> bool:
    """
    Record advertising spend for a campaign.
    
    Args:
        campaign_id: UUID of the campaign
        amount: Spend amount in USD
        
    Returns:
        bool: True if successfully recorded
        
    Raises:
        Campaign.DoesNotExist: If campaign not found
        ValueError: If amount is negative
    """
```

### Testing

Write tests for all new functionality:

```python
class TestSpendRecording(TestCase):
    def setUp(self):
        """Set up test data."""
        self.brand = Brand.objects.create(name="Test Brand")
        self.campaign = Campaign.objects.create(
            brand=self.brand,
            name="Test Campaign",
            daily_budget=Decimal("100.00"),
            monthly_budget=Decimal("3000.00")
        )
    
    def test_record_positive_spend(self):
        """Test recording positive spend amount."""
        result = record_spend(self.campaign.id, Decimal("50.00"))
        self.assertTrue(result)
        self.assertEqual(self.campaign.get_daily_spend(), Decimal("50.00"))
    
    def test_record_negative_spend_raises_error(self):
        """Test that negative spend raises ValueError."""
        with self.assertRaises(ValueError):
            record_spend(self.campaign.id, Decimal("-10.00"))
```

## Commit Message Guidelines

We follow the Conventional Commits specification:

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `perf`: Performance improvements

### Examples

```bash
# Feature
git commit -m "feat(campaigns): add budget pacing algorithm"

# Bug fix
git commit -m "fix(tasks): correct daily reset logic for campaigns"

# Documentation
git commit -m "docs: update API documentation for spend endpoint"

# With body
git commit -m "feat(api): add REST API for campaign management

- Add DRF serializers for Campaign model
- Implement CRUD endpoints
- Add API authentication
- Include pagination support

Closes #123"
```

## Pull Request Process

1. **Update documentation** - README, docstrings, comments
2. **Add tests** - Ensure coverage for new code
3. **Run all checks**:
   ```bash
   python manage.py test
   mypy .
   flake8 .
   black --check .
   ```
4. **Update CHANGELOG** if applicable
5. **Request review** from maintainers
6. **Address feedback** promptly
7. **Squash commits** if requested

## Testing Guidelines

### Unit Tests

Test individual functions and methods:

```python
def test_calculate_budget_utilization(self):
    """Test budget utilization calculation."""
    campaign = create_test_campaign(daily_budget=100)
    add_spend(campaign, 75)
    
    utilization = campaign.get_daily_utilization()
    self.assertEqual(utilization, 75.0)
```

### Integration Tests

Test component interactions:

```python
def test_spend_record_updates_aggregates(self):
    """Test that spend records update aggregates correctly."""
    campaign = create_test_campaign()
    
    # Record spend
    record_spend(campaign.id, Decimal("50.00"))
    
    # Check aggregates
    daily_agg = DailySpendAggregate.objects.get(
        campaign=campaign,
        date=date.today()
    )
    self.assertEqual(daily_agg.total_spend, Decimal("50.00"))
```

### Test Data

Use factories or fixtures for test data:

```python
# tests/factories.py
import factory
from campaigns.models import Brand, Campaign

class BrandFactory(factory.django.DjangoModelFactory):
    class Meta:
        model = Brand
    
    name = factory.Sequence(lambda n: f"Brand {n}")

class CampaignFactory(factory.django.DjangoModelFactory):
    class Meta:
        model = Campaign
    
    brand = factory.SubFactory(BrandFactory)
    name = factory.Sequence(lambda n: f"Campaign {n}")
    daily_budget = Decimal("1000.00")
    monthly_budget = Decimal("30000.00")
```

## Documentation

### Code Documentation

- Add docstrings to all public functions/classes
- Include type hints
- Add inline comments for complex logic
- Update README for new features

### API Documentation

Document new endpoints:

```python
class CampaignViewSet(viewsets.ModelViewSet):
    """
    API endpoint for managing campaigns.
    
    list:
        Return all campaigns for the authenticated user.
        
    create:
        Create a new campaign.
        
        Parameters:
            - name: Campaign name
            - brand_id: Associated brand UUID
            - daily_budget: Daily budget in USD
            - monthly_budget: Monthly budget in USD
    """
```

## Questions?

Feel free to:
- Open an issue for discussion
- Join our Discord/Slack community
- Email the maintainers

Thank you for contributing! 🎉