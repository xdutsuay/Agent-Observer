package smartctx

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"agent-memory-mcp/internal/core"
	"agent-memory-mcp/internal/store/sqlite"
)

// Packer selects and packs memories into a fixed token budget for agent injection.
type Packer struct {
	store *sqlite.Store
}

func NewPacker(store *sqlite.Store) *Packer {
	return &Packer{store: store}
}

// Pack selects the best memories for the given task and packs them under a token budget.
func (p *Packer) Pack(ctx context.Context, repoID, task string, maxTokens int) (core.SmartContext, error) {
	if maxTokens <= 0 {
		maxTokens = 8000
	}

	// 1. Search for task-relevant memories
	searchResults, err := p.store.Search(ctx, task, &repoID, nil, 30)
	if err != nil {
		return core.SmartContext{}, err
	}

	// 2. Also pull in high-priority memories (decisions, facts) regardless of search match
	kind := "decision"
	decisions, _ := p.store.ListMemories(ctx, &repoID, &kind, 10)
	kind = "fact"
	facts, _ := p.store.ListMemories(ctx, &repoID, &kind, 10)

	// Merge all candidates, deduplicating by ID
	seen := map[string]bool{}
	var candidates []core.Memory

	for _, m := range searchResults {
		if !seen[m.ID] {
			seen[m.ID] = true
			candidates = append(candidates, m)
		}
	}
	for _, m := range decisions {
		if !seen[m.ID] {
			seen[m.ID] = true
			candidates = append(candidates, m)
		}
	}
	for _, m := range facts {
		if !seen[m.ID] {
			seen[m.ID] = true
			candidates = append(candidates, m)
		}
	}

	// 3. Score and rank candidates
	type scoredMem struct {
		mem   core.Memory
		score float64
	}

	var scored []scoredMem
	for _, m := range candidates {
		s := scoreMemory(m, task)
		scored = append(scored, scoredMem{mem: m, score: s})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 4. Pack under token budget
	var selected []core.Memory
	totalTokens := 0

	for _, sm := range scored {
		tokens := estimateTokens(sm.mem.Content)
		if totalTokens+tokens > maxTokens {
			continue // Skip if it would blow the budget
		}
		selected = append(selected, sm.mem)
		totalTokens += tokens
	}

	if selected == nil {
		selected = []core.Memory{}
	}

	// 5. Build system prompt fragment
	fragment := buildPromptFragment(selected, repoID)

	return core.SmartContext{
		Memories:             selected,
		TokenEstimate:        totalTokens,
		SystemPromptFragment: fragment,
	}, nil
}

// scoreMemory computes a composite score for ranking a memory candidate.
func scoreMemory(m core.Memory, task string) float64 {
	score := 0.0

	// Kind weight
	switch m.Kind {
	case "decision":
		score += 0.35
	case "fact":
		score += 0.30
	case "preference":
		score += 0.25
	case "failure":
		score += 0.20
	case "attempt":
		score += 0.05
	}

	// Relevance score from DB
	score += m.RelevanceScore * 0.30

	// Task keyword overlap
	taskWords := strings.Fields(strings.ToLower(task))
	contentLower := strings.ToLower(m.Content)
	matchCount := 0
	for _, w := range taskWords {
		if len(w) >= 3 && strings.Contains(contentLower, w) {
			matchCount++
		}
	}
	if len(taskWords) > 0 {
		overlap := float64(matchCount) / float64(len(taskWords))
		score += overlap * 0.25
	}

	// Access count bonus (capped)
	if m.AccessCount > 0 {
		acBonus := float64(m.AccessCount) / 50.0
		if acBonus > 0.10 {
			acBonus = 0.10
		}
		score += acBonus
	}

	return score
}

// estimateTokens gives a rough token estimate for a string.
// ~4 chars per token is a common approximation.
func estimateTokens(s string) int {
	return len(s) / 4
}

// buildPromptFragment creates a formatted text block suitable for injecting into a system prompt.
func buildPromptFragment(mems []core.Memory, repoID string) string {
	if len(mems) == 0 {
		return fmt.Sprintf("No relevant memories found for repo %s.", repoID)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Agent Memory Context (repo: %s)\n\n", repoID))

	// Group by kind
	groups := map[string][]core.Memory{}
	for _, m := range mems {
		groups[m.Kind] = append(groups[m.Kind], m)
	}

	kindOrder := []string{"decision", "fact", "preference", "failure", "attempt"}
	kindLabels := map[string]string{
		"decision":   "Key Decisions",
		"fact":       "Known Facts",
		"preference": "Preferences",
		"failure":    "Known Issues",
		"attempt":    "Recent Attempts",
	}

	for _, kind := range kindOrder {
		ms, ok := groups[kind]
		if !ok || len(ms) == 0 {
			continue
		}
		label := kindLabels[kind]
		sb.WriteString(fmt.Sprintf("### %s\n", label))
		for _, m := range ms {
			content := m.Content
			if len(content) > 500 {
				content = content[:500] + "..."
			}
			sb.WriteString(fmt.Sprintf("- %s\n", content))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
