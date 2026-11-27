package hw10programoptimization

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"runtime"
	"strings"
	"sync"

	json "github.com/goccy/go-json"
)

type User struct {
	Email string
}

type DomainStat map[string]int

const batchBytes = 512 * 1024

func GetDomainStat(r io.Reader, domain string) (DomainStat, error) {
	suffix := "." + strings.ToLower(domain)

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan []byte, workers)
	partials := make([]DomainStat, workers)
	errs := make([]error, workers)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			partials[w], errs[w] = countBatches(jobs, suffix)
		}(w)
	}

	scanErr := produceBatches(r, jobs)
	close(jobs)
	wg.Wait()

	if scanErr != nil {
		return nil, scanErr
	}

	result := make(DomainStat)
	for w := 0; w < workers; w++ {
		if errs[w] != nil {
			return nil, errs[w]
		}
		for dom, n := range partials[w] {
			result[dom] += n
		}
	}

	return result, nil
}

// produceBatches reads r line by line and sends newline-delimited byte batches
// to jobs, so each batch holds whole JSON objects a worker can decode.
func produceBatches(r io.Reader, jobs chan<- []byte) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	batch := make([]byte, 0, batchBytes+64*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		batch = append(batch, line...)
		batch = append(batch, '\n')
		if len(batch) >= batchBytes {
			jobs <- batch
			batch = make([]byte, 0, batchBytes+64*1024)
		}
	}
	if len(batch) > 0 {
		jobs <- batch
	}

	return scanner.Err()
}

// countBatches drains every batch from jobs (even after an error, to keep the
// producer unblocked) and returns this worker's stat and the first parse error.
func countBatches(jobs <-chan []byte, suffix string) (DomainStat, error) {
	stat := make(DomainStat)
	var firstErr error
	var user User

	for batch := range jobs {
		if firstErr != nil {
			continue
		}

		dec := json.NewDecoder(bytes.NewReader(batch))
		for {
			user.Email = ""
			if err := dec.Decode(&user); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				firstErr = err
				break
			}

			at := strings.LastIndexByte(user.Email, '@')
			if at < 0 {
				continue
			}

			dom := user.Email[at+1:]
			if len(dom) >= len(suffix) && strings.EqualFold(dom[len(dom)-len(suffix):], suffix) {
				stat[strings.ToLower(dom)]++
			}
		}
	}

	return stat, firstErr
}
