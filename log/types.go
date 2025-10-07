package log

import (
	"log/slog"
)

// LogLevel represents the logging level
type LogLevel slog.Level

// Log levels
const (
	DEBUG    LogLevel = LogLevel(slog.LevelDebug)
	INFO     LogLevel = LogLevel(slog.LevelInfo)
	WARN     LogLevel = LogLevel(slog.LevelWarn)
	ERROR    LogLevel = LogLevel(slog.LevelError)
	CRITICAL LogLevel = 12 // Custom critical level
)

// LogFormat represents the logging format
type LogFormat string

const (
	TEXT LogFormat = "text"
	JSON LogFormat = "json"
)

// ModuleConfig represents module-specific logging configuration
type ModuleConfig struct {
	Level  LogLevel `json:"level"`
	Prefix string   `json:"prefix"`
}

// LogConfig represents the overall logging configuration
type LogConfig struct {
	Level      LogLevel                `json:"level"`
	OutputFile string                  `json:"output_file"`
	Format     LogFormat               `json:"format"`
	Modules    map[string]ModuleConfig `json:"modules"`
	Enabled    bool                    `json:"enabled"`
}

// DefaultLogConfig returns the default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:      INFO,
		OutputFile: "",
		Format:     TEXT,
		Modules:    make(map[string]ModuleConfig),
		Enabled:    true,
	}
}
