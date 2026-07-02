package patterns

import (
	"context"
	"regexp"
	"sort"

	"agent-memory-mcp/internal/core"
	"agent-memory-mcp/internal/store/sqlite"
)

// Correlator finds relationships between memories using content overlap,
// file references, and temporal proximity.
type Correlator struct {
	store *sqlite.Store
}

func NewCorrelator(store *sqlite.Store) *Correlator {
	return &Correlator{store: store}
}

// GetRelated finds memories related to the given memory.
func (c *Correlator) GetRelated(ctx context.Context, memoryID string, limit int) ([]core.Memory, error) {
	// Get the source memory
	mems, err := c.store.ListMemories(ctx, nil, nil, 500)
	if err != nil {
		return nil, err
	}

	var source *core.Memory
	for i := range mems {
		if mems[i].ID == memoryID {
			source = &mems[i]
			break
		}
	}
	if source == nil {
		return []core.Memory{}, nil
	}

	type scored struct {
		mem       core.Memory
		relevance float64
		reason    string
	}

	candidates := map[string]*scored{}

	// Strategy 1: Shared file references
	fileRefs := extractFileRefs(source.Content)
	if len(fileRefs) > 0 {
		for _, ref := range fileRefs {
			if len(ref) < 3 {
				continue
			}
			hits, _ := c.store.Search(ctx, ref, nil, nil, limit*2)
			for _, hit := range hits {
				if hit.ID != memoryID {
					if existing, ok := candidates[hit.ID]; ok {
						existing.relevance += 0.4
					} else {
						candidates[hit.ID] = &scored{mem: hit, relevance: 0.4, reason: "shared_file"}
					}
				}
			}
		}
	}

	// Strategy 2: Content similarity via FTS search
	queryText := source.Content
	if len(queryText) > 300 {
		queryText = queryText[:300]
	}
	similar, _ := c.store.Search(ctx, queryText, nil, nil, limit*2)
	for _, hit := range similar {
		if hit.ID != memoryID {
			simScore := hit.RelevanceScore
			if simScore == 0 {
				simScore = 0.3
			}
			if existing, ok := candidates[hit.ID]; ok {
				existing.relevance += simScore
			} else {
				candidates[hit.ID] = &scored{mem: hit, relevance: simScore, reason: "content_similar"}
			}
		}
	}

	// Strategy 3: Temporal proximity (same creation window)
	for _, m := range mems {
		if m.ID == memoryID {
			continue
		}
		// Simple temporal proximity: share the same 10-minute window (same timestamp prefix)
		if len(source.CreatedAt) >= 16 && len(m.CreatedAt) >= 16 {
			if source.CreatedAt[:16] == m.CreatedAt[:16] {
				if existing, ok := candidates[m.ID]; ok {
					existing.relevance += 0.2
				} else {
					candidates[m.ID] = &scored{mem: m, relevance: 0.2, reason: "temporal_proximity"}
				}
			}
		}
	}

	// Rank and return
	var ranked []scored
	for _, s := range candidates {
		ranked = append(ranked, *s)
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].relevance > ranked[j].relevance
	})

	result := []core.Memory{}
	for i, s := range ranked {
		if i >= limit {
			break
		}
		result = append(result, s.mem)
	}
	return result, nil
}

var fileRefPatterns = []*regexp.Regexp{
	regexp.MustCompile(`[\w/.-]+\.\w{1,5}`),
	regexp.MustCompile(`(?:src|lib|app|tests?)/[\w/.-]+`),
}

func extractFileRefs(text string) []string {
	seen := map[string]bool{}
	var refs []string
	for _, pattern := range fileRefPatterns {
		for _, match := range pattern.FindAllString(text, -1) {
			if len(match) > 5 && !seen[match] && len(match) < 200 {
				seen[match] = true
				refs = append(refs, match)
			}
		}
	}
	if len(refs) > 10 {
		refs = refs[:10]
	}
	return refs
}

// FindSimilarFailures finds failures in other repositories similar to failures in the given repo.
func (c *Correlator) FindSimilarFailures(ctx context.Context, repoID string, limit int) ([]core.Memory, error) {
	// 1. Get recent failures in this repo
	var failures []core.Memory
	mems, err := c.store.ListMemories(ctx, &repoID, nil, 100)
	if err != nil {
		return nil, err
	}
	for _, m := range mems {
		if m.Kind == "failure" {
			failures = append(failures, m)
		}
	}
	if len(failures) == 0 {
		return []core.Memory{}, nil
	}

	// 2. Build a combined query text from recent failures
	var queryText string
	for i, m := range failures {
		if i >= 5 {
			break
		}
		queryText += m.Content + " "
	}
	if len(queryText) > 500 {
		queryText = queryText[:500]
	}

	// 3. Search for failures globally (repoID=nil) using this combined text
	kinds := []string{"failure"}
	similar, err := c.store.Search(ctx, queryText, nil, kinds, limit*2)
	if err != nil {
		return nil, err
	}

	// 4. Filter out failures from the same repo
	var result []core.Memory
	for _, m := range similar {
		if m.RepoID != repoID {
			result = append(result, m)
			if len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}
