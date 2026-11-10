package executions

import (
	"context"

	"github.com/eduardolat/pgbackweb/internal/logger"
)

// FailInterruptedExecutions marks executions that are still "running" as
// failed. Call it once at startup, before the scheduler starts: at that time
// no backup can be running, so such rows were cut off by a restart or crash.
func (s *Service) FailInterruptedExecutions() {
	n, err := s.dbgen.ExecutionsServiceFailInterruptedExecutions(context.Background())
	if err != nil {
		logger.Error(
			"error marking interrupted executions as failed",
			logger.KV{"error": err},
		)
		return
	}
	if n > 0 {
		logger.Info(
			"interrupted executions marked as failed", logger.KV{"count": n},
		)
	}
}
