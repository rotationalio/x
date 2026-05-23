package benchmark_test

// Optional JSON benchmark snapshot (off by default).
//
//	PURSER_BENCH_SNAPSHOT=1 go test -run=TestBenchmarkSnapshot -count=1 ./purser/benchmark
//
// Writes purser/benchmark/results/go<ver>_<goos>_<goarch>_<commit>.json (overwrites per key).
// Copies the prior file to results/previous.json before overwrite.
// Compare: python3 ./purser/benchmark/compare.py

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"go.rtnl.ai/x/assert"
)

// benchSnapshot is the on-disk JSON document for one capture run.
type benchSnapshot struct {
	CapturedAt string       `json:"captured_at"`
	GoVersion  string       `json:"go_version"`
	GOOS       string       `json:"goos"`
	GOARCH     string       `json:"goarch"`
	Commit     string       `json:"commit,omitempty"`
	Benchmarks []benchEntry `json:"benchmarks"`
}

// benchEntry holds one sub-benchmark measurement.
type benchEntry struct {
	Name        string `json:"name"`
	NsPerOp     int64  `json:"ns_per_op"`
	BytesPerOp  int64  `json:"bytes_per_op"`
	AllocsPerOp int64  `json:"allocs_per_op"`
}

// TestBenchmarkSnapshot records size=256 hot-path numbers via testing.Benchmark.
func TestBenchmarkSnapshot(t *testing.T) {
	if os.Getenv("PURSER_BENCH_SNAPSHOT") == "" {
		t.Skip("set PURSER_BENCH_SNAPSHOT=1 to write a JSON snapshot")
	}
	if f := flag.Lookup("test.benchtime"); f != nil {
		if f.Value.String() == "1s" {
			_ = f.Value.Set("100ms")
		}
	}

	const size = benchSize256
	cases := []struct {
		name string
		fn   func(*testing.B)
	}{
		{"Locker/Seal/size=256", func(b *testing.B) { benchLockerSeal(b, size) }},
		{"Locker/Open/size=256", func(b *testing.B) { benchLockerOpen(b, size) }},
		{"Keyring/Route/1Locker/size=256", func(b *testing.B) { benchKeyringRoute(b, size) }},
		{"Registry/ParseKeyID/size=256", func(b *testing.B) { benchRegistryParseKeyID(b, size) }},
		{"Purser/Store/size=256", func(b *testing.B) { benchPurserStore(b, size) }},
		{"Purser/Retrieve/size=256", func(b *testing.B) { benchPurserRetrieve(b, size) }},
		{"Purser/Orchestration/Nulllocker/Store/size=256", func(b *testing.B) { benchPurserNulllockerStore(b, size) }},
	}

	snap := benchSnapshot{
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
		GoVersion:  runtime.Version(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		Commit:     vcsRevision(),
	}
	for _, c := range cases {
		r := testing.Benchmark(c.fn)
		assert.True(t, r.N > 0, c.name+": benchmark did not run")
		snap.Benchmarks = append(snap.Benchmarks, benchEntry{
			Name:        c.name,
			NsPerOp:     r.NsPerOp(),
			BytesPerOp:  r.AllocedBytesPerOp(),
			AllocsPerOp: r.AllocsPerOp(),
		})
	}

	_, here, _, ok := runtime.Caller(0)
	assert.True(t, ok, "runtime.Caller failed")
	dir := filepath.Join(filepath.Dir(here), "results")
	assert.Ok(t, os.MkdirAll(dir, 0o755))

	path := filepath.Join(dir, snapshotFilename(snap)+".json")
	rotateSnapshot(path)

	data, err := json.MarshalIndent(snap, "", "  ")
	assert.Ok(t, err)
	assert.Ok(t, os.WriteFile(path, data, 0o644))
	t.Logf("wrote %s", path)
}

// snapshotFilename returns a stable results basename (no timestamp) for this platform and commit.
func snapshotFilename(s benchSnapshot) string {
	commit := s.Commit
	if commit == "" {
		commit = "nogit"
	}
	goVer := strings.TrimPrefix(runtime.Version(), "go")
	return fmt.Sprintf("go%s_%s_%s_%s", goVer, s.GOOS, s.GOARCH, commit)
}

// rotateSnapshot copies path to results/previous.json when path already exists.
func rotateSnapshot(path string) {
	prev := filepath.Join(filepath.Dir(path), "previous.json")
	if _, err := os.Stat(path); err != nil {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_ = os.WriteFile(prev, data, 0o644)
}

// vcsRevision returns a short commit id from build info, else from git rev-parse.
func vcsRevision() string {
	if rev := vcsRevisionFromBuildInfo(); rev != "" {
		return rev
	}
	return vcsRevisionFromGit()
}

// vcsRevisionFromBuildInfo reads vcs.revision from test binary build metadata.
func vcsRevisionFromBuildInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return shortRevision(s.Value)
		}
	}
	return ""
}

// vcsRevisionFromGit shells out to git when build info has no VCS revision.
func vcsRevisionFromGit() string {
	root := moduleRoot()
	if root == "" {
		return ""
	}
	out, err := exec.Command("git", "-C", root, "rev-parse", "--short=12", "HEAD").Output()
	if err != nil {
		return ""
	}
	return shortRevision(strings.TrimSpace(string(out)))
}

// shortRevision truncates a revision string to 12 characters.
func shortRevision(rev string) string {
	if len(rev) > 12 {
		return rev[:12]
	}
	return rev
}

// moduleRoot walks up from the working directory to find the go.mod directory.
func moduleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
