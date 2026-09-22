package service

import (
	"errors"
	"log/slog"

	"github.com/gbexam/online-exam/internal/config"
	"github.com/gbexam/online-exam/internal/repository"
)

// Service-level sentinel errors translated by handlers into HTTP responses.
var (
	ErrBadRequest   = errors.New("bad request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = repository.ErrNotFound
	ErrConflict     = repository.ErrConflict
	ErrValidation   = errors.New("validation failed")
)

// ConflictError is a 409 conflict with a user-facing message.
type ConflictError struct {
	Message string
}

// Error returns the user-facing message.
func (e *ConflictError) Error() string {
	return e.Message
}

// Is makes ConflictError match ErrConflict via errors.Is.
func (e *ConflictError) Is(target error) bool {
	return target == ErrConflict
}

// baseService provides logger access for all services.
type baseService struct {
	logger *slog.Logger
}

// NewBaseService builds a baseService with an injected logger.
func NewBaseService(logger *slog.Logger) baseService {
	return baseService{logger: logger}
}

// ConfigProvider is used by services that need environment values (JWT, seed admin).
type ConfigProvider = config.Config
