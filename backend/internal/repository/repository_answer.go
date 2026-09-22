package repository

import (
	"context"
	"fmt"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/model"
)

// SaveAnswer inserts or updates one answer.
func (r *Repository) SaveAnswer(ctx context.Context, answer *model.Answer) error {
	var existing model.Answer
	err := r.db.WithContext(ctx).Where("attempt_id = ? AND exam_question_id = ?", answer.AttemptID, answer.ExamQuestionID).First(&existing).Error
	if err == nil {
		if updateErr := r.db.WithContext(ctx).Model(&model.Answer{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"answer_text": answer.AnswerText,
			"is_correct":  answer.IsCorrect,
			"score":       answer.Score,
			"marked":      answer.Marked,
			"graded_by":   answer.GradedBy,
		}).Error; updateErr != nil {
			return fmt.Errorf("update answer: %w", updateErr)
		}
		return nil
	}
	if isRecordNotFound(err) {
		if createErr := r.db.WithContext(ctx).Create(answer).Error; createErr != nil {
			return fmt.Errorf("create answer: %w", createErr)
		}
		return nil
	}
	return fmt.Errorf("find answer: %w", err)
}

// ListAnswersByAttempt returns all answers for an attempt.
func (r *Repository) ListAnswersByAttempt(ctx context.Context, attemptID uint) ([]model.Answer, error) {
	var items []model.Answer
	if err := r.db.WithContext(ctx).Where("attempt_id = ?", attemptID).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list answers: %w", err)
	}
	return items, nil
}

// HasOtherSubmittedWrongAnswer reports whether the student has the question
// marked wrong (is_correct = false) in any other submitted attempt.
func (r *Repository) HasOtherSubmittedWrongAnswer(ctx context.Context, studentID, questionID, excludeAttemptID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Answer{}).
		Joins("JOIN exam_attempts ON exam_attempts.id = answers.attempt_id").
		Where("answers.question_id = ? AND answers.attempt_id <> ?", questionID, excludeAttemptID).
		Where("answers.is_correct = 0").
		Where("exam_attempts.student_id = ? AND exam_attempts.status = ?", studentID, constants.AttemptSubmitted).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("count other wrong answers: %w", err)
	}
	return count > 0, nil
}
