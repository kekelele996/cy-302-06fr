-- Migration 0002: makeup exam closed loop.
-- Adds absence marking, attempt kind tracing and the makeup request table.
-- The Go server also runs GORM AutoMigrate at startup, so this file documents
-- the canonical schema and can be applied manually if needed.

ALTER TABLE exam_attempts
    ADD COLUMN kind VARCHAR(16) NOT NULL DEFAULT 'original' AFTER student_id,
    ADD COLUMN attempt_no INT NOT NULL DEFAULT 1 AFTER kind,
    ADD KEY idx_exam_attempts_kind (kind);

ALTER TABLE wrong_questions
    ADD COLUMN attempt_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER question_id,
    ADD KEY idx_wrong_questions_attempt_id (attempt_id);

CREATE TABLE IF NOT EXISTS makeup_requests (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    exam_id BIGINT UNSIGNED NOT NULL,
    student_id BIGINT UNSIGNED NOT NULL,
    reason VARCHAR(255) DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    reviewed_by BIGINT UNSIGNED DEFAULT 0,
    reviewed_at DATETIME(3) NULL,
    attempt_id BIGINT UNSIGNED DEFAULT 0,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY idx_makeup_exam_student (exam_id, student_id),
    KEY idx_makeup_requests_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
