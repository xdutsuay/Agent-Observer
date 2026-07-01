package projects

import (
	"os"
	"path/filepath"
)

// FindProjectRoot searches upwards from the given path to find the nearest .git directory.
// If not found, it returns the original absolute path.
func FindProjectRoot(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return startPath, err
	}

	current := absPath
	for {
		gitDir := filepath.Join(current, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		// Reached the root of the filesystem
		if parent == current {
			break
		}
		current = parent
	}

	// Default fallback if no .git found
	return absPath, nil
}
