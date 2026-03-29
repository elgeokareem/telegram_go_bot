DROP INDEX IF EXISTS idx_event_delivery_log_dedupe;
DROP TABLE IF EXISTS event_delivery_log;
DROP TABLE IF EXISTS event_reminders;
DROP INDEX IF EXISTS idx_event_recurrence_next_run;
DROP TABLE IF EXISTS event_recurrence;
DROP TABLE IF EXISTS events;
DROP TYPE IF EXISTS recurrence_type;
DROP TYPE IF EXISTS event_type;
