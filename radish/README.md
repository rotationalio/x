# Radish

This library implements a task manager for scheduling asynchronous tasks in Go. Tasks are user-defined functions with configurable delays and retries with backoffs.

```golang
// Start a task manager
tm, err := radish.New()

// Queue a single task that's a user-defined function
tm.Queue(radish.TaskFunc(func(ctx context.Context) error {
    // Do something
    return nil
}))
```

By default, tasks do not retry on failure. You can configure a task to retry up to specified number of times until it does not return an error.

```golang
// Queue a task to retry a maximum of 3 times until it returns nil
tm.Queue(radish.TaskFunc(func(ctx context.Context) error {
    // Do something
    return nil
}), radish.WithRetries(3))
```

By default, task retries are delayed according to the configured backoff strategy. The default is an exponential backoff. A custom backoff strategy can be provided with the `WithBackOff` option. Constant backoff and zero backoff strategies are also implemented in this library.

```golang
// Use exponential backoff that stops retries after 5 minutes.
backoff := radish.NewExponentialBackoff(radish.WithMaxElapsedTime(5 * time.Minute))

tm.Queue(radish.TaskFunc(func(ctx context.Context) error {
    // Do something
    return nil
}), radish.WithRetries(3), radish.WithBackOff(backoff))
```

The radish task manager uses an internal queue and several workers to handle tasks. You can set the size of the queue and the number of workers. Note that normally `tm.Queue` is a non-blocking call. However if the queue becomes full then any `tm.Queue` calls will block until workers can handle them.

```golang
// Start a task manager with a queue size and number of workers
tm, err := radish.New(radish.WithQueueSize(128), radish.WithWorkers(8))
```