-- name: ExecutionsServiceFailInterruptedExecutions :execrows
UPDATE executions
SET
  status = 'failed',
  message = 'Interrupted: the application stopped while this backup was running',
  finished_at = NOW()
WHERE status = 'running';
