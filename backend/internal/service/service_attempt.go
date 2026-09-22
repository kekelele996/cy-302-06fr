package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/gbexam/online-exam/internal/constants"
	"github.com/gbexam/online-exam/internal/dto"
	"github.com/gbexam/online-exam/internal/model"
	"github.com/gbexam/online-exam/internal/repository"
)

// AttemptService handles taking, submitting and grading exams.
type AttemptService struct {
	baseService
	examRepo     ExamRepo
	questionRepo QuestionRepo
	attemptRepo  AttemptRepo
	answerRepo   AnswerRepo
	wrongRepo    WrongRepo
}

// NewAttemptService constructs AttemptService.
func NewAttemptService(
	examRepo ExamRepo,
	questionRepo QuestionRepo,
	attemptRepo AttemptRepo,
	answerRepo AnswerRepo,
	wrongRepo WrongRepo,
	logger *slog.Logger,
) *AttemptService {
	return &AttemptService{
		baseService:  NewBaseService(logger),
		examRepo:     examRepo,
		questionRepo: questionRepo,
		attemptRepo:  attemptRepo,
		answerRepo:   answerRepo,
		wrongRepo:    wrongRepo,
	}
}

// Start creates or resumes a student normal attempt with a shuffled paper.
func (s *AttemptService) Start(ctx context.Context, studentID, examID uint) (*dto.AttemptStartResponse, error) {
	return s.start(ctx, studentID, examID, constants.AttemptKindNormal)
}

