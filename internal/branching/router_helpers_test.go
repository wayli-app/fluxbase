package branching

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// Unit tests filling the remaining GetActiveBranch / GetDefaultBranch branches
// (66.7% and 80%). These need a Router with an initialized activeBranch value,
// which SetActiveBranch provides. No DB pool required — the methods only touch
// the atomic.Value and config.
//
// Convention matches router_test.go: testify, package branching.

func TestRouter_GetActiveBranch_WhenSet_PrecedenceMatrix(t *testing.T) {
	t.Parallel()
	r := &Router{}
	// Before any SetActiveBranch, GetActiveBranch returns "" (activeBranch holds
	// the zero value "" set by NewRouter; a bare &Router{} has a nil Load).
	assert.Equal(t, "", r.GetActiveBranch())

	// After SetActiveBranch, the value is returned.
	r.SetActiveBranch("feature-x")
	assert.Equal(t, "feature-x", r.GetActiveBranch())

	r.SetActiveBranch("main")
	assert.Equal(t, "main", r.GetActiveBranch())
}

func TestRouter_GetDefaultBranch_Precedence(t *testing.T) {
	t.Parallel()
	t.Run("API-set active branch wins over config and default", func(t *testing.T) {
		t.Parallel()
		r := &Router{config: config.BranchingConfig{DefaultBranch: "develop"}}
		r.SetActiveBranch("hotfix")
		assert.Equal(t, "hotfix", r.GetDefaultBranch())
	})

	t.Run("config default used when no API-set branch", func(t *testing.T) {
		t.Parallel()
		r := &Router{config: config.BranchingConfig{DefaultBranch: "develop"}}
		// No SetActiveBranch call -> config default wins.
		assert.Equal(t, "develop", r.GetDefaultBranch())
	})

	t.Run("falls back to main when neither set", func(t *testing.T) {
		t.Parallel()
		r := &Router{}
		assert.Equal(t, "main", r.GetDefaultBranch())
	})
}

func TestRouter_GetActiveBranchSource_PrecedenceMatrix(t *testing.T) {
	t.Parallel()
	t.Run("api when active branch set", func(t *testing.T) {
		t.Parallel()
		r := &Router{config: config.BranchingConfig{DefaultBranch: "develop"}}
		r.SetActiveBranch("feature")
		assert.Equal(t, "api", r.GetActiveBranchSource())
	})

	t.Run("config when only config default set", func(t *testing.T) {
		t.Parallel()
		r := &Router{config: config.BranchingConfig{DefaultBranch: "develop"}}
		assert.Equal(t, "config", r.GetActiveBranchSource())
	})

	t.Run("default when nothing set", func(t *testing.T) {
		t.Parallel()
		r := &Router{}
		assert.Equal(t, "default", r.GetActiveBranchSource())
	})
}
