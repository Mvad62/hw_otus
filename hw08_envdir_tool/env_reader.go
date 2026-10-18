package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Environment map[string]EnvValue

// EnvValue помогает различать пустые файлы и файлы с первой пустой строкой.
type EnvValue struct {
	Value      string
	NeedRemove bool
}

// ReadDir читает указанный каталог и возвращает карту переменных окружения.
// Переменные представлены файлами, имя которых является именем переменной, а первая строка файла — её значением.
func ReadDir(dir string) (Environment, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}

	env := make(Environment, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Имя переменной не должно содержать "=".
		if strings.ContainsRune(name, '=') {
			continue
		}

		value, err := readEnvValue(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}

		env[name] = value
	}

	return env, nil
}

// readEnvValue читает первую строку файла и преобразует её в EnvValue.
// Пустой файл означает, что переменную следует удалить из окружения.
func readEnvValue(path string) (EnvValue, error) {
	f, err := os.Open(path)
	if err != nil {
		return EnvValue{}, fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	line, err := bufio.NewReader(f).ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return EnvValue{}, fmt.Errorf("read %q: %w", path, err)
	}

	// Пустой файл: нет байт вообще -> удалить переменную.
	if len(line) == 0 {
		return EnvValue{NeedRemove: true}, nil
	}

	// Удаляем завершающий перевод строки, если он есть: это разделитель строки, а не часть значения.
	line = bytes.TrimSuffix(line, []byte("\n"))
	line = bytes.TrimSuffix(line, []byte("\r"))

	// Терминальные нули (0x00) заменяются на переводы строки.
	line = bytes.ReplaceAll(line, []byte{0x00}, []byte("\n"))

	// Обрезаем пробелы и табы в конце строки.
	value := strings.TrimRight(string(line), " \t")

	return EnvValue{Value: value}, nil
}
