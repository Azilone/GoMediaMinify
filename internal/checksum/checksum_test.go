package checksum

import (
	"bytes"
	"os"
	"sync"
	"testing"
)

func TestXXHash64HasherCalculate(t *testing.T) {
	hasher := NewXXHash64Hasher()

	tmpFile, err := os.CreateTemp(t.TempDir(), "checksum-*.txt")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer tmpFile.Close()

	content := []byte("mediaflow-copy-mode")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	sumFromFile, err := hasher.Calculate(tmpFile.Name())
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}

	sumFromReader, err := hasher.CalculateReader(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("CalculateReader returned error: %v", err)
	}

	if sumFromFile != sumFromReader {
		t.Fatalf("expected identical sums, got %x and %x", sumFromFile, sumFromReader)
	}
}

func TestTracker(t *testing.T) {
	tracker := NewTracker()

	tracker.Register(1, "file-a")
	tracker.Register(1, "file-b")
	tracker.Register(2, "file-c")

	if !tracker.IsDuplicate(1) {
		t.Fatalf("expected checksum 1 to be duplicate")
	}

	first, ok := tracker.FirstPath(1)
	if !ok {
		t.Fatalf("expected checksum 1 to exist")
	}
	if first != "file-a" {
		t.Fatalf("expected first path to be file-a, got %s", first)
	}

	unique, duplicates := tracker.Stats()
	if unique != 2 {
		t.Fatalf("expected 2 unique checksums, got %d", unique)
	}
	if duplicates != 1 {
		t.Fatalf("expected 1 duplicate, got %d", duplicates)
	}
}

func TestTrackerConcurrency(t *testing.T) {
	tracker := NewTracker()
	const workers = 10
	const perWorker = 50

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(offset int) {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				sum := uint64(offset*perWorker + j)
				tracker.Register(sum, "path")
			}
		}(i)
	}

	wg.Wait()

	unique, duplicates := tracker.Stats()
	if unique != workers*perWorker {
		t.Fatalf("expected %d unique checksums, got %d", workers*perWorker, unique)
	}
	if duplicates != 0 {
		t.Fatalf("expected 0 duplicates, got %d", duplicates)
	}
}
