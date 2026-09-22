package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/gbexam/online-exam/internal/model"
)

// CreateMakeupRequest inserts a makeup application. The unique index on
// (exam_id, student_id) turns duplicate or concurrent applications into ErrConflict.
func (r *Repository) CreateMakeupRequest(ctx context.Context, req *model.MakeupRequest) error {
	if err := r.db.WithContext(ctx).Create(req).Error; err != nil {
		if isDuplicateKey(err) {
			return ErrConflict
		}
		return fmt.Errorf("create makeup request: %w", err)
	}
	return nil
}

// FindMakeupRequest returns the application of one student for one exam.
func (r *Repository) FindMakeupRequest(ctx context.Context, examID, studentID uint) (*model.MakeupRequest, error) {
	var req model.MakeupRequest
	err := r.db.WithContext(ctx).Where("exam_id = ? AND student_id = ?", examID, studentID).First(&req).Error
	if err != nil {
		return nil, wrapQuery("find makeup request", err)
	}
	return &req, nil
}

// FindMakeupRequestByID returns an application by id.
func (r *Repository) FindMakeupRequestByID(ctx context.Context, id uint) (*model.MakeupRequest, error) {
	var req model.MakeupRequest
	err := r.db.WithContext(ctx).First(&req, id).Error
	if err != nil {
		return nil, wrapQuery("find makeup request by id", err)
	}
	return &req, nil
}

// ListMakeupRequestsByExam returns all applications for an exam.
func (r *Repository) ListMakeupRequestsByExam(ctx context.Context, examID uint) ([]model.MakeupRequest, error) {
	var items []model.MakeupRequest
	if err := r.db.WithContext(ctx).Where("exam_id = ?", examID).Order("id DESC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list makeup requests: %w", err)
	}
	return items, nil
}

// ReapplyMakeupRequest flips a rejected application back to pending. The
// conditional update makes concurrent state changes fail with ErrConflict.
func (r *Repository) ReapplyMakeupRequest(ctx context.Context, id uint, reason string) error {
	res := r.db.WithContext(ctx).Model(&model.MakeupRequest{}).
		Where("id = ? AND status = ?", id, "rejected").
		Updates(map[string]any{
			"status":      "pending",
			"reason":      reason,
			"reviewed_by": 0,
			"reviewed_at": nil,
			"attempt_id":  0,
		})
	if res.Error != nil {
		return fmt.Errorf("reapply makeup request: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrConflict
	}
	return nil
}

// ApproveMakeupRequest atomically marks a pending application approved and
// creates the makeup attempt. The conditional status flip guarantees that
// concurrent approvals only succeed once.
func (r *Repository) ApproveMakeupRequest(ctx context.Context, id, reviewerID uint, attempt *model.ExamAttempt) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		res := tx.Model(&model.MakeupRequest{}).
			Where("id = ? AND status = ?", id, "pending").
			Updates(map[string]any{
				"status":      "approved",
				"reviewed_by": reviewerID,
				"reviewed_at": &now,
			})
		if res.Error != nil {
			return fmt.Errorf("approve makeup request: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrConflict
		}
		if err := tx.Create(attempt).Error; err != nil {
			return fmt.Errorf("create makeup attempt: %w", err)
		}
		if err := tx.Model(&model.MakeupRequest{}).Where("id = ?", id).Update("attempt_id", attempt.ID).Error; err != nil {
			return fmt.Errorf("bind makeup attempt: %w", err)
		}
		return nil
	})
}

// RejectMakeupRequest marks a pending application rejected. The conditional
// update makes concurrent reviews fail with ErrConflict.
func (r *Repository) RejectMakeupRequest(ctx context.Context, id, reviewerID uint) error {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&model.MakeupRequest{}).
		Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]any{
			"status":      "rejected",
			"reviewed_by": reviewerID,
			"reviewed_at": &now,
		})
	if res.Error != nil {
		return fmt.Errorf("reject makeup request: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrConflict
	}
	return nil
}
