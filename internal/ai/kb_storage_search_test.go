package ai

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyGraphBoost_NoSalienceData(t *testing.T) {
	chunks := []RetrievalResult{
		{ChunkID: "c1", DocumentID: "d1", Content: "chunk1", Similarity: 0.9},
		{ChunkID: "c2", DocumentID: "d2", Content: "chunk2", Similarity: 0.8},
	}
	opts := GraphBoostOptions{Limit: 2, GraphBoostWeight: 0.3}

	results := applyGraphBoost(chunks, map[string]float64{}, opts)

	require.Len(t, results, 2)
	// With no entity data, order should follow original similarity
	assert.Equal(t, "c1", results[0].ChunkID)
	assert.Equal(t, "c2", results[1].ChunkID)
}

func TestApplyGraphBoost_EntityBoostReordersResults(t *testing.T) {
	chunks := []RetrievalResult{
		{ChunkID: "c1", DocumentID: "d1", Content: "chunk1", Similarity: 0.9},
		{ChunkID: "c2", DocumentID: "d2", Content: "chunk2", Similarity: 0.5},
	}

	// Document d2 has high entity salience, d1 has none
	salience := map[string]float64{
		"d2": 1.0,
	}

	// With 50% graph boost weight, d2's final score should be boosted enough
	// to overtake d1: d2 = 0.5*0.5 + 1.0*0.5 = 0.75 vs d1 = 0.9*0.5 + 0*0.5 = 0.45
	opts := GraphBoostOptions{Limit: 2, GraphBoostWeight: 0.5}

	results := applyGraphBoost(chunks, salience, opts)

	require.Len(t, results, 2)
	// d2 should now rank first due to entity boost
	assert.Equal(t, "c2", results[0].ChunkID)
	assert.Equal(t, "c1", results[1].ChunkID)
	assert.InDelta(t, 0.75, results[0].Similarity, 0.001)
	assert.InDelta(t, 0.45, results[1].Similarity, 0.001)
}

func TestApplyGraphBoost_RespectsLimit(t *testing.T) {
	chunks := []RetrievalResult{
		{ChunkID: "c1", DocumentID: "d1", Similarity: 0.9},
		{ChunkID: "c2", DocumentID: "d2", Similarity: 0.8},
		{ChunkID: "c3", DocumentID: "d3", Similarity: 0.7},
	}
	opts := GraphBoostOptions{Limit: 2, GraphBoostWeight: 0.3}

	results := applyGraphBoost(chunks, map[string]float64{}, opts)

	require.Len(t, results, 2)
	assert.Equal(t, "c1", results[0].ChunkID)
	assert.Equal(t, "c2", results[1].ChunkID)
}

func TestApplyGraphBoost_EmptyChunks(t *testing.T) {
	results := applyGraphBoost([]RetrievalResult{}, map[string]float64{"d1": 1.0}, GraphBoostOptions{Limit: 5, GraphBoostWeight: 0.3})
	assert.Empty(t, results)
}

func TestApplyGraphBoost_ZeroGraphBoostWeight(t *testing.T) {
	chunks := []RetrievalResult{
		{ChunkID: "c1", DocumentID: "d1", Similarity: 0.9},
		{ChunkID: "c2", DocumentID: "d2", Similarity: 0.5},
	}
	salience := map[string]float64{"d2": 1.0}
	opts := GraphBoostOptions{Limit: 2, GraphBoostWeight: 0}

	results := applyGraphBoost(chunks, salience, opts)

	// With 0 graph weight, results should follow pure vector similarity
	require.Len(t, results, 2)
	assert.Equal(t, "c1", results[0].ChunkID)
	assert.Equal(t, 0.9, results[0].Similarity)
	assert.Equal(t, 0.5, results[1].Similarity)
}

func TestApplyGraphBoost_MetadataContainsBoostInfo(t *testing.T) {
	chunks := []RetrievalResult{
		{ChunkID: "c1", DocumentID: "d1", Similarity: 0.8},
	}
	salience := map[string]float64{"d1": 0.6}
	opts := GraphBoostOptions{Limit: 1, GraphBoostWeight: 0.3}

	results := applyGraphBoost(chunks, salience, opts)

	require.Len(t, results, 1)
	var metadata map[string]any
	err := json.Unmarshal(results[0].Metadata, &metadata)
	require.NoError(t, err)
	assert.Contains(t, metadata, "entity_boost")
	assert.Contains(t, metadata, "final_score")
}

func TestApplyGraphBoost_PreservesExistingMetadata(t *testing.T) {
	existingMeta, _ := json.Marshal(map[string]any{"source": "pdf"})
	chunks := []RetrievalResult{
		{ChunkID: "c1", DocumentID: "d1", Similarity: 0.8, Metadata: existingMeta},
	}
	opts := GraphBoostOptions{Limit: 1, GraphBoostWeight: 0.0}

	results := applyGraphBoost(chunks, map[string]float64{}, opts)

	require.Len(t, results, 1)
	var metadata map[string]any
	err := json.Unmarshal(results[0].Metadata, &metadata)
	require.NoError(t, err)
	assert.Equal(t, "pdf", metadata["source"])
	assert.Contains(t, metadata, "entity_boost")
}

func TestApplyGraphBoost_NormalizesSalience(t *testing.T) {
	chunks := []RetrievalResult{
		{ChunkID: "c1", DocumentID: "d1", Similarity: 0.5},
		{ChunkID: "c2", DocumentID: "d2", Similarity: 0.5},
	}
	// d1 has 0.4, d2 has 0.8 salience. d2 should get full boost (1.0), d1 gets half (0.5)
	salience := map[string]float64{
		"d1": 0.4,
		"d2": 0.8,
	}
	opts := GraphBoostOptions{Limit: 2, GraphBoostWeight: 0.5}

	results := applyGraphBoost(chunks, salience, opts)

	require.Len(t, results, 2)
	// d2: 0.5*0.5 + 1.0*0.5 = 0.75
	// d1: 0.5*0.5 + 0.5*0.5 = 0.5
	assert.Equal(t, "c2", results[0].ChunkID)
	assert.InDelta(t, 0.75, results[0].Similarity, 0.001)
	assert.Equal(t, "c1", results[1].ChunkID)
	assert.InDelta(t, 0.5, results[1].Similarity, 0.001)
}

func TestGetDocumentsByEntity_MethodExists(t *testing.T) {
	// Compile-time check that GetDocumentsByEntity exists on *KnowledgeGraph
	var _ func(string) = func(entityID string) {
		kg := &KnowledgeGraph{}
		// This will compile because the method exists
		_ = kg
	}
	// Verify the method signature matches expected pattern
	assert.NotNil(t, (*KnowledgeGraph).GetDocumentsByEntity)
}
