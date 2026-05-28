package pythonutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// ResolveExecutable returns a Python interpreter that can actually execute code.
// On Windows, command lookup often finds the Microsoft Store shim before a real
// interpreter, so every candidate is smoke-tested before it is accepted.
func ResolveExecutable(preferred string) (string, error) {
	if preferred != "" && !isGenericPythonCommand(preferred) {
		return resolveStrict(preferred)
	}

	candidates := pythonCandidates(preferred)
	seen := make(map[string]struct{}, len(candidates))
	var failures []string

	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}

		for _, resolved := range resolveCandidate(candidate) {
			key := strings.ToLower(resolved)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}

			if err := smokeTestPython(resolved); err != nil {
				failures = append(failures, fmt.Sprintf("%s (%v)", resolved, err))
				continue
			}
			return resolved, nil
		}
	}

	if len(failures) == 0 {
		return "", fmt.Errorf("no Python interpreter found")
	}
	return "", fmt.Errorf("no usable Python interpreter found; tried: %s", strings.Join(failures, "; "))
}

func resolveStrict(preferred string) (string, error) {
	var failures []string
	for _, resolved := range resolveCandidate(preferred) {
		if err := smokeTestPython(resolved); err != nil {
			failures = append(failures, fmt.Sprintf("%s (%v)", resolved, err))
			continue
		}
		return resolved, nil
	}
	if len(failures) == 0 {
		return "", fmt.Errorf("Python interpreter %q was not found", preferred)
	}
	return "", fmt.Errorf("Python interpreter %q is not usable; tried: %s", preferred, strings.Join(failures, "; "))
}

func isGenericPythonCommand(command string) bool {
	base := strings.ToLower(filepath.Base(command))
	base = strings.TrimSuffix(base, ".exe")
	return base == "python" || base == "python3"
}

func pythonCandidates(preferred string) []string {
	candidates := make([]string, 0, 8)
	if envPath := os.Getenv("PYTHON_PATH"); envPath != "" {
		candidates = append(candidates, envPath)
	}
	if preferred != "" {
		candidates = append(candidates, preferred)
	}

	if runtime.GOOS == "windows" {
		candidates = append(candidates, "python", "python3")
	} else {
		candidates = append(candidates, "python3", "python")
	}

	return candidates
}

func resolveCandidate(candidate string) []string {
	if filepath.IsAbs(candidate) {
		return []string{candidate}
	}

	paths := lookupAll(candidate)
	if len(paths) > 0 {
		return paths
	}

	if path, err := exec.LookPath(candidate); err == nil {
		return []string{path}
	}

	return nil
}

func lookupAll(name string) []string {
	if runtime.GOOS != "windows" {
		return nil
	}

	output, err := exec.Command("where.exe", name).Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	paths := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}
	sort.SliceStable(paths, func(i, j int) bool {
		return pythonPathPriority(paths[i]) < pythonPathPriority(paths[j])
	})
	return paths
}

func pythonPathPriority(path string) int {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, `\windowsapps\`):
		return 30
	case strings.Contains(lower, `\venv\`), strings.Contains(lower, `\.venv\`), strings.Contains(lower, `\virtualenv\`):
		return 20
	default:
		return 0
	}
}

func smokeTestPython(path string) error {
	if _, err := os.Stat(path); err != nil {
		if filepath.IsAbs(path) {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "-c", "import sys; print(sys.executable)")
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, text)
	}
	return nil
}
