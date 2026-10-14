package diagnostics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"runtime/metrics"
	"strings"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/logger"
)

type recordingLogger struct {
	warnings []string
}

var retainedDiagnosticBytes []byte

func (*recordingLogger) Info(_ context.Context, message string, _ ...logger.Field) {
}
func (*recordingLogger) Error(context.Context, string, ...logger.Field) {}
func (*recordingLogger) Debug(context.Context, string, ...logger.Field) {}
func (l *recordingLogger) Warn(_ context.Context, message string, fields ...logger.Field) {
	encoded, _ := json.Marshal(fields)
	l.warnings = append(l.warnings, message+" "+string(encoded))
}
func (l *recordingLogger) LogService(context.Context, logger.ServiceLog)         {}
func (l *recordingLogger) LogDB(context.Context, logger.DBLog)                   {}
func (l *recordingLogger) LogEvent(context.Context, logger.EventLog)             {}
func (l *recordingLogger) LogTransaction(context.Context, logger.TransactionLog) {}
func (l *recordingLogger) WithComponent(string) logger.Logger                    { return l }

func TestWatcherCooldownAndSnapshotDoesNotExposeGoroutineArguments(t *testing.T) {
	log := &recordingLogger{}
	watcher := New(Config{GoroutineThreshold: 1, Cooldown: time.Minute}, log)
	watcher.readFile = func(string) ([]byte, error) { return nil, errors.New("not mounted") }
	watcher.now = func() time.Time { return time.Unix(100, 0) }
	watcher.initializeMemoryLimit()

	release := make(chan struct{})
	go diagnosticBlockedWorker(release, "secret-value-that-must-not-appear")
	deadline := time.Now().Add(time.Second)
	for !goroutineStackContains("diagnosticBlockedWorker") && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	snapshot, triggered := watcher.Check()
	if !triggered || snapshot.Reason != "goroutine_threshold" {
		close(release)
		t.Fatalf("expected goroutine threshold trigger, got %+v triggered=%v", snapshot, triggered)
	}
	if !watcher.Emit(context.Background(), snapshot) {
		close(release)
		t.Fatal("expected first snapshot emission")
	}
	if watcher.Emit(context.Background(), snapshot) {
		close(release)
		t.Fatal("cooldown should suppress second snapshot")
	}
	if len(log.warnings) != 1 || !strings.Contains(log.warnings[0], "diagnostic_snapshot") || !strings.Contains(log.warnings[0], "diagnosticBlockedWorker") {
		close(release)
		t.Fatalf("snapshot missing message or blocking function: %#v", log.warnings)
	}
	if strings.Contains(log.warnings[0], "secret-value-that-must-not-appear") {
		close(release)
		t.Fatal("goroutine argument leaked into diagnostic output")
	}
	watcher.now = func() time.Time { return time.Unix(161, 0) }
	if !watcher.Emit(context.Background(), snapshot) || len(log.warnings) != 2 {
		close(release)
		t.Fatal("snapshot should emit again after cooldown")
	}
	close(release)
}

func goroutineStackContains(fragment string) bool {
	for _, entry := range topGoroutines(100, 10) {
		for _, frame := range entry.TopFrames {
			if strings.Contains(frame, fragment) {
				return true
			}
		}
	}
	return false
}

func diagnosticBlockedWorker(release <-chan struct{}, secret string) {
	_ = secret
	<-release
}

func TestWatcherUnreadableMemoryLimitDisablesHeapTrigger(t *testing.T) {
	log := &recordingLogger{}
	watcher := New(Config{HeapThresholdPercent: 1, GoroutineThreshold: int(^uint(0) >> 1)}, log)
	watcher.readFile = func(string) ([]byte, error) { return nil, errors.New("missing cgroup") }
	watcher.initializeMemoryLimit()
	snapshot, triggered := watcher.Check()
	if triggered || watcher.heapOn {
		t.Fatalf("unreadable cgroup must disable heap trigger, snapshot=%+v heapOn=%v", snapshot, watcher.heapOn)
	}
}

