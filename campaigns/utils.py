from __future__ import annotations
from typing import Optional, TypedDict
from decimal import Decimal
from datetime import date
from django.db import transaction
from django.db.models import Sum

from .models import Campaign, SpendRecord, DailySpendAggregate, MonthlySpendAggregate


class SpendSummary(TypedDict):
    """Type definition for spend summary."""
    daily_spend: Decimal
    monthly_spend: Decimal
    daily_budget_remaining: Decimal
    monthly_budget_remaining: Decimal
    daily_utilization_percent: float
    monthly_utilization_percent: float


def get_campaign_spend_summary(
    campaign: Campaign,
    target_date: Optional[date] = None
) -> SpendSummary:
    """
    Get comprehensive spend summary for a campaign.
    
    Args:
        campaign: Campaign instance
        target_date: Date to check (defaults to today)
    
    Returns:
        Dictionary with spend summary information
    """
    from django.utils import timezone
    
    if target_date is None:
        target_date = timezone.now().date()
    
    # Get daily spend
    daily_aggregate = campaign.daily_aggregates.filter(date=target_date).first()
    daily_spend = daily_aggregate.total_spend if daily_aggregate else Decimal("0.00")
    
    # Get monthly spend
    monthly_aggregate = campaign.monthly_aggregates.filter(
        year=target_date.year,
        month=target_date.month
    ).first()
    monthly_spend = monthly_aggregate.total_spend if monthly_aggregate else Decimal("0.00")
    
    # Calculate remaining budgets
    daily_remaining = max(campaign.daily_budget - daily_spend, Decimal("0.00"))
    monthly_remaining = max(campaign.monthly_budget - monthly_spend, Decimal("0.00"))
    
    # Calculate utilization percentages
    daily_utilization = float((daily_spend / campaign.daily_budget) * 100) if campaign.daily_budget else 0.0
    monthly_utilization = float((monthly_spend / campaign.monthly_budget) * 100) if campaign.monthly_budget else 0.0
    
    return SpendSummary(
        daily_spend=daily_spend,
        monthly_spend=monthly_spend,
        daily_budget_remaining=daily_remaining,
        monthly_budget_remaining=monthly_remaining,
        daily_utilization_percent=daily_utilization,
        monthly_utilization_percent=monthly_utilization
    )


def bulk_record_spend(spend_data: list[dict[str, str | Decimal]]) -> dict[str, int]:
    """
    Bulk record spend for multiple campaigns.
    
    Args:
        spend_data: List of dictionaries with campaign_id, amount, and spend_date
    
    Returns:
        Dictionary with success and error counts
    """
    from .tasks import record_spend
    
    success_count = 0
    error_count = 0
    
    for data in spend_data:
        try:
            campaign_id = str(data["campaign_id"])
            amount = str(data["amount"])
            spend_date = data.get("spend_date")
            
            if spend_date and isinstance(spend_date, date):
                spend_date_str = spend_date.strftime("%Y-%m-%d")
            else:
                spend_date_str = None
            
            record_spend.delay(campaign_id, amount, spend_date_str)
            success_count += 1
        except Exception as e:
            error_count += 1
            print(f"Error recording spend: {e}")
    
    return {"success": success_count, "errors": error_count}


def recalculate_aggregates(campaign: Campaign) -> None:
    """
    Recalculate all spend aggregates for a campaign.
    Useful for data integrity checks and corrections.
    """
    with transaction.atomic():
        # Clear existing aggregates
        campaign.daily_aggregates.all().delete()
        campaign.monthly_aggregates.all().delete()
        
        # Recalculate daily aggregates
        daily_spends = (
            SpendRecord.objects
            .filter(campaign=campaign)
            .values("spend_date")
            .annotate(total=Sum("amount"))
        )
        
        for daily_spend in daily_spends:
            DailySpendAggregate.objects.create(
                campaign=campaign,
                date=daily_spend["spend_date"],
                total_spend=daily_spend["total"] or Decimal("0.00")
            )
        
        # Recalculate monthly aggregates
        monthly_spends = (
            SpendRecord.objects
            .filter(campaign=campaign)
            .values("spend_date__year", "spend_date__month")
            .annotate(total=Sum("amount"))
        )
        
        for monthly_spend in monthly_spends:
            MonthlySpendAggregate.objects.create(
                campaign=campaign,
                year=monthly_spend["spend_date__year"],
                month=monthly_spend["spend_date__month"],
                total_spend=monthly_spend["total"] or Decimal("0.00")
            )