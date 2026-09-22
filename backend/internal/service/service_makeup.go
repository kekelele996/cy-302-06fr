package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/dto"
	"github.com/gbexam/online-exam/internal/model"
	"github.com/gbexam/online-exam/internal/repository"
)

// MakeupApplicationService handles student makeup applications, teacher review
// and starting the independent makeup attempt.
type MakeupApplicationService struct {
	baseService
	makeupRepo  MakeupRepo
	examRepo    ExamRepo
	attemptRepo AttemptRepo
	userRepo    UserRepo
	attempts    *AttemptService
}

// NewMakeupApplicationService constructs MakeupApplicationService.
func NewMakeupApplicationService(
	makeupRepo MakeupRepo,
	examRepo ExamRepo,
	attemptRepo AttemptRepo,
	userRepo UserRepo,
	attempts *AttemptService,
	logger *slog.Logger,
) *MakeupApplicationService {
	return &MakeupApplicationService{
		baseService: NewBaseService(logger),
		makeupRepo:  makeupRepo,
		examRepo:    examRepo,
		attemptRepo: attemptRepo,
		userRepo:    userRepo,
		attempts:    attempts,
	}
}

// Eligibility reports whether the student may apply for a makeup exam for the
// given exam, together with any existing application and its approval status.
func (s *MakeupApplicationService) Eligibility(ctx context.Context, studentID, examID uint) (*dto.MakeupEligibility, error) {
	eligible, reason, err := s.checkEligibility(ctx, studentID, examID)
	if err != nil {
		return nil, err
	}
	result := &dto.MakeupEligibility{ExamID: examID, Eligible: eligible, Reason: reason}
	app, err := s.makeupRepo.FindMakeupApplication(ctx, examID, studentID)
	if err == nil {
		item, err := s.toItem(ctx, app)
		if err != nil {
			return nil, err
		}
		result.Application = item
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("find makeup application: %w", err)
	}
	return result, nil
}

// MyApplication returns the student's application for one exam.
func (s *MakeupApplicationService) MyApplication(ctx context.Context, studentID, examID uint) (*dto.MakeupApplicationItem, error) {
	app, err := s.makeupRepo.FindMakeupApplication(ctx, examID, studentID)
	if err != nil {
		return nil, err
	}
	return s.toItem(ctx, app)
}

// Apply creates a makeup application. It rejects duplicate applications and
// students who are not eligible (no attempt record, already passing, makeup in
// progress, etc.). The unique (exam_id, student_id) index guarantees repeated
// or concurrent requests only succeed once.
func (s *MakeupApplicationService) Apply(ctx context.Context, studentID, examID uint, req dto.MakeupApplyRequest) (*dto.MakeupApplicationItem, error) {
	if existing, err := s.makeupRepo.FindMakeupApplication(ctx, examID, studentID); err == nil {
		switch existing.Status {
		case constants.MakeupPending:
			return nil, fmt.Errorf("%w: 已有待审批的补考申请，请勿重复提交", ErrConflict)
		case constants.MakeupApproved:
			return nil, fmt.Errorf("%w: 补考申请已批准，无需再次申请", ErrConflict)
		default:
			return nil, fmt.Errorf("%w: 补考申请已被处理，无法再次申请", ErrConflict)
		}
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("find makeup application: %w", err)
	}

	eligible, reason, err := s.checkEligibility(ctx, studentID, examID)
	if err != nil {
		return nil, err
	}
	if !eligible {
		return nil, fmt.Errorf("%w: %s", ErrValidation, reason)
	}

	app := &model.MakeupApplication{
		ExamID:    examID,
		StudentID: studentID,
		Reason:    req.Reason,
		Status:    constants.MakeupPending,
	}
	if err := s.makeupRepo.CreateMakeupApplication(ctx, app); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, fmt.Errorf("%w: 已有补考申请，请勿重复提交", ErrConflict)
		}
		return nil, err
	}
	return s.toItem(ctx, app)
}

