package gitdir

import (
	"bytes"
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMultiError(t *testing.T) {
	t.Parallel()

	err := newMultiError()
	assert.Nil(t, err)

	err = newMultiError(nil)
	assert.Nil(t, err)

	err = newMultiError(
		errors.New("test error please ignore"),
	)
	assert.NotNil(t, err)
	assert.Equal(t, "- test error please ignore\n", err.Error())

	err = newMultiError(
		nil,
		errors.New("test error please ignore"),
		nil,
	)
	assert.NotNil(t, err)
	assert.Equal(t, "- test error please ignore\n", err.Error())

	err = newMultiError(
		nil,
		errors.New("test error please ignore"),
		nil,
		errors.New("test error please ignore as well"),
	)
	assert.NotNil(t, err)
	assert.Equal(t, "- test error please ignore\n- test error please ignore as well\n", err.Error())
}

func TestListContainsStr(t *testing.T) {
	t.Skip("not implemented")

	t.Parallel()
}

func TestHandlePanic(t *testing.T) {
	t.Skip("not implemented")

	t.Parallel()
}

func TestWriteStringFmt(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)

	err := writeStringFmt(buf, "hello %s", "world")
	assert.Nil(t, err)
	assert.Equal(t, "hello world", buf.String())
}

func TestGetExitStatusFromError(t *testing.T) {
	t.Parallel()

	// It is way harder than it should be to mock out an ExitError, so we just
	// run a command we know will return a valid ExitError.
	cmd := exec.Command("sh", "-c", "exit 10")
	cmdErr := cmd.Run()

	var tests = []struct {
		Input    error
		Expected int
	}{
		{
			nil,
			0,
		},
		{
			errors.New("non ExitError"),
			1,
		},
		{
			cmdErr,
			10,
		},
	}

	for _, test := range tests {
		output := getExitStatusFromError(test.Input)
		assert.Equal(t, test.Expected, output)
	}
}

func TestSanitize(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		Input    string
		Expected string
	}{
		{"", ""},
		{"HELLO-WORLD", "hello-world"},
	}

	for _, test := range tests {
		output := sanitize(test.Input)
		assert.Equal(t, test.Expected, output)
	}
}

func TestRunCommand(t *testing.T) {
	t.Skip("not implemented")

	t.Parallel()
}
