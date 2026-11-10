package config

import (
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Env struct {
	PBW_ENCRYPTION_KEY       string `env:"PBW_ENCRYPTION_KEY,required"`
	PBW_POSTGRES_CONN_STRING string `env:"PBW_POSTGRES_CONN_STRING,required"`
	PBW_LISTEN_HOST          string `env:"PBW_LISTEN_HOST" envDefault:"0.0.0.0"`
	PBW_LISTEN_PORT          string `env:"PBW_LISTEN_PORT" envDefault:"8085"`
	PBW_PATH_PREFIX          string `env:"PBW_PATH_PREFIX" envDefault:""`

	// PBW_BACKUP_TIMEOUT stops a backup that runs longer than this (Go
	// duration, for example "6h"). 0 disables the limit.
	PBW_BACKUP_TIMEOUT time.Duration `env:"PBW_BACKUP_TIMEOUT" envDefault:"12h"`

	// PBW_RETENTION_KEEP_MIN is how many of the newest successful backups of
	// each backup job retention always keeps, whatever their age.
	PBW_RETENTION_KEEP_MIN int `env:"PBW_RETENTION_KEEP_MIN" envDefault:"3"`

	// PBW_COOKIE_SECURE adds the Secure flag to the session cookie. Set it to
	// true when users reach the app over HTTPS (also behind a TLS proxy).
	PBW_COOKIE_SECURE bool `env:"PBW_COOKIE_SECURE" envDefault:"false"`

	// PBW_TRUSTED_PROXIES is a comma-separated list of proxy IPs or CIDRs
	// whose X-Forwarded-For header is trusted for the client IP (rate limit,
	// logs). Empty means the header is ignored and the TCP peer IP is used.
	PBW_TRUSTED_PROXIES []string `env:"PBW_TRUSTED_PROXIES" envSeparator:","`
}

var (
	getEnvRes  Env
	getEnvErr  error
	getEnvOnce sync.Once
)

// GetEnv returns the environment variables.
//
// If there is an error, it will log it and exit the program.
func GetEnv(disableLogs ...bool) (Env, error) {
	getEnvOnce.Do(func() {
		_ = godotenv.Load()

		parsedEnv, err := env.ParseAs[Env]()
		if err != nil {
			getEnvErr = err
			return
		}

		if err := validateEnv(parsedEnv); err != nil {
			getEnvErr = err
			return
		}

		getEnvRes = parsedEnv
	})

	return getEnvRes, getEnvErr
}
