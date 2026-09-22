package service

import (
	"testing"
	"time"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/model"
)

func submitted(id uint, kind string, score float64, submittedAt time.Time) model.ExamAttempt {
	return model.ExamAttempt{
		ID:          id,
		Kind:        kind,
		Status:      constants.AttemptSubmitted,
		TotalScore:  score,
		SubmittedAt: &submittedAt,
	}
}

func withStudent(studentID uint, a model.ExamAttempt) model.ExamAttempt {
	a.StudentID = studentID
	return a
}

func TestBestSubmittedAttempt(t *testing.T) {
	base := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		attempts []model.ExamAttempt
		wantID   uint
	}{
		{
			name: "none submitted",
			attempts: []model.ExamAttempt{
				{ID: 1, Kind: constants.AttemptKindNormal, Status: constants.AttemptInProgress},
				{ID: 2, Kind: constants.AttemptKindNormal, Status: constants.AttemptAbsent},
			},
			wantID: 0,
		},
		{
			name: "makeup higher wins",
			attempts: []model.ExamAttempt{
				submitted(1, constants.AttemptKindNormal, 50, base),
				submitted(2, constants.AttemptKindMakeup, 80, base.Add(time.Hour)),
			},
			wantID: 2,
		},
		{
			name: "original higher wins",
			attempts: []model.ExamAttempt{
				submitted(1, constants.AttemptKindNormal, 90, base),
				submitted(2, constants.AttemptKindMakeup, 55, base.Add(time.Hour)),
			},
			wantID: 1,
		},
		{
			name: "tie prefers earlier submission",
			attempts: []model.ExamAttempt{
				submitted(2, constants.AttemptKindMakeup, 60, base.Add(time.Hour)),
				submitted(1, constants.AttemptKindNormal, 60, base),
			},
			wantID: 1,
		},
		{
			name: "absent and in progress ignored",
			attempts: []model.ExamAttempt{
				{ID: 1, Kind: constants.AttemptKindNormal, Status: constants.AttemptAbsent, TotalScore: 0},
				submitted(2, constants.AttemptKindMakeup, 40, base),
			},
			wantID: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BestSubmittedAttempt(tt.attempts)
			if tt.wantID == 0 {
				if got != nil {
					t.Fatalf("expected nil, got attempt %d", got.ID)
				}
				return
			}
			if got == nil || got.ID != tt.wantID {
				var gotID uint
				if got != nil {
					gotID = got.ID
				}
				t.Fatalf("expected attempt %d, got %d", tt.wantID, gotID)
			}
		})
	}
}

func TestEffectiveAttemptsGroupsPerStudent(t *testing.T) {
	base := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	rows := EffectiveAttempts([]model.ExamAttempt{
		{ID: 1, StudentID: 100, Kind: constants.AttemptKindNormal, Status: constants.AttemptAbsent},
		withStudent(100, submitted(2, constants.AttemptKindNormal, 30, base)),
		withStudent(100, submitted(3, constants.AttemptKindMakeup, 70, base.Add(time.Hour))),
	})
	student100 := rows[100]
	if student100.Attempt == nil || student100.Attempt.ID != 3 {
		t.Fatalf("expected best attempt 3 for student 100, got %+v", student100.Attempt)
	}
	if !student100.HasParticipated {
		t.Fatal("expected student 100 to be marked as participated")
	}

	rows = EffectiveAttempts([]model.ExamAttempt{
		{ID: 4, StudentID: 200, Kind: constants.AttemptKindNormal, Status: constants.AttemptAbsent},
	})
	student200 := rows[200]
	if student200.Attempt != nil {
		t.Fatal("absent-only student must not have a valid attempt")
	}
	if student200.HasParticipated {
		t.Fatal("absent-only student must not be marked as participated")
	}
}

func TestSortAttemptsByScore(t *testing.T) {
	base := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	attempts := []model.ExamAttempt{
		submitted(1, constants.AttemptKindNormal, 50, base),
		submitted(2, constants.AttemptKindMakeup, 90, base.Add(2*time.Hour)),
		submitted(3, constants.AttemptKindNormal, 90, base.Add(time.Hour)),
	}
	SortAttemptsByScore(attempts)
	if attempts[0].ID != 3 || attempts[1].ID != 2 || attempts[2].ID != 1 {
		t.Fatalf("unexpected order: %d %d %d", attempts[0].ID, attempts[1].ID, attempts[2].ID)
	}
}
