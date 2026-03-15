package handler

import (
	"context"
	"net/http"
	"time"

	"license-management-api/internal/cache"
	"license-management-api/internal/database"
	"license-management-api/internal/service"
)

type HealthHandler struct {
	db                database.Service
	circuitBreakerSvc *service.CircuitBreakerService
	licenseCache      *cache.LicenseCache
}

func NewHealthHandler(db database.Service, circuitBreakerSvc *service.CircuitBreakerService, licenseCache *cache.LicenseCache) *HealthHandler {
	return &HealthHandler{
		db:                db,
		circuitBreakerSvc: circuitBreakerSvc,
		licenseCache:      licenseCache,
	}
}

// Health checks the health of the API and database
// @Summary Check API health
// @Description Check if the API and database are healthy
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "API is healthy"
// @Failure 503 {object} map[string]interface{} "Service unavailable"
// @Router /health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	var dbHealth map[string]string

	// Use circuit breaker for database health check
	if h.circuitBreakerSvc != nil {
		result, err := h.circuitBreakerSvc.ExecuteDatabase(func() (interface{}, error) {
			health := h.db.Health()
			if health["status"] == "down" {
				return health, &CircuitBreakerError{Msg: health["error"]}
			}
			return health, nil
		})

		if err != nil {
			dbHealth = map[string]string{
				"status": "down",
				"error":  err.Error(),
			}
		} else {
			dbHealth = result.(map[string]string)
		}
	} else {
		dbHealth = h.db.Health()
	}

	// Check Redis health
	redisHealth := map[string]string{
		"status": "up",
	}

	if h.licenseCache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := h.licenseCache.HealthCheck(ctx); err != nil {
			redisHealth["status"] = "down"
			redisHealth["error"] = err.Error()
		}
	}

	status := http.StatusOK
	apiStatus := "up"
	if dbHealth["status"] == "down" || redisHealth["status"] == "down" {
		status = http.StatusServiceUnavailable
		apiStatus = "down"
	}

	// Add circuit breaker status
	response := map[string]interface{}{
		"api_status": apiStatus,
		"database":   dbHealth,
		"redis":      redisHealth,
	}
	if h.circuitBreakerSvc != nil {
		response["circuit_breakers"] = h.circuitBreakerSvc.GetStats()
	}

	writeJSON(w, status, response)
}

// CircuitBreakerError is returned when a circuit breaker operation fails
type CircuitBreakerError struct {
	Msg string
}

func (e *CircuitBreakerError) Error() string {
	return e.Msg
}

// Liveness probe: always returns 200 OK if the service is running
// Used by Kubernetes liveness probe to restart failed containers
// @Summary Liveness probe
// @Description Check if the service is running (always healthy)
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /livez [get]
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "alive",
	})
}

// Readiness probe: returns 200 only if the service can handle traffic
// Checks database and critical dependencies
// Used by Kubernetes readiness probe for load balancing decisions
// @Summary Readiness probe
// @Description Check if the service is ready to handle requests
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /readyz [get]
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	dbHealth := h.db.Health()

	status := http.StatusOK
	ready := dbHealth["status"] == "up"

	// Also check if database circuit breaker is open
	if h.circuitBreakerSvc != nil {
		cbState := h.circuitBreakerSvc.GetDatabaseState().String()
		if cbState == "open" {
			ready = false
		}
	}

	if !ready {
		status = http.StatusServiceUnavailable
	}

	response := map[string]interface{}{
		"ready":    ready,
		"database": dbHealth["status"],
	}

	if h.circuitBreakerSvc != nil {
		response["circuit_breakers"] = h.circuitBreakerSvc.GetStats()
	}

	writeJSON(w, status, response)
}

// Startup probe: returns 200 once initial bootstrap is complete
// Used by Kubernetes startup probe during pod initialization
// @Summary Startup probe
// @Description Check if the service has completed startup
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /startup [get]
func (h *HealthHandler) Startup(w http.ResponseWriter, r *http.Request) {
	// For now, startup is complete when database is available
	// In a more complex app, you'd track additional startup conditions
	dbHealth := h.db.Health()

	status := http.StatusOK
	complete := dbHealth["status"] == "up"

	if !complete {
		status = http.StatusServiceUnavailable
	}

	writeJSON(w, status, map[string]interface{}{
		"started":  complete,
		"database": dbHealth["status"],
	})
}
