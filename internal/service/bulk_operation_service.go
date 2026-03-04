package service

import (
	"fmt"
	"sync"
	"time"

	"license-management-api/internal/dto"
	"license-management-api/internal/errors"
	"license-management-api/internal/models"
	"license-management-api/internal/repository"

	"github.com/google/uuid"
)

const (
	// Threshold for async processing
	AsyncProcessingThreshold = 100

	// Job statuses
	JobStatusPending    = "pending"
	JobStatusInProgress = "in_progress"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
)

// BulkOperationService handles bulk operations on licenses
type BulkOperationService interface {
	BulkRevokeLicenses(licenseIDs []int, reason string, userID int) (*dto.BulkRevokeResponse, error)
	BulkRevokeLicensesAsync(licenseIDs []int, reason string, userID int) (*dto.AsyncBulkRevokeResponse, error)
	GetJobStatus(jobID string) (*dto.BulkJobStatus, error)
	CancelJob(jobID string) error
}

type bulkOperationService struct {
	licenseRepo repository.ILicenseRepository
	auditRepo   repository.IAuditLogRepository
	jobStore    *sync.Map // Thread-safe map for storing job statuses
}

// NewBulkOperationService creates a new bulk operation service
func NewBulkOperationService(
	licenseRepo repository.ILicenseRepository,
	auditRepo repository.IAuditLogRepository,
) BulkOperationService {
	return &bulkOperationService{
		licenseRepo: licenseRepo,
		auditRepo:   auditRepo,
		jobStore:    &sync.Map{},
	}
}

// BulkRevokeLicenses revokes multiple licenses in a single operation
func (bos *bulkOperationService) BulkRevokeLicenses(licenseIDs []int, reason string, userID int) (*dto.BulkRevokeResponse, error) {
	if len(licenseIDs) == 0 {
		return nil, errors.NewValidationError("At least one license must be specified for bulk revoke")
	}

	response := &dto.BulkRevokeResponse{
		TotalRequested: len(licenseIDs),
		SuccessCount:   0,
		FailureCount:   0,
		FailedIDs:      make([]int, 0),
		CompletedAt:    time.Now().UTC(),
	}

	// Process each license
	for _, licenseID := range licenseIDs {
		license, err := bos.licenseRepo.GetByID(licenseID)
		if err != nil {
			response.FailureCount++
			response.FailedIDs = append(response.FailedIDs, licenseID)
			continue
		}

		// Skip already revoked licenses
		if license.IsRevoked() {
			response.FailureCount++
			response.FailedIDs = append(response.FailedIDs, licenseID)
			continue
		}

		// Update license status to revoked
		now := time.Now().UTC()
		license.RevokedAt = &now

		if err := bos.licenseRepo.Update(license); err != nil {
			response.FailureCount++
			response.FailedIDs = append(response.FailedIDs, licenseID)
			continue
		}

		// Create audit log entry
		entityID := licenseID
		var userIDPtr *int
		userIDPtr = &userID

		auditLog := &models.AuditLog{
			UserID:     userIDPtr,
			Action:     "BULK_REVOKE_LICENSE",
			EntityType: "License",
			EntityID:   &entityID,
			Details:    &reason,
			Timestamp:  time.Now().UTC(),
		}

		if err := bos.auditRepo.Create(auditLog); err != nil {
			// Log the error but continue with the bulk operation
			fmt.Printf("Failed to create audit log for license %d: %v\n", licenseID, err)
		}

		response.SuccessCount++
	}

	return response, nil
}

// BulkRevokeLicensesAsync revokes multiple licenses asynchronously
// Returns job ID for tracking progress
func (bos *bulkOperationService) BulkRevokeLicensesAsync(licenseIDs []int, reason string, userID int) (*dto.AsyncBulkRevokeResponse, error) {
	if len(licenseIDs) == 0 {
		return nil, errors.NewValidationError("At least one license must be specified for bulk revoke")
	}

	// Generate unique job ID
	jobID := uuid.New().String()
	now := time.Now().UTC()

	// Initialize job status
	jobStatus := &dto.BulkJobStatus{
		JobID:          jobID,
		Status:         JobStatusPending,
		TotalItems:     len(licenseIDs),
		ProcessedItems: 0,
		SuccessCount:   0,
		FailureCount:   0,
		FailedIDs:      make([]int, 0),
		CreatedAt:      now,
		UpdatedAt:      now,
		ProgressPct:    0.0,
	}

	// Store job status
	bos.jobStore.Store(jobID, jobStatus)

	// Start async processing in goroutine
	go bos.processRevokeLicensesAsync(jobID, licenseIDs, reason, userID)

	return &dto.AsyncBulkRevokeResponse{
		JobID:      jobID,
		Status:     JobStatusPending,
		Message:    "Bulk revoke job started",
		TotalItems: len(licenseIDs),
		CreatedAt:  now,
	}, nil
}

