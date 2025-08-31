package repository

import (
	"budget-management/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BrandRepository handles brand database operations
type BrandRepository struct {
	db *gorm.DB
}

// NewBrandRepository creates a new brand repository
func NewBrandRepository(db *gorm.DB) *BrandRepository {
	return &BrandRepository{db: db}
}

// Create creates a new brand
func (r *BrandRepository) Create(brand *models.Brand) error {
	return r.db.Create(brand).Error
}

// GetByID retrieves a brand by ID
func (r *BrandRepository) GetByID(id uuid.UUID) (*models.Brand, error) {
	var brand models.Brand
	err := r.db.Where("id = ?", id).First(&brand).Error
	if err != nil {
		return nil, err
	}
	return &brand, nil
}

// GetByName retrieves a brand by name
func (r *BrandRepository) GetByName(name string) (*models.Brand, error) {
	var brand models.Brand
	err := r.db.Where("name = ?", name).First(&brand).Error
	if err != nil {
		return nil, err
	}
	return &brand, nil
}

// List retrieves all brands with pagination
func (r *BrandRepository) List(offset, limit int) ([]models.Brand, error) {
	var brands []models.Brand
	err := r.db.Offset(offset).Limit(limit).Order("name").Find(&brands).Error
	return brands, err
}

// Update updates a brand
func (r *BrandRepository) Update(brand *models.Brand) error {
	return r.db.Save(brand).Error
}

// Delete deletes a brand by ID
func (r *BrandRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.Brand{}).Error
}

// Count returns total number of brands
func (r *BrandRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.Brand{}).Count(&count).Error
	return count, err
}

// Exists checks if a brand exists by ID
func (r *BrandRepository) Exists(id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.Brand{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

// ExistsByName checks if a brand exists by name
func (r *BrandRepository) ExistsByName(name string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Brand{}).Where("name = ?", name).Count(&count).Error
	return count > 0, err
}