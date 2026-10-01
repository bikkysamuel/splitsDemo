// Package platform holds process-wide infrastructure every other package may
// use: configuration, logging, the clock and ID generation.
package platform

import (
	"errors"
	"fmt"
)

// EnvDevelopment is the only APP_ENV under which the fixed one-time code is
// allowed (ADR-0016).
const EnvDevelopment = "development"

// OTPMode selects how one-time codes are delivered.
type OTPMode string

const (
	// OTPModeFixed accepts the fixed code for every account and sends nothing.
	// Allowed only under APP_ENV=development (ADR-0016).
	OTPModeFixed OTPMode = "fixed"
	// OTPModeEmail generates random codes and emails them.
	OTPModeEmail OTPMode = "email"
)

const defaultHTTPAddr = ":8080"

// Config is the server configuration, read from environment variables only.
type Config struct {
	AppEnv      string
	DatabaseURL string
	HTTPAddr    string
	OTPMode     OTPMode
}

// LoadConfig reads the configuration through getenv and validates it. It
// refuses the fixed one-time code outside development (ADR-0016).
func LoadConfig(getenv func(string) string) (Config, error) {
	cfg := Config{
		AppEnv:      getenv("APP_ENV"),
		DatabaseURL: getenv("DATABASE_URL"),
		HTTPAddr:    getenv("HTTP_ADDR"),
		OTPMode:     OTPMode(getenv("OTP_MODE")),
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = defaultHTTPAddr
	}

	var errs []error
	if cfg.AppEnv == "" {
		errs = append(errs, errors.New("APP_ENV is required"))
	}
	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	switch cfg.OTPMode {
	case OTPModeFixed:
		if cfg.AppEnv != EnvDevelopment {
			errs = append(errs, fmt.Errorf("OTP_MODE=%s is allowed only with APP_ENV=%s (ADR-0016)", OTPModeFixed, EnvDevelopment))
		}
	case OTPModeEmail:
	default:
		errs = append(errs, fmt.Errorf("OTP_MODE must be %q or %q", OTPModeFixed, OTPModeEmail))
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}
