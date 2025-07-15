from __future__ import annotations
from decimal import Decimal
from typing import TYPE_CHECKING, Final
from django.db import models
from django.core.validators import MinValueValidator, MaxValueValidator
import uuid

if TYPE_CHECKING:
    pass


class Brand(models.Model):
    """Represents an advertising brand/client."""
    
    id: models.UUIDField = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False
    )
    name: models.CharField = models.CharField(
        max_length=255,
        unique=True,
        db_index=True
    )
    created_at: models.DateTimeField = models.DateTimeField(auto_now_add=True)
    updated_at: models.DateTimeField = models.DateTimeField(auto_now=True)
    
    class Meta:
        db_table: Final[str] = "brands"
        ordering: list[str] = ["name"]
        verbose_name: str = "Brand"
        verbose_name_plural: str = "Brands"
    
    def __str__(self) -> str:
        return self.name


class Campaign(models.Model):
    """Represents an advertising campaign with budget constraints."""
    
    id: models.UUIDField = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False
    )
    brand: models.ForeignKey = models.ForeignKey(
        Brand,
        on_delete=models.CASCADE,
        related_name="campaigns"
    )
    name: models.CharField = models.CharField(
        max_length=255,
        db_index=True
    )
    daily_budget: models.DecimalField = models.DecimalField(
        max_digits=10,
        decimal_places=2,
        validators=[MinValueValidator(Decimal("0.01"))]
    )
    monthly_budget: models.DecimalField = models.DecimalField(
        max_digits=12,
        decimal_places=2,
        validators=[MinValueValidator(Decimal("0.01"))]
    )
    is_active: models.BooleanField = models.BooleanField(
        default=True,
        db_index=True
    )
    is_paused_by_budget: models.BooleanField = models.BooleanField(
        default=False,
        db_index=True,
        help_text="Indicates if campaign is paused due to budget limits"
    )
    created_at: models.DateTimeField = models.DateTimeField(auto_now_add=True)
    updated_at: models.DateTimeField = models.DateTimeField(auto_now=True)
    
    class Meta:
        db_table: Final[str] = "campaigns"
        ordering: list[str] = ["-created_at"]
        verbose_name: str = "Campaign"
        verbose_name_plural: str = "Campaigns"
        unique_together: list[tuple[str, ...]] = [("brand", "name")]
        indexes: list[models.Index] = [
            models.Index(fields=["is_active", "is_paused_by_budget"]),
        ]
    
    def __str__(self) -> str:
        return f"{self.brand.name} - {self.name}"
    
    def pause_for_budget(self) -> None:
        """Pause the campaign due to budget constraints."""
        self.is_active = False
        self.is_paused_by_budget = True
        self.save(update_fields=["is_active", "is_paused_by_budget", "updated_at"])
    
    def reactivate(self) -> None:
        """Reactivate the campaign after budget reset."""
        self.is_active = True
        self.is_paused_by_budget = False
        self.save(update_fields=["is_active", "is_paused_by_budget", "updated_at"])


class DaypartingSchedule(models.Model):
    """Defines when a campaign should be active during the week."""
    
    DAYS_OF_WEEK: Final[list[tuple[int, str]]] = [
        (0, "Monday"),
        (1, "Tuesday"),
        (2, "Wednesday"),
        (3, "Thursday"),
        (4, "Friday"),
        (5, "Saturday"),
        (6, "Sunday"),
    ]
    
    id: models.UUIDField = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False
    )
    campaign: models.ForeignKey = models.ForeignKey(
        Campaign,
        on_delete=models.CASCADE,
        related_name="dayparting_schedules"
    )
    day_of_week: models.IntegerField = models.IntegerField(
        choices=DAYS_OF_WEEK,
        validators=[MinValueValidator(0), MaxValueValidator(6)]
    )
    start_time: models.TimeField = models.TimeField()
    end_time: models.TimeField = models.TimeField()
    is_active: models.BooleanField = models.BooleanField(default=True)
    created_at: models.DateTimeField = models.DateTimeField(auto_now_add=True)
    updated_at: models.DateTimeField = models.DateTimeField(auto_now=True)
    
    class Meta:
        db_table: Final[str] = "dayparting_schedules"
        ordering: list[str] = ["day_of_week", "start_time"]
        verbose_name: str = "Dayparting Schedule"
        verbose_name_plural: str = "Dayparting Schedules"
        indexes: list[models.Index] = [
            models.Index(fields=["campaign", "day_of_week", "is_active"]),
        ]
    
    def __str__(self) -> str:
        return f"{self.campaign.name} - {self.get_day_of_week_display()} {self.start_time}-{self.end_time}"


