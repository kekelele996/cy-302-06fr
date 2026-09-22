package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/dto"
	"github.com/gbexam/online-exam/internal/model"
	"github.com/gbexam/online-exam/internal/repository"
)

// fakeMakeupRepo is an in-memory MakeupRepo for service tests.
type fakeMakeupRepo struct {
	apps     map[uint]*model.MakeupApplication
	nextID   uint
	approves int
}

func newFakeMakeupRepo() *fakeMakeupRepo {
	return &fakeMakeupRepo{apps: map[uint]*model.MakeupApplication{}, nextID: 1}
}

func (f *fakeMakeupRepo) CreateMakeupApplication(_ context.Context, a *model.MakeupApplication) error {
	for _, ex := range f.apps {
		if ex.ExamID == a.ExamID && ex.StudentID == a.StudentID {
			return repository.ErrConflict
		}
	}
	a.ID = f.nextID
	f.nextID++
	cp := *a
	f.apps[a.ID] = &cp
	return nil
}

func (f *fakeMakeupRepo) FindMakeupApplication(_ context.Context, examID, studentID uint) (*model.MakeupApplication, error) {
	for _, a := range f.apps {
		if a.ExamID == examID && a.StudentID == studentID {
			cp := *a
			return &cp, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeMakeupRepo) FindMakeupApplicationByID(_ context.Context, id uint) (*model.MakeupApplication, error) {
	a, ok := f.apps[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *a
	return &cp, nil
}

func (f *fakeMakeupRepo) ListMakeupApplications(_ context.Context, examID uint, status string, _, _ int) ([]model.MakeupApplication, int64, error) {
	out := make([]model.MakeupApplication, 0)
	for _, a := range f.apps {
		if examID != 0 && a.ExamID != examID {
			continue
		}
		if status != "" && a.Status != status {
			continue
		}
		out = append(out, *a)
	}
	return out, int64(len(out)), nil
}

func (f *fakeMakeupRepo) ApproveMakeupWithAttempt(_ context.Context, appID, _, _, _ uint, attempt *model.ExamAttempt) error {
	a, ok := f.apps[appID]
	if !ok || a.Status != constants.MakeupPending {
		return repository.ErrConflict
	}
	f.approves++
	a.Status = constants.MakeupApproved
	a.MakeupAttemptID = &attempt.ID
	return nil
}

func (f *fakeMakeupRepo) RejectMakeupApplication(_ context.Context, id, _ uint, remark string) error {
	a, ok := f.apps[id]
	if !ok || a.Status != constants.MakeupPending {
		return repository.ErrConflict
	}
	a.Status = constants.MakeupRejected
	a.ReviewRemark = remark
	return nil
}

type fakeAttemptRepoMakeup struct {
	attempts []model.ExamAttempt
}

func (f *fakeAttemptRepoMakeup) CreateAttempt(_ context.Context, a *model.ExamAttempt) error {
	a.ID = uint(len(f.attempts) + 100)
	f.attempts = append(f.attempts, *a)
	return nil
}
func (f *fakeAttemptRepoMakeup) FindAttemptByID(_ context.Context, id uint) (*model.ExamAttempt, error) {
	for i := range f.attempts {
		if f.attempts[i].ID == id {
			return &f.attempts[i], nil
		}
	}
	return nil, repository.ErrNotFound
}
func (f *fakeAttemptRepoMakeup) UpdateAttempt(_ context.Context, _ *model.ExamAttempt) error {
	return nil
}
func (f *fakeAttemptRepoMakeup) FindInProgressAttemptByKind(_ context.Context, _, _ uint, _ string) (*model.ExamAttempt, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeAttemptRepoMakeup) ActivateAttempt(_ context.Context, _ uint, _, _ time.Time) (bool, error) {
	return true, nil
}
func (f *fakeAttemptRepoMakeup) FinalizeExpiredAttempt(_ context.Context, _ uint) (bool, error) {
	return false, nil
}
func (f *fakeAttemptRepoMakeup) ListAttemptsByStudent(_ context.Context, _, _ uint, _, _ int) ([]model.ExamAttempt, int64, error) {
	return nil, 0, nil
}
func (f *fakeAttemptRepoMakeup) ListAttemptsByStudentAndExam(_ context.Context, examID, studentID uint) ([]model.ExamAttempt, error) {
	out := make([]model.ExamAttempt, 0)
	for _, a := range f.attempts {
		if a.ExamID == examID && a.StudentID == studentID {
			out = append(out, a)
		}
	}
	return out, nil
}
func (f *fakeAttemptRepoMakeup) ListAttemptsByExam(_ context.Context, _ uint) ([]model.ExamAttempt, error) {
	return f.attempts, nil
}

func newMakeupServiceForTest(exam *model.Exam, attempts []model.ExamAttempt) (*MakeupApplicationService, *fakeMakeupRepo) {
	examRepo := &fakeExamRepo{
		exam: exam,
		questions: []model.ExamQuestion{
			{ID: 1, ExamID: exam.ID, QuestionID: 50, Score: 10, SortOrder: 0},
		},
	}
	attemptRepo := &fakeAttemptRepoMakeup{attempts: attempts}
	makeupRepo := newFakeMakeupRepo()
	questionRepo := &fakeQuestionRepo{questions: map[uint]model.Question{
		50: {ID: 50, Type: constants.QuestionSingle, Content: "q"},
	}}
	attemptSvc := NewAttemptService(examRepo, questionRepo, attemptRepo, nil, nil, nil)
	svc := NewMakeupApplicationService(makeupRepo, examRepo, attemptRepo, &fakeUserRepo{users: map[uint]model.User{
		1: {ID: 1, Username: "stu", Name: "学生甲"},
	}}, attemptSvc, nil)
	return svc, makeupRepo
}

func TestApplyRejectedWhenExamNotClosed(t *testing.T) {
	svc, _ := newMakeupServiceForTest(
		&model.Exam{ID: 1, Status: constants.ExamPublished, TotalScore: 100},
		[]model.ExamAttempt{{ID: 10, ExamID: 1, StudentID: 1, Kind: constants.AttemptKindNormal, Status: constants.AttemptSubmitted, TotalScore: 30}},
	)
	_, err := svc.Apply(context.Background(), 1, 1, dto.MakeupApplyRequest{})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestApplyRejectedWhenPassing(t *testing.T) {
	svc, _ := newMakeupServiceForTest(
		&model.Exam{ID: 1, Status: constants.ExamClosed, TotalScore: 100},
		[]model.ExamAttempt{{ID: 10, ExamID: 1, StudentID: 1, Kind: constants.AttemptKindNormal, Status: constants.AttemptSubmitted, TotalScore: 60}},
	)
	_, err := svc.Apply(context.Background(), 1, 1, dto.MakeupApplyRequest{})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for passing score, got %v", err)
	}
}

func TestAbsentStudentCanApplyAndApproveOnce(t *testing.T) {
	svc, makeupRepo := newMakeupServiceForTest(
		&model.Exam{ID: 1, Status: constants.ExamClosed, TotalScore: 100},
		[]model.ExamAttempt{{ID: 10, ExamID: 1, StudentID: 1, Kind: constants.AttemptKindNormal, Status: constants.AttemptAbsent}},
	)
	item, err := svc.Apply(context.Background(), 1, 1, dto.MakeupApplyRequest{Reason: "缺考"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if item.Status != constants.MakeupPending {
		t.Fatalf("expected pending, got %s", item.Status)
	}

	// duplicate application rejected
	if _, err := svc.Apply(context.Background(), 1, 1, dto.MakeupApplyRequest{}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict on duplicate apply, got %v", err)
	}

	approved, err := svc.Approve(context.Background(), constants.RoleAdmin, 9, item.ID)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approved.Status != constants.MakeupApproved || approved.MakeupAttemptID == nil {
		t.Fatalf("expected approved with bound attempt, got %+v", approved)
	}

	// concurrent / repeated approval fails exactly once
	if _, err := svc.Approve(context.Background(), constants.RoleAdmin, 9, item.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict on double approve, got %v", err)
	}
	if makeupRepo.approves != 1 {
		t.Fatalf("expected exactly 1 approval side effect, got %d", makeupRepo.approves)
	}
}

func TestApplyRejectedWhenMakeupInProgress(t *testing.T) {
	svc, _ := newMakeupServiceForTest(
		&model.Exam{ID: 1, Status: constants.ExamClosed, TotalScore: 100},
		[]model.ExamAttempt{
			{ID: 10, ExamID: 1, StudentID: 1, Kind: constants.AttemptKindNormal, Status: constants.AttemptSubmitted, TotalScore: 30},
			{ID: 11, ExamID: 1, StudentID: 1, Kind: constants.AttemptKindMakeup, Status: constants.AttemptInProgress},
		},
	)
	// no prior application row; eligibility itself must refuse
	_, err := svc.Apply(context.Background(), 1, 1, dto.MakeupApplyRequest{})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error with makeup in progress, got %v", err)
	}
}
