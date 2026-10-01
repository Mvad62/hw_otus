package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const (
	testdataDir = "testdata"
	inputFile   = "input.txt"
)

// inputPath возвращает путь к единственному входному файлу.
func inputPath() string {
	return filepath.Join(testdataDir, inputFile)
}

// readInput читает входной файл целиком.
func readInput(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(inputPath())
	if err != nil {
		t.Fatalf("read input: %v", err)
	}
	return data
}

// slice ожидает, что диапазон [offset, offset+limit) существует в data.
// Если limit == 0, берётся всё до конца.
func slice(t *testing.T, data []byte, offset, limit int64) []byte {
	t.Helper()
	if offset < 0 || offset > int64(len(data)) {
		t.Fatalf("bad offset %d for data len %d", offset, len(data))
	}
	end := int64(len(data))
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return data[offset:end]
}

// tmpOut возвращает путь к выходному файлу во временном каталоге теста.
// t.TempDir() сам удаляет каталог после теста.
func tmpOut(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), name)
}

// checkCopy копирует inputPath() в dst с заданными offset/limit
// и сверяет результат с want.
func checkCopy(t *testing.T, dst string, offset, limit int64, want []byte) {
	t.Helper()
	if err := Copy(inputPath(), dst, offset, limit); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got %d bytes, want %d bytes", len(got), len(want))
		// Для отладки — первые расхождения.
		n := len(got)
		if len(want) < n {
			n = len(want)
		}
		for i := 0; i < n; i++ {
			if got[i] != want[i] {
				t.Errorf("first mismatch at %d: got %q, want %q", i, got[i], want[i])
				break
			}
		}
	}
}

func TestCopyFull(t *testing.T) {
	dst := tmpOut(t, "copy.txt")
	checkCopy(t, dst, 0, 0, readInput(t))
}

func TestCopyOffset(t *testing.T) {
	dst := tmpOut(t, "off.txt")
	checkCopy(t, dst, 3, 0, slice(t, readInput(t), 3, 0))
}

func TestCopyLimit(t *testing.T) {
	dst := tmpOut(t, "lim.txt")
	checkCopy(t, dst, 2, 4, slice(t, readInput(t), 2, 4))
}

func TestCopyOffsetPlusLimit(t *testing.T) {
	dst := tmpOut(t, "offlim.txt")
	checkCopy(t, dst, 10, 3, slice(t, readInput(t), 10, 3))
}

func TestCopyLimitExceedsFileSize(t *testing.T) {
	dst := tmpOut(t, "biglimit.txt")
	// limit > размера файла → копируем весь файл до EOF.
	checkCopy(t, dst, 0, 1<<20, readInput(t))
}

func TestCopyOffsetEqualsFileSize(t *testing.T) {
	dst := tmpOut(t, "empty.txt")
	info, err := os.Stat(inputPath())
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	checkCopy(t, dst, info.Size(), 0, nil)
}

func TestCopyOffsetExceedsFileSize(t *testing.T) {
	dst := tmpOut(t, "bad.txt")
	err := Copy(inputPath(), dst, 1<<20, 0)
	if !errors.Is(err, ErrOffsetExceedsFileSize) {
		t.Fatalf("err = %v, want ErrOffsetExceedsFileSize", err)
	}
}

func TestCopyNegativeOffset(t *testing.T) {
	dst := tmpOut(t, "neg.txt")
	if err := Copy(inputPath(), dst, -1, 0); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCopyNegativeLimit(t *testing.T) {
	dst := tmpOut(t, "negl.txt")
	if err := Copy(inputPath(), dst, 0, -1); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCopySourceNotFound(t *testing.T) {
	dst := tmpOut(t, "dst.txt")
	if err := Copy(filepath.Join(testdataDir, "missing.bin"), dst, 0, 0); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCopyNonRegularSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no /dev/null on Windows")
	}
	dst := tmpOut(t, "devnull.txt")
	err := Copy("/dev/null", dst, 0, 0)
	if !errors.Is(err, ErrUnsupportedFile) {
		t.Fatalf("err = %v, want ErrUnsupportedFile", err)
	}
}

func TestCopySameFile(t *testing.T) {
	err := Copy(inputPath(), inputPath(), 0, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCopyToTempFile(t *testing.T) {
	tmp, err := os.CreateTemp("", "ddclone-test-*")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(tmp.Name())
	tmp.Close()

	if err := Copy(inputPath(), tmp.Name(), 0, 0); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	got, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, readInput(t)) {
		t.Errorf("dst != src")
	}
}
