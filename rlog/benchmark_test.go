package rlog_test

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"runtime"
	"testing"
	"time"

	"go.rtnl.ai/x/rlog"
)

// envRunLoggerVsSlogCompare must be set to "1" to run [TestLoggerVsSlogCompare]. That keeps plain
// go test and IDE "run test" (which pass -run by name) from executing this slow comparison unless
// you opt in from a real shell.
const envRunLoggerVsSlogCompare = "RLOG_LOGGER_VS_SLOG_COMPARE"

// BenchmarkLoggerVsSlog compares *rlog.Logger with *slog.Logger on identical nop handlers.
//
// # What every logging sub-benchmark measures
//
// Each “slog/…” and “rlog/…” pair uses the same [nopHandler]: Enabled always true, Handle a no-op.
// So the cost is almost entirely: capture caller PC (when applicable), build [slog.Record], merge
// attrs, dispatch to Handle—no I/O, no JSON, no locks in the handler.
//
// Sub-benchmark names:
//   - LogAttrs-none — LogAttrs with message only (no extra attributes).
//   - Info-kv — Info with one string key and int value ("a", 1).
//   - Log-kv — Log with the same key/value pair.
//   - *+pollute — same as above, but each iteration calls [benchPolluteCache] first (full rationale
//     in the helpers section under “Cache pollution”).
//   - paired/Info-kv-interleaved — one slog.Info then one rlog.Info per b.N iteration; ns/op is time
//     for both calls together (not per logger). Useful to see both code paths in one hot loop.
//   - rlog/Trace-none — custom TRACE level (no slog analogue with the same name).
//   - runtime.Callers/skip2 — isolated cost of Callers(2), matching rlog’s wrapper depth.
//   - runtime.Callers/skip3 — isolated cost of Callers(3), matching slog.Logger.log’s depth.
//   - directHandle-pc0 — NewRecord + Handle with PC=0, no Callers (rough lower bound if source were free).
//
// # Why rlog can look faster than slog in the plain pairs
//
// Back-to-back sub-benchmarks are separate [testing.Benchmark] timing windows. The second side
// often hits warmer I-cache and branch history. rlog also uses Callers(2) at its API vs slog’s
// Callers(3) inside log—not identical work. So “rlog faster” on an isolated pair is not proof the
// wrapper is free; use [TestLoggerVsSlogCompare] or *+pollute for a less misleading picture.
//
// # How to run
//
// Skip other tests (avoids concurrent tests printing JSON and competing for CPU):
//
//	go test -run=^$ -bench=BenchmarkLoggerVsSlog -benchmem ./rlog
//
// Percentage summary (slower/faster), with order averaging and a polluted section:
//
//	RLOG_LOGGER_VS_SLOG_COMPARE=1 go test ./rlog -run TestLoggerVsSlogCompare -v
//
// rlog cannot use [log/slog/internal.IgnorePC] (stdlib-only); slog’s -nopc benchmark flag does not apply here.
func BenchmarkLoggerVsSlog(b *testing.B) {
	b.Run("slog/LogAttrs-none", benchSlogLogAttrsNone)
	b.Run("rlog/LogAttrs-none", benchRlogLogAttrsNone)
	b.Run("slog/Info-kv", benchSlogInfoKV)
	b.Run("rlog/Info-kv", benchRlogInfoKV)
	b.Run("slog/Log-kv", benchSlogLogKV)
	b.Run("rlog/Log-kv", benchRlogLogKV)
	b.Run("slog/LogAttrs-none+pollute", benchSlogLogAttrsNonePollute)
	b.Run("rlog/LogAttrs-none+pollute", benchRlogLogAttrsNonePollute)
	b.Run("slog/Info-kv+pollute", benchSlogInfoKVPollute)
	b.Run("rlog/Info-kv+pollute", benchRlogInfoKVPollute)
	b.Run("slog/Log-kv+pollute", benchSlogLogKVPollute)
	b.Run("rlog/Log-kv+pollute", benchRlogLogKVPollute)
	b.Run("paired/Info-kv-interleaved", benchPairedInfoKVInterleaved)
	b.Run("rlog/Trace-none", benchRlogTraceNone)
	b.Run("runtime.Callers/skip2", benchRuntimeCallersSkip2)
	b.Run("runtime.Callers/skip3", benchRuntimeCallersSkip3)
	b.Run("directHandle-pc0", benchDirectHandlePC0)
}

