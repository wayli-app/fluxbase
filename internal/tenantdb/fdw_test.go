package tenantdb

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFDWConfig(t *testing.T) {
	t.Run("parses full URL with password", func(t *testing.T) {
		cfg, err := ParseFDWConfig("postgresql://admin:secret@db.example.com:5432/fluxbase")
		require.NoError(t, err)
		assert.Equal(t, "db.example.com", cfg.Host)
		assert.Equal(t, "5432", cfg.Port)
		assert.Equal(t, "fluxbase", cfg.DBName)
		assert.Equal(t, "admin", cfg.User)
		assert.Equal(t, "secret", cfg.Password)
	})

	t.Run("parses URL without password", func(t *testing.T) {
		cfg, err := ParseFDWConfig("postgresql://admin@localhost/mydb")
		require.NoError(t, err)
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, "5432", cfg.Port) // default port
		assert.Equal(t, "mydb", cfg.DBName)
		assert.Equal(t, "admin", cfg.User)
		assert.Empty(t, cfg.Password)
	})

	t.Run("parses URL without port", func(t *testing.T) {
		cfg, err := ParseFDWConfig("postgresql://user:pass@host/fluxbase")
		require.NoError(t, err)
		assert.Equal(t, "5432", cfg.Port)
	})

	t.Run("rejects empty URL", func(t *testing.T) {
		_, err := ParseFDWConfig("")
		assert.Error(t, err)
	})

	t.Run("rejects incomplete URL missing host", func(t *testing.T) {
		_, err := ParseFDWConfig("postgresql://user@/fluxbase")
		assert.Error(t, err)
	})

	t.Run("rejects incomplete URL missing dbname", func(t *testing.T) {
		_, err := ParseFDWConfig("postgresql://user@host")
		assert.Error(t, err)
	})
}

func TestFDWConfig_Validation(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := FDWConfig{
			Host:     "localhost",
			Port:     "5432",
			DBName:   "fluxbase",
			User:     "admin",
			Password: "secret",
		}
		assert.NotEmpty(t, cfg.Host)
		assert.NotEmpty(t, cfg.DBName)
		assert.NotEmpty(t, cfg.User)
	})

	t.Run("minimal config", func(t *testing.T) {
		cfg := FDWConfig{
			Host:   "localhost",
			DBName: "fluxbase",
			User:   "admin",
		}
		assert.Empty(t, cfg.Password)
		assert.Empty(t, cfg.Port)
	})
}

func TestQuoteIdent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "simple name", input: "users", expected: `"users"`},
		{name: "name with spaces", input: "my table", expected: `"my table"`},
		{name: "name with quotes", input: `my"table`, expected: `"my""table"`},
		{name: "empty string", input: "", expected: `""`},
		{name: "schema qualified", input: "main_server", expected: `"main_server"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, quoteIdent(tt.input))
		})
	}
}

func TestEscapeSQLString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "no escaping needed", input: "hello", expected: "hello"},
		{name: "single quote", input: "it's", expected: "it''s"},
		{name: "multiple single quotes", input: "a'b'c", expected: "a''b''c"},
		{name: "empty string", input: "", expected: ""},
		{name: "password with special chars", input: "p@ss'w0rd", expected: "p@ss''w0rd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, escapeSQLString(tt.input))
		})
	}
}

func TestSetupFDW_NilPool(t *testing.T) {
	err := SetupFDW(nil, nil, FDWConfig{Host: "localhost", DBName: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant pool is nil")
}

func TestSetupFDW_IncompleteConfig(t *testing.T) {
	err := SetupFDW(nil, nil, FDWConfig{})
	assert.Error(t, err)
}

func TestTeardownFDW_NilPool(t *testing.T) {
	err := TeardownFDW(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant pool is nil")
}

func TestIsFDWConnectionError(t *testing.T) {
	t.Run("detects SQLSTATE 08001", func(t *testing.T) {
		err := &pgconn.PgError{Code: "08001", Message: "could not connect to server"}
		assert.True(t, isFDWConnectionError(err))
	})

	t.Run("detects wrapped SQLSTATE 08001", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: "08001", Message: "could not connect to server"}
		wrapped := fmt.Errorf("import failed: %w", pgErr)
		assert.True(t, isFDWConnectionError(wrapped))
	})

	t.Run("non-connection error code", func(t *testing.T) {
		err := &pgconn.PgError{Code: "42P01", Message: "relation not found"}
		assert.False(t, isFDWConnectionError(err))
	})

	t.Run("non-PgError", func(t *testing.T) {
		err := fmt.Errorf("some other error")
		assert.False(t, isFDWConnectionError(err))
	})
}

func TestFilterSlice(t *testing.T) {
	t.Run("filters excluded elements", func(t *testing.T) {
		all := []string{"a", "b", "c", "d"}
		exclude := []string{"b", "d"}
		result := filterSlice(all, exclude)
		assert.Equal(t, []string{"a", "c"}, result)
	})

	t.Run("no exclusions", func(t *testing.T) {
		all := []string{"a", "b", "c"}
		result := filterSlice(all, nil)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("exclude all", func(t *testing.T) {
		all := []string{"a", "b"}
		exclude := []string{"a", "b"}
		result := filterSlice(all, exclude)
		assert.Empty(t, result)
	})

	t.Run("empty input", func(t *testing.T) {
		result := filterSlice(nil, []string{"a"})
		assert.Empty(t, result)
	})
}
