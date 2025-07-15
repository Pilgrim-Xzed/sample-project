from __future__ import annotations
from typing import Any
from django.contrib import admin
from django.db.models import QuerySet
from django.http import HttpRequest
from django.utils import timezone
from django.utils.html import format_html
from decimal import Decimal

from .models import (
    Brand,
    Campaign,
    DaypartingSchedule,
    SpendRecord,
    DailySpendAggregate,
    MonthlySpendAggregate,
)


@admin.register(Brand)
class BrandAdmin(admin.ModelAdmin):
    """Admin interface for Brand model."""
    
    list_display: list[str] = ["name", "created_at", "updated_at"]
    search_fields: list[str] = ["name"]
    ordering: list[str] = ["name"]
    readonly_fields: list[str] = ["id", "created_at", "updated_at"]


class DaypartingScheduleInline(admin.TabularInline):
    """Inline admin for DaypartingSchedule within Campaign."""
    
    model = DaypartingSchedule
    extra: int = 1
    fields: list[str] = ["day_of_week", "start_time", "end_time", "is_active"]


@admin.register(Campaign)
class CampaignAdmin(admin.ModelAdmin):
    """Admin interface for Campaign model."""
    
    list_display: list[str] = [
        "name",
        "brand",
        "daily_budget",
        "monthly_budget",
        "is_active_display",
        "is_paused_by_budget_display",
        "daily_spend_display",
        "monthly_spend_display",
        "created_at",
    ]
    list_filter: list[str | tuple[str, type[admin.SimpleListFilter]]] = [
        "is_active",
        "is_paused_by_budget",
        "brand",
        "created_at",
    ]
    search_fields: list[str] = ["name", "brand__name"]
    ordering: list[str] = ["-created_at"]
    readonly_fields: list[str] = [
        "id",
        "created_at",
        "updated_at",
        "daily_spend_display",
        "monthly_spend_display",
    ]
    inlines: list[type[admin.TabularInline]] = [DaypartingScheduleInline]
    
    fieldsets: list[tuple[str, dict[str, Any]]] = [
        ("Basic Information", {
            "fields": ("id", "brand", "name")
        }),
        ("Budget Settings", {
            "fields": ("daily_budget", "monthly_budget", "daily_spend_display", "monthly_spend_display")
        }),
        ("Status", {
            "fields": ("is_active", "is_paused_by_budget")
        }),
        ("Timestamps", {
            "fields": ("created_at", "updated_at"),
            "classes": ("collapse",)
        }),
    ]
    
    def is_active_display(self, obj: Campaign) -> str:
        """Display active status with color coding."""
        if obj.is_active:
            return format_html('<span style="color: green;">✓ Active</span>')
        return format_html('<span style="color: red;">✗ Inactive</span>')
    is_active_display.short_description = "Status"  # type: ignore
    
    def is_paused_by_budget_display(self, obj: Campaign) -> str:
        """Display budget pause status with color coding."""
        if obj.is_paused_by_budget:
            return format_html('<span style="color: orange;">⚠ Budget Paused</span>')
        return format_html('<span style="color: green;">✓ OK</span>')
    is_paused_by_budget_display.short_description = "Budget Status"  # type: ignore
    
    def daily_spend_display(self, obj: Campaign) -> str:
        """Display today's spend vs daily budget."""
        today = timezone.now().date()
        daily_aggregate = obj.daily_aggregates.filter(date=today).first()
        daily_spend = daily_aggregate.total_spend if daily_aggregate else Decimal("0.00")
        
        color = "red" if daily_spend >= obj.daily_budget else "green"
        return format_html(
            '<span style="color: {};">${:.2f} / ${:.2f}</span>',
            color,
            daily_spend,
            obj.daily_budget
        )
    daily_spend_display.short_description = "Today's Spend"  # type: ignore
    
    def monthly_spend_display(self, obj: Campaign) -> str:
        """Display this month's spend vs monthly budget."""
        now = timezone.now()
        monthly_aggregate = obj.monthly_aggregates.filter(
            year=now.year,
            month=now.month
        ).first()
        monthly_spend = monthly_aggregate.total_spend if monthly_aggregate else Decimal("0.00")
        
        color = "red" if monthly_spend >= obj.monthly_budget else "green"
        return format_html(
            '<span style="color: {};">${:.2f} / ${:.2f}</span>',
            color,
            monthly_spend,
            obj.monthly_budget
        )
    monthly_spend_display.short_description = "This Month's Spend"  # type: ignore
    
    actions: list[str] = ["pause_campaigns", "reactivate_campaigns"]
    
    def pause_campaigns(self, request: HttpRequest, queryset: QuerySet[Campaign]) -> None:
        """Admin action to pause selected campaigns."""
        for campaign in queryset:
            campaign.pause_for_budget()
        self.message_user(request, f"{queryset.count()} campaigns paused.")
    pause_campaigns.short_description = "Pause selected campaigns"  # type: ignore
    
    def reactivate_campaigns(self, request: HttpRequest, queryset: QuerySet[Campaign]) -> None:
        """Admin action to reactivate selected campaigns."""
        for campaign in queryset:
            campaign.reactivate()
        self.message_user(request, f"{queryset.count()} campaigns reactivated.")
    reactivate_campaigns.short_description = "Reactivate selected campaigns"  # type: ignore


