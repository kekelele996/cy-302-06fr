package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/dto"
	"github.com/gbexam/online-exam/internal/model"
)

// StatsService builds dashboard and exam statistics.
type StatsService struct {
	baseService
	repo StatsRepo
}

// NewStatsService constructs StatsService.
func NewStatsService(repo StatsRepo, logger *slog.Logger) *StatsService {
	return &StatsService{baseService: NewBaseService(logger), repo: repo}
}

// Overview returns aggregate counts for the dashboard.
func (s *StatsService) Overview(ctx context.Context) (*dto.OverviewResponse, error) {
	users, err := s.repo.CountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	questions, err := s.repo.CountQuestions(ctx)
	if err != nil {
		return nil, fmt.Errorf("count questions: %w", err)
	}
	exams, err := s.repo.CountExams(ctx)
	if err != nil {
		return nil, fmt.Errorf("count exams: %w", err)
	}
	attempts, err := s.repo.CountAttempts(ctx)
	if err != nil {
		return nil, fmt.Errorf("count attempts: %w", err)
	}
	return &dto.OverviewResponse{
		UserCount:     users,
		QuestionCount: questions,
		ExamCount:     exams,
		AttemptCount:  attempts,
	}, nil
}

// ExamStats returns score statistics and ranking for one exam. Original
// statistics are computed from raw submitted records; effective statistics are
// computed from each student's highest valid total (original or makeup).
func (s *StatsService) ExamStats(ctx context.Context, role string, userID, examID uint) (*dto.ExamStatResponse, error) {
	exam, err := s.repo.FindExamByID(ctx, examID)
	if err != nil {
		return nil, err
	}
	if role == constants.RoleTeacher && exam.CreatedBy != userID {
		return nil, ErrForbidden
	}
	attempts, err := s.repo.ListAttemptsByExam(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("list attempts by exam: %w", err)
	}

	submitted := make([]model.ExamAttempt, 0, len(attempts))
	absentCount := 0
	for _, a := range attempts {
		switch a.Status {
		case constants.AttemptSubmitted:
			submitted = append(submitted, a)
		case constants.AttemptAbsent:
			absentCount++
		}
	}

	effectiveRows := EffectiveAttempts(attempts)
	effective := make([]model.ExamAttempt, 0, len(effectiveRows))
	for _, row := range effectiveRows {
		if row.Attempt != nil {
			effective = append(effective, *row.Attempt)
		}
	}
	SortAttemptsByScore(effective)

	originalSummary := summarize(submitted, exam.TotalScore)
	effectiveSummary := summarize(effective, exam.TotalScore)

	ranking := make([]dto.RankItem, 0, len(effective))
	for i, a := range effective {
		name, username := s.studentName(ctx, a.StudentID)
		ranking = append(ranking, dto.RankItem{
			Rank:            i + 1,
			StudentName:     name,
			StudentUsername: username,
			TotalScore:      a.TotalScore,
			Kind:            a.Kind,
			AttemptID:       a.ID,
			SubmittedAt:     a.SubmittedAt,
		})
	}

	rawRows := make([]dto.RawAttemptRow, 0, len(attempts))
	for _, a := range attempts {
		name, username := s.studentName(ctx, a.StudentID)
		isEffective := false
		if row, ok := effectiveRows[a.StudentID]; ok && row.Attempt != nil {
			isEffective = row.Attempt.ID == a.ID && a.Status == constants.AttemptSubmitted
		}
		rawRows = append(rawRows, dto.RawAttemptRow{
			AttemptID:       a.ID,
			StudentName:     name,
			StudentUsername: username,
			Kind:            a.Kind,
			Status:          a.Status,
			TotalScore:      a.TotalScore,
			IsEffective:     isEffective,
			SubmittedAt:     a.SubmittedAt,
		})
	}

	return &dto.ExamStatResponse{
		ExamID:      exam.ID,
		ExamTitle:   exam.Title,
		AbsentCount: absentCount,
		Original:    originalSummary,
		Effective:   effectiveSummary,
		Ranking:     ranking,
		RawAttempts: rawRows,
	}, nil
}

func (s *StatsService) studentName(ctx context.Context, studentID uint) (string, string) {
	if user, err := s.repo.FindUserByID(ctx, studentID); err == nil {
		return user.Name, user.Username
	}
	return "", ""
}

func summarize(attempts []model.ExamAttempt, totalScore float64) dto.StatSummary {
	SortAttemptsByScore(attempts)
	summary := dto.StatSummary{
		Count:        len(attempts),
		Distribution: buildScoreBuckets(attempts, totalScore),
	}
	if len(attempts) == 0 {
		return summary
	}
	sum := 0.0
	summary.HighestScore = attempts[0].TotalScore
	summary.LowestScore = attempts[len(attempts)-1].TotalScore
	for _, a := range attempts {
		sum += a.TotalScore
		if totalScore > 0 && a.TotalScore >= totalScore*constants.PassThresholdRatio {
			summary.PassCount++
		}
	}
	summary.AverageScore = round2(sum / float64(len(attempts)))
	return summary
}

func buildScoreBuckets(attempts []model.ExamAttempt, totalScore float64) []dto.ScoreBucket {
	labels := []string{"0-59", "60-69", "70-79", "80-89", "90-100"}
	buckets := make([]dto.ScoreBucket, len(labels))
	for i, label := range labels {
		buckets[i] = dto.ScoreBucket{Label: label, Count: 0}
	}
	for _, a := range attempts {
		percent := 100.0
		if totalScore > 0 {
			percent = a.TotalScore / totalScore * 100
		}
		switch {
		case percent < 60:
			buckets[0].Count++
		case percent < 70:
			buckets[1].Count++
		case percent < 80:
			buckets[2].Count++
		case percent < 90:
			buckets[3].Count++
		default:
			buckets[4].Count++
		}
	}
	return buckets
}
