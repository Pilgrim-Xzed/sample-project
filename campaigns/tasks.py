from __future__ import annotations
from typing import Optional, Final
from decimal import Decimal
from datetime import datetime, timedelta
import logging

from celery import shared_task
from celery.utils.log import get_task_logger
from django.db import transaction
from django.db.models import F
from django.utils import timezone
import redis

from .models import (
    Campaign,
    SpendRecord,
    DailySpendAggregate,
    MonthlySpendAggregate,
)

logger: logging.Logger = get_task_logger(__name__)
redis_client: redis.Redis = redis.Redis(host='localhost', port=6379, db=1)

LOCK_EXPIRE: Final[int] = 60 * 5  # 5 minutes


def acquire_lock(lock_id: str, expire: int = LOCK_EXPIRE) -> bool:
    """Acquire a distributed lock using Redis."""
    identifier = f"lock:{lock_id}"
    return bool(redis_client.set(identifier, "1", nx=True, ex=expire))


def release_lock(lock_id: str) -> None:
    """Release a distributed lock."""
    identifier = f"lock:{lock_id}"
    redis_client.delete(identifier)


@shared_task(bind=True, max_retries=3)
def record_spend(
    self,
    campaign_id: str,
    amount: str,
    spend_date_str: Optional[str] = None
) -> dict[str, str]:
    """
    Record spend for a campaign and update aggregates.
    
    Args:
        campaign_id: UUID of the campaign
        amount: Spend amount as string (for JSON serialization)
        spend_date_str: Optional date string (YYYY-MM-DD format)
    
    Returns:
        Dictionary with status and message
    """
    lock_id = f"spend:{campaign_id}"
    
    if not acquire_lock(lock_id):
        self.retry(countdown=5)
    
    try:
        with transaction.atomic():
            try:
                campaign = Campaign.objects.select_for_update().get(id=campaign_id)
            except Campaign.DoesNotExist:
                return {"status": "error", "message": f"Campaign {campaign_id} not found"}
            
            amount_decimal = Decimal(amount)
            spend_date = (
                datetime.strptime(spend_date_str, "%Y-%m-%d").date()
                if spend_date_str
                else timezone.now().date()
            )
            
            # Create spend record
            SpendRecord.objects.create(
                campaign=campaign,
                amount=amount_decimal,
                spend_date=spend_date
            )
            
            # Update daily aggregate
            daily_aggregate, _ = DailySpendAggregate.objects.get_or_create(
                campaign=campaign,
                date=spend_date,
                defaults={"total_spend": Decimal("0.00")}
            )
            daily_aggregate.total_spend = F("total_spend") + amount_decimal
            daily_aggregate.save(update_fields=["total_spend", "last_updated"])
            
            # Update monthly aggregate
            monthly_aggregate, _ = MonthlySpendAggregate.objects.get_or_create(
                campaign=campaign,
                year=spend_date.year,
                month=spend_date.month,
                defaults={"total_spend": Decimal("0.00")}
            )
            monthly_aggregate.total_spend = F("total_spend") + amount_decimal
            monthly_aggregate.save(update_fields=["total_spend", "last_updated"])
            
            # Check budget limits
            check_budget_limits.delay(campaign_id)
            
            logger.info(f"Recorded spend of ${amount} for campaign {campaign.name}")
            return {
                "status": "success",
                "message": f"Spend of ${amount} recorded for {campaign.name}"
            }
            
    finally:
        release_lock(lock_id)


@shared_task
def check_budget_limits(campaign_id: str) -> dict[str, str]:
    """Check and enforce budget limits for a campaign."""
    lock_id = f"budget_check:{campaign_id}"
    
    if not acquire_lock(lock_id, expire=30):
        return {"status": "skipped", "message": "Another budget check in progress"}
    
    try:
        with transaction.atomic():
            try:
                campaign = Campaign.objects.select_for_update().get(id=campaign_id)
            except Campaign.DoesNotExist:
                return {"status": "error", "message": f"Campaign {campaign_id} not found"}
            
            if campaign.is_paused_by_budget:
                return {"status": "skipped", "message": "Campaign already paused"}
            
            today = timezone.now().date()
            
            # Check daily budget
            daily_aggregate = campaign.daily_aggregates.filter(date=today).first()
            daily_spend = daily_aggregate.total_spend if daily_aggregate else Decimal("0.00")
            
            if daily_spend >= campaign.daily_budget:
                campaign.pause_for_budget()
                logger.warning(
                    f"Campaign {campaign.name} paused - daily budget exceeded: "
                    f"${daily_spend} >= ${campaign.daily_budget}"
                )
                return {
                    "status": "paused",
                    "message": f"Campaign paused - daily budget exceeded"
                }
            
            # Check monthly budget
            now = timezone.now()
            monthly_aggregate = campaign.monthly_aggregates.filter(
                year=now.year,
                month=now.month
            ).first()
            monthly_spend = monthly_aggregate.total_spend if monthly_aggregate else Decimal("0.00")
            
            if monthly_spend >= campaign.monthly_budget:
                campaign.pause_for_budget()
                logger.warning(
                    f"Campaign {campaign.name} paused - monthly budget exceeded: "
                    f"${monthly_spend} >= ${campaign.monthly_budget}"
                )
                return {
                    "status": "paused",
                    "message": f"Campaign paused - monthly budget exceeded"
                }
            
            return {"status": "ok", "message": "Budget limits not exceeded"}
            
    finally:
        release_lock(lock_id)


