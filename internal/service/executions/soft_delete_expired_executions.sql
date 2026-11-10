-- name: ExecutionsServiceGetExpiredExecutions :many
-- The newest keep_min successful executions of each backup are never
-- returned, whatever their age. Without this, a backup that fails for longer
-- than its retention period loses every good copy.
WITH success_ranks AS (
  SELECT
    id,
    ROW_NUMBER() OVER (
      PARTITION BY backup_id ORDER BY finished_at DESC
    ) AS success_rank
  FROM executions
  WHERE status = 'success'
)
SELECT executions.*
FROM executions
JOIN backups ON executions.backup_id = backups.id
LEFT JOIN success_ranks ON success_ranks.id = executions.id
WHERE
  backups.retention_days > 0
  AND executions.status != 'deleted'
  AND executions.finished_at IS NOT NULL
  AND (
    executions.finished_at + (backups.retention_days || ' days')::INTERVAL
  ) < NOW()
  AND (
    success_ranks.success_rank IS NULL
    OR success_ranks.success_rank > sqlc.arg('keep_min')::INTEGER
  );
