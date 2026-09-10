package hw05parallelexecution

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
)

// ErrErrorsLimitExceeded возвращается, когда количество ошибок превысило лимит m.
var ErrErrorsLimitExceeded = errors.New("errors limit exceeded")

// Task — функция одной задачи.
type Task func() error

// Run выполняет задачи в n параллельных горутинах.
// Останавливается при достижении m ошибок.
// Если n <= 0, не выполняет задачи и возвращает nil.
// Если m <= 0, ошибки игнорируются.
func Run(tasks []Task, n, m int) error {
	if n <= 0 {
		return nil
	}

	ignoreErrors := m <= 0
	if !ignoreErrors && m > math.MaxInt32 {
		m = math.MaxInt32
	}
	errLimit := int32(m) //nolint:gosec // m > 0 && m <= math.MaxInt32

	tasksCh := make(chan Task)
	done := make(chan struct{})
	var (
		wg       sync.WaitGroup
		errCount int32
		once     sync.Once
	)

	startWorkers(&wg, n, tasksCh, done, &errCount, errLimit, ignoreErrors, &once)

	sendTasks(tasks, tasksCh, done)

	close(tasksCh)
	wg.Wait()

	if !ignoreErrors && atomic.LoadInt32(&errCount) >= errLimit {
		return ErrErrorsLimitExceeded
	}
	return nil
}

func startWorkers(
	wg *sync.WaitGroup,
	n int,
	tasksCh <-chan Task,
	done chan struct{},
	errCount *int32,
	errLimit int32,
	ignoreErrors bool,
	once *sync.Once,
) {
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(tasksCh, done, errCount, errLimit, ignoreErrors, once)
		}()
	}
}

func worker(
	tasksCh <-chan Task,
	done chan struct{}, // было <-chan, из-за этого close не компилировался
	errCount *int32,
	errLimit int32,
	ignoreErrors bool,
	once *sync.Once,
) {
	for {
		select {
		case task, ok := <-tasksCh:
			if !ok {
				return
			}
			if err := task(); err != nil && !ignoreErrors {
				if atomic.AddInt32(errCount, 1) >= errLimit {
					once.Do(func() { close(done) })
				}
			}
		case <-done:
			return
		}
	}
}

func sendTasks(tasks []Task, tasksCh chan<- Task, done <-chan struct{}) {
sendLoop:
	for _, task := range tasks {
		select {
		case tasksCh <- task:
		case <-done:
			break sendLoop
		}
	}
}
