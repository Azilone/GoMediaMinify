package checksum

import (
	"fmt"
	"io"
	"os"

	"github.com/cespare/xxhash/v2"
)

// Hasher defines the behaviour needed for checksum calculation.
type Hasher interface {
	Calculate(path string) (uint64, error)
	CalculateReader(r io.Reader) (uint64, error)
}

// XXHash64Hasher implements Hasher using xxHash64.
type XXHash64Hasher struct{}

// NewXXHash64Hasher returns a ready-to-use xxHash64 hasher.
func NewXXHash64Hasher() *XXHash64Hasher {
	return &XXHash64Hasher{}
}

// Calculate computes the checksum for a file path.
func (h *XXHash64Hasher) Calculate(path string) (uint64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	return h.CalculateReader(file)
}

// CalculateReader computes the checksum while reading from the provided reader.
func (h *XXHash64Hasher) CalculateReader(r io.Reader) (uint64, error) {
	hasher := xxhash.New()

	buf := make([]byte, 32*1024) // 32KB buffer keeps memory usage low
	if _, err := io.CopyBuffer(hasher, r, buf); err != nil {
		return 0, fmt.Errorf("read data: %w", err)
	}

	return hasher.Sum64(), nil
}

// FormatChecksum returns a human friendly hexadecimal string.
func FormatChecksum(sum uint64) string {
	return fmt.Sprintf("%016x", sum)
}
