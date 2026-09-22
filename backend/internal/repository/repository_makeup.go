package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/gbexam/online-exam/internal/model"
)

// CreateMakeupApplication inserts a makeup application. A duplicate
// (exam_id, student_id) pair is reported as ErrConflict so duplicate and
// concurrent applications only succeed once.
func (r *Repository) CreateMakeupApplication(ctx context.Context, a *model.MakeupApplication) error {
	if err := r.db.WithContext(ctx).Create(a).Error; err != nil {
		return wrapQuery("create makeup application", err)
	}
	return nil
}

// FindMakeupApplication returns the single application of a student for an exam.
func (r *Repository) FindMakeupApplication(ctx context.Context, examID, studentID uint) (*model.MakeupApplication, error) {
	var a model.MakeupApplication
	err := r.db.WithContext(ctx).
		Where("exam_id = ? AND student_id = ?", examID, studentID).First(&a).Error
	if err != nil {
		return nil, wrapQuery("find makeup application", err)
	}
	return &a, nil
}

// FindMakeupApplicationByID returns an application by its primary key.
func (r *Repository) FindMakeupApplicationByID(ctx context.Context, id uint) (*model.MakeupApplication, error) {
	var a model.MakeupApplication
	err := r.db.WithContext(ctx).First(&a, id).Error
	if err != nil {
		return nil, wrapQuery("find makeup application by id", err)
	}
	return &a, nil
}

// ListMakeupApplications returns applications filtered by exam and status.
func (r *Repository) ListMakeupApplications(ctx context.Context, examID uint, status string, page, pageSize int) ([]model.MakeupApplication, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.MakeupApplication{})
	if examID != 0 {
		q = q.Where("exam_id = ?", examID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count makeup applications: %w", err)
	}
	var items []model.MakeupApplication
	p, ps := NormalizePage(page, pageSize)
	if err := q.Order("id DESC").Limit(ps).Offset((p - 1) * ps).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list makeup applications: %w", err)
	}
	return items, total, nil
}

// ApproveMakeupWithAttempt atomically approves a pending application, creates
// the independent makeup attempt and binds it. Conditional updates ensure
// concurrent or repeated approvals succeed exactly once; any failure rolls the
// whole operation back and leaves no orphaned attempt.
func (r *Repository) ApproveMakeupWithAttempt(ctx context.Context, appID, reviewerID, studentID, examID uint, attempt *model.ExamAttempt) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.MakeupApplication{}).
			Where("id = ? AND status = ?", appID, "pending").
			Updates(map[string]any{
				"status":        "approved",
				"reviewed_by":   reviewerID,
				"reviewed_at":   time.Now(),
				"review_remark": "",
			})
		if res.Error != nil {
			return fmt.Errorf("lock makeup application: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrConflict
		}
		if err := tx.Create(attempt).Error; err != nil {
			return fmt.Errorf("create makeup attempt: %w", err)
		}
		if err := tx.Model(&model.MakeupApplication{}).Where("id = ?", appID).
			Update("makeup_attempt_id", attempt.ID).Error; err != nil {
			return fmt.Errorf("bind makeup attempt: %w", err)
		}
		return nil
	})
}

// RejectMakeupApplication atomically moves a pending application to rejected.
// Concurrent or repeated reviews match zero rows and fail with ErrConflict.
func (r *Repository) RejectMakeupApplication(ctx context.Context, id, reviewerID uint, remark string) error {
	res := r.db.WithContext(ctx).Model(&model.MakeupApplication{}).
		Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]any{
			"status":        "rejected",
			"reviewed_by":   reviewerID,
			"reviewed_at":   time.Now(),
			"review_remark": remark,
		})
	if res.Error != nil {
		return fmt.Errorf("reject makeup application: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrConflict
	}
	return nil
}
