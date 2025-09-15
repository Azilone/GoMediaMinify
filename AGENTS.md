# AGENTS.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build Commands
- `make build` - Build the binary to `./build/media-converter`
- `make build-local` - Build for local development as `./media-converter`
- `make dev` - Quick development build (clean, deps, build-local)
- `make cross-compile` - Build for multiple platforms (macOS, Linux, Windows)

### Testing and Quality
- `make test` - Run all tests
- `make test-coverage` - Run tests with coverage report
- `make check` - Run formatting, vetting, and tests
- `make lint` - Run formatting and go vet
- `make fmt` - Format Go code
- `make vet` - Run go vet

### Dependencies and Cleanup
- `make deps` - Download and tidy dependencies
- `make clean` - Clean build artifacts

### Running the Application
- `make run` - Build locally and run with --dry-run --help
- `./media-converter --dry-run /source /dest` - Test run without conversion
- `./media-converter /source /dest` - Basic conversion

## Architecture Overview

This is a secure, parallel media converter written in Go that converts images to modern formats (AVIF, WebP) and videos to efficient codecs (H.265, AV1).

### Core Architecture

**Entry Point**: `main.go` → `cmd/root.go` - CLI interface using Cobra
**Configuration**: `internal/config/` - Viper-based config with YAML file support (`$HOME/.media-converter.yaml`)
**Conversion Engine**: `internal/converter/` - Main conversion orchestrator with parallel processing
**Security Layer**: `internal/security/` - Disk space checks, file integrity verification, timeouts
**Logging**: `internal/logger/` - Structured logging to console and file
**Utilities**: `internal/utils/` - File handling, dependency checks, format detection

### Key Components

**Converter (`internal/converter/converter.go`)**:
- Orchestrates the entire conversion process
- Manages worker pools with semaphores for parallel processing
- Tracks progress and statistics
- Runs safety tests before batch processing

**Image Conversion (`internal/converter/image.go`)**:
- Handles photo conversions using ImageMagick
- Supports AVIF and WebP output formats
- Quality settings per format

**Video Conversion (`internal/converter/video.go`)**:
- Handles video conversions using FFmpeg
- Progress tracking with time-based progress bars
- AWS S3 cost estimation
- Supports H.265, H.264, AV1 codecs

**Security Checker (`internal/security/security.go`)**:
- File integrity verification
- Disk space monitoring (platform-specific implementations)
- Conversion timeouts
- Output file validation

### Processing Flow

1. **Initialization**: Validate directories, check dependencies, initialize logger
2. **Safety Test**: Test conversion on a sample file (unless dry-run)
3. **File Discovery**: Walk source directory and categorize files by type
4. **Parallel Conversion**: Use worker pools to convert files concurrently
5. **Progress Tracking**: Real-time progress bars and ETA calculations
6. **Final Report**: Statistics, timing, and cost estimations

### Configuration System

The application uses a layered configuration approach:
1. Default values in code
2. YAML config file (`$HOME/.media-converter.yaml`)
3. Environment variables
4. Command line flags (highest priority)

All configuration is managed through Viper and bound to Cobra flags.

### Dependencies

**Required External Tools**:
- FFmpeg (video conversion)
- ImageMagick (image conversion)

**Go Dependencies**:
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `github.com/sirupsen/logrus` - Logging
- `github.com/fatih/color` - Colored output
- `golang.org/x/sync` - Semaphore for worker pools

## File Organization

**Output Structure**: Files are organized by date (YYYY/MM-Month/YYYY-MM-DD/) with separate `images/` and `videos/` subdirectories. Month names support multiple languages (EN, FR, ES, DE).

**Logging**: All operations are logged to `{destination}/conversion.log` with detailed timing and error information.

**Security**: The application prioritizes safety with multiple verification steps, timeouts, and optional original file preservation.

## Safety and Idempotence Principles

This converter is designed to be **100% safe and idempotent**:

### Idempotent Operations
- **Run multiple times safely** - already converted files are automatically skipped via existence and integrity checks
- **Resume after interruption** - Ctrl+C, crashes, or failures can be resumed by re-running the same command
- **Atomic conversions** - files are converted to `.tmp` then renamed atomically, never overwriting originals during conversion

### Safety Guarantees
- **Originals preserved by default** (`--keep-originals=true`)
- **Triple verification before deletion**: file exists, integrity check, size validation
- **Processing markers** (`.processing` files) track active conversions with PID/timestamp
- **Automatic recovery** - detects and cleans up abandoned files from crashed processes
- **Corruption detection** - automatically re-converts files that fail integrity checks

### Recovery System
The converter includes comprehensive recovery mechanisms in `internal/security/security.go`:
- `FindAbandonedMarkers()` - detects processing markers from dead processes  
- `CleanupAbandonedFiles()` - removes `.tmp` files and abandoned markers
- `IsFileCorrupted()` - validates existing files and marks for re-conversion
- `VerifyOutputFile()` - comprehensive integrity verification using external tools

When resuming after interruption, the converter automatically:
1. Skips files that were already converted successfully
2. Cleans up temporary files from interrupted conversions  
3. Re-converts any files that were partially processed or corrupted
4. Continues from where it left off

### Date Extraction Logic
Critical for file organization: dates are extracted from file metadata (EXIF, video metadata) NOT filesystem timestamps. The `utils.GetFileDate()` function tries multiple methods in priority order:
1. macOS `mdls` metadata (most reliable for RAW files)
2. EXIF data via ImageMagick (`EXIF:DateTimeOriginal`)
3. Video metadata via FFmpeg (`creation_time`)
4. File modification time (fallback, with validation)

This ensures files are organized by their actual creation date, not when they were copied or modified.