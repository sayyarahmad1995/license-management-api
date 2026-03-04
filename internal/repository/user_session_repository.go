package repository

import (
	"time"

	"license-management-api/internal/models"

	"gorm.io/gorm"
)

type IUserSessionRepository interface {
	Create(session *models.UserSession) error
	GetBySessionID(sessionID string) (*models.UserSession, error)
	GetByUserID(userID int) ([]*models.UserSession, error)
	GetActiveSessions(userID int) ([]*models.UserSession, error)
	Update(session *models.UserSession) error
	Delete(id int64) error
	DeleteBySessionID(sessionID string) error
	DeleteByUserID(userID int) error
	DeleteExpiredSessions() error
}

type userSessionRepository struct {
	db *gorm.DB
}

func NewUserSessionRepository(db *gorm.DB) IUserSessionRepository {
	return &userSessionRepository{db: db}
}

// Create creates a new user session
func (r *userSessionRepository) Create(session *models.UserSession) error {
	return r.db.Create(session).Error
}

// GetBySessionID retrieves session by session ID
func (r *userSessionRepository) GetBySessionID(sessionID string) (*models.UserSession, error) {
	var session *models.UserSession
	err := r.db.Where("session_id = ?", sessionID).First(&session).Error
	return session, err
}

// GetByUserID retrieves all sessions for a user
func (r *userSessionRepository) GetByUserID(userID int) ([]*models.UserSession, error) {
	var sessions []*models.UserSession
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

// GetActiveSessions retrieves all active (non-revoked, non-expired) sessions for a user
func (r *userSessionRepository) GetActiveSessions(userID int) ([]*models.UserSession, error) {
	var sessions []*models.UserSession
	now := time.Now().UTC()
	err := r.db.Where("user_id = ? AND expires_at > ? AND is_revoked = ?", userID, now, false).
		Order("created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

// Update updates a user session
func (r *userSessionRepository) Update(session *models.UserSession) error {
	return r.db.Save(session).Error
}

// Delete deletes a session by ID
func (r *userSessionRepository) Delete(id int64) error {
	return r.db.Delete(&models.UserSession{}, id).Error
}

// DeleteBySessionID deletes a session by session ID
func (r *userSessionRepository) DeleteBySessionID(sessionID string) error {
	return r.db.Where("session_id = ?", sessionID).Delete(&models.UserSession{}).Error
}

// DeleteByUserID deletes all sessions for a user
func (r *userSessionRepository) DeleteByUserID(userID int) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.UserSession{}).Error
}

// DeleteExpiredSessions deletes all expired sessions
func (r *userSessionRepository) DeleteExpiredSessions() error {
	now := time.Now().UTC()
	return r.db.Where("expires_at < ?", now).Delete(&models.UserSession{}).Error
}
