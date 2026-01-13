CREATE TABLE IF NOT EXISTS group_events (
    id SERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    event_type VARCHAR(20) NOT NULL DEFAULT 'event',
    title VARCHAR(255) NOT NULL,
    description TEXT,
    event_date TIMESTAMPTZ,
    is_recurring BOOLEAN DEFAULT FALSE,
    recurrence_type VARCHAR(20),
    recurrence_day VARCHAR(20),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, group_id, event_type)
);

CREATE INDEX idx_events_group_id ON group_events(group_id);
CREATE INDEX idx_events_type ON group_events(event_type);
