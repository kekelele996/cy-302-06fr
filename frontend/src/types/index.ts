export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PageResult<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export interface UserProfile {
  id: number
  username: string
  name: string
  role: 'admin' | 'teacher' | 'student'
  status: string
  created_at: string
}

export interface LoginResponse {
  token: string
  user: UserProfile
}

export interface Option {
  key: string
  text: string
}

export type QuestionType = 'single' | 'multiple' | 'true_false' | 'fill_blank' | 'short_answer'

export interface Question {
  id: number
  type: QuestionType
  content: string
  options: Option[]
  answer?: unknown
  analysis?: string
  difficulty: 'easy' | 'medium' | 'hard'
  knowledge_point: string
  score: number
  created_by?: number
  created_at?: string
}

export interface Exam {
  id: number
  title: string
  description: string
  total_score: number
  duration_minutes: number
  start_time?: string | null
  end_time?: string | null
  status: 'draft' | 'published' | 'closed'
  question_count: number
  created_by: number
  created_at: string
}

export interface PaperQuestionConfig {
  type: QuestionType
  count: number
  score: number
  difficulty?: 'easy' | 'medium' | 'hard'
}

export interface ExamCreatePayload {
  title: string
  description: string
  duration_minutes: number
  total_score: number
  start_time?: string | null
  end_time?: string | null
  question_config: PaperQuestionConfig[]
}

export interface ExamQuestionView {
  exam_question_id: number
  type: QuestionType
  content: string
  options: Option[]
  score: number
  marked: boolean
  answer?: unknown
}

export interface AttemptStartResponse {
  attempt_id: number
  exam_id: number
  title: string
  kind: 'normal' | 'makeup'
  duration_minutes: number
  total_score: number
  started_at: string
  deadline: string
  questions: ExamQuestionView[]
}

export interface AttemptSummary {
  attempt_id: number
  exam_id: number
  exam_title: string
  kind: 'normal' | 'makeup'
  status: 'in_progress' | 'submitted' | 'absent'
  objective_score: number
  total_score: number
  effective_score: number
  is_effective: boolean
  started_at: string
  submitted_at?: string | null
}

export interface AttemptQuestionDetail {
  exam_question_id: number
  type: QuestionType
  content: string
  options: Option[]
  student_answer?: unknown
  correct_answer?: unknown
  is_correct?: boolean | null
  score: number
  max_score: number
  analysis?: string
  marked: boolean
  graded: boolean
}

export interface AttemptDetail {
  attempt_id: number
  exam_id: number
  exam_title: string
  kind: 'normal' | 'makeup'
  status: string
  objective_score: number
  total_score: number
  effective_score: number
  is_effective: boolean
  started_at: string
  submitted_at?: string | null
  deadline: string
  questions: AttemptQuestionDetail[]
}

export interface TypeScore {
  type: QuestionType
  name: string
  score: number
  max: number
  count: number
}

export interface ReportResponse {
  attempt_id: number
  exam_id: number
  exam_title: string
  kind: 'normal' | 'makeup'
  total_score: number
  objective_score: number
  subjective_score: number
  effective_score: number
  is_effective: boolean
  effective_kind: 'normal' | 'makeup' | ''
  effective_attempt_id: number
  accuracy: number
  rank: number
  participants: number
  type_breakdown: TypeScore[]
  submitted_at?: string | null
}

export interface RankItem {
  rank: number
  student_name: string
  student_username: string
  total_score: number
  kind: 'normal' | 'makeup'
  attempt_id: number
  submitted_at?: string | null
}

export interface ScoreBucket {
  label: string
  count: number
}

export interface StatSummary {
  count: number
  average_score: number
  highest_score: number
  lowest_score: number
  pass_count: number
  distribution: ScoreBucket[]
}

export interface RawAttemptRow {
  attempt_id: number
  student_name: string
  student_username: string
  kind: 'normal' | 'makeup'
  status: 'in_progress' | 'submitted' | 'absent'
  total_score: number
  is_effective: boolean
  submitted_at?: string | null
}

export interface ExamStatResponse {
  exam_id: number
  exam_title: string
  absent_count: number
  original: StatSummary
  effective: StatSummary
  ranking: RankItem[]
  raw_attempts: RawAttemptRow[]
}

export type MakeupStatus = 'pending' | 'approved' | 'rejected'

export interface MakeupApplicationItem {
  id: number
  exam_id: number
  exam_title: string
  student_id: number
  student_name: string
  student_username: string
  reason: string
  status: MakeupStatus
  review_remark: string
  reviewed_by: number
  reviewed_at?: string | null
  makeup_attempt_id?: number | null
  created_at: string
}

export interface MakeupEligibility {
  exam_id: number
  eligible: boolean
  reason: string
  application: MakeupApplicationItem | null
}

export interface OverviewResponse {
  user_count: number
  question_count: number
  exam_count: number
  attempt_count: number
}

export interface WrongQuestionItem {
  id: number
  question_id: number
  knowledge_point: string
  wrong_count: number
  status: string
  last_wrong_at: string
  question: Question
}

export interface PracticeQuestion {
  question_id: number
  type: QuestionType
  content: string
  options: Option[]
  score: number
  knowledge_point: string
}

export interface PracticeResultResponse {
  total: number
  correct: number
  items: { question_id: number; correct: boolean; score: number }[]
}
