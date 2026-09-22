-- Migration 0002: makeup exam closed loop.
-- 1. exam_attempts gains a kind ('normal' / 'makeup') and an activated flag;
--    a new status 'absent' marks students who did not submit when the exam closed.
-- 2. makeup_applications stores each student's single application per exam.
-- The Go server also runs GORM AutoMigrate at startup, so this file documents
-- the schema delta and can be applied manually if needed.

ALTER TABLE exam_attempts
    ADD COLUMN kind VARCHAR(16) NOT NULL DEFAULT 'normal' AFTER student_id,
    ADD COLUMN activated TINYINT(1) NOT NULL DEFAULT 1 AFTER total_score;

ALTER TABLE exam_attempts
    ADD KEY idx_attempt_exam_student (exam_id, student_id);

CREATE TABLE IF NOT EXISTS makeup_applications (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    exam_id BIGINT UNSIGNED NOT NULL,
    student_id BIGINT UNSIGNED NOT NULL,
    reason VARCHAR(500) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    review_remark VARCHAR(500) NOT NULL DEFAULT '',
    reviewed_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
    reviewed_at DATETIME(3) NULL,
    makeup_attempt_id BIGINT UNSIGNED NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uniq_makeup_exam_student (exam_id, student_id),
    KEY idx_makeup_applications_status (status),
    KEY idx_makeup_applications_attempt (makeup_attempt_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
