package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/dto"
	"github.com/gbexam/online-exam/internal/model"
	"github.com/gbexam/online-exam/internal/repository"
)

// MakeupService handles makeup-exam applications and approvals.
type MakeupService struct {
	baseService
	examRepo     ExamRepo
	attemptRepo  AttemptRepo
	makeupRepo   MakeupRepo
	questionRepo QuestionRepo
	userRepo     UserRepo
}

// NewMakeupService constructs MakeupService.
func NewMakeupService(
	examRepo ExamRepo,
	attemptRepo AttemptRepo,
	makeupRepo MakeupRepo,
	questionRepo QuestionRepo,
	userRepo UserRepo,
	logger *slog.Logger,
) *MakeupService {
	return &MakeupService{
		baseService:  NewBaseService(logger),
		examRepo:     examRepo,
		attemptRepo:  attemptRepo,
		makeupRepo:   makeupRepo,
		questionRepo: questionRepo,
		userRepo:     userRepo,
	}
}

// makeupState is the evaluated eligibility of a student for a makeup exam.
type makeupState struct {
	eligible        bool
	canApply        bool
	message         string
	effectiveScore  *float64
	makeupAttemptID uint
	makeupStatus    string
}

// evaluateMakeup is the pure eligibility rule: once the exam is closed,
// students who were absent or scored below 60% of the total may apply for
// exactly one makeup exam, unless an application is pending or a makeup
// attempt is already in progress.
func evaluateMakeup(exam *model.Exam, attempts []model.ExamAttempt, req *model.MakeupRequest) makeupState {
	state := makeupState{}
	passScore := exam.TotalScore * constants.MakeupPassRatio
	if best := bestSubmittedAttempt(attempts); best != nil {
		score := best.TotalScore
		state.effectiveScore = &score
	}
	for i := range attempts {
		if attempts[i].Kind == constants.AttemptKindMakeup {
			state.makeupAttemptID = attempts[i].ID
			state.makeupStatus = attempts[i].Status
			break
		}
	}

	if exam.Status != constants.ExamClosed {
		state.message = "考试尚未结束，暂不能申请补考"
		return state
	}
	if state.effectiveScore != nil && *state.effectiveScore >= passScore {
		state.message = "成绩已达标，无需补考"
		return state
	}
	switch state.makeupStatus {
	case constants.AttemptInProgress:
		state.message = "补考进行中，请完成补考"
		return state
	case constants.AttemptSubmitted:
		state.message = "补考已提交，本次补考机会已使用"
		return state
	}
	if req != nil {
		switch req.Status {
		case constants.MakeupPending:
			state.message = "补考申请待审核，请耐心等待"
			return state
		case constants.MakeupApproved:
			state.message = "补考申请已批准，请进入补考"
			return state
		case constants.MakeupRejected:
			state.eligible = true
			state.canApply = true
			state.message = "上次申请已被拒绝，可重新申请"
			return state
		}
	}
	state.eligible = true
	state.canApply = true
	state.message = "可申请补考"
	return state
}

// Status returns the student's makeup eligibility and application state.
func (s *MakeupService) Status(ctx context.Context, studentID, examID uint) (*dto.MakeupStatusResponse, error) {
	exam, err := s.examRepo.FindExamByID(ctx, examID)
	if err != nil {
		return nil, err
	}
	attempts, err := s.attemptRepo.ListAttemptsByExamAndStudent(ctx, examID, studentID)
	if err != nil {
		return nil, fmt.Errorf("list attempts for makeup status: %w", err)
	}
	req, err := s.findRequest(ctx, examID, studentID)
	if err != nil {
		return nil, err
	}
	state := evaluateMakeup(exam, attempts, req)
	resp := &dto.MakeupStatusResponse{
		ExamID:          exam.ID,
		PassScore:       round2(exam.TotalScore * constants.MakeupPassRatio),
		EffectiveScore:  state.effectiveScore,
		Eligible:        state.eligible,
		CanApply:        state.canApply,
		Message:         state.message,
		MakeupAttemptID: state.makeupAttemptID,
		MakeupStatus:    state.makeupStatus,
	}
	if req != nil {
		item := makeupRequestToItem(req, exam.Title, nil)
		resp.Request = &item
	}
	return resp, nil
}

// Apply creates a makeup application. Duplicate applications (or a pending
// request / running makeup) are rejected; a rejected application can be
// resubmitted exactly once per review round.
func (s *MakeupService) Apply(ctx context.Context, studentID, examID uint, reason string) (*dto.MakeupRequestItem, error) {
	exam, err := s.examRepo.FindExamByID(ctx, examID)
	if err != nil {
		return nil, err
	}
	attempts, err := s.attemptRepo.ListAttemptsByExamAndStudent(ctx, examID, studentID)
	if err != nil {
		return nil, fmt.Errorf("list attempts for makeup apply: %w", err)
	}
	existing, err := s.findRequest(ctx, examID, studentID)
	if err != nil {
		return nil, err
	}
	state := evaluateMakeup(exam, attempts, existing)
	if !state.canApply {
		return nil, &ConflictError{Message: state.message}
	}

	if existing != nil && existing.Status == constants.MakeupRejected {
		if err := s.makeupRepo.ReapplyMakeupRequest(ctx, existing.ID, reason); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return nil, &ConflictError{Message: "申请状态已变化，请刷新后重试"}
			}
			return nil, fmt.Errorf("reapply makeup request: %w", err)
		}
		existing.Status = constants.MakeupPending
		existing.Reason = reason
		item := makeupRequestToItem(existing, exam.Title, nil)
		return &item, nil
	}

	req := &model.MakeupRequest{
		ExamID:    examID,
		StudentID: studentID,
		Reason:    reason,
		Status:    constants.MakeupPending,
	}
	if err := s.makeupRepo.CreateMakeupRequest(ctx, req); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, &ConflictError{Message: "已存在补考申请，请勿重复提交"}
		}
		return nil, fmt.Errorf("create makeup request: %w", err)
	}
	item := makeupRequestToItem(req, exam.Title, nil)
	return &item, nil
}

