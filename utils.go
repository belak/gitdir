package gitdir

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog"
)

type multiError struct {
	errors []error
}

func newMultiError(errors ...error) error {
	ret := &multiError{}

	for _, err := range errors {
		if err != nil {
			ret.errors = append(ret.errors, err)
		}
	}

	if len(ret.errors) == 0 {
		return nil
	}

	return ret
}

func (ce *multiError) Error() string {
	buf := bytes.NewBuffer(nil)

	for _, err := range ce.errors {
		buf.WriteString("- ")
		buf.WriteString(err.Error())
		buf.WriteString("\n")
	}

	return buf.String()
}

func handlePanic(logger *zerolog.Logger) {
	if r := recover(); r != nil {
		logger.Error().Err(fmt.Errorf("%s", r)).Msg("Caught panic")
	}
}

func writeStringFmt(w io.Writer, format string, args ...interface{}) error {
	_, err := io.WriteString(w, fmt.Sprintf(format, args...))
	return err
}

func getExitStatusFromError(err error) int {
	if err == nil {
		return 0
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return 1
	}

	waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		// This is a fallback and should at least let us return something useful
		// when running on Windows, even if it isn't completely accurate.
		if exitErr.Success() {
			return 0
		}

		return 1
	}

	return waitStatus.ExitStatus()
}

func sanitize(in string) string {
	// TODO: this should do more
	return strings.ToLower(in)
}

// TODO: see if this can be cleaned up
func runCommand( //nolint:funlen
	log *zerolog.Logger,
	cwd string,
	session ssh.Session,
	args []string,
	environ []string,
) int {
	// NOTE: we are explicitly ignoring gosec here because we *only* pass in
	// known commands here.
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec
	cmd.Dir = cwd

	cmd.Env = append(cmd.Env, environ...)
	cmd.Env = append(cmd.Env, "PATH="+os.Getenv("PATH"))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get stdin pipe")
		return 1
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get stdout pipe")
		return 1
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get stderr pipe")
		return 1
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	if err = cmd.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start command")
		return 1
	}

	go func() {
		defer stdin.Close()
		if _, stdinErr := io.Copy(stdin, session); stdinErr != nil {
			log.Error().Err(err).Msg("Failed to write session to stdin")
		}
	}()

	go func() {
		defer wg.Done()
		if _, stdoutErr := io.Copy(session, stdout); stdoutErr != nil {
			log.Error().Err(err).Msg("Failed to write stdout to session")
		}
	}()

	go func() {
		defer wg.Done()
		if _, stderrErr := io.Copy(session.Stderr(), stderr); stderrErr != nil {
			log.Error().Err(err).Msg("Failed to write stderr to session")
		}
	}()

	// Ensure all the output has been written before we wait on the command to
	// exit.
	wg.Wait()

	// Wait for the command to exit and log any errors we get
	err = cmd.Wait()
	if err != nil {
		log.Error().Err(err).Msg("Failed to wait for command exit")
	}

	return getExitStatusFromError(err)
}