// TestLoggerVsSlogCompare runs the same benchmark bodies as [BenchmarkLoggerVsSlog] and prints
// how much slower or faster rlog is versus slog as a percentage of slog’s ns/op.
//
// # Why not just compare two bench outputs by hand?
//
// A single run of “slog bench then rlog bench” biases whoever runs second (hotter caches for that
// binary path). This test uses [nsPerOpOrderAvg] for each pair: it runs slog→rlog, then rlog→slog,
// and averages each side’s ns/op so neither implementation always benefits from running after the other.
//
// # Two sections: isolated vs +pollute
//
// The first block is the plain benchmarks (tight loop, log call only). The second block uses the
// *+pollute functions: before each log call they run [benchPolluteCache] to touch a large, strided
// region of memory. That makes “everything fits in L1 and stays hot” less likely and often widens
// the gap when one path does more work (e.g. extra indirection in rlog). It still does not simulate
// real app contention; it is only a harsher microbench knob.
//
// # Baselines
//
// The final lines log rlog-only and PC-only benchmarks so you can sanity-check Callers cost vs full log.
//
// Skipped under -short (this test runs many benchmarks and can take tens of seconds).
// Skipped unless [envRunLoggerVsSlogCompare] is set to "1" (avoids IDE "run test" and plain go test).
// Run: RLOG_LOGGER_VS_SLOG_COMPARE=1 go test ./rlog -run TestLoggerVsSlogCompare -v
func TestLoggerVsSlogCompare(t *testing.T) {
	if testing.Short() {
		t.Skip("skip bench comparison in short mode")
	}
	if os.Getenv(envRunLoggerVsSlogCompare) != "1" {
		t.Skipf("set %s=1 to run this comparison (IDE and default go test skip)", envRunLoggerVsSlogCompare)
	}

	pairs := []struct {
		name   string
		slogFn func(*testing.B)
		rlogFn func(*testing.B)
	}{
		{"LogAttrs-none", benchSlogLogAttrsNone, benchRlogLogAttrsNone},
		{"Info-kv", benchSlogInfoKV, benchRlogInfoKV},
		{"Log-kv", benchSlogLogKV, benchRlogLogKV},
	}
	pairsPollute := []struct {
		name   string
		slogFn func(*testing.B)
		rlogFn func(*testing.B)
	}{
		{"LogAttrs-none+pollute", benchSlogLogAttrsNonePollute, benchRlogLogAttrsNonePollute},
		{"Info-kv+pollute", benchSlogInfoKVPollute, benchRlogInfoKVPollute},
		{"Log-kv+pollute", benchSlogLogKVPollute, benchRlogLogKVPollute},
	}

	t.Log("rlog vs slog — isolated runs (positive % = rlog slower; negative = rlog faster)")
	var sumPct float64
	for _, p := range pairs {
		sNs, rNs := nsPerOpOrderAvg(p.slogFn, p.rlogFn)
		pct := pctRlogVsSlog(sNs, rNs)
		sumPct += pct
		dir := "slower"
		if pct < 0 {
			dir = "faster"
		}
		t.Logf("%s: slog %s/op  rlog %s/op  → rlog is %.1f%% %s than slog (order-averaged SR+RS)",
			p.name,
			formatNs(sNs),
			formatNs(rNs),
			math.Abs(pct),
			dir,
		)
	}
	avg := sumPct / float64(len(pairs))
	t.Logf("mean signed difference (rlog vs slog): %+.1f%%", avg)

	t.Log("same pairs with cache pollution each iteration (order-averaged SR+RS)")
	var sumPctP float64
	for _, p := range pairsPollute {
		sNs, rNs := nsPerOpOrderAvg(p.slogFn, p.rlogFn)
		pct := pctRlogVsSlog(sNs, rNs)
		sumPctP += pct
		dir := "slower"
		if pct < 0 {
			dir = "faster"
		}
		t.Logf("%s: slog %s/op  rlog %s/op  → rlog is %.1f%% %s than slog",
			p.name,
			formatNs(sNs),
			formatNs(rNs),
			math.Abs(pct),
			dir,
		)
	}
	t.Logf("mean signed difference (+pollute): %+.1f%%", sumPctP/float64(len(pairsPollute)))

	// Baselines: single-purpose benchmarks (no slog/rlog pair). Help interpret how much of the
	// full log path is Callers vs Trace-only vs a minimal Handle with PC=0.
	t.Log("baselines (not compared to slog):")
	for _, c := range []struct {
		name string
		fn   func(*testing.B)
	}{
		{"rlog/Trace-none", benchRlogTraceNone},
		{"runtime.Callers/skip2", benchRuntimeCallersSkip2},
		{"runtime.Callers/skip3", benchRuntimeCallersSkip3},
		{"directHandle-pc0", benchDirectHandlePC0},
	} {
		res := testing.Benchmark(c.fn)
		t.Logf("  %s: %s/op", c.name, formatNs(nsPerOp(res)))
	}
}

