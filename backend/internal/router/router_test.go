package router

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/gbexam/online-exam/internal/handler"
	"github.com/gbexam/online-exam/internal/service"
)

// TestNewRegistersRoutes ensures the route table (including makeup endpoints)
// is built without gin route conflicts and health checks respond.
func TestNewRegistersRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.Default()
	authService := &service.AuthService{}
	server := handler.NewServer(logger, authService, nil, nil, nil, nil, nil, nil, nil)
	engine := New(server, func(c *gin.Context) { c.Next() })

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz = %d, want 200", rec.Code)
	}

	routes := engine.Routes()
	want := []string{
		"GET /api/v1/exams/:id/makeup",
		"POST /api/v1/exams/:id/makeup",
		"GET /api/v1/exams/:id/makeup-requests",
		"POST /api/v1/makeup-requests/:id/approve",
		"POST /api/v1/makeup-requests/:id/reject",
	}
	registered := make(map[string]bool, len(routes))
	for _, r := range routes {
		registered[r.Method+" "+r.Path] = true
	}
	for _, route := range want {
		if !registered[route] {
			t.Fatalf("route %s not registered", route)
		}
	}
}