// List returns applications for teacher review, optionally filtered by exam/status.
func (s *MakeupApplicationService) List(ctx context.Context, role string, userID uint, query dto.MakeupApplicationListQuery) (dto.PageResult, error) {
	if query.ExamID != 0 {
		exam, err := s.examRepo.FindExamByID(ctx, query.ExamID)
		if err != nil {
			return dto.PageResult{}, err
		}
		if role == constants.RoleTeacher && exam.CreatedBy != userID {
			return dto.PageResult{}, ErrForbidden
		}
	}
	records, total, err := s.makeupRepo.ListMakeupApplications(ctx, query.ExamID, query.Status, query.Page, query.PageSize)
	if err != nil {
		return dto.PageResult{}, fmt.Errorf("list makeup applications: %w", err)
	}
	page, pageSize := normalizePage(query.Page, query.PageSize)
	items := make([]dto.MakeupApplicationItem, 0, len(records))
	for i := range records {
		item, err := s.toItem(ctx, &records[i])
		if err != nil {
			return dto.PageResult{}, err
		}
		items = append(items, *item)
	}
	return dto.PageResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// Approve approves a pending application and creates an independent makeup
// attempt in one transaction. Conditional updates ensure concurrent approvals
// succeed only once and no orphaned attempt is left behind on conflict.
func (s *MakeupApplicationService) Approve(ctx context.Context, role string, reviewerID, id uint) (*dto.MakeupApplicationItem, error) {
	app, err := s.makeupRepo.FindMakeupApplicationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	exam, err := s.examRepo.FindExamByID(ctx, app.ExamID)
	if err != nil {
		return nil, err
	}
	if role == constants.RoleTeacher && exam.CreatedBy != reviewerID {
		return nil, ErrForbidden
	}
	if app.Status != constants.MakeupPending {
		return nil, fmt.Errorf("%w: 该申请已处理，请勿重复审批", ErrConflict)
	}

	attempt, err := s.attempts.BuildMakeupAttemptModel(ctx, app.StudentID, app.ExamID)
	if err != nil {
		return nil, err
	}
	if err := s.makeupRepo.ApproveMakeupWithAttempt(ctx, id, reviewerID, app.StudentID, app.ExamID, attempt); err != nil {
		return nil, fmt.Errorf("approve makeup application: %w", err)
	}
	approved, err := s.makeupRepo.FindMakeupApplicationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toItem(ctx, approved)
}

// Reject rejects a pending application with an optional remark.
func (s *MakeupApplicationService) Reject(ctx context.Context, role string, reviewerID, id uint, req dto.MakeupReviewRequest) (*dto.MakeupApplicationItem, error) {
	app, err := s.makeupRepo.FindMakeupApplicationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	exam, err := s.examRepo.FindExamByID(ctx, app.ExamID)
	if err != nil {
		return nil, err
	}
	if role == constants.RoleTeacher && exam.CreatedBy != reviewerID {
		return nil, ErrForbidden
	}
	if err := s.makeupRepo.RejectMakeupApplication(ctx, id, reviewerID, req.Remark); err != nil {
		return nil, fmt.Errorf("reject makeup application: %w", err)
	}
	rejected, err := s.makeupRepo.FindMakeupApplicationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toItem(ctx, rejected)
}

// StartMakeup returns the paper of the makeup attempt granted by the student's
// approved application. It refuses if the application is not approved or the
// makeup attempt has already been submitted (makeup in progress check).
func (s *MakeupApplicationService) StartMakeup(ctx context.Context, studentID, examID uint) (*dto.AttemptStartResponse, error) {
	app, err := s.makeupRepo.FindMakeupApplication(ctx, examID, studentID)
	if err != nil {
		return nil, err
	}
	if app.Status != constants.MakeupApproved || app.MakeupAttemptID == nil {
		return nil, fmt.Errorf("%w: 补考申请尚未批准", ErrValidation)
	}
	return s.attempts.StartMakeupAttempt(ctx, studentID, *app.MakeupAttemptID)
}

// checkEligibility encapsulates the rules: the exam must be closed, the
// student must have a record for it (an unfinished normal attempt becomes
// "absent" at close), no attempt may have reached 60% of the exam total, and a
// makeup attempt must not already be in progress.
func (s *MakeupApplicationService) checkEligibility(ctx context.Context, studentID, examID uint) (bool, string, error) {
	exam, err := s.examRepo.FindExamByID(ctx, examID)
	if err != nil {
		return false, "", err
	}
	if exam.Status != constants.ExamClosed {
		return false, "考试尚未关考，暂不能申请补考", nil
	}
	attempts, err := s.attemptRepo.ListAttemptsByStudentAndExam(ctx, examID, studentID)
	if err != nil {
		return false, "", fmt.Errorf("list student attempts: %w", err)
	}
	if len(attempts) == 0 {
		return false, "未参加该考试，不符合补考资格", nil
	}
	passLine := exam.TotalScore * constants.PassThresholdRatio
	for _, a := range attempts {
		if a.Kind == constants.AttemptKindMakeup && a.Status == constants.AttemptInProgress {
			return false, "补考正在进行中，不能重复申请", nil
		}
		if a.Status == constants.AttemptSubmitted && exam.TotalScore > 0 && a.TotalScore >= passLine {
			return false, "已有成绩达到总分 60%，不符合补考资格", nil
		}
	}
	return true, "符合补考资格", nil
}

func (s *MakeupApplicationService) toItem(ctx context.Context, app *model.MakeupApplication) (*dto.MakeupApplicationItem, error) {
	examTitle := ""
	if exam, err := s.examRepo.FindExamByID(ctx, app.ExamID); err == nil {
		examTitle = exam.Title
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	name, username := "", ""
	if user, err := s.userRepo.FindUserByID(ctx, app.StudentID); err == nil {
		name = user.Name
		username = user.Username
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	return &dto.MakeupApplicationItem{
		ID:              app.ID,
		ExamID:          app.ExamID,
		ExamTitle:       examTitle,
		StudentID:       app.StudentID,
		StudentName:     name,
		StudentUsername: username,
		Reason:          app.Reason,
		Status:          app.Status,
		ReviewRemark:    app.ReviewRemark,
		ReviewedBy:      app.ReviewedBy,
		ReviewedAt:      app.ReviewedAt,
		MakeupAttemptID: app.MakeupAttemptID,
		CreatedAt:       app.CreatedAt,
	}, nil
}
