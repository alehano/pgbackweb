package executions

import (
	"context"

	"github.com/eduardolat/pgbackweb/internal/logger"
)

func (s *Service) SoftDeleteExpiredExecutions() {
	ctx := context.Background()

	expiredExecutions, err := s.dbgen.ExecutionsServiceGetExpiredExecutions(
		ctx, int32(s.env.PBW_RETENTION_KEEP_MIN),
	)
	if err != nil {
		logger.Error(
			"error soft deleting expired executions",
			logger.KV{"error": err},
		)
		return
	}

	// One failure must not stop the cleanup of all other executions.
	failed := 0
	for _, execution := range expiredExecutions {
		if err := s.SoftDeleteExecution(ctx, execution.ID); err != nil {
			failed++
			logger.Error(
				"error soft deleting expired execution",
				logger.KV{"id": execution.ID.String(), "error": err},
			)
		}
	}

	logger.Info("expired executions soft deleted", logger.KV{
		"total": len(expiredExecutions), "failed": failed,
	})
}
