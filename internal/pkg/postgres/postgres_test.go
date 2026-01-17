package postgres

import (
	"testing"
)

func TestConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "basic config",
			config: Config{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "pass",
				Database: "testdb",
			},
			expected: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		},
		{
			name: "with schema",
			config: Config{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "pass",
				Database: "testdb",
				Schema:   "orders",
			},
			expected: "postgres://user:pass@localhost:5432/testdb?sslmode=disable&search_path=orders",
		},
		{
			name: "with ssl mode",
			config: Config{
				Host:     "db.example.com",
				Port:     5432,
				User:     "admin",
				Password: "secret",
				Database: "prod",
				SSLMode:  "require",
			},
			expected: "postgres://admin:secret@db.example.com:5432/prod?sslmode=require",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.config.DSN()
			if dsn != tt.expected {
				t.Errorf("expected DSN %q, got %q", tt.expected, dsn)
			}
		})
	}
}
