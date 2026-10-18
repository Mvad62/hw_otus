package main

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunCmd(t *testing.T) {
	helper := []string{os.Args[0], "-test.run=TestHelperProcess", "--"}
	baseEnv := Environment{"GO_WANT_HELPER_PROCESS": {Value: "1"}}

	t.Run("exit code is propagated", func(t *testing.T) {
		env := Environment{
			"GO_WANT_HELPER_PROCESS": {Value: "1"},
			"HELPER_EXIT_CODE":       {Value: "42"},
		}
		require.Equal(t, 42, RunCmd(helper, env))
	})

	t.Run("zero exit code on success", func(t *testing.T) {
		require.Equal(t, 0, RunCmd(helper, baseEnv))
	})

	t.Run("sets and removes variables", func(t *testing.T) {
		require.NoError(t, os.Setenv("TO_REMOVE", "present"))
		t.Cleanup(func() { _ = os.Unsetenv("TO_REMOVE") })

		env := Environment{
			"GO_WANT_HELPER_PROCESS": {Value: "1"},
			"HELPER_CHECK_SET":       {Value: "expected"},
			"HELPER_CHECK_REMOVED":   {Value: "1"},
			"TO_REMOVE":              {NeedRemove: true},
		}
		// Вспомогательный процесс завершается успешно только тогда, когда TO_REMOVE отсутствует и TO_SET совпадает.
		env["TO_SET"] = EnvValue{Value: "expected"}
		require.Equal(t, 0, RunCmd(helper, env))
	})

	t.Run("missing binary returns non-zero", func(t *testing.T) {
		require.NotEqual(t, 0, RunCmd([]string{"definitely-not-a-real-binary-xyz"}, baseEnv))
	})

	t.Run("empty cmd returns non-zero", func(t *testing.T) {
		require.NotEqual(t, 0, RunCmd(nil, baseEnv))
	})
}

// TestHelperProcess не является реальным тестом. Он запускается как дочерний процесс
// из RunCmd во время TestRunCmd и управляется переменными окружения.
func TestHelperProcess(_ *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	if os.Getenv("HELPER_CHECK_REMOVED") == "1" {
		if _, ok := os.LookupEnv("TO_REMOVE"); ok {
			os.Exit(10)
		}
	}

	if os.Getenv("HELPER_CHECK_SET") != "" {
		if os.Getenv("TO_SET") != os.Getenv("HELPER_CHECK_SET") {
			os.Exit(11)
		}
	}

	code := 0
	if v := os.Getenv("HELPER_EXIT_CODE"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			code = parsed
		}
	}

	os.Exit(code)
}
