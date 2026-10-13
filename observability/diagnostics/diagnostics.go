package diagnostics

import (
	"bytes"
	"context"
	"fmt"
	"os"
	goruntime "runtime"
	"runtime/metrics"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yogayulanda/go-core/logger"
)

type Config struct {
	HeapThresholdPercent int
	GoroutineThreshold   int
	Cooldown             time.Duration
	CheckInterval        time.Duration
}

type Watcher struct {
	config   Config
	log      logger.Logger
	readFile func(string) ([]byte, error)
	now      func() time.Time
	mu       sync.Mutex
	lastEmit time.Time
	limit    int64
	heapOn   bool
}

type Snapshot struct {
	Reason           string
	HeapInuseBytes   uint64
	MemoryLimitBytes int64
	Goroutines       int
	GC               map[string]uint64
	TopHeap          []HeapEntry
	TopGoroutines    []GoroutineEntry
}

type HeapEntry struct {
	Function   string `json:"function"`
	InuseBytes int64  `json:"inuse_bytes"`
}

type GoroutineEntry struct {
	Count     int      `json:"count"`
	TopFrames []string `json:"top_frames"`
}

func New(cfg Config, log logger.Logger) *Watcher {
	return &Watcher{config: cfg, log: log, readFile: readFile, now: time.Now}
}

func (w *Watcher) Start(ctx context.Context) {
	if w == nil || w.log == nil {
		return
	}
	w.initializeMemoryLimit()
	go w.run(ctx)
}

func (w *Watcher) initializeMemoryLimit() {
	for _, path := range []string{"/sys/fs/cgroup/memory.max", "/sys/fs/cgroup/memory/memory.limit_in_bytes"} {
		data, err := w.readFile(path)
		if err != nil {
			continue
		}
		value := strings.TrimSpace(string(data))
		if value == "max" {
			break
		}
		limit, err := strconv.ParseInt(value, 10, 64)
		if err != nil || limit <= 0 || limit >= 1<<60 {
			break
		}
		w.limit = limit
		w.heapOn = true
		return
	}
	w.log.Info(context.Background(), "diagnostic heap trigger disabled: cgroup memory limit unavailable", logger.Field{Key: "app.diagnostic.reason", Value: "memory_limit_unavailable"})
}

func (w *Watcher) run(ctx context.Context) {
	interval := w.config.CheckInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if snapshot, triggered := w.Check(); triggered {
				w.Emit(ctx, snapshot)
			}
		}
	}
}

func (w *Watcher) Check() (Snapshot, bool) {
	var samples = []metrics.Sample{{Name: "/memory/classes/heap/objects:bytes"}}
	metrics.Read(samples)
	var heapInuse uint64
	if samples[0].Value.Kind() == metrics.KindUint64 {
		heapInuse = samples[0].Value.Uint64()
	}
	goroutines := goruntime.NumGoroutine()
	snapshot := Snapshot{HeapInuseBytes: heapInuse, MemoryLimitBytes: w.limit, Goroutines: goroutines}
	if w.heapOn && float64(heapInuse)*100 >= float64(w.limit)*float64(w.config.HeapThresholdPercent) {
		snapshot.Reason = "heap_threshold"
	}
	if w.config.GoroutineThreshold > 0 && goroutines >= w.config.GoroutineThreshold {
		if snapshot.Reason == "" {
			snapshot.Reason = "goroutine_threshold"
		} else {
			snapshot.Reason = "heap_and_goroutine_threshold"
		}
	}
	return snapshot, snapshot.Reason != ""
}

