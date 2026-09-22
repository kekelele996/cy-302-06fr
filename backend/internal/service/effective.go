package service

import (
	"sort"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/model"
)

// EffectiveRow pairs a student with the attempt that determines their valid
// score for an exam: the highest-scoring submitted attempt (original or makeup).
type EffectiveRow struct {
	StudentID       uint
	Attempt         *model.ExamAttempt
	HasParticipated bool
}

// BestSubmittedAttempt returns a student's valid attempt for an exam, i.e. the
// submitted attempt (normal or makeup) with the highest total score. Ties are
// broken by earlier submission time and then lower attempt ID, which keeps the
// original attempt preferred when both score equally. In-progress and absent
// attempts are never valid.
func BestSubmittedAttempt(attempts []model.ExamAttempt) *model.ExamAttempt {
	var best *model.ExamAttempt
	for i := range attempts {
		a := &attempts[i]
		if a.Status != constants.AttemptSubmitted {
			continue
		}
		if best == nil || higherValidScore(a, best) {
			best = a
		}
	}
	return best
}

// EffectiveAttempts groups all attempts of an exam by student and keeps only
// the valid (highest-scoring submitted) attempt per student.
func EffectiveAttempts(attempts []model.ExamAttempt) map[uint]EffectiveRow {
	byStudent := make(map[uint][]model.ExamAttempt)
	for _, a := range attempts {
		byStudent[a.StudentID] = append(byStudent[a.StudentID], a)
	}
	result := make(map[uint]EffectiveRow, len(byStudent))
	for studentID, list := range byStudent {
		row := EffectiveRow{StudentID: studentID}
		for i := range list {
			if list[i].Status == constants.AttemptSubmitted {
				row.HasParticipated = true
				break
			}
		}
		row.Attempt = BestSubmittedAttempt(list)
		result[studentID] = row
	}
	return result
}

// SortAttemptsByScore sorts submitted attempts by descending total score,
// earlier submission time and then lower ID.
func SortAttemptsByScore(attempts []model.ExamAttempt) {
	sort.Slice(attempts, func(i, j int) bool {
		if attempts[i].TotalScore != attempts[j].TotalScore {
			return attempts[i].TotalScore > attempts[j].TotalScore
		}
		if attempts[i].SubmittedAt != nil && attempts[j].SubmittedAt != nil {
			return attempts[i].SubmittedAt.Before(*attempts[j].SubmittedAt)
		}
		if attempts[i].SubmittedAt != nil {
			return true
		}
		if attempts[j].SubmittedAt != nil {
			return false
		}
		return attempts[i].ID < attempts[j].ID
	})
}

func higherValidScore(candidate, current *model.ExamAttempt) bool {
	if candidate.TotalScore != current.TotalScore {
		return candidate.TotalScore > current.TotalScore
	}
	if candidate.SubmittedAt != nil && current.SubmittedAt != nil && !candidate.SubmittedAt.Equal(*current.SubmittedAt) {
		return candidate.SubmittedAt.Before(*current.SubmittedAt)
	}
	return candidate.ID < current.ID
}
