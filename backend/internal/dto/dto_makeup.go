package dto

import "time"

// MakeupApplyRequest is the student application payload.
type MakeupApplyRequest struct {
	Reason string `json:"reason" binding:"max=255"`
}

// MakeupRequestItem describes one makeup application.
type MakeupRequestItem struct {
	ID              uint       `json:"id"`
	ExamID          uint       `json:"exam_id"`
	ExamTitle       string     `json:"exam_title,omitempty"`
	StudentID       uint       `json:"student_id"`
	StudentName     string     `json:"student_name,omitempty"`
	StudentUsername string     `json:"student_username,omitempty"`
	Reason          string     `json:"reason"`
	Status          string     `json:"status"`
	ReviewedBy      uint       `json:"reviewed_by"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	AttemptID       uint       `json:"attempt_id"`
	CreatedAt       time.Time  `json:"created_at"`
}

// MakeupStatusResponse tells a student whether they may apply for a makeup
// exam, the state of their application and their effective score.
type MakeupStatusResponse struct {
	ExamID          uint               `json:"exam_id"`
	PassScore       float64            `json:"pass_score"`
	EffectiveScore  *float64           `json:"effective_score"`
	Eligible        bool               `json:"eligible"`
	CanApply        bool               `json:"can_apply"`
	Message         string             `json:"message"`
	MakeupAttemptID uint               `json:"makeup_attempt_id,omitempty"`
	MakeupStatus    string             `json:"makeup_status,omitempty"`
	Request         *MakeupRequestItem `json:"request,omitempty"`
}
