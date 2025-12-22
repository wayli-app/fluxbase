package migrations

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCalculateHash(t *testing.T) {
	t.Run("generates consistent hash", func(t *testing.T) {
		content := "SELECT * FROM users;"
		hash1 := calculateHash(content)
		hash2 := calculateHash(content)

		assert.Equal(t, hash1, hash2)
		assert.Len(t, hash1, 64) // SHA256 produces 64 hex chars
	})

	t.Run("different content produces different hash", func(t *testing.T) {
		hash1 := calculateHash("content one")
		hash2 := calculateHash("content two")

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("empty string produces valid hash", func(t *testing.T) {
		hash := calculateHash("")
		assert.Len(t, hash, 64)
		// SHA256 of empty string is well-known
		assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hash)
	})

	t.Run("hash is hex encoded", func(t *testing.T) {
		hash := calculateHash("test content")

		for _, c := range hash {
			assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
				"hash should only contain hex chars, got: %c", c)
		}
	})

	t.Run("whitespace matters", func(t *testing.T) {
		hash1 := calculateHash("SELECT * FROM users;")
		hash2 := calculateHash(" SELECT * FROM users;")
		hash3 := calculateHash("SELECT * FROM users; ")

		assert.NotEqual(t, hash1, hash2)
		assert.NotEqual(t, hash1, hash3)
		assert.NotEqual(t, hash2, hash3)
	})

	t.Run("case sensitive", func(t *testing.T) {
		hash1 := calculateHash("SELECT")
		hash2 := calculateHash("select")

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("handles unicode", func(t *testing.T) {
		hash := calculateHash("-- Comment with émojis 🎉")
		assert.Len(t, hash, 64)
	})

	t.Run("handles multiline content", func(t *testing.T) {
		content := `CREATE TABLE users (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);`
		hash := calculateHash(content)
		assert.Len(t, hash, 64)
	})
}

func TestValueOrEmpty(t *testing.T) {
	t.Run("returns empty string for nil", func(t *testing.T) {
		result := valueOrEmpty(nil)
		assert.Equal(t, "", result)
	})

	t.Run("returns string value for non-nil", func(t *testing.T) {
		s := "test value"
		result := valueOrEmpty(&s)
		assert.Equal(t, "test value", result)
	})

	t.Run("returns empty string pointer value", func(t *testing.T) {
		s := ""
		result := valueOrEmpty(&s)
		assert.Equal(t, "", result)
	})

	t.Run("returns whitespace string", func(t *testing.T) {
		s := "   "
		result := valueOrEmpty(&s)
		assert.Equal(t, "   ", result)
	})

	t.Run("returns long string", func(t *testing.T) {
		longStr := "a very long string that contains many characters"
		result := valueOrEmpty(&longStr)
		assert.Equal(t, longStr, result)
	})

	t.Run("returns string with special characters", func(t *testing.T) {
		s := "string with\nnewlines\tand\ttabs"
		result := valueOrEmpty(&s)
		assert.Equal(t, s, result)
	})
}

func TestMigration_Struct(t *testing.T) {
	t.Run("creates migration with required fields", func(t *testing.T) {
		m := Migration{
			ID:        uuid.New(),
			Namespace: "default",
			Name:      "001_create_users.sql",
			UpSQL:     "CREATE TABLE users (...);",
			Status:    "pending",
		}

		assert.NotEqual(t, uuid.Nil, m.ID)
		assert.Equal(t, "default", m.Namespace)
		assert.Equal(t, "001_create_users.sql", m.Name)
		assert.Equal(t, "pending", m.Status)
	})

	t.Run("handles optional fields", func(t *testing.T) {
		desc := "Creates initial users table"
		downSQL := "DROP TABLE users;"
		m := Migration{
			ID:          uuid.New(),
			Namespace:   "auth",
			Name:        "002_create_roles.sql",
			Description: &desc,
			UpSQL:       "CREATE TABLE roles...;",
			DownSQL:     &downSQL,
			Status:      "applied",
		}

		assert.NotNil(t, m.Description)
		assert.Equal(t, "Creates initial users table", *m.Description)
		assert.NotNil(t, m.DownSQL)
		assert.Equal(t, "DROP TABLE users;", *m.DownSQL)
	})

	t.Run("status values", func(t *testing.T) {
		statuses := []string{"pending", "applied", "failed", "rolled_back"}
		for _, status := range statuses {
			m := Migration{Status: status}
			assert.Equal(t, status, m.Status)
		}
	})
}

func TestExecutionLog_Struct(t *testing.T) {
	t.Run("creates execution log with success", func(t *testing.T) {
		migrationID := uuid.New()
		durationMs := 150
		logs := "Migration applied successfully"

		log := ExecutionLog{
			ID:          uuid.New(),
			MigrationID: migrationID,
			Action:      "apply",
			Status:      "success",
			DurationMs:  &durationMs,
			Logs:        &logs,
			ExecutedAt:  time.Now(),
		}

		assert.Equal(t, migrationID, log.MigrationID)
		assert.Equal(t, "apply", log.Action)
		assert.Equal(t, "success", log.Status)
		assert.Equal(t, 150, *log.DurationMs)
	})

	t.Run("creates execution log with failure", func(t *testing.T) {
		errorMsg := "syntax error at position 42"

		log := ExecutionLog{
			ID:           uuid.New(),
			MigrationID:  uuid.New(),
			Action:       "apply",
			Status:       "failed",
			ErrorMessage: &errorMsg,
			ExecutedAt:   time.Now(),
		}

		assert.Equal(t, "failed", log.Status)
		assert.NotNil(t, log.ErrorMessage)
		assert.Contains(t, *log.ErrorMessage, "syntax error")
	})

	t.Run("supports rollback action", func(t *testing.T) {
		log := ExecutionLog{
			ID:          uuid.New(),
			MigrationID: uuid.New(),
			Action:      "rollback",
			Status:      "success",
			ExecutedAt:  time.Now(),
		}

		assert.Equal(t, "rollback", log.Action)
	})
}
