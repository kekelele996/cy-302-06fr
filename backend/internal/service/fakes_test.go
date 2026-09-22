package service

import (
	"context"

	"github.com/gbexam/online-exam/internal/model"
	"github.com/gbexam/online-exam/internal/repository"
)

type fakeExamRepo struct {
	exam      *model.Exam
	closed    int
	questions []model.ExamQuestion
}

func (f *fakeExamRepo) CreateExam(_ context.Context, _ *model.Exam) error { return nil }
func (f *fakeExamRepo) FindExamByID(_ context.Context, id uint) (*model.Exam, error) {
	if f.exam != nil && f.exam.ID == id {
		cp := *f.exam
		return &cp, nil
	}
	return nil, repository.ErrNotFound
}
func (f *fakeExamRepo) UpdateExam(_ context.Context, _ *model.Exam) error { return nil }
func (f *fakeExamRepo) DeleteExam(_ context.Context, _ uint) error        { return nil }
func (f *fakeExamRepo) ListExams(_ context.Context, _ repository.ExamFilter, _, _ int) ([]model.Exam, int64, error) {
	return nil, 0, nil
}
func (f *fakeExamRepo) ReplaceExamQuestions(_ context.Context, _ uint, _ []model.ExamQuestion) error {
	return nil
}
func (f *fakeExamRepo) ListExamQuestions(_ context.Context, _ uint) ([]model.ExamQuestion, error) {
	return f.questions, nil
}
func (f *fakeExamRepo) CountExamQuestions(_ context.Context, _ uint) (int64, error) {
	return int64(len(f.questions)), nil
}
func (f *fakeExamRepo) CloseExamAndMarkAbsent(_ context.Context, _ uint) (int64, error) {
	f.closed++
	return 0, nil
}

type fakeUserRepo struct {
	users map[uint]model.User
}

func (f *fakeUserRepo) CreateUser(_ context.Context, _ *model.User) error { return nil }
func (f *fakeUserRepo) FindUserByUsername(_ context.Context, _ string) (*model.User, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeUserRepo) FindUserByID(_ context.Context, id uint) (*model.User, error) {
	if u, ok := f.users[id]; ok {
		return &u, nil
	}
	return nil, repository.ErrNotFound
}
func (f *fakeUserRepo) ListUsers(_ context.Context, _, _ string, _, _ int) ([]model.User, int64, error) {
	return nil, 0, nil
}
func (f *fakeUserRepo) UpdateUserStatus(_ context.Context, _ uint, _ string) error { return nil }
