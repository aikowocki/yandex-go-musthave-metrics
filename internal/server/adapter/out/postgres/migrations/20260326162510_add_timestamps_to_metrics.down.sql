ALTER TABLE gauges
    DROP COLUMN IF EXISTS created_at,
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE counters
    DROP COLUMN IF EXISTS created_at,
    DROP COLUMN IF EXISTS updated_at;