//==============================================================================
// Helpers (nop handler, benchmark bodies, small utilities)
//==============================================================================
//
// # Cache pollution (benchScratch + benchPolluteCache)
//
// Microbenchmarks that only call logging in a tight loop keep the logger’s code and small stack
// data in fast CPU caches (L1/L2) and train the branch predictor. That can hide real costs and
// make two similar paths look arbitrarily close—or make the second timed path look faster.
//
// benchPolluteCache is a deliberate “dirty the cache a bit” step:
//   - benchScratch is 1 MiB (1<<20 bytes), larger than typical L1 and often larger than L2, so
//     sequential or random touches can evict or replace lines that held logging code or stack.
//   - We do not try to flush caches portably (no CLFLUSH, no privileged instructions); this is
//     best-effort noise, not a hardware-level cache clear.
//   - Each call performs 256 byte reads at indices spread with a prime stride (4093) and a per-iter
//     mix (j). That scatters accesses across the buffer instead of hammering one cache line, which
//     would be a different artifact.
//   - The XOR accumulator is written to sink so the reads are not dead and the compiler keeps the loop.
//
// Limitations: other cores, prefetchers, and OS scheduling still dominate on some machines; results
// should be interpreted with [testing.B] variance and tools like benchstat, not as absolute truth.
//
// # Order averaging (nsPerOpOrderAvg)
//
// Four separate [testing.Benchmark] invocations per pair: SR (slog then rlog), RS (rlog then slog).
// Slog’s ns/op is (s1+s2)/2 and rlog’s is (r1+r2)/2 so each side is measured once as “first” and once
// as “second” after the competitor’s benchmark has run.

// sink prevents the compiler from optimizing away benchmark bodies (Callers PC, pollution XOR).
var sink uintptr

// benchScratch is the backing store for cache pollution; see package comment and benchPolluteCache.
var benchScratch []byte

func init() {
	// Non-zero contents so the buffer is not all zero pages with special OS behavior; also makes
	// XOR depend on data the compiler cannot fold away.
	benchScratch = make([]byte, 1<<20)
	for i := range benchScratch {
		benchScratch[i] = byte(i)
	}
}

// benchPolluteCache performs the read traffic described in the helpers section header. iter varies
// the starting phase so consecutive benchmark iterations do not hit the exact same addresses every time.
func benchPolluteCache(iter int) {
	if len(benchScratch) == 0 {
		return
	}
	// 256 loads, indices wrapped with &(len-1) because len is a power of two (cheap mask).
	j := iter * 1103515245 // mix iteration index (linear congruential-style spread)
	var s byte
	for k := 0; k < 256; k++ {
		s ^= benchScratch[(j+k*4093)&(len(benchScratch)-1)]
	}
	sink = uintptr(s)
}

// nsPerOpOrderAvg returns averaged ns/op for slogFn and rlogFn after running SR then RS sequences.
func nsPerOpOrderAvg(slogFn, rlogFn func(*testing.B)) (slogNs, rlogNs float64) {
	s1 := nsPerOp(testing.Benchmark(slogFn))
	r1 := nsPerOp(testing.Benchmark(rlogFn))
	r2 := nsPerOp(testing.Benchmark(rlogFn))
	s2 := nsPerOp(testing.Benchmark(slogFn))
	return (s1 + s2) / 2, (r1 + r2) / 2
}

// nopHandler is a slog.Handler that accepts every record and does nothing in Handle. All logging
// cost stays in the Logger wrapper and record construction—ideal for comparing rlog vs slog shape.
type nopHandler struct{}

func (nopHandler) Enabled(context.Context, slog.Level) bool { return true }

func (nopHandler) Handle(context.Context, slog.Record) error { return nil }

func (nopHandler) WithAttrs([]slog.Attr) slog.Handler { return nopHandler{} }

func (nopHandler) WithGroup(string) slog.Handler { return nopHandler{} }

// benchEnv builds a fresh context and two loggers sharing the same nop handler: a raw *slog.Logger
// and an *rlog.Logger wrapping slog.New(same handler). Call once per benchmark function, not per b.N.
func benchEnv() (ctx context.Context, slogL *slog.Logger, rlogL *rlog.Logger, h nopHandler) {
	ctx = context.Background()
	h = nopHandler{}
	slogL = slog.New(h)
	rlogL = rlog.New(slog.New(h))
	return ctx, slogL, rlogL, h
}

// --- Plain pairs (no pollution): tight loop, one log API call per iteration. ---

func benchSlogLogAttrsNone(b *testing.B) {
	ctx, slogL, _, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		slogL.LogAttrs(ctx, slog.LevelInfo, "msg")
	}
}

