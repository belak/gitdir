package gitdir

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog"
)

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
func runCommand(log *zerolog.Logger, session ssh.Session, args []string) int { //nolint:funlen
	// NOTE: we are explicitly ignoring gosec here because we *only* pass in
	// known commands here.
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec

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
