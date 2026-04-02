package handler

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	apiv1 "github.com/XavierMary56/automatic_review/go-server/internal/api/v1"
	apiv2 "github.com/XavierMary56/automatic_review/go-server/internal/api/v2"
	"github.com/XavierMary56/automatic_review/go-server/internal/audit"
	"github.com/XavierMary56/automatic_review/go-server/internal/config"
	"github.com/XavierMary56/automatic_review/go-server/internal/logger"
	"github.com/XavierMary56/automatic_review/go-server/internal/service"
	"github.com/XavierMary56/automatic_review/go-server/internal/storage"
)

// Handler handles all HTTP requests.
type Handler struct {
	svc   *service.ModerationService
	log   *logger.Logger
	cfg   *config.Config
	db    *storage.DB
	audit *audit.AuditLogger
	tasks sync.Map
	usage sync.Map // key -> *rateCounter
}

type rateCounter struct {
	mu      sync.Mutex
	count   int
	resetAt time.Time
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) StatusCode() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

// New creates a new handler instance.
func New(svc *service.ModerationService, log *logger.Logger, cfg *config.Config, db *storage.DB, auditLogger *audit.AuditLogger) *Handler {
	return &Handler{
		svc:   svc,
		log:   log,
		cfg:   cfg,
		db:    db,
		audit: auditLogger,
	}
}

// RegisterRoutes registers all public API routes.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	v1Handler := apiv1.New(h.svc, h.log, h.cfg, h.db, h.audit, &h.tasks)
	v1Handler.RegisterRoutes(mux, h.withMiddleware)

	v2Handler := apiv2.New(h.svc, h.log, h.cfg, h.db, h.audit, &h.tasks)
	v2Handler.RegisterRoutes(mux, h.withMiddleware)
}

func (h *Handler) withMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Project-Key")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if r.Method == http.MethodOptions {
			rec.WriteHeader(http.StatusOK)
			return
		}

		var projectKey *storage.ProjectKey
		if h.cfg.EnableAuth {
			var err error
			projectKey, err = h.validateProjectKey(r.Header.Get("X-Project-Key"))
			if err != nil {
				if h.audit != nil {
					h.audit.LogAuthAttempt("unknown", r.Header.Get("X-Project-Key"), false, h.getClientIP(r))
				}
				h.jsonError(rec, http.StatusUnauthorized, "invalid project key")
				return
			}
			if h.audit != nil {
				h.audit.LogAuthAttempt(projectKey.ProjectName, projectKey.Key, true, h.getClientIP(r))
			}
			if err := h.checkRateLimit(projectKey); err != nil {
				if h.audit != nil {
					h.audit.LogRateLimitExceeded(projectKey.ProjectName, projectKey.Key, h.getClientIP(r))
				}
				h.jsonError(rec, http.StatusTooManyRequests, err.Error())
				return
			}
		}

		next(rec, r)

		if h.audit != nil && projectKey != nil {
			errorMsg := ""
			if rec.StatusCode() >= http.StatusBadRequest {
				errorMsg = http.StatusText(rec.StatusCode())
			}
			h.audit.LogAPICall(
				projectKey.ProjectName,
				projectKey.Key,
				r.Method,
				r.URL.Path,
				rec.StatusCode(),
				time.Since(start).Milliseconds(),
				h.getClientIP(r),
				errorMsg,
			)
		}
	}
}

func (h *Handler) validateProjectKey(key string) (*storage.ProjectKey, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("empty project key")
	}

	if h.db != nil {
		projectKey, err := h.db.GetEnabledProjectKey(key)
		if err != nil {
			h.log.Error("database project key lookup failed: " + err.Error())
			return nil, err
		}
		if projectKey != nil {
			return projectKey, nil
		}
	}

	for _, allowed := range h.cfg.AllowedKeys {
		if strings.TrimSpace(allowed) == key {
			return &storage.ProjectKey{
				ProjectName: key,
				Key:         key,
				Enabled:     true,
			}, nil
		}
	}

	return nil, fmt.Errorf("project key not found")
}

func (h *Handler) checkRateLimit(projectKey *storage.ProjectKey) error {
	if projectKey == nil || projectKey.RateLimit <= 0 {
		return nil
	}

	newCounter := &rateCounter{resetAt: time.Now().Add(time.Minute)}
	val, _ := h.usage.LoadOrStore(projectKey.Key, newCounter)
	counter := val.(*rateCounter)

	counter.mu.Lock()
	defer counter.mu.Unlock()

	now := time.Now()
	if now.After(counter.resetAt) {
		counter.count = 0
		counter.resetAt = now.Add(time.Minute)
	}

	if counter.count >= projectKey.RateLimit {
		return fmt.Errorf("rate limit exceeded for project %s", projectKey.ProjectName)
	}

	counter.count++
	return nil
}

func (h *Handler) jsonError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"code":%d,"error":%q}`, status, msg)))
}

func (h *Handler) getClientIP(r *http.Request) string {
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// 取最后一个 IP：最后一跳由可信反向代理追加，不可被客户端伪造
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return r.RemoteAddr
}
