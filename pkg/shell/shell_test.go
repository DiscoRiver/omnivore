package shell

import "testing"

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
	testEnvVar := "TERM"

	_, err := Getenv(testEnvVar)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}
