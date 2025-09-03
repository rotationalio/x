# Backoff

This is a port of the exponential backoff algorithm from [cenkalti/backoff](https://github.com/cenkalti/backoff) that implements a Rotational-specific channel interface.

[Exponential backoff](http://en.wikipedia.org/wiki/Exponential_backoff) is an algorithm that uses feedback to multiplicatively decrease the rate of some process, in order to gradually find an acceptable rate. The retries exponentially increase and stop increasing when a certain threshold is met.

## Usage

The simplest interface is the `Retry` interface, in which you specify a function to be retried with backoff if that function returns an error. The function is guaranteed to be called at least once.

```go
// Return error to retry, nil error on success
func operation() (bool, error) {}

func Do() error {
    if _, err := backoff.Retry(context.Background(), operation); err != nil {
        return err
    }
    return nil
}
```

Data can also be returned from the operation. For example to return an integer:

```go
func guess() (int, error) {
    val := rand.Intn(0, 1000)
    if val < 42 || val > 789 {
        return 0, errors.New("bounds exceeded")
    }
    return val, nil
}

func Do() int {
    // Will keep retrying guess with backoff until a random number greater than 42 and
    // less than 789 is guessed by the random integer generator
    val, err := backoff.Retry(context.Background()), guess)
}
```

You can specify a notification function to be called on every failure:

```go
func notify(err error, delay time.Duration) {
    log.Warn().Err(err).Duration("delay", delay).Msg("server not available, retrying after delay")
}


func Do() {
    backoff.Retry(context.Background(), Ping, backoff.WithNotify(notify))
}
```

You can also specify different backoff policies using the `WithBackoff` option, specify a maximum delay with `backoff.WithMaxElapsedTime` (defaults to 15 minutes, set to 0 to retry indefinitely). and a maximum number of attempts with `backoff.WithMaxTries`.