// ListByExam returns all makeup applications of an exam for staff review.
func (s *MakeupService) ListByExam(ctx context.Context, role string, userID, examID uint) ([]dto.MakeupRequestItem, error) {
	exam, err := s.examRepo.FindExamByID(ctx, examID)
	if err != nil {
		return nil, err
	}
	if role == constants.RoleTeacher && exam.CreatedBy != userID {
		return nil, ErrForbidden
	}
	requests, err := s.makeupRepo.ListMakeupRequestsByExam(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("list makeup requests: %w", err)
	}
	items := make([]dto.MakeupRequestItem, 0, len(requests))
	for i := range requests {
		var user *model.User
		if u, userErr := s.userRepo.FindUserByID(ctx, requests[i].StudentID); userErr == nil {
			user = u
		}
		items = append(items, makeupRequestToItem(&requests[i], exam.Title, user))
	}
	return items, nil
}

// Approve marks a pending application approved and generates the independent
// makeup attempt. Concurrent approvals only succeed once thanks to the
// conditional status flip in the repository.
func (s *MakeupService) Approve(ctx context.Context, role string, reviewerID, requestID uint) error {
	req, exam, err := s.reviewable(ctx, role, reviewerID, requestID)
	if err != nil {
		return err
	}
	items, err := s.examRepo.ListExamQuestions(ctx, exam.ID)
	if err != nil {
		return fmt.Errorf("list exam questions: %w", err)
	}
	if len(items) == 0 {
		return fmt.Errorf("%w: 试卷没有题目，无法生成补考", ErrValidation)
	}
	ids := make([]uint, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.QuestionID)
	}
	questions, err := s.questionRepo.FindQuestionsByIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("find questions by ids: %w", err)
	}
	attempt := newShuffledAttempt(exam, req.StudentID, items, questions, constants.AttemptKindMakeup, 2, time.Now())
	if err := s.makeupRepo.ApproveMakeupRequest(ctx, req.ID, reviewerID, attempt); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return &ConflictError{Message: "该申请已被处理，请刷新列表"}
		}
		return fmt.Errorf("approve makeup request: %w", err)
	}
	return nil
}

// Reject marks a pending application rejected.
func (s *MakeupService) Reject(ctx context.Context, role string, reviewerID, requestID uint) error {
	req, _, err := s.reviewable(ctx, role, reviewerID, requestID)
	if err != nil {
		return err
	}
	if err := s.makeupRepo.RejectMakeupRequest(ctx, req.ID, reviewerID); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return &ConflictError{Message: "该申请已被处理，请刷新列表"}
		}
		return fmt.Errorf("reject makeup request: %w", err)
	}
	return nil
}

// reviewable loads a pending request and its exam, enforcing ownership.
func (s *MakeupService) reviewable(ctx context.Context, role string, reviewerID, requestID uint) (*model.MakeupRequest, *model.Exam, error) {
	req, err := s.makeupRepo.FindMakeupRequestByID(ctx, requestID)
	if err != nil {
		return nil, nil, err
	}
	exam, err := s.examRepo.FindExamByID(ctx, req.ExamID)
	if err != nil {
		return nil, nil, err
	}
	if role == constants.RoleTeacher && exam.CreatedBy != reviewerID {
		return nil, nil, ErrForbidden
	}
	if req.Status != constants.MakeupPending {
		return nil, nil, &ConflictError{Message: "该申请已被处理，请刷新列表"}
	}
	return req, exam, nil
}

func (s *MakeupService) findRequest(ctx context.Context, examID, studentID uint) (*model.MakeupRequest, error) {
	req, err := s.makeupRepo.FindMakeupRequest(ctx, examID, studentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find makeup request: %w", err)
	}
	return req, nil
}

func makeupRequestToItem(req *model.MakeupRequest, examTitle string, user *model.User) dto.MakeupRequestItem {
	item := dto.MakeupRequestItem{
		ID:         req.ID,
		ExamID:     req.ExamID,
		ExamTitle:  examTitle,
		StudentID:  req.StudentID,
		Reason:     req.Reason,
		Status:     req.Status,
		ReviewedBy: req.ReviewedBy,
		ReviewedAt: req.ReviewedAt,
		AttemptID:  req.AttemptID,
		CreatedAt:  req.CreatedAt,
	}
	if user != nil {
		item.StudentName = user.Name
		item.StudentUsername = user.Username
	}
	return item
}
