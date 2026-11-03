-- Accounting done in the first 10 days of a month records the previous month's closing balances.
-- The backend now files new snapshots that way; this re-files the three existing ones.
-- Order matters (unique year/month): each target month is free by the time it is used.

UPDATE net_worth_snapshots SET month = 2
WHERE year = 2026 AND month = 3 AND (date AT TIME ZONE 'UTC')::date = '2026-03-02';

UPDATE net_worth_snapshots SET month = 3
WHERE year = 2026 AND month = 4 AND (date AT TIME ZONE 'UTC')::date = '2026-04-01';

UPDATE net_worth_snapshots SET month = 4
WHERE year = 2026 AND month = 5 AND (date AT TIME ZONE 'UTC')::date = '2026-05-01';