func (w *Watcher) Emit(ctx context.Context, snapshot Snapshot) bool {
	if w == nil || w.log == nil || snapshot.Reason == "" {
		return false
	}
	w.mu.Lock()
	if !w.lastEmit.IsZero() && w.now().Sub(w.lastEmit) < w.config.Cooldown {
		w.mu.Unlock()
		return false
	}
	w.lastEmit = w.now()
	w.mu.Unlock()

	snapshot.GC = gcStats()
	snapshot.TopHeap = topHeap(10)
	snapshot.TopGoroutines = topGoroutines(10, 5)
	w.log.Warn(ctx, "diagnostic_snapshot",
		logger.Field{Key: "app.diagnostic.reason", Value: snapshot.Reason},
		logger.Field{Key: "app.diagnostic.heap_inuse_bytes", Value: snapshot.HeapInuseBytes},
		logger.Field{Key: "app.diagnostic.memory_limit_bytes", Value: snapshot.MemoryLimitBytes},
		logger.Field{Key: "app.diagnostic.goroutines", Value: snapshot.Goroutines},
		logger.Field{Key: "app.diagnostic.gc", Value: snapshot.GC},
		logger.Field{Key: "app.diagnostic.top_heap", Value: snapshot.TopHeap},
		logger.Field{Key: "app.diagnostic.top_goroutines", Value: snapshot.TopGoroutines},
	)
	return true
}

func (snapshot Snapshot) String() string {
	return fmt.Sprintf("%s heap=%d goroutines=%d", snapshot.Reason, snapshot.HeapInuseBytes, snapshot.Goroutines)
}

func gcStats() map[string]uint64 {
	var stats goruntime.MemStats
	goruntime.ReadMemStats(&stats)
	return map[string]uint64{
		"num_gc":            uint64(stats.NumGC),
		"pause_total_ns":    stats.PauseTotalNs,
		"last_pause_ns":     stats.PauseNs[(stats.NumGC+255)%256],
		"heap_objects":      stats.HeapObjects,
		"total_alloc_bytes": stats.TotalAlloc,
	}
}

func topHeap(limit int) []HeapEntry {
	profile := pprof.Lookup("heap")
	if profile == nil {
		return nil
	}
	var output bytes.Buffer
	if profile.WriteTo(&output, 1) != nil {
		return nil
	}
	byFunction := make(map[string]int64)
	var bytesInUse int64
	for _, line := range strings.Split(output.String(), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, " @ ") {
			var objects int64
			if _, err := fmt.Sscanf(line, "%d: %d", &objects, &bytesInUse); err != nil || objects <= 0 || bytesInUse <= 0 {
				bytesInUse = 0
			}
			continue
		}
		if bytesInUse <= 0 || !strings.HasPrefix(line, "#\t") {
			continue
		}
		columns := strings.Split(line, "\t")
		if len(columns) >= 3 {
			byFunction[columns[2]] += bytesInUse
			bytesInUse = 0
		}
	}
	entries := make([]HeapEntry, 0, len(byFunction))
	for function, inuseBytes := range byFunction {
		entries = append(entries, HeapEntry{Function: function, InuseBytes: inuseBytes})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].InuseBytes > entries[j].InuseBytes })
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries
}

func topGoroutines(limit, frameLimit int) []GoroutineEntry {
	buffer := make([]byte, 1<<20)
	n := goruntime.Stack(buffer, true)
	if n == 0 {
		return nil
	}
	groups := make(map[string]int)
	for _, block := range strings.Split(string(buffer[:n]), "\n\n") {
		lines := strings.Split(block, "\n")
		var frames []string
		for _, line := range lines[1:] {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "created by ") || strings.HasPrefix(line, "/") {
				continue
			}
			if strings.Contains(line, ".") && !strings.Contains(line, "0x") && len(frames) < frameLimit {
				frames = append(frames, line)
			}
		}
		if len(frames) > 0 {
			groups[strings.Join(frames, "\n")]++
		}
	}
	entries := make([]GoroutineEntry, 0, len(groups))
	for stack, count := range groups {
		entries = append(entries, GoroutineEntry{Count: count, TopFrames: strings.Split(stack, "\n")})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Count > entries[j].Count })
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries
}

var readFile = func(path string) ([]byte, error) { return os.ReadFile(path) }
