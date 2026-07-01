package sqlite

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var entryRe = regexp.MustCompile(`(?m)^### (.+)$`)

func RunLegacyMigration(ctx context.Context, s *Store, root string) error {
	markerPath := filepath.Join(root, ".migrated_to_sqlite")
	if _, err := os.Stat(markerPath); err == nil {
		return nil // Already migrated
	}

	memoryDir := filepath.Join(root, "agent-memory")
	if _, err := os.Stat(memoryDir); os.IsNotExist(err) {
		return nil // Nothing to migrate
	}

	reposMapPath := filepath.Join(root, "repos.json")
	reposMap := make(map[string]string)
	if b, err := ioutil.ReadFile(reposMapPath); err == nil {
		json.Unmarshal(b, &reposMap)
	}

	entries, err := ioutil.ReadDir(memoryDir)
	if err != nil {
		return err
	}

	imported := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		repoID := entry.Name()
		memPath := filepath.Join(memoryDir, repoID, "memory")
		if _, err := os.Stat(memPath); os.IsNotExist(err) {
			continue
		}

		path, ok := reposMap[repoID]
		if !ok {
			path = filepath.Join(memoryDir, repoID)
		}
		s.UpsertRepo(ctx, repoID, path)

		kinds := map[string]string{
			"failures.md":  "failure",
			"decisions.md": "decision",
			"attempts.md":  "attempt",
		}

		for file, kind := range kinds {
			fpath := filepath.Join(memPath, file)
			b, err := ioutil.ReadFile(fpath)
			if err != nil {
				continue
			}

			blocks := splitMarkdownEntries(string(b))
			for _, block := range blocks {
				if strings.TrimSpace(block.Body) == "" {
					continue
				}
				meta := map[string]any{"legacy_timestamp": block.Timestamp}
				_, inserted, _ := s.InsertMemory(ctx, repoID, kind, strings.TrimSpace(block.Body), "import", meta, true)
				if inserted {
					imported++
				}
			}
		}

		statePath := filepath.Join(memPath, "state.json")
		if b, err := ioutil.ReadFile(statePath); err == nil {
			var state map[string]any
			if json.Unmarshal(b, &state) == nil {
				if fa, ok := state["failure_analytics"].(map[string]any); ok {
					for sig, dataAny := range fa {
						if data, ok := dataAny.(map[string]any); ok {
							count := 1
							if c, ok := data["count"].(float64); ok { count = int(c) }
							fs := ""
							if f, ok := data["first_seen"].(string); ok { fs = f }
							ls := ""
							if l, ok := data["last_seen"].(string); ok { ls = l }
							res := 0
							if r, ok := data["resolved"].(bool); ok && r { res = 1 }

							s.memoryDB.ExecContext(ctx, `
								INSERT OR IGNORE INTO failure_signatures
								(repo_id, signature, count, first_seen, last_seen, resolved, memory_id)
								VALUES (?, ?, ?, ?, ?, ?, NULL)
							`, repoID, sig, count, fs, ls, res)
						}
					}
				}
			}
		}
	}

	if imported > 0 || len(entries) > 0 {
		ioutil.WriteFile(markerPath, []byte("ok"), 0644)
	}

	return nil
}

type mdBlock struct {
	Timestamp string
	Body      string
}

func splitMarkdownEntries(text string) []mdBlock {
	var parts []mdBlock
	matches := entryRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		if strings.TrimSpace(text) != "" {
			parts = append(parts, mdBlock{Timestamp: "", Body: text})
		}
		return parts
	}

	for i, m := range matches {
		ts := text[m[2]:m[3]]
		start := m[1]
		end := len(text)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		parts = append(parts, mdBlock{
			Timestamp: ts,
			Body:      text[start:end],
		})
	}
	return parts
}
