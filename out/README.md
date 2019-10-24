# Out

**Package out implements simple hierarchical logging functionality.**

This is a pretty standard hierarchical output logging module that allows you to use tiered functions to manage what is printed to the command line. For example if the level is set to `Info`, then `Trace` and `Debug` messages will be implemented as no-ops, reducing the amount of information printed.

This package also provides a Caution log level - caution messages are only printed if
a specific threshold of messages has been reached. This helps to reduce the number of
repeated messages (e.g. connection down) that occur in logging while still giving
effective debugging and systems administration feedback to the user.

Setting up the console usually happens in the `init()` method of a package:

```go
func init() {
    // Initialize our debug logging with our prefix
    out.Init("[myapp] ", log.Lmicroseconds)
    out.SetLevel(out.LevelInfo)
}
```

Now the logging functions can automatically be used:

```go
out.Trace("routine %s happening", thing)
out.Debug("sending message #%d from %s to %s", msg, send, recv)
out.Info("listening on %s", addr)
out.Caution("could not reach %s -- connection is down", addr)
out.Status("completed %d out of %d tasks", completed, nTasks)
out.Warn("limit of %d queries reached", nQueries)
out.Warne(err)
```

The purpose of these functions were to have simple `pout` and `perr` methods inside of applications.
