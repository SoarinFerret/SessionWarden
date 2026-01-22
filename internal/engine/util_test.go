package engine

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvFromProc(t *testing.T) {
	// Test with current process
	pid := os.Getpid()

	// Try to read PATH which should exist in most environments
	value, err := getEnvFromProc(pid, "PATH")
	if !assert.NoError(t, err, "Should be able to read PATH from current process") {
		return
	}
	assert.NotEmpty(t, value, "PATH should not be empty")

	// Compare with os.Getenv to verify correctness
	expected := os.Getenv("PATH")
	assert.Equal(t, expected, value, "Value from /proc should match os.Getenv")
}

func TestGetEnvFromProc_NotFound(t *testing.T) {
	pid := os.Getpid()

	// Try to read a variable that doesn't exist
	_, err := getEnvFromProc(pid, "NONEXISTENT_VARIABLE_THAT_SHOULD_NOT_EXIST")
	assert.Error(t, err, "Should return error for non-existent variable")
	assert.Contains(t, err.Error(), "not found", "Error should mention variable not found")
}

func TestGetEnvFromProc_InvalidPID(t *testing.T) {
	// Try to read from a PID that doesn't exist
	_, err := getEnvFromProc(999999, "PATH")
	assert.Error(t, err, "Should return error for invalid PID")
}

func TestScanNullTerminated(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		atEOF    bool
		wantAdv  int
		wantTok  []byte
		wantErr  error
	}{
		{
			name:     "Single null-terminated string",
			input:    []byte("FOO=bar\x00"),
			atEOF:    false,
			wantAdv:  8,
			wantTok:  []byte("FOO=bar"),
			wantErr:  nil,
		},
		{
			name:     "Multiple null-terminated strings",
			input:    []byte("FOO=bar\x00BAZ=qux\x00"),
			atEOF:    false,
			wantAdv:  8,
			wantTok:  []byte("FOO=bar"),
			wantErr:  nil,
		},
		{
			name:     "EOF without null terminator",
			input:    []byte("FOO=bar"),
			atEOF:    true,
			wantAdv:  7,
			wantTok:  []byte("FOO=bar"),
			wantErr:  nil,
		},
		{
			name:     "EOF with empty input",
			input:    []byte{},
			atEOF:    true,
			wantAdv:  0,
			wantTok:  nil,
			wantErr:  nil,
		},
		{
			name:     "No null and not EOF",
			input:    []byte("FOO=bar"),
			atEOF:    false,
			wantAdv:  0,
			wantTok:  nil,
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adv, tok, err := scanNullTerminated(tt.input, tt.atEOF)
			assert.Equal(t, tt.wantAdv, adv, "advance should match")
			assert.Equal(t, tt.wantTok, tok, "token should match")
			assert.Equal(t, tt.wantErr, err, "error should match")
		})
	}
}

func TestScanNullTerminated_Integration(t *testing.T) {
	// Test that we can properly scan a realistic /proc/pid/environ-style string
	input := "USER=alice\x00HOME=/home/alice\x00PATH=/usr/bin:/bin\x00"

	var tokens []string
	data := []byte(input)
	offset := 0

	for offset < len(data) {
		adv, tok, _ := scanNullTerminated(data[offset:], offset >= len(data))
		if adv == 0 {
			break
		}
		if tok != nil {
			tokens = append(tokens, string(tok))
		}
		offset += adv
	}

	expected := []string{
		"USER=alice",
		"HOME=/home/alice",
		"PATH=/usr/bin:/bin",
	}

	assert.Equal(t, expected, tokens, "Should correctly parse all environment variables")
}
