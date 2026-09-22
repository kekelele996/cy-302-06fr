package constants

// Roles
const (
	RoleAdmin   = "admin"
	RoleTeacher = "teacher"
	RoleStudent = "student"
)

// Question types
const (
	QuestionSingle     = "single"
	QuestionMultiple   = "multiple"
	QuestionTrueFalse  = "true_false"
	QuestionFillBlank  = "fill_blank"
	QuestionShortAnswer = "short_answer"
)

// Difficulties
const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

// Exam statuses
const (
	ExamDraft     = "draft"
	ExamPublished = "published"
	ExamClosed    = "closed"
)

// Attempt statuses
const (
	AttemptInProgress = "in_progress"
	AttemptSubmitted  = "submitted"
	AttemptAbsent     = "absent"
)

// Attempt kinds
const (
	AttemptKindOriginal = "original"
	AttemptKindMakeup   = "makeup"
)

// Makeup request statuses
const (
	MakeupPending  = "pending"
	MakeupApproved = "approved"
	MakeupRejected = "rejected"
)

// MakeupPassRatio is the pass threshold ratio of the exam total score.
const MakeupPassRatio = 0.6

// Wrong question statuses
const (
	WrongUnresolved = "unresolved"
	WrongResolved   = "resolved"
)

// User statuses
const (
	UserActive   = "active"
	UserDisabled = "disabled"
)
