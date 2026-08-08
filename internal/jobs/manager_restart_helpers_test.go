package jobs

import (
	"fmt"

	"github.com/google/uuid"
)

// newTestUUID returns a fresh random UUID for tests.
func newTestUUID() uuid.UUID { return uuid.New() }

// errFail returns a simple error used only by restart tests.
func errFail(msg string) error { return fmt.Errorf("test failure: %s", msg) }
