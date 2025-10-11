-- Migration: 003 - Add Pause Fields to Reminders Table
-- Description: Add PausedUntil and PauseReason fields for C3 pause/resume feature
-- Date: 2025-10-11
-- Phase: C3

-- Add paused_until column (nullable timestamp)
ALTER TABLE reminders ADD COLUMN IF NOT EXISTS paused_until DATETIME DEFAULT NULL;

-- Add pause_reason column (nullable text)
ALTER TABLE reminders ADD COLUMN IF NOT EXISTS pause_reason TEXT DEFAULT NULL;

-- Create index on paused_until for efficient querying of paused reminders
CREATE INDEX IF NOT EXISTS idx_reminders_paused_until ON reminders(paused_until);

-- Verification Query (uncomment to verify)
-- SELECT
--     COUNT(*) as total_reminders,
--     SUM(CASE WHEN paused_until IS NOT NULL AND paused_until > datetime('now') THEN 1 ELSE 0 END) as currently_paused,
--     SUM(CASE WHEN paused_until IS NOT NULL AND paused_until <= datetime('now') THEN 1 ELSE 0 END) as pause_expired
-- FROM reminders;
