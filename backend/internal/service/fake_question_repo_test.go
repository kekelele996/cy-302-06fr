package service

import (
	"context"

	"github.com/gbexam/online-exam/internal/model"
	"github.com/gbexam/online-exam/internal/repository"
)

type fakeQuestionRepo struct {
	questions map[uint]model.Question
}

func (f *fakeQuestionRepo) CreateQuestion(_ context.Context, _ *model.Question) error { return nil }
func (f *fakeQuestionRepo) CreateQuestionsBatch(_ context.Context, _ []model.Question) error {
	return nil
}
func (f *fakeQuestionRepo) FindQuestionByID(_ context.Context, id uint) (*model.Question, error) {
	if q, ok := f.questions[id]; ok {
		return &q, nil
	}
	return nil, repository.ErrNotFound
}
func (f *fakeQuestionRepo) UpdateQuestion(_ context.Context, _ *model.Question) error { return nil }
func (f *fakeQuestionRepo) DeleteQuestion(_ context.Context, _ uint) error            { return nil }
func (f *fakeQuestionRepo) ListQuestions(_ context.Context, _ repository.QuestionFilter, _, _ int) ([]model.Question, int64, error) {
	return nil, 0, nil
}
func (f *fakeQuestionRepo) ListQuestionsByTypeDifficulty(_ context.Context, _, _ string) ([]model.Question, error) {
	return nil, nil
}
func (f *fakeQuestionRepo) FindQuestionsByIDs(_ context.Context, ids []uint) (map[uint]model.Question, error) {
	out := make(map[uint]model.Question, len(ids))
	for _, id := range ids {
		if q, ok := f.questions[id]; ok {
			out[id] = q
		}
	}
	return out, nil
}
