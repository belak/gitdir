package main

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

func expandGroups(groups map[string][]string, users []string) []string {
	out := []string{}

	for _, user := range users {
		if strings.HasPrefix(user, "$") {
			out = append(out, groups[user[1:]]...)
		} else {
			out = append(out, user)
		}
	}

	return sliceUniqMap(out)
}

// groupMembers recursively finds all members of the given group
func groupMembers(groups map[string][]string, groupName string, groupPath []string) ([]string, error) {
	out := []string{}

	if listContains(groupPath, groupName) {
		return nil, fmt.Errorf("Found group loop: %s", strings.Join(groupPath, ", "))
	}

	groupPath = append(groupPath, groupName)

	for _, user := range groups[groupName] {
		if strings.HasPrefix(user, "$") {
			nested, err := groupMembers(groups, user[1:], groupPath)
			if err != nil {
				return nil, err
			}

			out = append(out, nested...)
		} else {
			out = append(out, user)
		}
	}

	// Ensure we're always returning the smallest version of this list that we
	// can.
	return sliceUniqMap(out), nil
}

func sliceUniqMap(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}

func listContains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}

	return false
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

func runCommand(log *zerolog.Logger, session ssh.Session, args []string) int {
	cmd := exec.Command(args[0], args[1:]...)
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
		if _, err := io.Copy(stdin, session); err != nil {
			log.Error().Err(err).Msg("Failed to write session to stdin")
		}
	}()

	go func() {
		defer wg.Done()
		if _, err := io.Copy(session, stdout); err != nil {
			log.Error().Err(err).Msg("Failed to write stdout to session")
		}
	}()

	go func() {
		defer wg.Done()
		if _, err := io.Copy(session.Stderr(), stderr); err != nil {
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
