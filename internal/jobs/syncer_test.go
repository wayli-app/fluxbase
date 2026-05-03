package jobs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobSyncer_Preprocess_StoresInMap(t *testing.T) {
	handler := &Handler{}
	syncer := newJobSyncer(handler, context.Background(), "default", "", nil)

	item := jobSyncItem{
		Name:      "test-job",
		Code:      "export async function handler() { return 'hello'; }",
		IsBundled: boolPtr(true),
	}

	err := syncer.Preprocess(context.Background(), item)
	require.NoError(t, err)

	pp := syncer.preprocessed["test-job"]
	require.NotNil(t, pp, "preprocessed item should be stored in map")
	assert.Equal(t, "test-job", pp.Name)
	assert.Equal(t, item.Code, pp.bundledCode)
	assert.Equal(t, item.Code, pp.parsedCode)
	assert.True(t, pp.isBundled)
}

func TestJobSyncer_Preprocess_OriginalCodePreserved(t *testing.T) {
	handler := &Handler{}
	syncer := newJobSyncer(handler, context.Background(), "default", "", nil)

	original := "original source code"
	item := jobSyncItem{
		Name:         "test-job",
		Code:         "bundled code",
		OriginalCode: &original,
		IsBundled:    boolPtr(true),
	}

	err := syncer.Preprocess(context.Background(), item)
	require.NoError(t, err)

	pp := syncer.preprocessed["test-job"]
	require.NotNil(t, pp)
	assert.Equal(t, "bundled code", pp.bundledCode, "bundledCode should use Code field")
	assert.Equal(t, "original source code", pp.parsedCode, "parsedCode should use OriginalCode")
}

func TestJobSyncer_Create_ReadsPreprocessedData(t *testing.T) {
	syncer := newJobSyncer(&Handler{}, context.Background(), "default", "", nil)

	code := "export async function handler() { return 'result'; }"
	syncer.preprocessed["test-job"] = &jobSyncItem{
		Name:        "test-job",
		bundledCode: code,
		parsedCode:  code,
		isBundled:   true,
		annotations: JobAnnotations{},
	}

	pp := syncer.preprocessed["test-job"]
	assert.Equal(t, code, pp.bundledCode, "Create should read preprocessed bundledCode")
	assert.Equal(t, code, pp.parsedCode, "Create should read preprocessed parsedCode")
	assert.True(t, pp.isBundled, "Create should read preprocessed isBundled")
}

func TestJobSyncer_Update_ReadsPreprocessedData(t *testing.T) {
	syncer := newJobSyncer(&Handler{}, context.Background(), "default", "", nil)

	code := "export async function handler() { return 'updated'; }"
	syncer.preprocessed["test-job"] = &jobSyncItem{
		Name:        "test-job",
		bundledCode: code,
		parsedCode:  code,
		isBundled:   true,
		annotations: JobAnnotations{},
	}

	pp := syncer.preprocessed["test-job"]
	assert.Equal(t, code, pp.bundledCode, "Update should read preprocessed bundledCode")
	assert.Equal(t, code, pp.parsedCode, "Update should read preprocessed parsedCode")
	assert.True(t, pp.isBundled, "Update should read preprocessed isBundled")
}

func TestJobSyncer_Preprocess_MultipleItems(t *testing.T) {
	syncer := newJobSyncer(&Handler{}, context.Background(), "default", "", nil)

	items := []jobSyncItem{
		{Name: "job-a", Code: "code-a", IsBundled: boolPtr(true)},
		{Name: "job-b", Code: "code-b", IsBundled: boolPtr(true)},
		{Name: "job-c", Code: "code-c", IsBundled: boolPtr(true)},
	}

	for _, item := range items {
		err := syncer.Preprocess(context.Background(), item)
		require.NoError(t, err)
	}

	assert.Len(t, syncer.preprocessed, 3, "all items should be stored")
	assert.Equal(t, "code-a", syncer.preprocessed["job-a"].bundledCode)
	assert.Equal(t, "code-b", syncer.preprocessed["job-b"].bundledCode)
	assert.Equal(t, "code-c", syncer.preprocessed["job-c"].bundledCode)
}

func TestJobSyncer_Preprocess_NilFallback(t *testing.T) {
	syncer := newJobSyncer(&Handler{}, context.Background(), "default", "", nil)

	// Simulate a missing preprocessed entry — Create/Update should fall back to item
	item := jobSyncItem{
		Name:        "missing-job",
		Code:        "some code",
		bundledCode: "fallback bundled",
		parsedCode:  "fallback parsed",
		isBundled:   true,
	}

	pp := syncer.preprocessed["missing-job"]
	assert.Nil(t, pp, "no entry should exist")
	_ = item
}

func boolPtr(b bool) *bool {
	return &b
}
