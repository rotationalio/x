package radish_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
	"go.rtnl.ai/x/radish"
)

func TestMain(m *testing.M) {
	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestTasks(t *testing.T) {
	// NOTE: ensure the queue size is zero so that queueing blocks until all tasks are
	// queued to prevent a race condition with the call to stop.
	tm, err := radish.New(radish.WithWorkers(4), radish.WithQueueSize(0))
	assert.Nil(t, err, "expected to be able to create a task manager")
	tm.Start()

	// Should be able to call start twice without panic
	tm.Start()

	// Queue basic tasks with no retries
	var completed int32
	for i := 0; i < 100; i++ {
		tm.Queue(radish.TaskFunc(func(context.Context) error {
			time.Sleep(1 * time.Millisecond)
			atomic.AddInt32(&completed, 1)
			return nil
		}))
	}

	assert.True(t, tm.IsRunning())
	tm.Stop()

	// Should be able to call stop twice without panic
	tm.Stop()

	assert.Equal(t, int32(100), completed)
	assert.False(t, tm.IsRunning())

	// Should not be able to queue when the task manager is stopped
	err = tm.Queue(radish.TaskFunc(func(context.Context) error { return nil }))
	assert.ErrorIs(t, err, radish.ErrTaskManagerStopped)
}

func TestQueueContext(t *testing.T) {
	// NOTE: ensure the queue size is zero so that queueing blocks until all tasks are
	// queued to prevent a race condition with the call to stop.
	tm, err := radish.New(radish.WithWorkers(4), radish.WithQueueSize(0))
	assert.Nil(t, err, "expected to be able to create a task manager")
	tm.Start()

	// Should be able to call start twice without panic
	tm.Start()

	// Queue basic tasks with no retries
	var completed int32
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		tm.QueueContext(ctx, radish.TaskFunc(func(context.Context) error {
			time.Sleep(1 * time.Millisecond)
			atomic.AddInt32(&completed, 1)
			return nil
		}))
	}

	assert.True(t, tm.IsRunning())
	tm.Stop()
}

func TestTasksError(t *testing.T) {
	// Number of workers must be greater than zero
	_, err := radish.New(radish.WithWorkers(0))
	assert.ErrorIs(t, err, radish.ErrNoWorkers, "expected an error for 0 workers")

	// Queue size must be greater than or equal to zero
	_, err = radish.New(radish.WithQueueSize(-1))
	assert.ErrorIs(t, err, radish.ErrInvalidQueueSize, "expected an error for negative queue size")
}

func TestTasksRetry(t *testing.T) {
	// This is a long running test, skip if in short mode
	if testing.Short() {
		t.Skip("skipping long running test in short mode")
	}

	// NOTE: ensure the queue size is zero so that queueing blocks until all tasks are
	// queued to prevent a race condition with the call to stop.
	tm, err := radish.New(radish.WithWorkers(4), radish.WithQueueSize(0))
	assert.Nil(t, err, "expected to be able to create a task manager")
	tm.Start()

	// Create a state of tasks that hold the number of attempts and success
	var wg sync.WaitGroup
	state := make([]*TestTask, 0, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		state = append(state, &TestTask{failUntil: 3, wg: &wg})
	}

	// Queue state tasks with a retry limit that will ensure they all succeed
	for _, retryTask := range state {
		tm.Queue(retryTask, radish.WithRetries(5), radish.WithBackOff(radish.NewConstantBackOff(0)))
	}

	// Wait for all tasks to be completed and stop the task manager.
	wg.Wait()
	tm.Stop()

	// Analyze the results from the state
	var completed, attempts int
	for _, retryTask := range state {
		attempts += retryTask.attempts
		if retryTask.success {
			completed++
		}
	}

	assert.Equal(t, 100, completed, "expected all tasks to have been completed")
	assert.Equal(t, 300, attempts, "expected all tasks to have failed twice before success")
}

func TestTasksRetryFailure(t *testing.T) {
	// This is a long running test, skip if in short mode
	if testing.Short() {
		t.Skip("skipping long running test in short mode")
	}

	// NOTE: ensure the queue size is zero so that queueing blocks until all tasks are
	// queued to prevent a race condition with the call to stop.
	tm, err := radish.New(radish.WithWorkers(4), radish.WithQueueSize(0))
	assert.Nil(t, err, "expected to be able to create a task manager")
	tm.Start()

	// Create a state of tasks that hold the number of attempts and success
	var wg sync.WaitGroup
	state := make([]*TestTask, 0, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		state = append(state, &TestTask{failUntil: 5, wg: &wg})
	}

	// Queue state tasks with a retry limit that will ensure they all fail
	for _, retryTask := range state {
		tm.Queue(retryTask, radish.WithRetries(1), radish.WithBackOff(radish.NewConstantBackOff(0)))
	}

	// Wait for all tasks to be completed and stop the task manager.
	time.Sleep(500 * time.Millisecond)
	tm.Stop()

	// Analyze the results from the state
	var completed, attempts int
	for _, retryTask := range state {
		attempts += retryTask.attempts
		if retryTask.success {
			completed++
		}
	}

	assert.Equal(t, 0, completed, "expected all tasks to have failed")
	assert.Equal(t, 200, attempts, "expected all tasks to have failed twice before no more retries")
}

