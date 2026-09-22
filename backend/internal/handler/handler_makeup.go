package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/dto"
	"github.com/gbexam/online-exam/internal/middleware"
	"github.com/gbexam/online-exam/pkg/httpx"
)

// MakeupEligibility handles GET /exams/:id/makeup/eligibility.
func (s *Server) MakeupEligibility(c *gin.Context) {
	examID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "考试 ID 不合法")
		return
	}
	result, err := s.makeupApplications.Eligibility(c.Request.Context(), middleware.UserID(c), uint(examID))
	if err != nil {
		s.respondError(c, err)
		return
	}
	httpx.OK(c, result)
}

// ApplyMakeup handles POST /exams/:id/makeup/applications.
func (s *Server) ApplyMakeup(c *gin.Context) {
	examID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "考试 ID 不合法")
		return
	}
	var req dto.MakeupApplyRequest
	// reason is optional; an empty body is accepted and left as zero values.
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "请求参数不合法")
		return
	}
	item, err := s.makeupApplications.Apply(c.Request.Context(), middleware.UserID(c), uint(examID), req)
	if err != nil {
		s.respondError(c, err)
		return
	}
	httpx.Created(c, item)
}

// StartMakeup handles POST /exams/:id/makeup/attempts.
func (s *Server) StartMakeup(c *gin.Context) {
	examID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "考试 ID 不合法")
		return
	}
	paper, err := s.makeupApplications.StartMakeup(c.Request.Context(), middleware.UserID(c), uint(examID))
	if err != nil {
		s.respondError(c, err)
		return
	}
	httpx.Created(c, paper)
}

// MyMakeupApplication handles GET /exams/:id/makeup/application.
func (s *Server) MyMakeupApplication(c *gin.Context) {
	examID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "考试 ID 不合法")
		return
	}
	item, err := s.makeupApplications.MyApplication(c.Request.Context(), middleware.UserID(c), uint(examID))
	if err != nil {
		s.respondError(c, err)
		return
	}
	httpx.OK(c, item)
}

// ListMakeupApplications handles GET /makeup-applications.
func (s *Server) ListMakeupApplications(c *gin.Context) {
	var query dto.MakeupApplicationListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "请求参数不合法")
		return
	}
	result, err := s.makeupApplications.List(c.Request.Context(), middleware.Role(c), middleware.UserID(c), query)
	if err != nil {
		s.respondError(c, err)
		return
	}
	httpx.OK(c, result)
}

// ApproveMakeup handles POST /makeup-applications/:id/approve.
func (s *Server) ApproveMakeup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "申请 ID 不合法")
		return
	}
	item, err := s.makeupApplications.Approve(c.Request.Context(), middleware.Role(c), middleware.UserID(c), uint(id))
	if err != nil {
		s.respondError(c, err)
		return
	}
	httpx.OK(c, item)
}

// RejectMakeup handles POST /makeup-applications/:id/reject.
func (s *Server) RejectMakeup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, http.StatusUnprocessableEntity, constants.CodeValidation, "申请 ID 不合法")
		return
	}
	var req dto.MakeupReviewRequest
	_ = c.ShouldBindJSON(&req)
	item, err := s.makeupApplications.Reject(c.Request.Context(), middleware.Role(c), middleware.UserID(c), uint(id), req)
	if err != nil {
		s.respondError(c, err)
		return
	}
	httpx.OK(c, item)
}