@admin.register(DaypartingSchedule)
class DaypartingScheduleAdmin(admin.ModelAdmin):
    """Admin interface for DaypartingSchedule model."""
    
    list_display: list[str] = [
        "campaign",
        "get_day_of_week_display",
        "start_time",
        "end_time",
        "is_active",
    ]
    list_filter: list[str] = ["day_of_week", "is_active", "campaign__brand"]
    search_fields: list[str] = ["campaign__name", "campaign__brand__name"]
    ordering: list[str] = ["campaign", "day_of_week", "start_time"]


@admin.register(SpendRecord)
class SpendRecordAdmin(admin.ModelAdmin):
    """Admin interface for SpendRecord model."""
    
    list_display: list[str] = [
        "campaign",
        "amount",
        "spend_date",
        "created_at",
    ]
    list_filter: list[str | tuple[str, type[admin.SimpleListFilter]]] = [
        "spend_date",
        "campaign__brand",
        "created_at",
    ]
    search_fields: list[str] = ["campaign__name", "campaign__brand__name"]
    ordering: list[str] = ["-spend_date", "-created_at"]
    readonly_fields: list[str] = ["id", "created_at"]
    date_hierarchy: str = "spend_date"
    
    def get_queryset(self, request: HttpRequest) -> QuerySet[SpendRecord]:
        """Optimize queryset with select_related."""
        return super().get_queryset(request).select_related("campaign", "campaign__brand")


@admin.register(DailySpendAggregate)
class DailySpendAggregateAdmin(admin.ModelAdmin):
    """Admin interface for DailySpendAggregate model."""
    
    list_display: list[str] = [
        "campaign",
        "date",
        "total_spend",
        "budget_utilization",
        "last_updated",
    ]
    list_filter: list[str | tuple[str, type[admin.SimpleListFilter]]] = [
        "date",
        "campaign__brand",
    ]
    search_fields: list[str] = ["campaign__name", "campaign__brand__name"]
    ordering: list[str] = ["-date", "campaign"]
    readonly_fields: list[str] = ["last_updated"]
    date_hierarchy: str = "date"
    
    def budget_utilization(self, obj: DailySpendAggregate) -> str:
        """Display budget utilization percentage."""
        utilization = (obj.total_spend / obj.campaign.daily_budget) * 100
        color = "red" if utilization >= 100 else "orange" if utilization >= 80 else "green"
        return format_html(
            '<span style="color: {};">{:.1f}%</span>',
            color,
            utilization
        )
    budget_utilization.short_description = "Budget Used"  # type: ignore
    
    def get_queryset(self, request: HttpRequest) -> QuerySet[DailySpendAggregate]:
        """Optimize queryset with select_related."""
        return super().get_queryset(request).select_related("campaign", "campaign__brand")


@admin.register(MonthlySpendAggregate)
class MonthlySpendAggregateAdmin(admin.ModelAdmin):
    """Admin interface for MonthlySpendAggregate model."""
    
    list_display: list[str] = [
        "campaign",
        "year_month_display",
        "total_spend",
        "budget_utilization",
        "last_updated",
    ]
    list_filter: list[str | tuple[str, type[admin.SimpleListFilter]]] = [
        "year",
        "month",
        "campaign__brand",
    ]
    search_fields: list[str] = ["campaign__name", "campaign__brand__name"]
    ordering: list[str] = ["-year", "-month", "campaign"]
    readonly_fields: list[str] = ["last_updated"]
    
    def year_month_display(self, obj: MonthlySpendAggregate) -> str:
        """Display year and month in readable format."""
        return f"{obj.year}/{obj.month:02d}"
    year_month_display.short_description = "Period"  # type: ignore
    
    def budget_utilization(self, obj: MonthlySpendAggregate) -> str:
        """Display budget utilization percentage."""
        utilization = (obj.total_spend / obj.campaign.monthly_budget) * 100
        color = "red" if utilization >= 100 else "orange" if utilization >= 80 else "green"
        return format_html(
            '<span style="color: {};">{:.1f}%</span>',
            color,
            utilization
        )
    budget_utilization.short_description = "Budget Used"  # type: ignore
    
    def get_queryset(self, request: HttpRequest) -> QuerySet[MonthlySpendAggregate]:
        """Optimize queryset with select_related."""
        return super().get_queryset(request).select_related("campaign", "campaign__brand")