func TestTasksRetryBackoff(t *testing.T) {
	// NOTE: ensure the queue size is zero so that queueing blocks until all tasks are
	// queued to prevent a race condition with the call to stop.
	tm, err := radish.New(radish.WithWorkers(4), radish.WithQueueSize(0))
	assert.Nil(t, err, "expected to be able to create a task manager")
	tm.Start()

	// Create a state of tasks that hold the number of attempts and success
	var wg sync.WaitGroup
	state := make([]*TestTask, 0, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		state = append(state, &TestTask{failUntil: 3, wg: &wg})
	}

	// Queue state tasks with a retry limit that will ensure they all succeed
	for _, retryTask := range state {
		tm.Queue(retryTask, radish.WithRetries(5), radish.WithBackOff(radish.NewConstantBackOff(0)))
	}

	// Wait for all tasks to be completed and stop the task manager.
	wg.Wait()
	tm.Stop()

	// Analyze the results from the state
	var completed, attempts int
	for _, retryTask := range state {
		attempts += retryTask.attempts
		if retryTask.success {
			completed++
		}
	}

	// TODO: how to check if backoff was respected?
	assert.Equal(t, 100, completed, "expected all tasks to have been completed")
	assert.Equal(t, 300, attempts, "expected all tasks to have failed twice before success")
}

func TestTasksRetryContextCanceled(t *testing.T) {
	// This is a long running test, skip if in short mode
	if testing.Short() {
		t.Skip("skipping long running test in short mode")
	}

	// NOTE: ensure the queue size is zero so that queueing blocks until all tasks are
	// queued to prevent a race condition with the call to stop.
	tm, err := radish.New(radish.WithWorkers(4), radish.WithQueueSize(0))
	assert.Nil(t, err, "expected to be able to create a task manager")
	tm.Start()

	var completed, attempts int32

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Queue tasks that are getting canceled
	for i := 0; i < 100; i++ {
		tm.Queue(radish.TaskFunc(func(ctx context.Context) error {
			atomic.AddInt32(&attempts, 1)
			if err := ctx.Err(); err != nil {
				return err
			}

			atomic.AddInt32(&completed, 1)
			return nil
		}), radish.WithRetries(1),
			radish.WithBackOff(radish.NewConstantBackOff(0)),
			radish.WithContext(ctx),
		)
	}

	// Wait for all tasks to be completed and stop the task manager.
	time.Sleep(500 * time.Millisecond)
	tm.Stop()

	assert.Equal(t, int32(0), completed, "expected all tasks to have been canceled")
	assert.Equal(t, int32(200), attempts, "expected all tasks to have failed twice before no more retries")
}

func TestTasksRetrySuccessAndFailure(t *testing.T) {
	// This is a long running test, skip if in short mode
	if testing.Short() {
		t.Skip("skipping long running test in short mode")
	}

	// Test non-retry tasks alongside retry tasks
	// NOTE: ensure the queue size is zero so that queueing blocks until all tasks are
	// queued to prevent a race condition with the call to stop.
	tm, err := radish.New(radish.WithWorkers(4), radish.WithQueueSize(0))
	assert.Nil(t, err, "expected to be able to create a task manager")
	tm.Start()

	// Create a state of tasks that hold the number of attempts and success
	var wg sync.WaitGroup
	state := make([]*TestTask, 0, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		state = append(state, &TestTask{failUntil: 2, wg: &wg})
	}

	// Queue state tasks with a retry limit that will ensure they all fail
	// First 50 have a retry, second 50 do not.
	for i, retryTask := range state {
		if i < 50 {
			tm.Queue(retryTask, radish.WithRetries(2), radish.WithBackOff(radish.NewConstantBackOff(0)))
		} else {
			tm.Queue(retryTask, radish.WithBackOff(radish.NewConstantBackOff(0)))
		}
	}

	// Wait for all tasks to be completed and stop the task manager.
	time.Sleep(500 * time.Millisecond)
	tm.Stop()

	// Analyze the results from the state
	var completed, attempts int
	for _, retryTask := range state {
		attempts += retryTask.attempts
		if retryTask.success {
			completed++
		}
	}

	assert.Equal(t, 50, completed, "expected all tasks to have failed")
	assert.Equal(t, 150, attempts, "expected all tasks to have failed twice before no more retries")
}

func TestTasksTimeout(t *testing.T) {
	// This is a long running test, skip if in short mode
	if testing.Short() {
		t.Skip("skipping long running test in short mode")
	}

	// NOTE: ensure the queue size is zero so that queueing blocks until all tasks are
	// queued
	tm, err := radish.New(radish.WithWorkers(4), radish.WithQueueSize(0))
	assert.Nil(t, err, "expected to be able to create a task manager")
	tm.Start()

	// Create a state of tasks that hold the number of attempts and success
	var wg sync.WaitGroup
	state := make([]*TestTask, 0, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		state = append(state, &TestTask{failUntil: 3, wg: &wg, delay: 50 * time.Millisecond})
	}

	// Queue state tasks with a short timeout
	for _, retryTask := range state {
		tm.Queue(retryTask, radish.WithBackOff(radish.NewConstantBackOff(0)), radish.WithTimeout(10*time.Millisecond))
	}

	// Wait for the timeout to expire
	time.Sleep(500 * time.Millisecond)
	tm.Stop()

	// Analyze the results from the state
	var completed, attempts int
	for _, retryTask := range state {
		attempts += retryTask.attempts
		if retryTask.success {
			completed++
		}
	}

	assert.NotEqual(t, completed, 100, "expected some tasks to have timed out")
}

func TestQueue(t *testing.T) {
	// A simple test to ensure that tm.Stop() will wait until all items in the queue are finished.
	var wg sync.WaitGroup
	queue := make(chan int32, 64)
	var final int32

	wg.Add(1)
	go func() {
		for num := range queue {
			time.Sleep(1 * time.Millisecond)
			atomic.SwapInt32(&final, num)
		}
		wg.Done()
	}()

	for i := int32(1); i < 101; i++ {
		queue <- i
	}

	close(queue)
	wg.Wait()
	assert.Equal(t, int32(100), final)
}
