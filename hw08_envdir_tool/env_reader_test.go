package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadDir(t *testing.T) {
	t.Run("reference testdata", func(t *testing.T) {
		env, err := ReadDir("testdata/env")
		require.NoError(t, err)

		require.Equal(t, EnvValue{Value: `"hello"`}, env["HELLO"])
		require.Equal(t, EnvValue{Value: "bar"}, env["BAR"])
		require.Equal(t, EnvValue{Value: "   foo\nwith new line"}, env["FOO"])
		require.Equal(t, EnvValue{Value: ""}, env["EMPTY"])
		require.Equal(t, EnvValue{NeedRemove: true}, env["UNSET"])
	})

	t.Run("edge cases", func(t *testing.T) {
		dir := t.TempDir()

		writeFile(t, dir, "TRIMMED", "value  \t ")
		writeFile(t, dir, "WITH_NULL", "a\x00b")
		writeFile(t, dir, "SECOND_LINE", "first\nsecond")
		writeFile(t, dir, "EMPTYLINE", "\nignored")
		writeFile(t, dir, "ZERO", "")
		writeFile(t, dir, "BAD=NAME", "should be ignored")

		// Вложенная директория должна игнорироваться.
		require.NoError(t, os.Mkdir(filepath.Join(dir, "nested"), 0o755))

		env, err := ReadDir(dir)
		require.NoError(t, err)

		require.Equal(t, EnvValue{Value: "value"}, env["TRIMMED"])
		require.Equal(t, EnvValue{Value: "a\nb"}, env["WITH_NULL"])
		require.Equal(t, EnvValue{Value: "first"}, env["SECOND_LINE"])
		require.Equal(t, EnvValue{Value: ""}, env["EMPTYLINE"])
		require.Equal(t, EnvValue{NeedRemove: true}, env["ZERO"])

		_, ok := env["BAD=NAME"]
		require.False(t, ok, "имена, содержащие '=' должны пропускаться")
		_, ok = env["nested"]
		require.False(t, ok, "директории должны пропускаться")
	})

	t.Run("missing dir returns error", func(t *testing.T) {
		_, err := ReadDir(filepath.Join(t.TempDir(), "does-not-exist"))
		require.Error(t, err)
	})
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}
