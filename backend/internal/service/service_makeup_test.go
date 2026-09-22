package service

import (
	"testing"
	"time"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/model"
)

func submittedAttempt(id uint, kind string, score float64) model.ExamAttempt {
	now := time.Now()
	return model.ExamAttempt{
		ID:          id,
		Kind:        kind,
		Status:      constants.AttemptSubmitted,
		TotalScore:  score,
		SubmittedAt: &now,
	}
}

func TestBestSubmittedAttempt(t *testing.T) {
	tests := []struct {
		name     string
		attempts []model.ExamAttempt
		wantID   uint
		wantNil  bool
	}{
		{
			name:    "no attempts",
			wantNil: true,
		},
		{
			name: "no submitted attempts",
			attempts: []model.ExamAttempt{
				{ID: 1, Kind: constants.AttemptKindOriginal, Status: constants.AttemptAbsent},
				{ID: 2, Kind: constants.AttemptKindMakeup, Status: constants.AttemptInProgress},
			},
			wantNil: true,
		},
		{
			name: "makeup wins with higher score",
			attempts: []model.ExamAttempt{
				submittedAttempt(1, constants.AttemptKindOriginal, 50),
				submittedAttempt(2, constants.AttemptKindMakeup, 80),
			},
			wantID: 2,
		},
		{
			name: "original stays effective when makeup is lower",
			attempts: []model.ExamAttempt{
				submittedAttempt(1, constants.AttemptKindOriginal, 90),
				submittedAttempt(2, constants.AttemptKindMakeup, 40),
			},
			wantID: 1,
		},
		{
			name: "tie keeps earliest attempt",
			attempts: []model.ExamAttempt{
				submittedAttempt(1, constants.AttemptKindOriginal, 70),
				submittedAttempt(2, constants.AttemptKindMakeup, 70),
			},
			wantID: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bestSubmittedAttempt(tt.attempts)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("bestSubmittedAttempt() = %+v, want nil", got)
				}
				return
			}
			if got == nil || got.ID != tt.wantID {
				t.Fatalf("bestSubmittedAttempt() = %+v, want id %d", got, tt.wantID)
			}
		})
	}
}

func TestEvaluateMakeup(t *testing.T) {
	closed := &model.Exam{ID: 1, Status: constants.ExamClosed, TotalScore: 100}
	published := &model.Exam{ID: 1, Status: constants.ExamPublished, TotalScore: 100}
	absent := model.ExamAttempt{ID: 1, Kind: constants.AttemptKindOriginal, Status: constants.AttemptAbsent}
	pendingReq := &model.MakeupRequest{Status: constants.MakeupPending}
	approvedReq := &model.MakeupRequest{Status: constants.MakeupApproved}
	rejectedReq := &model.MakeupRequest{Status: constants.MakeupRejected}

	tests := []struct {
		name          string
		exam          *model.Exam
		attempts      []model.ExamAttempt
		req           *model.MakeupRequest
		wantEligible  bool
		wantCanApply  bool
		wantEffective *float64
	}{
		{
			name:         "exam not closed",
			exam:         published,
			attempts:     nil,
			wantEligible: false,
		},
		{
			name:         "absent student can apply",
			exam:         closed,
			attempts:     []model.ExamAttempt{absent},
			wantEligible: true,
			wantCanApply: true,
		},
		{
			name:         "no record student can apply",
			exam:         closed,
			attempts:     nil,
			wantEligible: true,
			wantCanApply: true,
		},
		{
			name:         "below 60 percent can apply",
			exam:         closed,
			attempts:     []model.ExamAttempt{submittedAttempt(1, constants.AttemptKindOriginal, 59)},
			wantEligible: true,
			wantCanApply: true,
		},
		{
			name:         "passed student not eligible",
			exam:         closed,
			attempts:     []model.ExamAttempt{submittedAttempt(1, constants.AttemptKindOriginal, 60)},
			wantEligible: false,
		},
		{
			name: "effective score above threshold not eligible",
			exam: closed,
			attempts: []model.ExamAttempt{
				submittedAttempt(1, constants.AttemptKindOriginal, 50),
				submittedAttempt(2, constants.AttemptKindMakeup, 61),
			},
			wantEligible: false,
		},
		{
			name:         "pending request blocks apply",
			exam:         closed,
			attempts:     []model.ExamAttempt{absent},
			req:          pendingReq,
			wantEligible: false,
			wantCanApply: false,
		},
		{
			name:         "approved request blocks apply",
			exam:         closed,
			attempts:     []model.ExamAttempt{absent},
			req:          approvedReq,
			wantEligible: false,
			wantCanApply: false,
		},
		{
			name:         "rejected request can reapply",
			exam:         closed,
			attempts:     []model.ExamAttempt{absent},
			req:          rejectedReq,
			wantEligible: true,
			wantCanApply: true,
		},
		{
			name: "makeup in progress blocks apply",
			exam: closed,
			attempts: []model.ExamAttempt{
				absent,
				{ID: 2, Kind: constants.AttemptKindMakeup, Status: constants.AttemptInProgress},
			},
			req:          approvedReq,
			wantEligible: false,
			wantCanApply: false,
		},
		{
			name: "makeup submitted blocks apply",
			exam: closed,
			attempts: []model.ExamAttempt{
				submittedAttempt(1, constants.AttemptKindOriginal, 30),
				submittedAttempt(2, constants.AttemptKindMakeup, 45),
			},
			req:          approvedReq,
			wantEligible: false,
			wantCanApply: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := evaluateMakeup(tt.exam, tt.attempts, tt.req)
			if state.eligible != tt.wantEligible || state.canApply != tt.wantCanApply {
				t.Fatalf("evaluateMakeup() = (eligible=%v, canApply=%v), want (%v, %v)",
					state.eligible, state.canApply, tt.wantEligible, tt.wantCanApply)
			}
			if state.message == "" {
				t.Fatalf("evaluateMakeup() returned empty message")
			}
		})
	}
}

func TestEvaluateMakeupEffectiveScore(t *testing.T) {
	exam := &model.Exam{ID: 1, Status: constants.ExamClosed, TotalScore: 100}
	attempts := []model.ExamAttempt{
		submittedAttempt(1, constants.AttemptKindOriginal, 40),
		submittedAttempt(2, constants.AttemptKindMakeup, 55),
	}
	state := evaluateMakeup(exam, attempts, &model.MakeupRequest{Status: constants.MakeupApproved})
	if state.effectiveScore == nil || *state.effectiveScore != 55 {
		t.Fatalf("effectiveScore = %v, want 55", state.effectiveScore)
	}
	if state.makeupAttemptID != 2 || state.makeupStatus != constants.AttemptSubmitted {
		t.Fatalf("makeup attempt = (%d, %s), want (2, submitted)", state.makeupAttemptID, state.makeupStatus)
	}
}
