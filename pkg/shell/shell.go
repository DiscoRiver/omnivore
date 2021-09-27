// Package Shell is a collection of shell utilities
package shell

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	ErrStdinPipeDoesNotExist = errors.New("stdin pipe does not exist")
)

// ReadStdinToSlice reads space-separated values from Stdin and returns a slice, or error on failure.
func ReadStdinToSlice() ([]string, error) {
	if !StdinPipeExists() {
		return nil, ErrStdinPipeDoesNotExist
	}

	reader := bufio.NewReader(os.Stdin)
	stdinContent, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("error reading stdin: %s", err)
	}

	return strings.Fields(stdinContent), nil
}

// StdinPipeExists returns true if a valid Stdin pipe exists, or false if not.
func StdinPipeExists() bool {
	stdinFile, _ := os.Stdin.Stat()
	if stdinFile.Mode()&os.ModeCharDevice != 0 {
		return false
	}

	return true
}
