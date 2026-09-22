package dto

import "time"

// RankItem is one student in a score ranking.
type RankItem struct {
	Rank            int        `json:"rank"`
	StudentName     string     `json:"student_name"`
	StudentUsername string     `json:"student_username"`
	TotalScore      float64    `json:"total_score"`
	Source          string     `json:"source"`
	SubmittedAt     *time.Time `json:"submitted_at"`
}

// ScoreBucket is a histogram bucket.
type ScoreBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// ExamStatResponse is teacher-facing exam statistics. The headline numbers are
// computed on effective scores (best of original/makeup per student), while the
// raw counters keep the original records distinguishable.
type ExamStatResponse struct {
	ExamID            uint          `json:"exam_id"`
	ExamTitle         string        `json:"exam_title"`
	ParticipantCount  int           `json:"participant_count"`
	RecordCount       int           `json:"record_count"`
	AbsentCount       int           `json:"absent_count"`
	MakeupCount       int           `json:"makeup_count"`
	AverageScore      float64       `json:"average_score"`
	RawAverageScore   float64       `json:"raw_average_score"`
	HighestScore      float64       `json:"highest_score"`
	LowestScore       float64       `json:"lowest_score"`
	PassCount         int           `json:"pass_count"`
	ScoreDistribution []ScoreBucket `json:"score_distribution"`
	Ranking           []RankItem    `json:"ranking"`
}

// OverviewResponse is a compact dashboard summary.
type OverviewResponse struct {
	UserCount     int64 `json:"user_count"`
	QuestionCount int64 `json:"question_count"`
	ExamCount     int64 `json:"exam_count"`
	AttemptCount  int64 `json:"attempt_count"`
}
