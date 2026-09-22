package model

import "time"

// MakeupRequest is a student's application for a makeup exam.
// The unique index on (exam_id, student_id) guarantees one application per
// student per exam, so duplicate or concurrent applications only succeed once.
type MakeupRequest struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	ExamID     uint       `gorm:"uniqueIndex:idx_makeup_exam_student;not null" json:"exam_id"`
	StudentID  uint       `gorm:"uniqueIndex:idx_makeup_exam_student;not null" json:"student_id"`
	Reason     string     `gorm:"size:255" json:"reason"`
	Status     string     `gorm:"size:16;not null;default:pending;index" json:"status"`
	ReviewedBy uint       `json:"reviewed_by"`
	ReviewedAt *time.Time `json:"reviewed_at"`
	AttemptID  uint       `json:"attempt_id"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
