package handlers

import (
	"net/http"
	"strconv"

	"budget-management/internal/models"
	"budget-management/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// CampaignHandler handles campaign-related HTTP requests
type CampaignHandler struct {
	campaignService *services.CampaignService
	logger          *logrus.Logger
}

// NewCampaignHandler creates a new campaign handler
func NewCampaignHandler(campaignService *services.CampaignService, logger *logrus.Logger) *CampaignHandler {
	return &CampaignHandler{
		campaignService: campaignService,
		logger:          logger,
	}
}

// List retrieves campaigns with optional filtering
func (h *CampaignHandler) List(c *gin.Context) {
	offset, _ := c.Get("offset")
	limit, _ := c.Get("limit")

	// Parse optional filters
	var brandID *uuid.UUID
	if brandIDStr := c.Query("brand_id"); brandIDStr != "" {
		if id, err := uuid.Parse(brandIDStr); err == nil {
			brandID = &id
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid brand_id format",
			})
			return
		}
	}

	var isActive *bool
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		if active, err := strconv.ParseBool(isActiveStr); err == nil {
			isActive = &active
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid is_active format (must be true or false)",
			})
			return
		}
	}

	campaigns, err := h.campaignService.ListCampaigns(offset.(int), limit.(int), brandID, isActive)
	if err != nil {
		h.logger.Errorf("Failed to list campaigns: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve campaigns",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": campaigns,
		"meta": gin.H{
			"offset":    offset,
			"limit":     limit,
			"brand_id":  brandID,
			"is_active": isActive,
		},
	})
}

// Get retrieves a campaign by ID
func (h *CampaignHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	campaign, err := h.campaignService.GetCampaign(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Campaign not found",
			})
		} else {
			h.logger.Errorf("Failed to get campaign %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve campaign",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": campaign,
	})
}

// Create creates a new campaign
func (h *CampaignHandler) Create(c *gin.Context) {
	var req struct {
		BrandID       uuid.UUID `json:"brand_id" binding:"required"`
		Name          string    `json:"name" binding:"required,min=1,max=255"`
		DailyBudget   string    `json:"daily_budget" binding:"required"`
		MonthlyBudget string    `json:"monthly_budget" binding:"required"`
		IsActive      *bool     `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Parse budget amounts
	dailyBudget, err := decimal.NewFromString(req.DailyBudget)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid daily_budget format",
		})
		return
	}

	monthlyBudget, err := decimal.NewFromString(req.MonthlyBudget)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid monthly_budget format",
		})
		return
	}

	// Default is_active to true if not provided
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	campaign := &models.Campaign{
		BrandID:       req.BrandID,
		Name:          req.Name,
		DailyBudget:   dailyBudget,
		MonthlyBudget: monthlyBudget,
		IsActive:      isActive,
	}

	if err := h.campaignService.CreateCampaign(campaign); err != nil {
		h.logger.Errorf("Failed to create campaign: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create campaign",
		})
		return
	}

	h.logger.Infof("Created campaign: %s (ID: %s)", campaign.Name, campaign.ID)

	c.JSON(http.StatusCreated, gin.H{
		"data": campaign,
	})
}

// Update updates an existing campaign
func (h *CampaignHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	var req struct {
		Name          string `json:"name" binding:"required,min=1,max=255"`
		DailyBudget   string `json:"daily_budget" binding:"required"`
		MonthlyBudget string `json:"monthly_budget" binding:"required"`
		IsActive      *bool  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Get existing campaign
	campaign, err := h.campaignService.GetCampaign(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Campaign not found",
			})
		} else {
			h.logger.Errorf("Failed to get campaign %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update campaign",
			})
		}
		return
	}

	// Parse budget amounts
	dailyBudget, err := decimal.NewFromString(req.DailyBudget)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid daily_budget format",
		})
		return
	}

	monthlyBudget, err := decimal.NewFromString(req.MonthlyBudget)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid monthly_budget format",
		})
		return
	}

	// Update campaign fields
	campaign.Name = req.Name
	campaign.DailyBudget = dailyBudget
	campaign.MonthlyBudget = monthlyBudget

	if req.IsActive != nil {
		campaign.IsActive = *req.IsActive
	}

	if err := h.campaignService.UpdateCampaign(campaign); err != nil {
		h.logger.Errorf("Failed to update campaign %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update campaign",
		})
		return
	}

	h.logger.Infof("Updated campaign: %s (ID: %s)", campaign.Name, campaign.ID)

	c.JSON(http.StatusOK, gin.H{
		"data": campaign,
	})
}

// Delete deletes a campaign
func (h *CampaignHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	if err := h.campaignService.DeleteCampaign(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Campaign not found",
			})
		} else {
			h.logger.Errorf("Failed to delete campaign %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete campaign",
			})
		}
		return
	}

	h.logger.Infof("Deleted campaign: %s", id)

	c.JSON(http.StatusNoContent, nil)
}

// Pause manually pauses a campaign
func (h *CampaignHandler) Pause(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	if err := h.campaignService.PauseCampaign(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Campaign not found",
			})
		} else {
			h.logger.Errorf("Failed to pause campaign %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to pause campaign",
			})
		}
		return
	}

	h.logger.Infof("Paused campaign: %s", id)

	c.JSON(http.StatusOK, gin.H{
		"message": "Campaign paused successfully",
	})
}

// Activate manually activates a campaign
func (h *CampaignHandler) Activate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	if err := h.campaignService.ActivateCampaign(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Campaign not found",
			})
		} else {
			h.logger.Errorf("Failed to activate campaign %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to activate campaign",
			})
		}
		return
	}

	h.logger.Infof("Activated campaign: %s", id)

	c.JSON(http.StatusOK, gin.H{
		"message": "Campaign activated successfully",
	})
}

// GetSpend retrieves spend summary for a campaign
func (h *CampaignHandler) GetSpend(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	summary, err := h.campaignService.GetSpendSummary(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Campaign not found",
			})
		} else {
			h.logger.Errorf("Failed to get spend summary for campaign %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve spend summary",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": summary,
	})
}

// CheckBudget manually triggers a budget check for a campaign
func (h *CampaignHandler) CheckBudget(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	if err := h.campaignService.CheckBudgetLimits(c.Request.Context(), id); err != nil {
		h.logger.Errorf("Failed to check budget limits for campaign %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check budget limits",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Budget check completed",
	})
}