package helpers

import (
	"log"
	"os"
	"strings"
	"sync"
)

var (
	warnAllowAllOriginsOnce sync.Once
	warnAllowNoOriginsOnce  sync.Once
)

// IsDevMode returns true if GO_ENV is set to dev, development, or test.
// Use for local-only behavior (e.g. skipping auth, allowing all CORS).
func IsDevMode() bool {
	env := os.Getenv("GO_ENV")
	return env == "dev" || env == "development" || env == "test"
}

// GetAllowedOrigins returns the list of allowed CORS origins from ALLOWED_ORIGINS (comma-separated).
// If ALLOWED_ORIGINS is not set: in dev mode allows all origins (with a one-time warning);
// otherwise allows none and logs a one-time warning.
func GetAllowedOrigins() []string {
	originsEnv := os.Getenv("ALLOWED_ORIGINS")
	if originsEnv != "" {
		parts := strings.Split(originsEnv, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}
	if IsDevMode() {
		warnAllowAllOriginsOnce.Do(func() {
			log.Println("WARNING: ALLOWED_ORIGINS not set and GO_ENV is dev - allowing all origins (set ALLOWED_ORIGINS in production)")
		})
		return []string{"*"}
	}
	warnAllowNoOriginsOnce.Do(func() {
		log.Println("WARNING: ALLOWED_ORIGINS not set and GO_ENV is not dev - no origins allowed (set ALLOWED_ORIGINS or use GO_ENV=dev for local dev)")
	})
	return []string{}
}