func benchRlogLogAttrsNone(b *testing.B) {
	ctx, _, rlogL, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rlogL.LogAttrs(ctx, slog.LevelInfo, "msg")
	}
}

func benchSlogInfoKV(b *testing.B) {
	_, slogL, _, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		slogL.Info("msg", "a", 1)
	}
}

func benchRlogInfoKV(b *testing.B) {
	_, _, rlogL, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rlogL.Info("msg", "a", 1)
	}
}

func benchSlogLogKV(b *testing.B) {
	ctx, slogL, _, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		slogL.Log(ctx, slog.LevelInfo, "msg", "a", 1)
	}
}

func benchRlogLogKV(b *testing.B) {
	ctx, _, rlogL, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rlogL.Log(ctx, slog.LevelInfo, "msg", "a", 1)
	}
}

// --- Polluted pairs: benchPolluteCache(i) runs immediately before each log call (same i as loop). ---

func benchSlogLogAttrsNonePollute(b *testing.B) {
	ctx, slogL, _, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchPolluteCache(i)
		slogL.LogAttrs(ctx, slog.LevelInfo, "msg")
	}
}

func benchRlogLogAttrsNonePollute(b *testing.B) {
	ctx, _, rlogL, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchPolluteCache(i)
		rlogL.LogAttrs(ctx, slog.LevelInfo, "msg")
	}
}

func benchSlogInfoKVPollute(b *testing.B) {
	_, slogL, _, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchPolluteCache(i)
		slogL.Info("msg", "a", 1)
	}
}

func benchRlogInfoKVPollute(b *testing.B) {
	_, _, rlogL, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchPolluteCache(i)
		rlogL.Info("msg", "a", 1)
	}
}

func benchSlogLogKVPollute(b *testing.B) {
	ctx, slogL, _, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchPolluteCache(i)
		slogL.Log(ctx, slog.LevelInfo, "msg", "a", 1)
	}
}

func benchRlogLogKVPollute(b *testing.B) {
	ctx, _, rlogL, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchPolluteCache(i)
		rlogL.Log(ctx, slog.LevelInfo, "msg", "a", 1)
	}
}

// benchPairedInfoKVInterleaved runs slog.Info then rlog.Info every iteration, fixed order.
// Reported ns/op = total time / b.N, i.e. per pair of calls, not per single log. Useful to see both
// implementations in one loop (shared iteration overhead); the number is not directly comparable
// to the per-side sub-benchmarks without dividing by two and accounting for interaction effects.
func benchPairedInfoKVInterleaved(b *testing.B) {
	_, slogL, rlogL, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		slogL.Info("msg", "a", 1)
		rlogL.Info("msg", "a", 1)
	}
}

// --- rlog-only and PC baselines (no slog equivalent for Trace). ---

func benchRlogTraceNone(b *testing.B) {
	_, _, rlogL, _ := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rlogL.Trace("msg")
	}
}

// benchRuntimeCallersSkip2 measures a single Callers(2) per iteration (rlog-style skip depth).
func benchRuntimeCallersSkip2(b *testing.B) {
	var pcs [1]uintptr
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		runtime.Callers(2, pcs[:])
	}
	sink = pcs[0]
}

// benchRuntimeCallersSkip3 measures a single Callers(3) per iteration (slog.Logger.log depth).
func benchRuntimeCallersSkip3(b *testing.B) {
	var pcs [1]uintptr
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		runtime.Callers(3, pcs[:])
	}
	sink = pcs[0]
}

// benchDirectHandlePC0 bypasses Logger and Callers: builds a Record with PC==0 and invokes the nop
// handler. Approximates “if we did not capture source at all” plus minimal record + Handle overhead.
func benchDirectHandlePC0(b *testing.B) {
	ctx, _, _, h := benchEnv()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
		_ = h.Handle(ctx, r)
	}
}

// nsPerOp converts [testing.BenchmarkResult] to nanoseconds per iteration.
func nsPerOp(r testing.BenchmarkResult) float64 {
	if r.N == 0 {
		return 0
	}
	return float64(r.T.Nanoseconds()) / float64(r.N)
}

// pctRlogVsSlog returns (rlog-slog)/slog * 100. Positive means rlog is slower than slog.
func pctRlogVsSlog(slogNs, rlogNs float64) float64 {
	if slogNs == 0 {
		return math.NaN()
	}
	return 100 * (rlogNs - slogNs) / slogNs
}

// formatNs picks a readable precision for logging ns/op in test output.
func formatNs(ns float64) string {
	if ns < 10 {
		return fmt.Sprintf("%.2f ns", ns)
	}
	return fmt.Sprintf("%.1f ns", ns)
}
