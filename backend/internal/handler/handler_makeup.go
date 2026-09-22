package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/dto"
	"github.com/gbexam/online-exam/internal/middleware"
	"github.com/gbexam/online-exam/pkg/httpx"
)

// GetMakeupStatus handles GET /exams/:id/makeup.
func (s *Server) GetMakeupStatus(c *gin.Context) {
	examID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "考试 ID 不合法")
		return
	}
	status, err := s.makeup.Status(c.Request.Context(), middleware.UserID(c), uint(examID))
	if err != nil {
		s.respondError(c, err)
		return
	}
	httpx.OK(c, status)
}

// ApplyMakeup handles POST /exams/:id/makeup.
func (s *Server) ApplyMakeup(c *gin.Context) {
	examID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "考试 ID 不合法")
		return
	}
	var req dto.MakeupApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "请求参数不合法")
		return
	}
	item, err := s.makeup.Apply(c.Request.Context(), middleware.UserID(c), uint(examID), req.Reason)
	if err != nil {
		s.respondError(c, err)
		return
	}
	httpx.Created(c, item)
}

// ListMakeupRequests handles GET /exams/:id/makeup-requests.
func (s *Server) ListMakeupRequests(c *gin.Context) {
	examID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "考试 ID 不合法")
		return
	}
	items, err := s.makeup.ListByExam(c.Request.Context(), middleware.Role(c), middleware.UserID(c), uint(examID))
	if err != nil {
		s.respondError(c, err)
		return
	}
	httpx.OK(c, items)
}

// ApproveMakeupRequest handles POST /makeup-requests/:id/approve.
func (s *Server) ApproveMakeupRequest(c *gin.Context) {
	requestID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "申请 ID 不合法")
		return
	}
	if err := s.makeup.Approve(c.Request.Context(), middleware.Role(c), middleware.UserID(c), uint(requestID)); err != nil {
		s.respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"message": "已批准补考"})
}

// RejectMakeupRequest handles POST /makeup-requests/:id/reject.
func (s *Server) RejectMakeupRequest(c *gin.Context) {
	requestID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "申请 ID 不合法")
		return
	}
	if err := s.makeup.Reject(c.Request.Context(), middleware.Role(c), middleware.UserID(c), uint(requestID)); err != nil {
		s.respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"message": "已拒绝申请"})
}
