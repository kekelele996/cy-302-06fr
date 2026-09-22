package constants

// Roles
const (
	RoleAdmin   = "admin"
	RoleTeacher = "teacher"
	RoleStudent = "student"
)

// Question types
const (
	QuestionSingle      = "single"
	QuestionMultiple    = "multiple"
	QuestionTrueFalse   = "true_false"
	QuestionFillBlank   = "fill_blank"
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
	AttemptKindNormal = "normal"
	AttemptKindMakeup = "makeup"
)

// Makeup application statuses
const (
	MakeupPending  = "pending"
	MakeupApproved = "approved"
	MakeupRejected = "rejected"
)

// Pass threshold ratio relative to the exam total score.
const PassThresholdRatio = 0.6

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