@shared_task
def daily_reset_task() -> dict[str, int]:
    """
    Reset campaigns that were paused due to daily budget limits.
    Runs at midnight daily.
    """
    logger.info("Starting daily reset task")
    
    yesterday = timezone.now().date() - timedelta(days=1)
    campaigns_to_check = Campaign.objects.filter(
        is_paused_by_budget=True
    ).select_related("brand")
    
    reactivated_count = 0
    
    for campaign in campaigns_to_check:
        # Check if campaign was paused due to daily budget
        daily_aggregate = campaign.daily_aggregates.filter(date=yesterday).first()
        
        if daily_aggregate and daily_aggregate.total_spend >= campaign.daily_budget:
            # Check if monthly budget still allows activation
            now = timezone.now()
            monthly_aggregate = campaign.monthly_aggregates.filter(
                year=now.year,
                month=now.month
            ).first()
            monthly_spend = monthly_aggregate.total_spend if monthly_aggregate else Decimal("0.00")
            
            if monthly_spend < campaign.monthly_budget:
                campaign.reactivate()
                reactivated_count += 1
                logger.info(f"Reactivated campaign {campaign.name} after daily reset")
    
    logger.info(f"Daily reset completed. Reactivated {reactivated_count} campaigns")
    return {"reactivated_campaigns": reactivated_count}


@shared_task
def monthly_reset_task() -> dict[str, int]:
    """
    Reset all budget-paused campaigns at the start of a new month.
    Runs at midnight on the first day of each month.
    """
    logger.info("Starting monthly reset task")
    
    campaigns_to_reactivate = Campaign.objects.filter(
        is_paused_by_budget=True
    ).select_related("brand")
    
    reactivated_count = 0
    
    for campaign in campaigns_to_reactivate:
        campaign.reactivate()
        reactivated_count += 1
        logger.info(f"Reactivated campaign {campaign.name} after monthly reset")
    
    logger.info(f"Monthly reset completed. Reactivated {reactivated_count} campaigns")
    return {"reactivated_campaigns": reactivated_count}


@shared_task
def dayparting_check_task() -> dict[str, dict[str, int]]:
    """
    Check and enforce dayparting schedules for all campaigns.
    Runs every minute.
    """
    current_time = timezone.now()
    current_day = current_time.weekday()  # 0-6 (Monday-Sunday)
    current_hour_minute = current_time.time()
    
    # Get all campaigns with dayparting schedules
    campaigns_with_schedules = Campaign.objects.filter(
        dayparting_schedules__isnull=False
    ).distinct().select_related("brand")
    
    activated_count = 0
    deactivated_count = 0
    
    for campaign in campaigns_with_schedules:
        # Skip if campaign is paused by budget
        if campaign.is_paused_by_budget:
            continue
        
        # Check if campaign should be active based on dayparting
        active_schedules = campaign.dayparting_schedules.filter(
            day_of_week=current_day,
            is_active=True,
            start_time__lte=current_hour_minute,
            end_time__gte=current_hour_minute
        )
        
        should_be_active = active_schedules.exists()
        
        # Update campaign status if needed
        if should_be_active and not campaign.is_active:
            campaign.is_active = True
            campaign.save(update_fields=["is_active", "updated_at"])
            activated_count += 1
            logger.info(f"Activated campaign {campaign.name} based on dayparting schedule")
        elif not should_be_active and campaign.is_active:
            campaign.is_active = False
            campaign.save(update_fields=["is_active", "updated_at"])
            deactivated_count += 1
            logger.info(f"Deactivated campaign {campaign.name} based on dayparting schedule")
    
    return {
        "status": {
            "activated": activated_count,
            "deactivated": deactivated_count
        }
    }


@shared_task
def periodic_budget_check_task() -> dict[str, int]:
    """
    Periodic check of all active campaigns for budget limits.
    Runs every 5 minutes.
    """
    active_campaigns = Campaign.objects.filter(
        is_active=True,
        is_paused_by_budget=False
    ).values_list("id", flat=True)
    
    checked_count = 0
    
    for campaign_id in active_campaigns:
        check_budget_limits.delay(str(campaign_id))
        checked_count += 1
    
    logger.info(f"Queued budget checks for {checked_count} active campaigns")
    return {"campaigns_checked": checked_count}