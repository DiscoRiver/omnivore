package shell

import (
	"os"
	"testing"
)

var shellTestEnv = "OMNI_TEST"

func TestRunCommand(t *testing.T) {
	cmd := "echo"
	arg1 := "hello"
	arg2 := "world"

	output, err := RunCommand(cmd, arg1, arg2)
	if err != nil {
		t.Logf("RunCommand error: %s", err)
		t.FailNow()
	}

	expected := "hello world\n"
	if string(output) != expected {
		t.Logf("Expected %s, got %s", expected, string(output))
		t.Fail()
	}
}

func TestGetenv(t *testing.T) {
	err := os.Setenv("OMNI_TEST", "some_value")
	if err != nil {
		t.Logf("couldn't set environment variable: %s", err)
		t.FailNow()
	}

	_, err = Getenv(shellTestEnv)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}