func TestWatcherUnlimitedMemoryLimitDisablesHeapTrigger(t *testing.T) {
	watcher := New(Config{HeapThresholdPercent: 1, GoroutineThreshold: int(^uint(0) >> 1)}, &recordingLogger{})
	watcher.readFile = func(path string) ([]byte, error) {
		if path == "/sys/fs/cgroup/memory.max" {
			return []byte("max"), nil
		}
		return nil, errors.New("not present")
	}
	watcher.initializeMemoryLimit()
	if watcher.heapOn {
		t.Fatal("unlimited cgroup memory must disable heap trigger")
	}
}

func TestTopHeapAggregatesFunctionAndInUseBytes(t *testing.T) {
	retainedDiagnosticBytes = make([]byte, 2<<20)
	runtime.GC()
	entries := topHeap(10)
	if len(entries) == 0 {
		t.Fatal("expected heap profile entries")
	}
	if entries[0].InuseBytes <= 0 || entries[0].Function == "" {
		t.Fatalf("heap profile entry lacks function or in-use bytes: %+v", entries[0])
	}
}

func BenchmarkPeriodicCheck(b *testing.B) {
	watcher := New(Config{HeapThresholdPercent: 80, GoroutineThreshold: int(^uint(0) >> 1)}, &recordingLogger{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher.Check()
	}
}

func BenchmarkSnapshot5000ParkedGoroutines(b *testing.B) {
	watcher := New(Config{GoroutineThreshold: 1, Cooldown: 0}, &recordingLogger{})
	for i := 0; i < b.N; i++ {
		release := make(chan struct{})
		for worker := 0; worker < 5000; worker++ {
			go diagnosticBlockedWorker(release, "benchmark-private-value")
		}
		waitForGoroutineCount(b, 5000)
		beforePause := gcPauseHistogram()
		started := time.Now()
		watcher.Emit(context.Background(), Snapshot{Reason: "benchmark", Goroutines: 5000})
		elapsed := time.Since(started)
		afterPause := gcPauseHistogram()
		pauseUpperBound := newPauseUpperBound(beforePause, afterPause)
		b.ReportMetric(float64(elapsed.Nanoseconds()), "snapshot-ns")
		b.ReportMetric(float64(pauseUpperBound), "gc-pause-upper-bound-ns")
		close(release)
		if pauseUpperBound > 10_000_000 {
			b.Fatalf("snapshot GC pause upper bound %d ns exceeds 10ms", pauseUpperBound)
		}
	}
}

func waitForGoroutineCount(b *testing.B, expected int) {
	b.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() >= expected+2 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	b.Fatalf("only observed %d goroutines", runtime.NumGoroutine())
}

type pauseHistogram struct {
	buckets []float64
	counts  []uint64
}

func gcPauseHistogram() pauseHistogram {
	samples := []metrics.Sample{{Name: "/sched/pauses/total/gc:seconds"}}
	metrics.Read(samples)
	if samples[0].Value.Kind() != metrics.KindFloat64Histogram {
		return pauseHistogram{}
	}
	histogram := samples[0].Value.Float64Histogram()
	return pauseHistogram{buckets: append([]float64(nil), histogram.Buckets...), counts: append([]uint64(nil), histogram.Counts...)}
}

func newPauseUpperBound(before, after pauseHistogram) uint64 {
	var maxBound float64
	for index, count := range after.counts {
		if index < len(before.counts) && count <= before.counts[index] {
			continue
		}
		if index < len(after.buckets) && after.buckets[index] > maxBound {
			maxBound = after.buckets[index]
		}
	}
	if maxBound < 0 || maxBound > 1e6 {
		return 0
	}
	return uint64(maxBound * float64(time.Second))
}

func ExampleSnapshot_String() {
	fmt.Println(Snapshot{Reason: "goroutine_threshold", HeapInuseBytes: 12, Goroutines: 5}.String())
	// Output: goroutine_threshold heap=12 goroutines=5
}
