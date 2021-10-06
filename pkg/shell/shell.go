// Package Shell is a collection of shell utilities
package shell

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	ErrStdinPipeDoesNotExist = errors.New("stdin pipe does not exist")
	ErrEnvironmentVariableNotSet = errors.New("environment variable not set")
	ErrCouldNotSetEnvironmentVariable = errors.New("couldn't set environment variable")
)

func RunCommand(cmd string, args ...string) ([]byte, error) {
	c := exec.Command(cmd, args...)
	stdout, err := c.Output()
	if err != nil {
		return nil, err
	}
	return stdout, nil
}

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

func Getenv(key string) (string, error) {
	env := os.Getenv(key)
	if env == "" {
		return "", ErrEnvironmentVariableNotSet
	}

	return env, nil
}
