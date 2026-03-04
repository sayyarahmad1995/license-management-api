package repository

import (
	"license-management-api/internal/models"
	"gorm.io/gorm"
)

// IUserRepository defines user-specific repository operations
type IUserRepository interface {
	IRepository[models.User]
	GetByEmail(email string) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	GetByEmailOrUsername(email, username string) (*models.User, error)
}

// UserRepository implements IUserRepository
type UserRepository struct {
	*GenericRepository[models.User]
	db *gorm.DB
}

// NewUserRepository creates a new instance of UserRepository
func NewUserRepository(db *gorm.DB) IUserRepository {
	return &UserRepository{
		GenericRepository: &GenericRepository[models.User]{db: db},
		db:                db,
	}
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmailOrUsername retrieves a user by email or username
func (r *UserRepository) GetByEmailOrUsername(email, username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ? OR username = ?", email, username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