// processRevokeLicensesAsync processes license revocation asynchronously
func (bos *bulkOperationService) processRevokeLicensesAsync(jobID string, licenseIDs []int, reason string, userID int) {
	// Update status to in_progress
	bos.updateJobStatus(jobID, func(status *dto.BulkJobStatus) {
		status.Status = JobStatusInProgress
		status.UpdatedAt = time.Now().UTC()
	})

	// Process licenses with error recovery
	for i, licenseID := range licenseIDs {
		// Check if job was cancelled
		if jobStatus, ok := bos.jobStore.Load(jobID); ok {
			if js, ok := jobStatus.(*dto.BulkJobStatus); ok && js.Status == "cancelled" {
				bos.updateJobStatus(jobID, func(status *dto.BulkJobStatus) {
					status.Status = JobStatusFailed
					status.ErrorMessage = "Job was cancelled"
					status.UpdatedAt = time.Now().UTC()
					now := time.Now().UTC()
					status.CompletedAt = &now
				})
				return
			}
		}

		// Process individual license with retry logic
		success := bos.processLicenseRevocationWithRetry(licenseID, reason, userID)

		// Update job status
		bos.updateJobStatus(jobID, func(status *dto.BulkJobStatus) {
			status.ProcessedItems = i + 1
			status.ProgressPct = float64(status.ProcessedItems) / float64(status.TotalItems) * 100.0

			if success {
				status.SuccessCount++
			} else {
				status.FailureCount++
				status.FailedIDs = append(status.FailedIDs, licenseID)
			}

			status.UpdatedAt = time.Now().UTC()
		})
	}

	// Mark job as completed
	bos.updateJobStatus(jobID, func(status *dto.BulkJobStatus) {
		status.Status = JobStatusCompleted
		status.ProgressPct = 100.0
		status.UpdatedAt = time.Now().UTC()
		now := time.Now().UTC()
		status.CompletedAt = &now
	})
}

// processLicenseRevocationWithRetry attempts to revoke a license with retry logic
func (bos *bulkOperationService) processLicenseRevocationWithRetry(licenseID int, reason string, userID int) bool {
	maxRetries := 3
	retryDelay := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}

		// Attempt to get license
		license, err := bos.licenseRepo.GetByID(licenseID)
		if err != nil {
			if attempt == maxRetries-1 {
				fmt.Printf("Failed to get license %d after %d attempts: %v\n", licenseID, maxRetries, err)
				return false
			}
			continue
		}

		// Skip already revoked licenses
		if license.IsRevoked() {
			return false
		}

		// Update license status to revoked
		now := time.Now().UTC()
		license.RevokedAt = &now

		if err := bos.licenseRepo.Update(license); err != nil {
			if attempt == maxRetries-1 {
				fmt.Printf("Failed to update license %d after %d attempts: %v\n", licenseID, maxRetries, err)
				return false
			}
			continue
		}

		// Create audit log entry (best effort)
		entityID := licenseID
		var userIDPtr *int
		userIDPtr = &userID

		auditLog := &models.AuditLog{
			UserID:     userIDPtr,
			Action:     "BULK_REVOKE_LICENSE_ASYNC",
			EntityType: "License",
			EntityID:   &entityID,
			Details:    &reason,
			Timestamp:  time.Now().UTC(),
		}

		if err := bos.auditRepo.Create(auditLog); err != nil {
			fmt.Printf("Failed to create audit log for license %d: %v\n", licenseID, err)
			// Continue anyway - audit log failure shouldn't fail the operation
		}

		return true
	}

	return false
}

// updateJobStatus updates a job's status atomically
func (bos *bulkOperationService) updateJobStatus(jobID string, updateFn func(*dto.BulkJobStatus)) {
	if value, ok := bos.jobStore.Load(jobID); ok {
		if status, ok := value.(*dto.BulkJobStatus); ok {
			updateFn(status)
			bos.jobStore.Store(jobID, status)
		}
	}
}

// GetJobStatus retrieves the status of a bulk operation job
func (bos *bulkOperationService) GetJobStatus(jobID string) (*dto.BulkJobStatus, error) {
	if value, ok := bos.jobStore.Load(jobID); ok {
		if status, ok := value.(*dto.BulkJobStatus); ok {
			return status, nil
		}
	}
	return nil, errors.NewNotFoundError("Job not found")
}

// CancelJob attempts to cancel a running job
func (bos *bulkOperationService) CancelJob(jobID string) error {
	if value, ok := bos.jobStore.Load(jobID); ok {
		if status, ok := value.(*dto.BulkJobStatus); ok {
			if status.Status == JobStatusCompleted || status.Status == JobStatusFailed {
				return errors.NewBadRequestError("Cannot cancel completed or failed job")
			}

			bos.updateJobStatus(jobID, func(s *dto.BulkJobStatus) {
				s.Status = "cancelled"
				s.UpdatedAt = time.Now().UTC()
			})

			return nil
		}
	}
	return errors.NewNotFoundError("Job not found")
}
