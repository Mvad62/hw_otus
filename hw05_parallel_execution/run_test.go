package hw05parallelexecution

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestRun(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("if were errors in first M tasks, than finished not more N+M tasks", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32

		for i := 0; i < tasksCount; i++ {
			err := fmt.Errorf("error from task %d", i)
			tasks = append(tasks, func() error {
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
				atomic.AddInt32(&runTasksCount, 1)
				return err
			})
		}

		workersCount := 10
		maxErrorsCount := 23
		err := Run(tasks, workersCount, maxErrorsCount)

		require.Truef(t, errors.Is(err, ErrErrorsLimitExceeded), "actual err - %v", err)
		require.LessOrEqual(t, runTasksCount, int32(workersCount+maxErrorsCount), "extra tasks were started")
	})

	t.Run("tasks without errors", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32
		var sumTime time.Duration

		for i := 0; i < tasksCount; i++ {
			taskSleep := time.Millisecond * time.Duration(rand.Intn(100))
			sumTime += taskSleep

			tasks = append(tasks, func() error {
				time.Sleep(taskSleep)
				atomic.AddInt32(&runTasksCount, 1)
				return nil
			})
		}

		workersCount := 5
		maxErrorsCount := 1

		start := time.Now()
		err := Run(tasks, workersCount, maxErrorsCount)
		elapsedTime := time.Since(start)
		require.NoError(t, err)

		require.Equal(t, int32(tasksCount), runTasksCount, "not all tasks were completed")
		require.LessOrEqual(t, int64(elapsedTime), int64(sumTime/2), "tasks were run sequentially?")
	})

	t.Run("ignore errors when m <= 0", func(t *testing.T) {
		tasksCount := 10
		tasks := make([]Task, tasksCount)
		var runTasksCount int32

		for i := 0; i < tasksCount; i++ {
			tasks[i] = func() error {
				atomic.AddInt32(&runTasksCount, 1)
				return errors.New("some error")
			}
		}

		// m = 0, ожидаем, что ошибки будут проигнорированы и выполнятся все задачи
		err := Run(tasks, 5, 0)
		require.NoError(t, err)
		require.Equal(t, int32(tasksCount), runTasksCount, "not all tasks were completed when m <= 0")
	})

	t.Run("concurrency without sleep", func(t *testing.T) {
		defer goleak.VerifyNone(t)

		const (
			workers    = 4
			tasksCount = 10
		)

		var (
			running    int32 // текущее количество выполняемых задач
			maxRunning int32 // максимальное зафиксированное количество одновременно выполняемых задач
			release    = make(chan struct{})
			wg         sync.WaitGroup
		)

		tasks := make([]Task, tasksCount)
		for i := range tasks {
			tasks[i] = func() error {
				cur := atomic.AddInt32(&running, 1)

				// Обновляем максимум при необходимости
				for {
					oldMax := atomic.LoadInt32(&maxRunning)
					if cur <= oldMax || atomic.CompareAndSwapInt32(&maxRunning, oldMax, cur) {
						break
					}
				}

				<-release
				atomic.AddInt32(&running, -1)
				return nil
			}
		}

		var runErr error
		wg.Add(1)
		go func() {
			defer wg.Done()
			runErr = Run(tasks, workers, 1)
		}()

		// Ждём, пока все воркеры не начнут выполнение (running == workers)
		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&running) == workers
		}, time.Second, 10*time.Millisecond, "должны запуститься все воркеры")

		// Проверяем, что одновременно выполнялось не больше workers задач
		require.Equal(t, int32(workers), atomic.LoadInt32(&maxRunning))

		// Разрешаем задачам завершиться
		close(release)

		// Дожидаемся окончания работы Run
		wg.Wait()

		require.NoError(t, runErr)
		require.Equal(t, int32(0), atomic.LoadInt32(&running))
	})
}
