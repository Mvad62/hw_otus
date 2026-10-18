package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrUnsupportedFile       = errors.New("unsupported file")
	ErrOffsetExceedsFileSize = errors.New("offset exceeds file size")
)

type progressWriter struct {
	writer      io.Writer
	total       int64
	written     int64
	lastPercent int64
}

func (p *progressWriter) Write(data []byte) (int, error) {
	n, err := p.writer.Write(data)
	p.written += int64(n)
	if p.total > 0 {
		percent := p.written * 100 / p.total
		if percent != p.lastPercent || p.written == p.total {
			barWidth := int64(20)
			filled := percent * barWidth / 100
			bar := "[" + strings.Repeat("=", int(filled)) + strings.Repeat(" ", int(barWidth-filled)) + "]"
			if _, progressErr := fmt.Fprintf(os.Stdout, "\r%s %3d%%", bar, percent); err == nil {
				err = progressErr
			}
			p.lastPercent = percent
		}
	}
	return n, err
}

// Copy копирует из fromPath в toPath до limit байт, начиная со смещения offset.
//
// Правила:
//   - offset < 0 или limit < 0 — ошибка аргументов;
//   - offset > размер(fromPath) — ErrOffsetExceedsFileSize;
//   - limit == 0 или limit > (размер - offset) — копируем до EOF;
//   - fromPath должен быть обычным файлом (иначе ErrUnsupportedFile).
func Copy(fromPath, toPath string, offset, limit int64) (err error) {
	if offset < 0 {
		return errors.New("offset must be non-negative")
	}
	if limit < 0 {
		return errors.New("limit must be non-negative")
	}

	srcInfo, err := os.Stat(fromPath)
	if err != nil {
		return err
	}
	// Файлы с неизвестной длиной (/dev/urandom, пайпы, сокеты) не поддерживаем.
	if !srcInfo.Mode().IsRegular() {
		return ErrUnsupportedFile
	}

	// 🛡️ Защита от случайного уничтожения файла при копировании "самого в себя"
	if dstInfo, statErr := os.Stat(toPath); statErr == nil {
		if os.SameFile(srcInfo, dstInfo) {
			return errors.New("source and destination are the same file")
		}
	}

	size := srcInfo.Size()
	if offset > size {
		return ErrOffsetExceedsFileSize
	}

	remaining := size - offset
	toCopy := remaining
	if limit > 0 && limit < remaining {
		toCopy = limit
	}

	src, err := os.Open(fromPath)
	if err != nil {
		return err
	}
	defer src.Close()

	if offset > 0 {
		if _, err := src.Seek(offset, io.SeekStart); err != nil {
			return err
		}
	}

	// 🛡️ Создание родительских директорий, если их не существует
	if err := os.MkdirAll(filepath.Dir(toPath), 0o755); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	dst, err := os.Create(toPath)
	if err != nil {
		return err
	}

	// 🛡️ Именованный возврат для отложенной очистки "битого" файла при ошибке
	defer func() {
		if cerr := dst.Close(); cerr != nil && err == nil {
			err = cerr
		}
		if err != nil {
			_ = os.Remove(toPath)
		}
	}()

	progress := &progressWriter{
		writer: dst,
		total:  toCopy,
	}
	_, err = io.CopyN(progress, src, toCopy)

	// ⚠️ По ТЗ: если источник закончился раньше, EOF считаем нормальным.
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("copy failed: %w", err)
	}

	if toCopy == 0 {
		_, err = fmt.Fprint(os.Stdout, "\r100%")
		if err != nil {
			return err
		}
	}

	if _, err = fmt.Fprintln(os.Stdout); err != nil {
		return err
	}

	return dst.Sync()
}
