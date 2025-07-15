from __future__ import annotations
from typing import Any
from decimal import Decimal
from datetime import time
from django.core.management.base import BaseCommand, CommandError
from campaigns.models import Brand, Campaign, DaypartingSchedule


class Command(BaseCommand):
    """Create test data for the budget management system."""
    
    help: str = "Creates test brands, campaigns, and dayparting schedules"
    
    def handle(self, *args: Any, **options: Any) -> None:
        """Execute the command."""
        self.stdout.write("Creating test data...")
        
        # Create brands
        brand1, _ = Brand.objects.get_or_create(
            name="Nike",
            defaults={}
        )
        brand2, _ = Brand.objects.get_or_create(
            name="Adidas",
            defaults={}
        )
        
        # Create campaigns
        campaign1, created1 = Campaign.objects.get_or_create(
            brand=brand1,
            name="Summer Sale 2024",
            defaults={
                "daily_budget": Decimal("1000.00"),
                "monthly_budget": Decimal("25000.00"),
            }
        )
        
        campaign2, created2 = Campaign.objects.get_or_create(
            brand=brand1,
            name="Back to School",
            defaults={
                "daily_budget": Decimal("500.00"),
                "monthly_budget": Decimal("12000.00"),
            }
        )
        
        campaign3, created3 = Campaign.objects.get_or_create(
            brand=brand2,
            name="World Cup Campaign",
            defaults={
                "daily_budget": Decimal("2000.00"),
                "monthly_budget": Decimal("50000.00"),
            }
        )
        
        # Create dayparting schedules
        if created1:
            # Business hours on weekdays
            for day in range(5):  # Monday to Friday
                DaypartingSchedule.objects.create(
                    campaign=campaign1,
                    day_of_week=day,
                    start_time=time(9, 0),
                    end_time=time(17, 0),
                )
        
        if created2:
            # Peak hours schedule
            for day in range(7):  # All week
                DaypartingSchedule.objects.create(
                    campaign=campaign2,
                    day_of_week=day,
                    start_time=time(7, 0),
                    end_time=time(9, 0),
                )
                DaypartingSchedule.objects.create(
                    campaign=campaign2,
                    day_of_week=day,
                    start_time=time(17, 0),
                    end_time=time(21, 0),
                )
        
        if created3:
            # 24/7 schedule
            for day in range(7):
                DaypartingSchedule.objects.create(
                    campaign=campaign3,
                    day_of_week=day,
                    start_time=time(0, 0),
                    end_time=time(23, 59),
                )
        
        self.stdout.write(
            self.style.SUCCESS(
                f"Test data created successfully!\n"
                f"Brands: {Brand.objects.count()}\n"
                f"Campaigns: {Campaign.objects.count()}\n"
                f"Dayparting Schedules: {DaypartingSchedule.objects.count()}"
            )
        )