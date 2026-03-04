package repository

import "gorm.io/gorm"

// IRepository defines the interface for generic repository operations
type IRepository[T any] interface {
	Create(entity *T) error
	GetByID(id int) (*T, error)
	Update(entity *T) error
	Delete(id int) error
	GetAll(page, pageSize int) ([]T, int, error)
	FindByCondition(fieldName string, fieldValue interface{}) ([]T, error)
	Count() (int64, error)
}

// GenericRepository implements the IRepository interface
type GenericRepository[T any] struct {
	db *gorm.DB
}

// NewGenericRepository creates a new instance of GenericRepository
func NewGenericRepository[T any](db *gorm.DB) IRepository[T] {
	return &GenericRepository[T]{db: db}
}

// Create creates a new entity
func (r *GenericRepository[T]) Create(entity *T) error {
	return r.db.Create(entity).Error
}

// GetByID retrieves an entity by its ID
func (r *GenericRepository[T]) GetByID(id int) (*T, error) {
	var entity T
	err := r.db.First(&entity, id).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// Update updates an entity
func (r *GenericRepository[T]) Update(entity *T) error {
	return r.db.Save(entity).Error
}

// Delete deletes an entity by its ID
func (r *GenericRepository[T]) Delete(id int) error {
	return r.db.Delete(new(T), id).Error
}

// GetAll retrieves all entities with pagination
func (r *GenericRepository[T]) GetAll(page, pageSize int) ([]T, int, error) {
	var entities []T
	var total int64

	// Get total count
	if err := r.db.Model(new(T)).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	err := r.db.Offset(offset).Limit(pageSize).Find(&entities).Error
	if err != nil {
		return nil, 0, err
	}

	return entities, int(total), nil
}

// FindByCondition finds entities by a specific condition
func (r *GenericRepository[T]) FindByCondition(fieldName string, fieldValue interface{}) ([]T, error) {
	var entities []T
	err := r.db.Where(fieldName+" = ?", fieldValue).Find(&entities).Error
	return entities, err
}

// Count returns the total count of entities
func (r *GenericRepository[T]) Count() (int64, error) {
	var count int64
	err := r.db.Model(new(T)).Count(&count).Error
	return count, err
}
