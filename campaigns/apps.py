from __future__ import annotations
from typing import Final
from django.apps import AppConfig


class CampaignsConfig(AppConfig):
    """Configuration for the campaigns app."""
    
    default_auto_field: Final[str] = "django.db.models.BigAutoField"
    name: Final[str] = "campaigns"
    verbose_name: Final[str] = "Campaign Management"
