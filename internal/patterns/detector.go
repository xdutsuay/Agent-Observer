package patterns

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"agent-memory-mcp/internal/core"
	"agent-memory-mcp/internal/store/sqlite"
)

// Detector identifies recurring patterns, trends, and anomalies in memories.
type Detector struct {
	store *sqlite.Store
}

func NewDetector(store *sqlite.Store) *Detector {
	return &Detector{store: store}
}

// GetReport generates a full pattern report for a repo (or all repos if repoID is nil).
func (d *Detector) GetReport(ctx context.Context, repoID *string) (map[string]any, error) {
	recurring, err := d.recurringFailures(ctx, repoID)
	if err != nil {
		return nil, err
	}

	trends, err := d.activityTrends(ctx, repoID)
	if err != nil {
		return nil, err
	}

	categories, err := d.errorCategories(ctx, repoID)
	if err != nil {
		return nil, err
	}

	decisionPatterns, err := d.decisionPatterns(ctx, repoID)
	if err != nil {
		return nil, err
	}

	health, err := d.healthScore(ctx, repoID, recurring)
	if err != nil {
		return nil, err
	}

	rid := "global"
	if repoID != nil {
		rid = *repoID
	}

	// Count total memories
	mems, _ := d.store.ListMemories(ctx, repoID, nil, 1000)

	failCount := 0
	decCount := 0
	attCount := 0
	for _, m := range mems {
		switch m.Kind {
		case "failure":
			failCount++
		case "decision":
			decCount++
		case "attempt":
			attCount++
		}
	}

	return map[string]any{
		"repo_id":                     rid,
		"total_memories":              len(mems),
		"recurring_failures":          recurring,
		"activity_trends":             trends,
		"common_error_categories":     categories,
		"decision_patterns":           decisionPatterns,
		"health_score":                health,
		"breakdown": map[string]int{
			"failures":  failCount,
			"decisions": decCount,
			"attempts":  attCount,
		},
		"recent_unresolved_signatures": d.unresolvedSignatures(ctx, repoID),
	}, nil
}

func (d *Detector) recurringFailures(ctx context.Context, repoID *string) ([]map[string]any, error) {
	var allSigs []map[string]any

	if repoID != nil {
		sigs, err := d.store.GetFailureSignatures(ctx, *repoID)
		if err != nil {
			return nil, err
		}
		allSigs = sigs
	} else {
		repos, err := d.store.ListReposDB(ctx)
		if err != nil {
			return nil, err
		}
		for _, repo := range repos {
			sigs, _ := d.store.GetFailureSignatures(ctx, repo.ID)
			allSigs = append(allSigs, sigs...)
		}
	}

	var recurring []map[string]any
	for _, s := range allSigs {
		count, _ := s["count"].(int)
		resolved, _ := s["resolved"].(bool)
		if count >= 2 && !resolved {
			recurring = append(recurring, s)
		}
	}

	sort.Slice(recurring, func(i, j int) bool {
		ci, _ := recurring[i]["count"].(int)
		cj, _ := recurring[j]["count"].(int)
		return ci > cj
	})

	if len(recurring) > 20 {
		recurring = recurring[:20]
	}
	if recurring == nil {
		recurring = []map[string]any{}
	}
	return recurring, nil
}

func (d *Detector) activityTrends(ctx context.Context, repoID *string) (map[string]any, error) {
	mems, err := d.store.ListMemories(ctx, repoID, nil, 500)
	if err != nil {
		return nil, err
	}
	if len(mems) == 0 {
		return map[string]any{
			"total":   0,
			"by_day":  map[string]int{},
			"by_kind": map[string]int{},
		}, nil
	}

	byDay := map[string]int{}
	byKind := map[string]int{}

	for _, m := range mems {
		if len(m.CreatedAt) >= 10 {
			day := m.CreatedAt[:10]
			byDay[day]++
		}
		byKind[m.Kind]++
	}

	// Keep only last 30 days
	if len(byDay) > 30 {
		days := make([]string, 0, len(byDay))
		for d := range byDay {
			days = append(days, d)
		}
		sort.Strings(days)
		trimmed := map[string]int{}
		for _, d := range days[len(days)-30:] {
			trimmed[d] = byDay[d]
		}
		byDay = trimmed
	}

	return map[string]any{
		"total":   len(mems),
		"by_day":  byDay,
		"by_kind": byKind,
	}, nil
}

var errorPatterns = []struct {
	pattern  *regexp.Regexp
	category string
}{
	{regexp.MustCompile(`(?i)timeout|timed?\s*out`), "Timeout"},
	{regexp.MustCompile(`(?i)connection\s*(refused|reset|error)`), "Connection Error"},
	{regexp.MustCompile(`(?i)import\s*error|module\s*not\s*found|no\s*module`), "Import Error"},
	{regexp.MustCompile(`(?i)syntax\s*error`), "Syntax Error"},
	{regexp.MustCompile(`(?i)type\s*error|typeerror`), "Type Error"},
	{regexp.MustCompile(`(?i)permission|access\s*denied|forbidden`), "Permission Error"},
	{regexp.MustCompile(`(?i)not\s*found|404|missing`), "Not Found"},
	{regexp.MustCompile(`(?i)memory|oom|out\s*of\s*memory`), "Memory Error"},
	{regexp.MustCompile(`(?i)null|none|undefined|nil`), "Null Reference"},
	{regexp.MustCompile(`(?i)assertion|assert`), "Assertion Error"},
	{regexp.MustCompile(`(?i)test\s*fail`), "Test Failure"},
	{regexp.MustCompile(`(?i)build\s*fail|compile`), "Build Error"},
}

