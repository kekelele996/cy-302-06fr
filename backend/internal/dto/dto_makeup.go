package dto

import "time"

// MakeupApplyRequest is submitted by a student to request a makeup exam.
type MakeupApplyRequest struct {
	Reason string `json:"reason" binding:"max=500"`
}

// MakeupReviewRequest is submitted by a teacher to reject an application.
type MakeupReviewRequest struct {
	Remark string `json:"remark" binding:"max=500"`
}

// MakeupApplicationListQuery filters the teacher application list.
type MakeupApplicationListQuery struct {
	PageQuery
	ExamID uint   `form:"exam_id"`
	Status string `form:"status" binding:"omitempty,oneof=pending approved rejected"`
}

// MakeupApplicationItem is one application with exam and student metadata.
type MakeupApplicationItem struct {
	ID              uint       `json:"id"`
	ExamID          uint       `json:"exam_id"`
	ExamTitle       string     `json:"exam_title"`
	StudentID       uint       `json:"student_id"`
	StudentName     string     `json:"student_name"`
	StudentUsername string     `json:"student_username"`
	Reason          string     `json:"reason"`
	Status          string     `json:"status"`
	ReviewRemark    string     `json:"review_remark"`
	ReviewedBy      uint       `json:"reviewed_by"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	MakeupAttemptID *uint      `json:"makeup_attempt_id"`
	CreatedAt       time.Time  `json:"created_at"`
}

// MakeupEligibility describes whether a student may apply for a makeup exam.
type MakeupEligibility struct {
	ExamID      uint                   `json:"exam_id"`
	Eligible    bool                   `json:"eligible"`
	Reason      string                 `json:"reason"`
	Application *MakeupApplicationItem `json:"application"`
}

// MakeupApplicationDetail is the application view used by both sides.
type MakeupApplicationDetail = MakeupApplicationItem
