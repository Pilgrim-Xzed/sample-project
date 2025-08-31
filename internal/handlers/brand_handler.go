package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"budget-management/internal/models"
	"budget-management/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// BrandHandler handles brand-related HTTP requests
type BrandHandler struct {
	brandRepo *repository.BrandRepository
	logger    *logrus.Logger
}

// NewBrandHandler creates a new brand handler
func NewBrandHandler(brandRepo *repository.BrandRepository, logger *logrus.Logger) *BrandHandler {
	return &BrandHandler{
		brandRepo: brandRepo,
		logger:    logger,
	}
}

// List retrieves all brands with pagination
func (h *BrandHandler) List(c *gin.Context) {
	offset, _ := c.Get("offset")
	limit, _ := c.Get("limit")

	brands, err := h.brandRepo.List(offset.(int), limit.(int))
	if err != nil {
		h.logger.Errorf("Failed to list brands: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve brands",
		})
		return
	}

	// Get total count
	totalCount, err := h.brandRepo.Count()
	if err != nil {
		h.logger.Errorf("Failed to count brands: %v", err)
		totalCount = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"data": brands,
		"meta": gin.H{
			"total":  totalCount,
			"offset": offset,
			"limit":  limit,
		},
	})
}

// Get retrieves a brand by ID
func (h *BrandHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid brand ID",
		})
		return
	}

	brand, err := h.brandRepo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Brand not found",
			})
		} else {
			h.logger.Errorf("Failed to get brand %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve brand",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": brand,
	})
}

// Create creates a new brand
func (h *BrandHandler) Create(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required,min=1,max=255"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Check if brand with same name already exists
	exists, err := h.brandRepo.ExistsByName(req.Name)
	if err != nil {
		h.logger.Errorf("Failed to check brand existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create brand",
		})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Brand with this name already exists",
		})
		return
	}

	brand := &models.Brand{
		Name: req.Name,
	}

	if err := h.brandRepo.Create(brand); err != nil {
		h.logger.Errorf("Failed to create brand: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create brand",
		})
		return
	}

	h.logger.Infof("Created brand: %s (ID: %s)", brand.Name, brand.ID)

	c.JSON(http.StatusCreated, gin.H{
		"data": brand,
	})
}

// Update updates an existing brand
func (h *BrandHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid brand ID",
		})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required,min=1,max=255"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Get existing brand
	brand, err := h.brandRepo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Brand not found",
			})
		} else {
			h.logger.Errorf("Failed to get brand %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update brand",
			})
		}
		return
	}

	// Check if another brand with same name already exists
	if brand.Name != req.Name {
		exists, err := h.brandRepo.ExistsByName(req.Name)
		if err != nil {
			h.logger.Errorf("Failed to check brand existence: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update brand",
			})
			return
		}

		if exists {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Brand with this name already exists",
			})
			return
		}
	}

	// Update brand
	brand.Name = req.Name

	if err := h.brandRepo.Update(brand); err != nil {
		h.logger.Errorf("Failed to update brand %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update brand",
		})
		return
	}

	h.logger.Infof("Updated brand: %s (ID: %s)", brand.Name, brand.ID)

	c.JSON(http.StatusOK, gin.H{
		"data": brand,
	})
}

// Delete deletes a brand
func (h *BrandHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid brand ID",
		})
		return
	}

	// Check if brand exists
	exists, err := h.brandRepo.Exists(id)
	if err != nil {
		h.logger.Errorf("Failed to check brand existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete brand",
		})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Brand not found",
		})
		return
	}

	if err := h.brandRepo.Delete(id); err != nil {
		h.logger.Errorf("Failed to delete brand %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete brand",
		})
		return
	}

	h.logger.Infof("Deleted brand: %s", id)

	c.JSON(http.StatusNoContent, nil)
}

// GetWithCampaigns retrieves a brand with its campaigns
func (h *BrandHandler) GetWithCampaigns(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid brand ID",
		})
		return
	}

	brand, err := h.brandRepo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Brand not found",
			})
		} else {
			h.logger.Errorf("Failed to get brand %s: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve brand",
			})
		}
		return
	}

	// Note: In a real implementation, you might want to load campaigns here
	// For now, we'll just return the brand
	c.JSON(http.StatusOK, gin.H{
		"data": brand,
	})
}

// Search searches brands by name
func (h *BrandHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Search query is required",
		})
		return
	}

	offsetStr := c.Query("offset")
	limitStr := c.Query("limit")

	offset := 0
	limit := 20

	if offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// For now, we'll use a simple LIKE search
	// In a production system, you might want to use full-text search
	brands, err := h.brandRepo.List(offset, limit)
	if err != nil {
		h.logger.Errorf("Failed to search brands: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search brands",
		})
		return
	}

	// Filter brands by name (simple implementation)
	var filteredBrands []models.Brand
	for _, brand := range brands {
		if len(brand.Name) >= len(query) && 
		   strings.Contains(strings.ToLower(brand.Name), strings.ToLower(query)) {
			filteredBrands = append(filteredBrands, brand)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": filteredBrands,
		"meta": gin.H{
			"query":  query,
			"offset": offset,
			"limit":  limit,
			"count":  len(filteredBrands),
		},
	})
}