func (s *AttemptService) start(ctx context.Context, studentID, examID uint, kind string) (*dto.AttemptStartResponse, error) {
	exam, err := s.examRepo.FindExamByID(ctx, examID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if kind == constants.AttemptKindNormal {
		if exam.Status != constants.ExamPublished {
			return nil, ErrForbidden
		}
		if exam.StartTime != nil && now.Before(*exam.StartTime) {
			return nil, fmt.Errorf("%w: 考试尚未开始", ErrValidation)
		}
		if exam.EndTime != nil && now.After(*exam.EndTime) {
			return nil, fmt.Errorf("%w: 考试已结束", ErrValidation)
		}
	}

	if existing, err := s.attemptRepo.FindInProgressAttemptByKind(ctx, examID, studentID, kind); err == nil {
		return s.startResponse(ctx, existing, exam)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("find in progress attempt: %w", err)
	}

	orderRaw, optionRaw, err := s.buildShuffledPaper(ctx, examID)
	if err != nil {
		return nil, err
	}
	attempt := &model.ExamAttempt{
		ExamID:        examID,
		StudentID:     studentID,
		Kind:          kind,
		Status:        constants.AttemptInProgress,
		StartedAt:     now,
		Deadline:      now.Add(time.Duration(exam.DurationMinutes) * time.Minute),
		QuestionOrder: orderRaw,
		OptionOrder:   optionRaw,
	}
	if err := s.attemptRepo.CreateAttempt(ctx, attempt); err != nil {
		return nil, err
	}
	return s.startResponse(ctx, attempt, exam)
}

// buildShuffledPaper draws the exam questions and returns randomized question
// and option orders. Used for both normal and makeup attempts.
func (s *AttemptService) buildShuffledPaper(ctx context.Context, examID uint) (string, string, error) {
	items, err := s.examRepo.ListExamQuestions(ctx, examID)
	if err != nil {
		return "", "", err
	}
	if len(items) == 0 {
		return "", "", fmt.Errorf("%w: 试卷没有题目", ErrValidation)
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	shuffle(items, rng)

	order := make([]uint, 0, len(items))
	optionOrder := map[uint][]string{}
	for _, it := range items {
		order = append(order, it.ID)
		q, ok := s.findQuestion(ctx, it.QuestionID)
		if !ok {
			continue
		}
		if isChoiceType(q.Type) {
			options, _ := unmarshalOptions(q.Options)
			shuffle(options, rng)
			keys := make([]string, 0, len(options))
			for _, opt := range options {
				keys = append(keys, opt.Key)
			}
			optionOrder[it.ID] = keys
		}
	}
	orderRaw, _ := json.Marshal(order)
	optionRaw, _ := json.Marshal(optionOrder)
	return string(orderRaw), string(optionRaw), nil
}

// BuildMakeupAttemptModel builds (without persisting) an independent makeup
// attempt for an approved application. The countdown is not started until the
// student first opens the paper (activated = false); until then the deadline is
// kept far in the future so nothing is auto-finalized prematurely. Persistence
// is performed together with the approval update in one transaction by
// MakeupApplicationService.
func (s *AttemptService) BuildMakeupAttemptModel(ctx context.Context, studentID, examID uint) (*model.ExamAttempt, error) {
	orderRaw, optionRaw, err := s.buildShuffledPaper(ctx, examID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &model.ExamAttempt{
		ExamID:        examID,
		StudentID:     studentID,
		Kind:          constants.AttemptKindMakeup,
		Status:        constants.AttemptInProgress,
		StartedAt:     now,
		Deadline:      now.AddDate(10, 0, 0),
		QuestionOrder: orderRaw,
		OptionOrder:   optionRaw,
		Activated:     false,
	}, nil
}

// StartMakeupAttempt resumes the specific makeup attempt bound to an approved
// application. The first open starts the countdown; later opens resume it.
func (s *AttemptService) StartMakeupAttempt(ctx context.Context, studentID, attemptID uint) (*dto.AttemptStartResponse, error) {
	attempt, err := s.attemptRepo.FindAttemptByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.StudentID != studentID || attempt.Kind != constants.AttemptKindMakeup {
		return nil, ErrForbidden
	}
	if attempt.Status != constants.AttemptInProgress {
		return nil, ErrConflict
	}
	if !attempt.Activated {
		exam, err := s.examRepo.FindExamByID(ctx, attempt.ExamID)
		if err != nil {
			return nil, err
		}
		now := time.Now()
		deadline := now.Add(time.Duration(exam.DurationMinutes) * time.Minute)
		activated, err := s.attemptRepo.ActivateAttempt(ctx, attemptID, now, deadline)
		if err != nil {
			return nil, err
		}
		if activated {
			attempt.StartedAt = now
			attempt.Deadline = deadline
			attempt.Activated = true
		} else {
			// A concurrent request activated it first: reload to get the
			// authoritative countdown window.
			fresh, err := s.attemptRepo.FindAttemptByID(ctx, attemptID)
			if err != nil {
				return nil, err
			}
			attempt = fresh
		}
	}
	exam, err := s.examRepo.FindExamByID(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}
	return s.startResponse(ctx, attempt, exam)
}

// Current returns the student's current unfinished normal attempt.
func (s *AttemptService) Current(ctx context.Context, studentID, examID uint) (*dto.AttemptStartResponse, error) {
	attempt, err := s.attemptRepo.FindInProgressAttemptByKind(ctx, examID, studentID, constants.AttemptKindNormal)
	if err != nil {
		return nil, err
	}
	exam, err := s.examRepo.FindExamByID(ctx, examID)
	if err != nil {
		return nil, err
	}
	return s.startResponse(ctx, attempt, exam)
}

// SaveAnswer persists one answer (does not auto-grade).
func (s *AttemptService) SaveAnswer(ctx context.Context, studentID, attemptID uint, req dto.AnswerSubmitRequest) error {
	attempt, err := s.attemptRepo.FindAttemptByID(ctx, attemptID)
	if err != nil {
		return err
	}
	if attempt.StudentID != studentID {
		return ErrForbidden
	}
	if attempt.Status != constants.AttemptInProgress {
		return fmt.Errorf("%w: 试卷已提交，无法继续答题", ErrValidation)
	}
	if time.Now().After(attempt.Deadline) {
		return fmt.Errorf("%w: 考试时间已到，请交卷", ErrValidation)
	}
	eq, ok := s.findExamQuestion(ctx, attempt, req.ExamQuestionID)
	if !ok {
		return fmt.Errorf("%w: 题目不在当前试卷中", ErrValidation)
	}
	answerRaw, err := marshalAnswer(req.Answer)
	if err != nil {
		return err
	}
	marked := false
	if req.Marked != nil {
		marked = *req.Marked
	}
	answer := &model.Answer{
		AttemptID:      attemptID,
		ExamQuestionID: eq.ID,
		QuestionID:     eq.QuestionID,
		AnswerText:     answerRaw,
		Marked:         marked,
		Score:          0,
	}
	if err := s.answerRepo.SaveAnswer(ctx, answer); err != nil {
		return fmt.Errorf("save answer: %w", err)
	}
	return nil
}

// Submit finalizes an attempt, auto-grades objective questions, and then
// reconciles the student's wrong-question book against the valid attempt.
func (s *AttemptService) Submit(ctx context.Context, studentID, attemptID uint) error {
	attempt, err := s.attemptRepo.FindAttemptByID(ctx, attemptID)
	if err != nil {
		return err
	}
	if attempt.StudentID != studentID {
		return ErrForbidden
	}
	if attempt.Status != constants.AttemptInProgress {
		return ErrConflict
	}
	// An expired, activated attempt is lazily finalized as absent; the student
	// then cannot submit it. Unactivated makeup attempts are not on a countdown.
	if attempt.Activated && time.Now().After(attempt.Deadline) {
		if _, err := s.attemptRepo.FinalizeExpiredAttempt(ctx, attemptID); err != nil {
			return err
		}
		return fmt.Errorf("%w: 考试时间已到，该记录已按缺考处理", ErrValidation)
	}

	items, err := s.examRepo.ListExamQuestions(ctx, attempt.ExamID)
	if err != nil {
		return err
	}
	answers, err := s.answerRepo.ListAnswersByAttempt(ctx, attemptID)
	if err != nil {
		return err
	}
	answerMap := make(map[uint]model.Answer, len(answers))
	for _, a := range answers {
		answerMap[a.ExamQuestionID] = a
	}

	objectiveTotal := 0.0
	now := time.Now()
	for _, it := range items {
		q, ok := s.findQuestion(ctx, it.QuestionID)
		if !ok {
			continue
		}
		studentAnswer := answerMap[it.ID]
		correctAnswer, _ := unmarshalAnswer(q.Answer)
		studentRaw, _ := unmarshalAnswer(studentAnswer.AnswerText)

		isObjective := ObjectiveQuestionTypes()[q.Type]
		var isCorrect *bool
		score := 0.0
		if isObjective {
			correct := studentAnswer.AnswerText != "" && isCorrectObjective(q.Type, correctAnswer, studentRaw)
			isCorrect = &correct
			if correct {
				score = it.Score
				objectiveTotal += score
			}
		}
		saved := &model.Answer{
			AttemptID:      attemptID,
			ExamQuestionID: it.ID,
			QuestionID:     q.ID,
			AnswerText:     studentAnswer.AnswerText,
			IsCorrect:      isCorrect,
			Score:          score,
			Marked:         studentAnswer.Marked,
		}
		if err := s.answerRepo.SaveAnswer(ctx, saved); err != nil {
			return fmt.Errorf("save answer: %w", err)
		}
	}

	submittedAt := now
	attempt.Status = constants.AttemptSubmitted
	attempt.SubmittedAt = &submittedAt
	attempt.ObjectiveScore = objectiveTotal
	attempt.TotalScore = objectiveTotal
	if err := s.attemptRepo.UpdateAttempt(ctx, attempt); err != nil {
		return fmt.Errorf("update attempt: %w", err)
	}
	if err := s.reconcileWrongBook(ctx, studentID, attempt.ExamID); err != nil {
		return fmt.Errorf("reconcile wrong book: %w", err)
	}
	return nil
}

// Grade applies teacher scores to subjective answers and reconciles the
// wrong-question book because grading can change the valid attempt.
func (s *AttemptService) Grade(ctx context.Context, teacherID uint, role string, attemptID uint, req dto.GradeRequest) error {
	attempt, err := s.attemptRepo.FindAttemptByID(ctx, attemptID)
	if err != nil {
		return err
	}
	if attempt.Status != constants.AttemptSubmitted {
		return fmt.Errorf("%w: 只有已提交的试卷可以批改", ErrValidation)
	}
	exam, err := s.examRepo.FindExamByID(ctx, attempt.ExamID)
	if err != nil {
		return err
	}
	if role == constants.RoleTeacher && exam.CreatedBy != teacherID {
		return ErrForbidden
	}

	items, err := s.examRepo.ListExamQuestions(ctx, attempt.ExamID)
	if err != nil {
		return err
	}
	questionMap := make(map[uint]model.ExamQuestion, len(items))
	for _, it := range items {
		questionMap[it.ID] = it
	}
	answers, err := s.answerRepo.ListAnswersByAttempt(ctx, attemptID)
	if err != nil {
		return err
	}
	answerMap := make(map[uint]model.Answer, len(answers))
	for _, a := range answers {
		answerMap[a.ExamQuestionID] = a
	}

	for _, item := range req.Items {
		eq, ok := questionMap[item.ExamQuestionID]
		if !ok {
			return fmt.Errorf("%w: 题目不在该试卷中", ErrValidation)
		}
		q, ok := s.findQuestion(ctx, eq.QuestionID)
		if !ok {
			continue
		}
		if ObjectiveQuestionTypes()[q.Type] {
			continue
		}
		answer, exists := answerMap[item.ExamQuestionID]
		if !exists {
			continue
		}
		if item.Score > eq.Score {
			return fmt.Errorf("%w: 得分不能超过题目分值 %.2f", ErrValidation, eq.Score)
		}
		answer.Score = item.Score
		answer.GradedBy = teacherID
		if err := s.answerRepo.SaveAnswer(ctx, &answer); err != nil {
			return fmt.Errorf("grade answer: %w", err)
		}
		answerMap[item.ExamQuestionID] = answer
	}

	total := attempt.ObjectiveScore
	for _, a := range answerMap {
		if _, isObjective := s.questionTypeByAnswer(ctx, a); !isObjective {
			total += a.Score
		}
	}
	attempt.TotalScore = total
	if err := s.attemptRepo.UpdateAttempt(ctx, attempt); err != nil {
		return fmt.Errorf("update attempt: %w", err)
	}
	if err := s.reconcileWrongBook(ctx, attempt.StudentID, attempt.ExamID); err != nil {
		return fmt.Errorf("reconcile wrong book: %w", err)
	}
	return nil
}

// Detail returns the full review of an attempt.
func (s *AttemptService) Detail(ctx context.Context, role string, userID, attemptID uint) (*dto.AttemptDetail, error) {
	attempt, err := s.attemptRepo.FindAttemptByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	exam, err := s.examRepo.FindExamByID(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}
	if role == constants.RoleStudent && attempt.StudentID != userID {
		return nil, ErrForbidden
	}
	if role == constants.RoleTeacher && exam.CreatedBy != userID {
		return nil, ErrForbidden
	}

	details, err := s.buildDetail(ctx, attempt, exam, role)
	if err != nil {
		return nil, err
	}
	best := s.effectiveForStudent(ctx, attempt.ExamID, attempt.StudentID)
	resp := &dto.AttemptDetail{
		AttemptID:      attempt.ID,
		ExamID:         exam.ID,
		ExamTitle:      exam.Title,
		Kind:           attempt.Kind,
		Status:         attempt.Status,
		ObjectiveScore: attempt.ObjectiveScore,
		TotalScore:     attempt.TotalScore,
		StartedAt:      attempt.StartedAt,
		SubmittedAt:    attempt.SubmittedAt,
		Deadline:       attempt.Deadline,
		Questions:      details,
	}
	s.fillEffective(resp, attempt, best)
	return resp, nil
}

// List returns the student's attempt history.
func (s *AttemptService) List(ctx context.Context, studentID uint, query dto.AttemptListQuery) (dto.PageResult, error) {
	attempts, total, err := s.attemptRepo.ListAttemptsByStudent(ctx, studentID, query.ExamID, query.Page, query.PageSize)
	if err != nil {
		return dto.PageResult{}, fmt.Errorf("list attempts: %w", err)
	}
	page, pageSize := normalizePage(query.Page, query.PageSize)
	items := make([]dto.AttemptSummary, 0, len(attempts))
	for i := range attempts {
		exam, examErr := s.examRepo.FindExamByID(ctx, attempts[i].ExamID)
		title := ""
		if examErr == nil {
			title = exam.Title
		}
		best := s.effectiveForStudent(ctx, attempts[i].ExamID, studentID)
		summary := dto.AttemptSummary{
			AttemptID:      attempts[i].ID,
			ExamID:         attempts[i].ExamID,
			ExamTitle:      title,
			Kind:           attempts[i].Kind,
			Status:         attempts[i].Status,
			ObjectiveScore: attempts[i].ObjectiveScore,
			TotalScore:     attempts[i].TotalScore,
			StartedAt:      attempts[i].StartedAt,
			SubmittedAt:    attempts[i].SubmittedAt,
		}
		s.fillSummaryEffective(&summary, &attempts[i], best)
		items = append(items, summary)
	}
	return dto.PageResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// Report builds score analysis with ranking based on effective (best-of-two) scores.
func (s *AttemptService) Report(ctx context.Context, role string, userID, attemptID uint) (*dto.ReportResponse, error) {
	attempt, err := s.attemptRepo.FindAttemptByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	exam, err := s.examRepo.FindExamByID(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}
	if role == constants.RoleStudent && attempt.StudentID != userID {
		return nil, ErrForbidden
	}
	if role == constants.RoleTeacher && exam.CreatedBy != userID {
		return nil, ErrForbidden
	}

	items, err := s.examRepo.ListExamQuestions(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}
	answers, err := s.answerRepo.ListAnswersByAttempt(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	answerMap := make(map[uint]model.Answer, len(answers))
	for _, a := range answers {
		answerMap[a.ExamQuestionID] = a
	}

	type agg struct {
		name  string
		score float64
		max   float64
		count int
	}
	aggMap := map[string]*agg{}
	objectiveCorrect := 0
	objectiveCount := 0

	for _, it := range items {
		a, ok := answerMap[it.ID]
		if !ok {
			continue
		}
		q, ok := s.findQuestion(ctx, it.QuestionID)
		if !ok {
			continue
		}
		entry, exists := aggMap[q.Type]
		if !exists {
			entry = &agg{name: questionTypeName(q.Type)}
			aggMap[q.Type] = entry
		}
		entry.score += a.Score
		entry.max += it.Score
		entry.count++
		if ObjectiveQuestionTypes()[q.Type] {
			objectiveCount++
			if a.IsCorrect != nil && *a.IsCorrect {
				objectiveCorrect++
			}
		}
	}

	type namedAgg struct {
		key   string
		entry *agg
	}
	ordered := make([]namedAgg, 0, len(aggMap))
	for key, entry := range aggMap {
		ordered = append(ordered, namedAgg{key: key, entry: entry})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].key < ordered[j].key })
	breakdown := make([]dto.TypeScore, 0, len(ordered))
	for _, item := range ordered {
		breakdown = append(breakdown, dto.TypeScore{
			Type:  item.key,
			Name:  item.entry.name,
			Score: item.entry.score,
			Max:   item.entry.max,
			Count: item.entry.count,
		})
	}

	accuracy := 0.0
	if objectiveCount > 0 {
		accuracy = float64(objectiveCorrect) / float64(objectiveCount) * 100
	}
	rank, participants := s.effectiveRanking(ctx, attempt)

	subjectiveScore := attempt.TotalScore - attempt.ObjectiveScore
	best := s.effectiveForStudent(ctx, attempt.ExamID, attempt.StudentID)
	resp := &dto.ReportResponse{
		AttemptID:       attempt.ID,
		ExamID:          exam.ID,
		ExamTitle:       exam.Title,
		Kind:            attempt.Kind,
		TotalScore:      attempt.TotalScore,
		ObjectiveScore:  attempt.ObjectiveScore,
		SubjectiveScore: subjectiveScore,
		Accuracy:        round2(accuracy),
		Rank:            rank,
		Participants:    participants,
		TypeBreakdown:   breakdown,
		SubmittedAt:     attempt.SubmittedAt,
	}
	if best != nil {
		resp.EffectiveScore = best.TotalScore
		resp.IsEffective = best.ID == attempt.ID
		resp.EffectiveKind = best.Kind
		resp.EffectiveAttemptID = best.ID
	} else {
		resp.EffectiveScore = attempt.TotalScore
	}
	return resp, nil
}

// ListGrading returns submitted attempts of an exam for teacher grading.
func (s *AttemptService) ListGrading(ctx context.Context, role string, userID, examID uint) ([]dto.AttemptSummary, error) {
	exam, err := s.examRepo.FindExamByID(ctx, examID)
	if err != nil {
		return nil, err
	}
	if role == constants.RoleTeacher && exam.CreatedBy != userID {
		return nil, ErrForbidden
	}
	attempts, err := s.attemptRepo.ListAttemptsByExam(ctx, examID)
	if err != nil {
		return nil, err
	}
	effective := EffectiveAttempts(attempts)
	result := make([]dto.AttemptSummary, 0, len(attempts))
	for i := range attempts {
		if attempts[i].Status != constants.AttemptSubmitted {
			continue
		}
		summary := dto.AttemptSummary{
			AttemptID:      attempts[i].ID,
			ExamID:         attempts[i].ExamID,
			ExamTitle:      exam.Title,
			Kind:           attempts[i].Kind,
			Status:         attempts[i].Status,
			ObjectiveScore: attempts[i].ObjectiveScore,
			TotalScore:     attempts[i].TotalScore,
			StartedAt:      attempts[i].StartedAt,
			SubmittedAt:    attempts[i].SubmittedAt,
		}
		if row, ok := effective[attempts[i].StudentID]; ok {
			s.fillSummaryEffective(&summary, &attempts[i], row.Attempt)
		}
		result = append(result, summary)
	}
	return result, nil
}

func (s *AttemptService) buildDetail(ctx context.Context, attempt *model.ExamAttempt, exam *model.Exam, role string) ([]dto.AttemptQuestionDetail, error) {
	items, err := s.examRepo.ListExamQuestions(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}
	answers, err := s.answerRepo.ListAnswersByAttempt(ctx, attempt.ID)
	if err != nil {
		return nil, err
	}
	answerMap := make(map[uint]model.Answer, len(answers))
	for _, a := range answers {
		answerMap[a.ExamQuestionID] = a
	}
	order := parseOrder(attempt.QuestionOrder)
	items = orderExamQuestions(items, order)

	result := make([]dto.AttemptQuestionDetail, 0, len(items))
	for _, it := range items {
		q, ok := s.findQuestion(ctx, it.QuestionID)
		if !ok {
			continue
		}
		a := answerMap[it.ID]
		studentAnswer, _ := unmarshalAnswer(a.AnswerText)
		correctAnswer, _ := unmarshalAnswer(q.Answer)
		showCorrect := role == constants.RoleTeacher || role == constants.RoleAdmin || ObjectiveQuestionTypes()[q.Type]
		if !showCorrect {
			correctAnswer = nil
		}
		var isCorrect *bool
		if a.IsCorrect != nil {
			val := *a.IsCorrect
			isCorrect = &val
		}
		result = append(result, dto.AttemptQuestionDetail{
			ExamQuestionID: it.ID,
			Type:           q.Type,
			Content:        q.Content,
			Options:        mustOptions(q.Options),
			StudentAnswer:  studentAnswer,
			CorrectAnswer:  correctAnswer,
			IsCorrect:      isCorrect,
			Score:          a.Score,
			MaxScore:       it.Score,
			Analysis:       q.Analysis,
			Marked:         a.Marked,
			Graded:         a.GradedBy != 0,
		})
	}
	return result, nil
}

func (s *AttemptService) startResponse(ctx context.Context, attempt *model.ExamAttempt, exam *model.Exam) (*dto.AttemptStartResponse, error) {
	items, err := s.examRepo.ListExamQuestions(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}
	answers, err := s.answerRepo.ListAnswersByAttempt(ctx, attempt.ID)
	if err != nil {
		return nil, err
	}
	answerMap := make(map[uint]model.Answer, len(answers))
	for _, a := range answers {
		answerMap[a.ExamQuestionID] = a
	}
	order := parseOrder(attempt.QuestionOrder)
	items = orderExamQuestions(items, order)
	optionOrder := parseOptionOrder(attempt.OptionOrder)

	views := make([]dto.ExamQuestionView, 0, len(items))
	for _, it := range items {
		q, ok := s.findQuestion(ctx, it.QuestionID)
		if !ok {
			continue
		}
		options, _ := unmarshalOptions(q.Options)
		if keys, ok := optionOrder[it.ID]; ok {
			options = reorderOptions(options, keys)
		}
		a := answerMap[it.ID]
		studentAnswer, _ := unmarshalAnswer(a.AnswerText)
		views = append(views, dto.ExamQuestionView{
			ExamQuestionID: it.ID,
			Type:           q.Type,
			Content:        q.Content,
			Options:        options,
			Score:          it.Score,
			Marked:         a.Marked,
			Answer:         studentAnswer,
		})
	}
	return &dto.AttemptStartResponse{
		AttemptID:       attempt.ID,
		ExamID:          exam.ID,
		Title:           exam.Title,
		Kind:            attempt.Kind,
		DurationMinutes: exam.DurationMinutes,
		TotalScore:      exam.TotalScore,
		StartedAt:       attempt.StartedAt,
		Deadline:        attempt.Deadline,
		Questions:       views,
	}, nil
}

func (s *AttemptService) findQuestion(ctx context.Context, id uint) (model.Question, bool) {
	m, err := s.questionRepo.FindQuestionsByIDs(ctx, []uint{id})
	if err != nil {
		return model.Question{}, false
	}
	q, ok := m[id]
	return q, ok
}

func (s *AttemptService) findExamQuestion(ctx context.Context, attempt *model.ExamAttempt, eqID uint) (model.ExamQuestion, bool) {
	items, err := s.examRepo.ListExamQuestions(ctx, attempt.ExamID)
	if err != nil {
		return model.ExamQuestion{}, false
	}
	for _, it := range items {
		if it.ID == eqID {
			return it, true
		}
	}
	return model.ExamQuestion{}, false
}

func (s *AttemptService) questionTypeByAnswer(ctx context.Context, a model.Answer) (string, bool) {
	q, ok := s.findQuestion(ctx, a.QuestionID)
	if !ok {
		return "", true
	}
	return q.Type, ObjectiveQuestionTypes()[q.Type]
}

// effectiveForStudent returns the student's valid (highest-scoring submitted)
// attempt for an exam.
func (s *AttemptService) effectiveForStudent(ctx context.Context, examID, studentID uint) *model.ExamAttempt {
	attempts, err := s.attemptRepo.ListAttemptsByStudentAndExam(ctx, examID, studentID)
	if err != nil {
		return nil
	}
	return BestSubmittedAttempt(attempts)
}

func (s *AttemptService) fillSummaryEffective(summary *dto.AttemptSummary, current, best *model.ExamAttempt) {
	if best == nil {
		summary.EffectiveScore = current.TotalScore
		summary.IsEffective = false
		return
	}
	summary.EffectiveScore = best.TotalScore
	summary.IsEffective = best.ID == current.ID
}

func (s *AttemptService) fillEffective(resp *dto.AttemptDetail, current, best *model.ExamAttempt) {
	if best == nil {
		resp.EffectiveScore = current.TotalScore
		resp.IsEffective = false
		return
	}
	resp.EffectiveScore = best.TotalScore
	resp.IsEffective = best.ID == current.ID
}

// effectiveRanking ranks the student among per-student effective attempts.
// Absent students are not participants.
func (s *AttemptService) effectiveRanking(ctx context.Context, attempt *model.ExamAttempt) (int, int) {
	attempts, err := s.attemptRepo.ListAttemptsByExam(ctx, attempt.ExamID)
	if err != nil {
		return 0, 0
	}
	rows := EffectiveAttempts(attempts)
	effective := make([]model.ExamAttempt, 0, len(rows))
	for _, row := range rows {
		if row.Attempt != nil {
			effective = append(effective, *row.Attempt)
		}
	}
	SortAttemptsByScore(effective)
	best := s.effectiveForStudent(ctx, attempt.ExamID, attempt.StudentID)
	rank := 0
	if best != nil {
		for i, a := range effective {
			if a.ID == best.ID {
				rank = i + 1
				break
			}
		}
	}
	return rank, len(effective)
}

// reconcileWrongBook makes the student's wrong-question entries for the exam's
// questions match exactly the wrong objective answers of their current valid
// (highest-scoring) attempt for the exam.
func (s *AttemptService) reconcileWrongBook(ctx context.Context, studentID, examID uint) error {
	attempts, err := s.attemptRepo.ListAttemptsByStudentAndExam(ctx, examID, studentID)
	if err != nil {
		return err
	}
	best := BestSubmittedAttempt(attempts)
	if best == nil {
		return nil
	}
	items, err := s.examRepo.ListExamQuestions(ctx, examID)
	if err != nil {
		return err
	}
	answers, err := s.answerRepo.ListAnswersByAttempt(ctx, best.ID)
	if err != nil {
		return err
	}
	answerMap := make(map[uint]model.Answer, len(answers))
	for _, a := range answers {
		answerMap[a.ExamQuestionID] = a
	}

	now := time.Now()
	examQuestionIDs := make(map[uint]struct{}, len(items))
	wrongSet := make(map[uint]model.ExamQuestion)
	for _, it := range items {
		examQuestionIDs[it.QuestionID] = struct{}{}
		q, ok := s.findQuestion(ctx, it.QuestionID)
		if !ok || !ObjectiveQuestionTypes()[q.Type] {
			continue
		}
		a, answered := answerMap[it.ID]
		if !answered || a.IsCorrect == nil || *a.IsCorrect {
			continue
		}
		wrongSet[q.ID] = it
	}

	records, _, err := s.wrongRepo.ListWrongQuestions(ctx, studentID, "", 1, 1000)
	if err != nil {
		return err
	}
	existing := make(map[uint]model.WrongQuestion)
	for _, r := range records {
		existing[r.QuestionID] = r
	}

	for qID := range wrongSet {
		q, ok := s.findQuestion(ctx, qID)
		if !ok {
			continue
		}
		if _, present := existing[qID]; present {
			continue
		}
		w := &model.WrongQuestion{
			StudentID:      studentID,
			QuestionID:     qID,
			KnowledgePoint: q.KnowledgePoint,
			WrongCount:     1,
			LastWrongAt:    now,
			Status:         constants.WrongUnresolved,
		}
		if err := s.wrongRepo.UpsertWrongQuestion(ctx, w); err != nil {
			return fmt.Errorf("upsert wrong question: %w", err)
		}
	}

	// Remove wrong entries for exam questions that the valid attempt answered
	// correctly, unless the question is still wrong in another submitted attempt.
	for _, r := range records {
		if _, stillWrong := wrongSet[r.QuestionID]; stillWrong {
			continue
		}
		if _, belongs := examQuestionIDs[r.QuestionID]; !belongs {
			continue
		}
		other, err := s.answerRepo.HasOtherSubmittedWrongAnswer(ctx, studentID, r.QuestionID, best.ID)
		if err != nil {
			return err
		}
		if !other {
			if err := s.wrongRepo.DeleteWrongQuestion(ctx, r.ID, studentID); err != nil && !errors.Is(err, repository.ErrNotFound) {
				return fmt.Errorf("delete reconciled wrong question: %w", err)
			}
		}
	}
	return nil
}

func parseOrder(raw string) []uint {
	var order []uint
	_ = json.Unmarshal([]byte(raw), &order)
	return order
}

func parseOptionOrder(raw string) map[uint][]string {
	result := map[uint][]string{}
	_ = json.Unmarshal([]byte(raw), &result)
	return result
}

func orderExamQuestions(items []model.ExamQuestion, order []uint) []model.ExamQuestion {
	if len(order) == 0 {
		return items
	}
	byID := make(map[uint]model.ExamQuestion, len(items))
	for _, it := range items {
		byID[it.ID] = it
	}
	result := make([]model.ExamQuestion, 0, len(items))
	for _, id := range order {
		if it, ok := byID[id]; ok {
			result = append(result, it)
		}
	}
	return result
}

func reorderOptions(options []dto.Option, keys []string) []dto.Option {
	if len(keys) == 0 {
		return options
	}
	byKey := make(map[string]dto.Option, len(options))
	for _, opt := range options {
		byKey[opt.Key] = opt
	}
	result := make([]dto.Option, 0, len(keys))
	for _, key := range keys {
		if opt, ok := byKey[key]; ok {
			result = append(result, opt)
		}
	}
	return result
}

func mustOptions(raw string) []dto.Option {
	options, _ := unmarshalOptions(raw)
	return options
}

func isChoiceType(qtype string) bool {
	return qtype == constants.QuestionSingle || qtype == constants.QuestionMultiple || qtype == constants.QuestionTrueFalse
}

func isCorrectObjective(qtype string, correct, student any) bool {
	switch qtype {
	case constants.QuestionSingle, constants.QuestionTrueFalse:
		c, ok1 := toString(correct)
		s, ok2 := toString(student)
		return ok1 && ok2 && strings.EqualFold(strings.TrimSpace(c), strings.TrimSpace(s))
	case constants.QuestionMultiple:
		c, ok1 := toStringSlice(correct)
		s, ok2 := toStringSlice(student)
		if !ok1 || !ok2 || len(c) != len(s) {
			return false
		}
		set := make(map[string]bool, len(c))
		for _, v := range c {
			set[strings.TrimSpace(v)] = true
		}
		for _, v := range s {
			if !set[strings.TrimSpace(v)] {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func questionTypeName(qtype string) string {
	switch qtype {
	case constants.QuestionSingle:
		return "单选题"
	case constants.QuestionMultiple:
		return "多选题"
	case constants.QuestionTrueFalse:
		return "判断题"
	case constants.QuestionFillBlank:
		return "填空题"
	case constants.QuestionShortAnswer:
		return "简答题"
	default:
		return qtype
	}
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
