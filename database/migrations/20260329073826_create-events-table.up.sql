CREATE TYPE event_type AS ENUM ('birthday', 'reminder', 'custom');

CREATE TYPE recurrence_type AS ENUM ('none', 'daily', 'weekly', 'monthly', 'yearly');

CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    created_by_user_id BIGINT NOT NULL,
    target_user_id BIGINT,
    type event_type NOT NULL DEFAULT 'custom',
    title TEXT NOT NULL,
    description TEXT,
    is_all_day BOOLEAN NOT NULL DEFAULT FALSE,
    event_date DATE,
    event_at TIMESTAMPTZ,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_event_time_shape CHECK (
        (is_all_day = TRUE AND event_date IS NOT NULL AND event_at IS NULL)
        OR
        (is_all_day = FALSE AND event_at IS NOT NULL AND event_date IS NULL)
    )
);

CREATE INDEX idx_events_chat_id ON events (chat_id);

CREATE TABLE event_recurrence (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL UNIQUE REFERENCES events(id) ON DELETE CASCADE,
    frequency recurrence_type NOT NULL DEFAULT 'none',
    interval_value INT NOT NULL DEFAULT 1,
    until_at TIMESTAMPTZ,
    occurrence_count INT,
    next_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_event_recurrence_interval_positive CHECK (interval_value > 0),
    CONSTRAINT chk_event_recurrence_count_positive CHECK (occurrence_count IS NULL OR occurrence_count > 0)
);

CREATE INDEX idx_event_recurrence_next_run ON event_recurrence (next_run_at);

CREATE TABLE event_reminders (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    offset_minutes INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    message_template TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_event_reminders_event_id ON event_reminders (event_id);

CREATE TABLE event_delivery_log (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    reminder_id BIGINT REFERENCES event_reminders(id) ON DELETE SET NULL,
    scheduled_for TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ,
    status TEXT NOT NULL CHECK (status IN ('sent', 'failed', 'skipped')),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_event_delivery_log_dedupe
ON event_delivery_log (event_id, COALESCE(reminder_id, 0), scheduled_for);
