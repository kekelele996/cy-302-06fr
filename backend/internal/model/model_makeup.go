package model

import "time"

// MakeupApplication is a student's request for one makeup exam opportunity.
// The unique index on (exam_id, student_id) guarantees at most one application
// per student per exam, so duplicate or concurrent requests can only succeed once.
type MakeupApplication struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	ExamID          uint       `gorm:"uniqueIndex:uniq_makeup_exam_student;not null" json:"exam_id"`
	StudentID       uint       `gorm:"uniqueIndex:uniq_makeup_exam_student;not null" json:"student_id"`
	Reason          string     `gorm:"size:500;not null;default:''" json:"reason"`
	Status          string     `gorm:"size:16;not null;default:pending;index" json:"status"`
	ReviewRemark    string     `gorm:"size:500;not null;default:''" json:"review_remark"`
	ReviewedBy      uint       `gorm:"default:0" json:"reviewed_by"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	MakeupAttemptID *uint      `gorm:"index" json:"makeup_attempt_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
