<div align="center">
  <img src="https://github.com/Azilone/GoMediaMinify/blob/main/docs/images/logo.png" alt="GoMediaMinify Logo" width="120" height="120">
  
  # GoMediaMinify
  
  **Preserve your memories efficiently with modern codecs like AVIF and H.265. Great visual quality, minimal storage space.**
</div>

GoMediaMinify is an open-source command-line utility for shrinking everyday photo and video libraries with modern codecs like AVIF, WebP, H.265, and AV1. It keeps your originals untouched, organizes new files by capture date, and resumes large conversions automatically so home archives stay tidy without manual effort.

> ⚠️ GoMediaMinify uses modern **lossy** codecs (AVIF, H.265) to shrink your photos and videos. This means the output isn't pixel-for-pixel identical to RAW or original files, but it's visually very close while dramatically reducing size—perfect for cloud backups like Google Photos.

## Quick Start (Non-Technical Users)

**Just want to use it right away?** Download the ready-to-use program:

### 📥 Download (No Building Required)
1. Go to [Releases](https://github.com/Azilone/GoMediaMinify/releases)
2. Download the version for your system:
   - **Windows**: `media-converter-windows-amd64.exe`
   - **macOS Intel**: `media-converter-macos-intel` (older Macs)
   - **macOS Apple Silicon**: `media-converter-macos-apple-silicon` (M1/M2/M3/M4)
   - **Linux x64**: `media-converter-linux-amd64`
   - **Linux ARM**: `media-converter-linux-arm64` (Raspberry Pi, etc.)

### 🔧 Install Dependencies
Install FFmpeg (which also ships `ffprobe`) and ImageMagick:

- **macOS**: `brew install ffmpeg imagemagick`  
- **Windows**: Download [FFmpeg](https://ffmpeg.org/download.html) + [ImageMagick](https://imagemagick.org/script/download.php#windows)  
- **Linux**: `sudo apt install ffmpeg imagemagick`

### 🚀 Use It
```bash
# Test first (see what happens without changing anything)
./media-converter --dry-run /path/to/your/photos /path/to/converted

# Convert safely (keeps your originals)
./media-converter /path/to/your/photos /path/to/converted
```

## Real Example: Before & After

Here's what happens when you convert a typical SD card from a camera:

### Before Conversion
![SD Card Before](https://github.com/Azilone/GoMediaMinify/blob/main/docs/images/before-conversion.png)
*SD card with mixed photos and videos - +100GB*

### After Conversion  
![SD Card After](https://github.com/Azilone/GoMediaMinify/blob/main/docs/images/after-conversion.png)
*Same content converted - only 10GB used, organized by date*

**Result**: 90% space saved, photos organized by actual date taken, originals preserved safely.

## Purpose: Efficient Memory Backup

This tool is designed for **preserving personal memories** (family photos, vacation videos, etc.) using modern, efficient **lossy** codecs. It intentionally trades a little quality for huge space savings, making it ideal for cloud backups with providers like Google Photos. It's **not intended for professional RAW file storage** or pixel-perfect archival workflows.

### What It Does

✅ **Converts everyday photos**: JPG/HEIC → AVIF/WebP (up to 90% smaller, visually very close)
✅ **Compresses videos smartly**: MOV/MP4 → H.265/AV1 (60% smaller, great quality)
✅ **Organizes by date**: Uses photo metadata to sort by actual date taken
✅ **Keeps originals safe**: Never overwrites your files
✅ **Resume anywhere**: Stop and continue later without losing progress

### Ideal Use Cases ✅
- Household photo libraries that mix phone shots and camera images and need dependable backups.
- Trip or event footage where large 4K/HD videos benefit from smaller H.265/AV1 versions.
- Preparing libraries for cloud or NAS syncing when storage quotas are tight.

### Not Recommended For ❌
- Professional RAW file workflows
- Files requiring pixel-perfect accuracy
- Commercial photography archives
- Content requiring original formats  

## Memory Backup Examples

**Family Photo Collection**
```bash
./media-converter ~/Pictures/Family ~/Pictures/Family_Memories
# Result: 20GB → 6GB of precious memories, perfectly organized
```

**Vacation Videos**
```bash
./media-converter --video-codec=h265 ~/Videos/Vacation ~/Videos/Vacation_Memories  
# Result: 4K vacation videos 50% smaller, same beautiful quality
```

**Phone Camera Backup**
```bash
./media-converter /Volumes/iPhone_Photos ~/Desktop/Phone_Memories
# Result: Years of photos organized by date, huge space savings
```

## Supported Formats

**Input**: JPG, HEIC, HEIF, CR2, ARW, NEF, DNG, TIFF, PNG, RAW, BMP, GIF, WebP, MOV, MP4, AVI, MKV, M4V, MTS, M2TS, MPG, MPEG, WMV, FLV, 3GP

**Output**: AVIF, WebP (photos) • H.265, H.264, AV1 (videos)

## Safety Features

- **Never lose files**: Originals are preserved by default
- **Crash-proof**: Interrupt anytime, resume exactly where you left off  
- **Test mode**: `--dry-run` shows what will happen without doing it
- **Atomic operations**: Files are either perfect or untouched
- **Auto-recovery**: Cleans up if something goes wrong

## Installation Options

### Option 1: Download Binary (Easiest)
Go to [Releases](https://github.com/Azilone/GoMediaMinify/releases) and download for your platform.

### Option 2: Build from Source
```bash
git clone https://github.com/Azilone/GoMediaMinify.git
cd GoMediaMinify
go build -o media-converter
```

## Configuration

### Basic Options

| Option | Default | Description |
|--------|---------|-------------|
| `--dry-run` | false | Preview without converting |
| `--keep-originals` | true | Preserve original files |
| `--jobs` | CPU-1 | Number of parallel jobs |
| `--photo-format` | avif | Photo output (avif, webp) |
| `--video-codec` | h265 | Video codec (h265, h264, av1) |
| `--organize-by-date` | true | Organize by date |
| `--language` | en | Month names (en, fr, es, de) |

### Config File (`$HOME/.media-converter.yaml`)
```yaml
dry_run: false
keep_originals: true
max_jobs: 4
photo_format: "avif"
photo_quality_avif: 80
video_codec: "h265"
organize_by_date: true
language: "en"
adaptive_workers:
  enabled: true
  min: 1
  max: 6
  cpu_high: 80
  cpu_low: 50
  mem_low_percent: 20
  interval_seconds: 3
```

### Adaptive Worker Mode

Use `--adaptive-workers` (or the `adaptive_workers` config block) when you want the converter to automatically scale the number of simultaneous video conversions based on current system load. The monitor checks CPU load averages and free memory every few seconds and nudges the worker limit between the configured `min` and `max` bounds:

- Reduce workers when CPU stays above `cpu_high` or free memory drops below `mem_low_percent`.
- Increase workers after two calm readings (CPU under `cpu_low`, memory comfortably above the threshold).

This keeps long video batches responsive on laptops without micro-managing job counts. Omit the block entirely to keep the traditional fixed limit.

## Output Structure

With date organization (default):
```
converted/
├── 2024/
│   ├── 01-January/
│   │   ├── 2024-01-15/
│   │   │   ├── images/
│   │   │   └── videos/
└── conversion.log
```

## Progress Tracking

Real-time progress with time estimates:
```
🎬 vacation_video.mov (245.3 MB)
   [████████████░░░░░░░░░░░░░░░░░░] 42.3% | Speed: 1.2x | ETA: 3m24s

📈 Overall: [█████████░░░░░░░░░░░░░░░░] 15/40 (37.5%) | ETA: 12m30s
```

## Advanced Usage

### Custom Quality Settings
```bash
# High quality for archival
./media-converter --photo-quality-avif=90 --video-crf=23 ~/Photos ~/Archive

# Space-saving mode
./media-converter --photo-quality-avif=70 --video-crf=32 ~/Photos ~/Compressed
```

### Resume Interrupted Conversion
```bash
# If conversion is interrupted, just run the same command again
./media-converter ~/Photos ~/Photos_Converted
# ✅ Automatically skips completed files and continues
```

## Troubleshooting

**Missing dependencies**:
```bash
# Check if installed
ffmpeg -version
ffprobe -version
magick -version

# Install on macOS
brew install ffmpeg imagemagick
```

**Not enough space**: The tool needs about 50% of your source folder size for temporary files during conversion.

**Large files timing out**: Increase timeout with `--timeout-video=3600`


## Development

### Build
```bash
go build -o media-converter

# Cross-compile
GOOS=windows GOARCH=amd64 go build -o media-converter.exe
GOOS=linux GOARCH=amd64 go build -o media-converter-linux
```

### Project Structure
```
├── main.go              # Entry point
├── cmd/root.go         # CLI interface
├── internal/
│   ├── converter/      # Conversion logic
│   ├── security/       # Safety checks
│   └── utils/          # File handling
```

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch  
3. Make your changes
4. Submit a pull request

---

*Built for preserving your precious memories efficiently. Modern codecs, great quality, minimal space.*
