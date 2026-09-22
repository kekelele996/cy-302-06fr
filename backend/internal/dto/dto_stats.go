package dto

import "time"

// RankItem is one student in a score ranking.
type RankItem struct {
	Rank            int        `json:"rank"`
	StudentName     string     `json:"student_name"`
	StudentUsername string     `json:"student_username"`
	TotalScore      float64    `json:"total_score"`
	Kind            string     `json:"kind"`
	AttemptID       uint       `json:"attempt_id"`
	SubmittedAt     *time.Time `json:"submitted_at"`
}

// RawAttemptRow is one original attempt record in the statistics view.
type RawAttemptRow struct {
	AttemptID       uint       `json:"attempt_id"`
	StudentName     string     `json:"student_name"`
	StudentUsername string     `json:"student_username"`
	Kind            string     `json:"kind"`
	Status          string     `json:"status"`
	TotalScore      float64    `json:"total_score"`
	IsEffective     bool       `json:"is_effective"`
	SubmittedAt     *time.Time `json:"submitted_at"`
}

// StatSummary aggregates a set of attempts (original records or effective scores).
type StatSummary struct {
	Count        int           `json:"count"`
	AverageScore float64       `json:"average_score"`
	HighestScore float64       `json:"highest_score"`
	LowestScore  float64       `json:"lowest_score"`
	PassCount    int           `json:"pass_count"`
	Distribution []ScoreBucket `json:"distribution"`
}

// ScoreBucket is a histogram bucket.
type ScoreBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// ExamStatResponse is teacher-facing exam statistics. It distinguishes the raw
// attempt records from the per-student effective (best-of-two) results.
type ExamStatResponse struct {
	ExamID      uint            `json:"exam_id"`
	ExamTitle   string          `json:"exam_title"`
	AbsentCount int             `json:"absent_count"`
	Original    StatSummary     `json:"original"`
	Effective   StatSummary     `json:"effective"`
	Ranking     []RankItem      `json:"ranking"`
	RawAttempts []RawAttemptRow `json:"raw_attempts"`
}

// OverviewResponse is a compact dashboard summary.
type OverviewResponse struct {
	UserCount     int64 `json:"user_count"`
	QuestionCount int64 `json:"question_count"`
	ExamCount     int64 `json:"exam_count"`
	AttemptCount  int64 `json:"attempt_count"`
}