class SpendRecord(models.Model):
    """Individual spend record for a campaign."""
    
    id: models.UUIDField = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False
    )
    campaign: models.ForeignKey = models.ForeignKey(
        Campaign,
        on_delete=models.CASCADE,
        related_name="spend_records"
    )
    amount: models.DecimalField = models.DecimalField(
        max_digits=10,
        decimal_places=2,
        validators=[MinValueValidator(Decimal("0.01"))]
    )
    spend_date: models.DateField = models.DateField(db_index=True)
    created_at: models.DateTimeField = models.DateTimeField(auto_now_add=True)
    
    class Meta:
        db_table: Final[str] = "spend_records"
        ordering: list[str] = ["-spend_date", "-created_at"]
        verbose_name: str = "Spend Record"
        verbose_name_plural: str = "Spend Records"
        indexes: list[models.Index] = [
            models.Index(fields=["campaign", "spend_date"]),
        ]
    
    def __str__(self) -> str:
        return f"{self.campaign.name} - ${self.amount} on {self.spend_date}"


class DailySpendAggregate(models.Model):
    """Aggregated daily spend for efficient budget checking."""
    
    campaign: models.ForeignKey = models.ForeignKey(
        Campaign,
        on_delete=models.CASCADE,
        related_name="daily_aggregates"
    )
    date: models.DateField = models.DateField(db_index=True)
    total_spend: models.DecimalField = models.DecimalField(
        max_digits=10,
        decimal_places=2,
        default=Decimal("0.00")
    )
    last_updated: models.DateTimeField = models.DateTimeField(auto_now=True)
    
    class Meta:
        db_table: Final[str] = "daily_spend_aggregates"
        unique_together: list[tuple[str, ...]] = [("campaign", "date")]
        ordering: list[str] = ["-date"]
        verbose_name: str = "Daily Spend Aggregate"
        verbose_name_plural: str = "Daily Spend Aggregates"
        indexes: list[models.Index] = [
            models.Index(fields=["date", "campaign"]),
        ]
    
    def __str__(self) -> str:
        return f"{self.campaign.name} - {self.date}: ${self.total_spend}"


class MonthlySpendAggregate(models.Model):
    """Aggregated monthly spend for efficient budget checking."""
    
    campaign: models.ForeignKey = models.ForeignKey(
        Campaign,
        on_delete=models.CASCADE,
        related_name="monthly_aggregates"
    )
    year: models.IntegerField = models.IntegerField(
        validators=[MinValueValidator(2020)]
    )
    month: models.IntegerField = models.IntegerField(
        validators=[MinValueValidator(1), MaxValueValidator(12)]
    )
    total_spend: models.DecimalField = models.DecimalField(
        max_digits=12,
        decimal_places=2,
        default=Decimal("0.00")
    )
    last_updated: models.DateTimeField = models.DateTimeField(auto_now=True)
    
    class Meta:
        db_table: Final[str] = "monthly_spend_aggregates"
        unique_together: list[tuple[str, ...]] = [("campaign", "year", "month")]
        ordering: list[str] = ["-year", "-month"]
        verbose_name: str = "Monthly Spend Aggregate"
        verbose_name_plural: str = "Monthly Spend Aggregates"
        indexes: list[models.Index] = [
            models.Index(fields=["year", "month", "campaign"]),
        ]
    
    def __str__(self) -> str:
        return f"{self.campaign.name} - {self.year}/{self.month:02d}: ${self.total_spend}"
