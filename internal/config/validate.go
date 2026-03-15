package config

import (
	"fmt"
	"strings"
)

// Validate checks the configuration for invalid or inconsistent values.
func (c *Config) Validate() error {
	var errs []string
	if c.Server.Port < 0 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Sprintf("server.port must be 0-65535, got %d", c.Server.Port))
	}
	if c.SMTP.Host != "" && c.SMTP.Port == 0 {
		errs = append(errs, "smtp.port is required when smtp.host is set")
	}
	if c.OCR.Provider != "" && c.OCR.APIKey == "" {
		errs = append(errs, "ocr.api_key is required when ocr.provider is set")
	}
	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}
