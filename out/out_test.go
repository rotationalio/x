package out_test

import (
	"errors"

	"go.rtnl.ai/x/out"
)

func ExampleOutput() {
	out.Init("[test] ", 0)
	out.SetLogLevel(out.LevelInfo)
	out.SetCautionThreshold(2)

	out.Trace("routine %s happening", "thing")
	out.Debug("sending message #%d from %s to %s", 42, "me", "you")
	out.Info("listening on %s", "127.0.0.1")
	out.Caution("could not reach %s -- connection is down", "uptime.robot")
	out.Status("completed %d out of %d tasks", 42, 121)
	out.Warn("limit of %d queries reached", 21)
	out.Warne(errors.New("something bad happened"))

	out.Caution("could not reach %s -- connection is down", "uptime.robot")
	out.Caution("could not reach %s -- connection is down", "uptime.robot")

	// Output:
	// [test] listening on 127.0.0.1
	// [test] could not reach uptime.robot -- connection is down
	// [test] completed 42 out of 121 tasks
	// [test] limit of 21 queries reached
	// [test] something bad happened
	// [test] could not reach uptime.robot -- connection is down
}
