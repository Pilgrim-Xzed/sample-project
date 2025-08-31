package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"budget-management/internal/models"
	"budget-management/internal/services"
	"budget-management/internal/tasks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SpendHandler handles spend-related HTTP requests
type SpendHandler struct {
	spendService *services.SpendService
	taskQueue    *tasks.TaskQueue
	logger       *logrus.Logger
}

// NewSpendHandler creates a new spend handler
func NewSpendHandler(spendService *services.SpendService, taskQueue *tasks.TaskQueue, logger *logrus.Logger) *SpendHandler {
	return &SpendHandler{
		spendService: spendService,
		taskQueue:    taskQueue,
		logger:       logger,
	}
}

// Record records spend for a campaign
func (h *SpendHandler) Record(c *gin.Context) {
	var req struct {
		CampaignID uuid.UUID `json:"campaign_id" binding:"required"`
		Amount     string    `json:"amount" binding:"required"`
		SpendDate  *string   `json:"spend_date"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Parse and validate amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid amount format",
		})
		return
	}

	if err := h.spendService.ValidateSpendAmount(amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Parse spend date if provided
	var spendDate *time.Time
	if req.SpendDate != nil && *req.SpendDate != "" {
		date, err := time.Parse("2006-01-02", *req.SpendDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid spend_date format (expected YYYY-MM-DD)",
			})
			return
		}
		spendDate = &date
	}

	// Record spend directly (synchronous)
	if err := h.spendService.RecordSpend(c.Request.Context(), req.CampaignID, amount, spendDate); err != nil {
		h.logger.Errorf("Failed to record spend: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to record spend",
		})
		return
	}

	h.logger.Infof("Recorded spend of %s for campaign %s", amount, req.CampaignID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Spend recorded successfully",
		"data": gin.H{
			"campaign_id": req.CampaignID,
			"amount":      amount,
			"spend_date":  spendDate,
		},
	})
}

// RecordAsync records spend asynchronously using task queue
func (h *SpendHandler) RecordAsync(c *gin.Context) {
	var req struct {
		CampaignID uuid.UUID `json:"campaign_id" binding:"required"`
		Amount     string    `json:"amount" binding:"required"`
		SpendDate  *string   `json:"spend_date"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Validate amount format
	if _, err := decimal.NewFromString(req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid amount format",
		})
		return
	}

	// Validate spend date if provided
	spendDateStr := ""
	if req.SpendDate != nil && *req.SpendDate != "" {
		if _, err := time.Parse("2006-01-02", *req.SpendDate); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid spend_date format (expected YYYY-MM-DD)",
			})
			return
		}
		spendDateStr = *req.SpendDate
	}

	// Queue the task
	if err := h.taskQueue.EnqueueRecordSpend(req.CampaignID.String(), req.Amount, spendDateStr); err != nil {
		h.logger.Errorf("Failed to enqueue spend recording task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to queue spend recording",
		})
		return
	}

	h.logger.Infof("Queued spend recording of %s for campaign %s", req.Amount, req.CampaignID)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Spend recording queued successfully",
		"data": gin.H{
			"campaign_id": req.CampaignID,
			"amount":      req.Amount,
			"spend_date":  spendDateStr,
		},
	})
}

// GetCampaignSpend retrieves spend records for a campaign
func (h *SpendHandler) GetCampaignSpend(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	offset, _ := c.Get("offset")
	limit, _ := c.Get("limit")

	records, err := h.spendService.GetCampaignSpend(id, offset.(int), limit.(int))
	if err != nil {
		h.logger.Errorf("Failed to get spend records for campaign %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve spend records",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": records,
		"meta": gin.H{
			"campaign_id": id,
			"offset":      offset,
			"limit":       limit,
		},
	})
}

// GetDailySpend retrieves daily spend aggregates for a campaign
func (h *SpendHandler) GetDailySpend(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	// Parse date range parameters
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	var startDate, endDate time.Time

	if startDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", startDateStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid start_date format (expected YYYY-MM-DD)",
			})
			return
		} else {
			startDate = parsed
		}
	} else {
		// Default to 30 days ago
		startDate = time.Now().UTC().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	}

	if endDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", endDateStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid end_date format (expected YYYY-MM-DD)",
			})
			return
		} else {
			endDate = parsed
		}
	} else {
		// Default to today
		endDate = time.Now().UTC().Truncate(24 * time.Hour)
	}

	if startDate.After(endDate) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "start_date must be before or equal to end_date",
		})
		return
	}

	aggregates, err := h.spendService.GetDailySpend(id, startDate, endDate)
	if err != nil {
		h.logger.Errorf("Failed to get daily spend for campaign %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve daily spend",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": aggregates,
		"meta": gin.H{
			"campaign_id": id,
			"start_date":  startDate.Format("2006-01-02"),
			"end_date":    endDate.Format("2006-01-02"),
		},
	})
}

// GetMonthlySpend retrieves monthly spend aggregates for a campaign
func (h *SpendHandler) GetMonthlySpend(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	// Parse limit parameter
	limit := 12 // Default to 12 months
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 60 {
			limit = parsed
		}
	}

	aggregates, err := h.spendService.GetMonthlySpend(id, limit)
	if err != nil {
		h.logger.Errorf("Failed to get monthly spend for campaign %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve monthly spend",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": aggregates,
		"meta": gin.H{
			"campaign_id": id,
			"limit":       limit,
		},
	})
}

// GetSpendSummary retrieves comprehensive spend summary for a campaign
func (h *SpendHandler) GetSpendSummary(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	summary, err := h.spendService.GetSpendSummary(id)
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

// BulkRecord records multiple spend entries
func (h *SpendHandler) BulkRecord(c *gin.Context) {
	var req struct {
		Records []struct {
			CampaignID uuid.UUID `json:"campaign_id" binding:"required"`
			Amount     string    `json:"amount" binding:"required"`
			SpendDate  *string   `json:"spend_date"`
		} `json:"records" binding:"required,min=1,max=1000"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Validate and convert records
	var spendRecords []models.SpendRecord
	for i, record := range req.Records {
		// Parse amount
		amount, err := decimal.NewFromString(record.Amount)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid amount format",
				"details": fmt.Sprintf("Record %d has invalid amount: %s", i, record.Amount),
			})
			return
		}

		if err := h.spendService.ValidateSpendAmount(amount); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   err.Error(),
				"details": fmt.Sprintf("Record %d: %s", i, err.Error()),
			})
			return
		}

		// Parse spend date
		spendDate := time.Now().UTC().Truncate(24 * time.Hour)
		if record.SpendDate != nil && *record.SpendDate != "" {
			if parsed, err := time.Parse("2006-01-02", *record.SpendDate); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid spend_date format",
					"details": fmt.Sprintf("Record %d has invalid spend_date: %s", i, *record.SpendDate),
				})
				return
			} else {
				spendDate = parsed
			}
		}

		spendRecords = append(spendRecords, models.SpendRecord{
			CampaignID: record.CampaignID,
			Amount:     amount,
			SpendDate:  spendDate,
		})
	}

	// Bulk record spend
	if err := h.spendService.BulkRecordSpend(c.Request.Context(), spendRecords); err != nil {
		h.logger.Errorf("Failed to bulk record spend: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to record spend",
		})
		return
	}

	h.logger.Infof("Bulk recorded %d spend entries", len(spendRecords))

	c.JSON(http.StatusCreated, gin.H{
		"message": "Spend records created successfully",
		"data": gin.H{
			"records_created": len(spendRecords),
		},
	})
}