package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/model"
)

// CreateAttempt inserts an exam attempt.
func (r *Repository) CreateAttempt(ctx context.Context, a *model.ExamAttempt) error {
	if err := r.db.WithContext(ctx).Create(a).Error; err != nil {
		return fmt.Errorf("create attempt: %w", err)
	}
	return nil
}

// FindAttemptByID returns an attempt.
func (r *Repository) FindAttemptByID(ctx context.Context, id uint) (*model.ExamAttempt, error) {
	var a model.ExamAttempt
	err := r.db.WithContext(ctx).First(&a, id).Error
	if err != nil {
		return nil, wrapQuery("find attempt by id", err)
	}
	return &a, nil
}

// UpdateAttempt updates attempt status and scores.
func (r *Repository) UpdateAttempt(ctx context.Context, a *model.ExamAttempt) error {
	res := r.db.WithContext(ctx).Model(&model.ExamAttempt{}).Where("id = ?", a.ID).Updates(map[string]any{
		"status":          a.Status,
		"submitted_at":    a.SubmittedAt,
		"deadline":        a.Deadline,
		"objective_score": a.ObjectiveScore,
		"total_score":     a.TotalScore,
	})
	if res.Error != nil {
		return fmt.Errorf("update attempt: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// FindInProgressAttemptByKind returns the student's current unfinished attempt
// of the given kind (normal/makeup) for an exam.
func (r *Repository) FindInProgressAttemptByKind(ctx context.Context, examID, studentID uint, kind string) (*model.ExamAttempt, error) {
	var a model.ExamAttempt
	err := r.db.WithContext(ctx).
		Where("exam_id = ? AND student_id = ? AND status = ? AND kind = ?", examID, studentID, constants.AttemptInProgress, kind).
		Order("id DESC").First(&a).Error
	if err != nil {
		return nil, wrapQuery("find in progress attempt", err)
	}
	return &a, nil
}

// ListAttemptsByStudent returns a page of attempts for a student.
func (r *Repository) ListAttemptsByStudent(ctx context.Context, studentID, examID uint, page, pageSize int) ([]model.ExamAttempt, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.ExamAttempt{}).Where("student_id = ?", studentID)
	if examID != 0 {
		q = q.Where("exam_id = ?", examID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count attempts: %w", err)
	}
	var items []model.ExamAttempt
	p, ps := NormalizePage(page, pageSize)
	if err := q.Order("id DESC").Limit(ps).Offset((p - 1) * ps).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list attempts: %w", err)
	}
	return items, total, nil
}

// ListAttemptsByStudentAndExam returns every attempt of a student for one exam.
func (r *Repository) ListAttemptsByStudentAndExam(ctx context.Context, examID, studentID uint) ([]model.ExamAttempt, error) {
	var items []model.ExamAttempt
	if err := r.db.WithContext(ctx).
		Where("exam_id = ? AND student_id = ?", examID, studentID).
		Order("id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list attempts by student and exam: %w", err)
	}
	return items, nil
}

// ListAttemptsByExam returns all attempts for an exam.
func (r *Repository) ListAttemptsByExam(ctx context.Context, examID uint) ([]model.ExamAttempt, error) {
	var items []model.ExamAttempt
	if err := r.db.WithContext(ctx).Where("exam_id = ?", examID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list attempts by exam: %w", err)
	}
	return items, nil
}

// ActivateAttempt starts the countdown of a not-yet-activated attempt (a
// makeup attempt created at approval time). The conditional update guarantees
// the countdown starts only on the student's first open. It reports whether
// this call performed the activation.
func (r *Repository) ActivateAttempt(ctx context.Context, id uint, startedAt, deadline time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.ExamAttempt{}).
		Where("id = ? AND activated = ?", id, false).
		Updates(map[string]any{
			"activated":  true,
			"started_at": startedAt,
			"deadline":   deadline,
		})
	if res.Error != nil {
		return false, fmt.Errorf("activate attempt: %w", res.Error)
	}
	return res.RowsAffected > 0, nil
}

// FinalizeExpiredAttempt marks a single activated, past-deadline attempt as
// absent. It is the lazy counterpart of the batch absent marking at exam close.
func (r *Repository) FinalizeExpiredAttempt(ctx context.Context, id uint) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.ExamAttempt{}).
		Where("id = ? AND status = ? AND activated = ? AND deadline < ?",
			id, constants.AttemptInProgress, true, time.Now()).
		Update("status", constants.AttemptAbsent)
	if res.Error != nil {
		return false, fmt.Errorf("finalize expired attempt: %w", res.Error)
	}
	return res.RowsAffected > 0, nil
}

// CloseExamAndMarkAbsent atomically closes an exam and marks every unfinished
// normal attempt as absent.
func (r *Repository) CloseExamAndMarkAbsent(ctx context.Context, examID uint) (int64, error) {
	var absentCount int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		examRes := tx.Model(&model.Exam{}).Where("id = ?", examID).Update("status", constants.ExamClosed)
		if examRes.Error != nil {
			return fmt.Errorf("close exam: %w", examRes.Error)
		}
		if examRes.RowsAffected == 0 {
			return ErrNotFound
		}
		res := tx.Model(&model.ExamAttempt{}).
			Where("exam_id = ? AND status = ? AND kind = ?", examID, constants.AttemptInProgress, constants.AttemptKindNormal).
			Update("status", constants.AttemptAbsent)
		if res.Error != nil {
			return fmt.Errorf("mark absent attempts: %w", res.Error)
		}
		absentCount = res.RowsAffected
		return nil
	})
	if err != nil {
		return 0, err
	}
	return absentCount, nil
}

// CountAttempts returns the total number of attempts.
func (r *Repository) CountAttempts(ctx context.Context) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.ExamAttempt{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("count attempts: %w", err)
	}
	return total, nil
}
