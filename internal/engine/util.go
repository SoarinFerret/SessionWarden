package engine

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// getEnvFromProc reads an environment variable from /proc/<pid>/environ
func getEnvFromProc(pid int, envVar string) (string, error) {
	path := fmt.Sprintf("/proc/%d/environ", pid)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", path, err)
	}

	// environ file contains null-separated key=value pairs
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Split(scanNullTerminated)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, envVar+"=") {
			return strings.TrimPrefix(line, envVar+"="), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning environ: %w", err)
	}

	return "", fmt.Errorf("environment variable %s not found", envVar)
}

// scanNullTerminated is a split function for bufio.Scanner that splits on null bytes
func scanNullTerminated(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Find the next null byte
	if i := strings.IndexByte(string(data), 0); i >= 0 {
		return i + 1, data[0:i], nil
	}

	// If we're at EOF, return what we have
	if atEOF {
		return len(data), data, nil
	}

	// Request more data
	return 0, nil, nil
}
