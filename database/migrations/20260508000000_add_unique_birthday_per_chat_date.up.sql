CREATE UNIQUE INDEX idx_events_unique_birthday_chat_event_date
ON events (chat_id, event_date)
WHERE type = 'birthday';
