# Kubernetes Probe Handlers

This package has HTTP handlers for kubernetes liveness and readiness HTTP requests that can be easily added to any existing service. It also implements a Probe client if you want to make liveness/readiness checks against a service that implements a handler and finally it adds a server for responding to those checks for containerized applications that don't necessarily respond to HTTP requests.

## Simple Usage

If you have an `http.ServeMux` in your application, use that directly:

```go
package main

import (
    "net/http"

    "go.rtnl.ai/x/probez"
)


func main() {
    // Create the serve mux and the probe handler
    mux := http.NewServeMux()
    probe := probez.New()

    // And the probe routes to the muxer
    probe.Mux(mux)

    // Add your other routes as necessary
    // ...

    // Serve the mux
    http.ListenAndServe(":8080", mux)
}
```

The probe starts in the healthy/not ready state. To modify the state of the probe, use `probe.Healthy()`, `probe.Unhealthy()`, `probe.Ready()`, `probe.NotReady()`. For example, if you have a server:

```go
type Server struct {
    probe *probez.Handler
}

func (s *Server) Serve() {
    // Set the probe to healthy
    s.server.Healthy()

    // Start the server up
    // ...

    // Perform database pings and other preparatory code
    // ...

    // Mark the server as ready for requests
    s.server.Ready()
}

func (s *Server) Shutdown() {
    // Set the server to not ready
    s.server.NotReady()

    // Perform server shutdown
    // ...

    // Mark the server as not healthy
    s.server.Unhealthy()
}
```

## Using with Gin

Use `gin.WrapF` to utilize the probez handler with the gin web framework as follows:

```go
router := gin.Default()
probe := probez.New()

router.GET(probez.Livez, gin.Wrapf(probe.Healthz))
router.GET(probez.Healthz, gin.Wrapf(probe.Healthz))
router.GET(probez.Readyz, gin.Wrapf(probe.Readyz))
```