func categorizeError(content string) string {
	for _, ep := range errorPatterns {
		if ep.pattern.MatchString(content) {
			return ep.category
		}
	}
	return "Other"
}

func (d *Detector) errorCategories(ctx context.Context, repoID *string) ([]map[string]any, error) {
	kind := "failure"
	failures, err := d.store.ListMemories(ctx, repoID, &kind, 200)
	if err != nil {
		return nil, err
	}

	counts := map[string]int{}
	for _, f := range failures {
		cat := categorizeError(f.Content)
		counts[cat]++
	}

	type catCount struct {
		cat   string
		count int
	}
	var sorted []catCount
	for cat, count := range counts {
		sorted = append(sorted, catCount{cat, count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })

	result := []map[string]any{}
	for i, cc := range sorted {
		if i >= 15 {
			break
		}
		result = append(result, map[string]any{"category": cc.cat, "count": cc.count})
	}
	return result, nil
}

var stopwords = map[string]bool{
	"this": true, "that": true, "with": true, "from": true, "have": true,
	"will": true, "been": true, "were": true, "than": true, "into": true,
	"also": true, "just": true, "more": true, "some": true, "when": true,
	"what": true, "each": true, "make": true, "like": true, "over": true,
	"such": true, "take": true, "only": true, "come": true, "could": true,
	"them": true, "made": true, "after": true, "before": true, "should": true,
	"would": true, "about": true, "which": true, "their": true, "there": true,
	"other": true, "because": true, "these": true, "those": true, "being": true,
	"does": true, "done": true, "most": true, "very": true, "using": true,
}

var wordRegex = regexp.MustCompile(`[a-z]{4,}`)

func (d *Detector) decisionPatterns(ctx context.Context, repoID *string) (map[string]any, error) {
	kind := "decision"
	decisions, err := d.store.ListMemories(ctx, repoID, &kind, 200)
	if err != nil {
		return nil, err
	}

	topicCounts := map[string]int{}
	for _, dec := range decisions {
		words := wordRegex.FindAllString(strings.ToLower(dec.Content), -1)
		for _, w := range words {
			if !stopwords[w] {
				topicCounts[w]++
			}
		}
	}

	type topicCount struct {
		topic string
		count int
	}
	var sorted []topicCount
	for t, c := range topicCounts {
		sorted = append(sorted, topicCount{t, c})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })

	topics := []map[string]any{}
	for i, tc := range sorted {
		if i >= 15 {
			break
		}
		topics = append(topics, map[string]any{"topic": tc.topic, "count": tc.count})
	}

	return map[string]any{
		"total_decisions": len(decisions),
		"top_topics":      topics,
	}, nil
}

func (d *Detector) healthScore(ctx context.Context, repoID *string, recurring []map[string]any) (map[string]any, error) {
	score := 100
	reasons := []string{}

	var failCount int
	if repoID != nil {
		failCount = d.store.FailureErrorCount(ctx, *repoID)
	} else {
		repos, _ := d.store.ListReposDB(ctx)
		for _, r := range repos {
			failCount += d.store.FailureErrorCount(ctx, r.ID)
		}
	}

	if failCount > 0 {
		penalty := failCount * 5
		if penalty > 40 {
			penalty = 40
		}
		score -= penalty
		reasons = append(reasons, strings.Replace(
			strings.Replace("N unresolved failures (-P)", "N", itoa(failCount), 1),
			"P", itoa(penalty), 1))
	}

	var highCount int
	for _, r := range recurring {
		if c, ok := r["count"].(int); ok && c >= 5 {
			highCount++
		}
	}
	if highCount > 0 {
		penalty := highCount * 10
		if penalty > 30 {
			penalty = 30
		}
		score -= penalty
		reasons = append(reasons, strings.Replace(
			strings.Replace("N highly recurring failures (-P)", "N", itoa(highCount), 1),
			"P", itoa(penalty), 1))
	}

	kind := "decision"
	decisions, _ := d.store.ListMemories(ctx, repoID, &kind, 10)
	if len(decisions) >= 3 {
		bonus := 10
		score += bonus
		if score > 100 {
			score = 100
		}
		reasons = append(reasons, strings.Replace("Good decision documentation (+B)", "B", itoa(bonus), 1))
	}

	if score < 0 {
		score = 0
	}

	grade := "F"
	switch {
	case score >= 90:
		grade = "A"
	case score >= 75:
		grade = "B"
	case score >= 60:
		grade = "C"
	case score >= 40:
		grade = "D"
	}

	return map[string]any{
		"score":   score,
		"grade":   grade,
		"reasons": reasons,
	}, nil
}

func (d *Detector) unresolvedSignatures(ctx context.Context, repoID *string) []map[string]any {
	var allSigs []map[string]any
	if repoID != nil {
		sigs, _ := d.store.GetFailureSignatures(ctx, *repoID)
		allSigs = sigs
	} else {
		repos, _ := d.store.ListReposDB(ctx)
		for _, r := range repos {
			sigs, _ := d.store.GetFailureSignatures(ctx, r.ID)
			allSigs = append(allSigs, sigs...)
		}
	}

	var unresolved []map[string]any
	for _, s := range allSigs {
		if resolved, ok := s["resolved"].(bool); ok && !resolved {
			unresolved = append(unresolved, s)
			if len(unresolved) >= 10 {
				break
			}
		}
	}
	if unresolved == nil {
		unresolved = []map[string]any{}
	}
	return unresolved
}

// ListMemoriesByKind is a helper to query typed memories.
func listByKind(ctx context.Context, store *sqlite.Store, repoID *string, kind string, limit int) []core.Memory {
	mems, _ := store.ListMemories(ctx, repoID, &kind, limit)
	return mems
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
