package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

const exitCodeError = 1

// RunCmd запускает команду с аргументами (cmd) с окружением из env.
func RunCmd(cmd []string, env Environment) (returnCode int) {
	if len(cmd) == 0 {
		return exitCodeError
	}

	c := exec.Command(cmd[0], cmd[1:]...) //nolint:gosec
	c.Env = buildEnv(env)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		return exitCodeError
	}

	return c.ProcessState.ExitCode()
}

// buildEnv объединяет текущее окружение процесса с env:
// записи с NeedRemove удаляются, остальные задаются или перезаписываются.
func buildEnv(env Environment) []string {
	current := make(map[string]string, len(os.Environ()))
	for _, kv := range os.Environ() {
		key, value, found := strings.Cut(kv, "=")
		if !found {
			continue
		}
		current[key] = value
	}

	for name, value := range env {
		if value.NeedRemove {
			delete(current, name)
			continue
		}
		current[name] = value.Value
	}

	result := make([]string, 0, len(current))
	for key, value := range current {
		result = append(result, key+"="+value)
	}

	return result
}
