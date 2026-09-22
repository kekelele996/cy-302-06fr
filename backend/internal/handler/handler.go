package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/service"
	"github.com/gbexam/online-exam/pkg/httpx"
)

// Server aggregates all services for HTTP handlers.
type Server struct {
	logger             *slog.Logger
	auth               *service.AuthService
	users              *service.UserService
	questions          *service.QuestionService
	exams              *service.ExamService
	attempts           *service.AttemptService
	stats              *service.StatsService
	wrong              *service.WrongQuestionService
	makeupApplications *service.MakeupApplicationService
}

// NewServer constructs Server with injected services.
func NewServer(
	logger *slog.Logger,
	auth *service.AuthService,
	users *service.UserService,
	questions *service.QuestionService,
	exams *service.ExamService,
	attempts *service.AttemptService,
	stats *service.StatsService,
	wrong *service.WrongQuestionService,
	makeupApplications *service.MakeupApplicationService,
) *Server {
	return &Server{
		logger:             logger,
		auth:               auth,
		users:              users,
		questions:          questions,
		exams:              exams,
		attempts:           attempts,
		stats:              stats,
		wrong:              wrong,
		makeupApplications: makeupApplications,
	}
}

// Logger exposes the handler logger for middleware wiring.
func (s *Server) Logger() *slog.Logger {
	return s.logger
}

// Health handles the /healthz and /health endpoints.
func (s *Server) Health(c *gin.Context) {
	httpx.OK(c, gin.H{"status": "ok"})
}

func (s *Server) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, constants.CodeNotFound, "资源不存在")
	case errors.Is(err, service.ErrConflict):
		httpx.Fail(c, http.StatusConflict, constants.CodeConflict, conflictMessage(err))
	case errors.Is(err, service.ErrUnauthorized):
		httpx.Fail(c, http.StatusUnauthorized, constants.CodeUnauthorized, "认证失败")
	case errors.Is(err, service.ErrForbidden):
		httpx.Fail(c, http.StatusForbidden, constants.CodeForbidden, "无权限访问")
	case errors.Is(err, service.ErrBadRequest):
		httpx.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, err.Error())
	case errors.Is(err, service.ErrValidation):
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, err.Error())
	default:
		s.logger.Error("internal error", "error", err)
		httpx.Fail(c, http.StatusInternalServerError, constants.CodeInternal, "服务器内部错误")
	}
}

// conflictMessage surfaces a domain-specific conflict message when the service
// wrapped the sentinel error with one, otherwise it falls back to a generic text.
func conflictMessage(err error) string {
	msg := err.Error()
	if msg == "" || msg == service.ErrConflict.Error() {
		return "资源冲突"
	}
	return msg
}

func parsePage(c *gin.Context) (int, int) {
	page := c.GetInt("page")
	pageSize := c.GetInt("page_size")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
