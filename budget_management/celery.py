from __future__ import annotations
import os
from typing import Any
from celery import Celery
from celery.schedules import crontab

os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'budget_management.settings')

app: Celery = Celery('budget_management')

app.config_from_object('django.conf:settings', namespace='CELERY')

app.autodiscover_tasks()

app.conf.beat_schedule: dict[str, dict[str, Any]] = {
    'daily_reset_task': {
        'task': 'campaigns.tasks.daily_reset_task',
        'schedule': crontab(hour=0, minute=0),  # Midnight daily
        'options': {'queue': 'high_priority'},
    },
    'monthly_reset_task': {
        'task': 'campaigns.tasks.monthly_reset_task',
        'schedule': crontab(day_of_month=1, hour=0, minute=0),  # First day of month
        'options': {'queue': 'high_priority'},
    },
    'dayparting_check_task': {
        'task': 'campaigns.tasks.dayparting_check_task',
        'schedule': crontab(minute='*'),  # Every minute
        'options': {'queue': 'default'},
    },
    'periodic_budget_check_task': {
        'task': 'campaigns.tasks.periodic_budget_check_task',
        'schedule': crontab(minute='*/5'),  # Every 5 minutes
        'options': {'queue': 'default'},
    },
}