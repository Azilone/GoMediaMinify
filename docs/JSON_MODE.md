# JSON Mode Documentation

## Overview

The JSON mode enables structured, machine-readable output for integration with desktop applications like Tauri. When enabled, the CLI emits JSON events to stdout while writing traditional logs to the log file.

## Usage

```bash
# Enable JSON mode
./media-converter --json-mode /source /destination

# Combine with other flags
./media-converter --json-mode --dry-run --copy-only /source /destination
```

## Event Stream Format

Each event is a single JSON object on one line (newline-delimited JSON):

```json
{"type":"started","timestamp":"2024-11-03T12:00:00Z","data":{...}}
{"type":"progress","timestamp":"2024-11-03T12:00:05Z","data":{...}}
{"type":"file_start","timestamp":"2024-11-03T12:00:06Z","data":{...}}
{"type":"file_end","timestamp":"2024-11-03T12:00:10Z","data":{...}}
{"type":"complete","timestamp":"2024-11-03T12:05:00Z","data":{...}}
```

## Event Types

### `started`
Emitted when conversion/copy begins.

```json
{
  "type": "started",
  "timestamp": "2024-11-03T12:00:00Z",
  "data": {
    "source_dir": "/path/to/source",
    "dest_dir": "/path/to/dest",
    "total_files": 150,
    "mode": "conversion",
    "dry_run": false,
    "keep_originals": true,
    "organize_by_date": true
  }
}
```

### `progress`
Emitted periodically during processing.

```json
{
  "type": "progress",
  "timestamp": "2024-11-03T12:00:30Z",
  "data": {
    "processed_files": 25,
    "total_files": 150,
    "progress_percent": 16.67,
    "eta": "5m30s"
  }
}
```

### `file_start`
Emitted when a file begins processing.

```json
{
  "type": "file_start",
  "timestamp": "2024-11-03T12:00:06Z",
  "data": {
    "file_path": "/source/IMG_1234.jpg",
    "file_name": "IMG_1234.jpg",
    "file_size": 5242880,
    "file_type": "image",
    "operation": "convert"
  }
}
```

### `file_end`
Emitted when a file completes processing.

```json
{
  "type": "file_end",
  "timestamp": "2024-11-03T12:00:10Z",
  "data": {
    "file_path": "/source/IMG_1234.jpg",
    "file_name": "IMG_1234.jpg",
    "success": true,
    "output_path": "/dest/2024/11-November/2024-11-03/images/IMG_1234.avif",
    "output_size": 524288,
    "compression_ratio": 0.9,
    "duration": "4.2s",
    "checksum": "3f8a9b2c4d5e6f7a"
  }
}
```

### `log`
General log messages.

```json
{
  "type": "log",
  "timestamp": "2024-11-03T12:00:15Z",
  "data": {
    "level": "info",
    "message": "Converting photos..."
  }
}
```

Levels: `info`, `warn`, `error`, `debug`

### `error`
Error events.

```json
{
  "type": "error",
  "timestamp": "2024-11-03T12:00:20Z",
  "data": {
    "message": "Failed to convert file: timeout",
    "file_path": "/source/large_video.mov",
    "fatal": false
  }
}
```

### `complete`
Emitted when conversion finishes.

```json
{
  "type": "complete",
  "timestamp": "2024-11-03T12:05:00Z",
  "data": {
    "success": true,
    "total_files": 150,
    "processed_files": 148,
    "failed_files": 2,
    "skipped_files": 0,
    "total_duration": "5m0s"
  }
}
```

### `statistics`
Detailed statistics (emitted after `complete`).

```json
{
  "type": "statistics",
  "timestamp": "2024-11-03T12:05:00Z",
  "data": {
    "images_converted": 120,
    "videos_converted": 28,
    "files_copied": 0,
    "files_skipped": 0,
    "duplicates_found": 0,
    "total_input_size": 5368709120,
    "total_output_size": 536870912,
    "space_saved": 4831838208,
    "compression_ratio": 0.9,
    "average_speed": "17.9 MB/s",
    "total_duration": "5m0s"
  }
}
```

## Integration Examples

### Node.js / JavaScript

```javascript
const { spawn } = require('child_process');

const converter = spawn('./media-converter', [
  '--json-mode',
  '/source',
  '/destination'
]);

converter.stdout.on('data', (data) => {
  // Parse newline-delimited JSON
  const lines = data.toString().split('\n').filter(l => l.trim());

  lines.forEach(line => {
    try {
      const event = JSON.parse(line);

      switch(event.type) {
        case 'started':
          console.log(`Starting conversion of ${event.data.total_files} files`);
          break;

        case 'progress':
          console.log(`Progress: ${event.data.progress_percent.toFixed(1)}%`);
          break;

        case 'file_end':
          if (event.data.success) {
            console.log(`✓ ${event.data.file_name}`);
          } else {
            console.error(`✗ ${event.data.file_name}: ${event.data.error_message}`);
          }
          break;

        case 'complete':
          console.log(`Done! ${event.data.processed_files}/${event.data.total_files} files`);
          break;
      }
    } catch (e) {
      console.error('Failed to parse JSON:', e);
    }
  });
});

converter.stderr.on('data', (data) => {
  console.error(`Error: ${data}`);
});

converter.on('close', (code) => {
  console.log(`Process exited with code ${code}`);
});
```

### Rust (for Tauri)

```rust
use std::process::{Command, Stdio};
use std::io::{BufRead, BufReader};
use serde_json::Value;

#[tauri::command]
async fn start_conversion(source: String, dest: String) -> Result<(), String> {
    let mut child = Command::new("./media-converter")
        .arg("--json-mode")
        .arg(source)
        .arg(dest)
        .stdout(Stdio::piped())
        .spawn()
        .map_err(|e| e.to_string())?;

    let stdout = child.stdout.take().ok_or("Failed to capture stdout")?;
    let reader = BufReader::new(stdout);

    for line in reader.lines() {
        let line = line.map_err(|e| e.to_string())?;

        if let Ok(event) = serde_json::from_str::<Value>(&line) {
            let event_type = event["type"].as_str().unwrap_or("unknown");

            match event_type {
                "progress" => {
                    // Emit to frontend via Tauri event
                    let progress = event["data"]["progress_percent"].as_f64().unwrap_or(0.0);
                    // app.emit_all("conversion-progress", progress)?;
                }
                "complete" => {
                    // Conversion finished
                    // app.emit_all("conversion-complete", event["data"])?;
                    break;
                }
                _ => {}
            }
        }
    }

    Ok(())
}
```

### Python

```python
import subprocess
import json
import sys

process = subprocess.Popen(
    ['./media-converter', '--json-mode', '/source', '/destination'],
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,
    text=True
)

for line in process.stdout:
    try:
        event = json.loads(line.strip())
        event_type = event['type']
        data = event['data']

        if event_type == 'started':
            print(f"Starting: {data['total_files']} files")
        elif event_type == 'progress':
            print(f"Progress: {data['progress_percent']:.1f}%")
        elif event_type == 'complete':
            print(f"Complete: {data['processed_files']} processed")

    except json.JSONDecodeError as e:
        print(f"JSON parse error: {e}", file=sys.stderr)

process.wait()
```

## Best Practices

1. **Parse line-by-line**: Events are newline-delimited, parse each line separately
2. **Handle parse errors**: Invalid JSON lines should be logged but not crash your app
3. **Buffer management**: Process stdout in real-time to avoid buffer overflow on large conversions
4. **Event ordering**: Events are emitted in chronological order, but `file_*` events may interleave
5. **Error handling**: Check both `error` events and the `complete.success` field
6. **Log file**: Traditional logs are still written to `conversion.log` for debugging

## Differences from Normal Mode

| Aspect | Normal Mode | JSON Mode |
|--------|-------------|-----------|
| stdout | Colorized human-readable logs | JSON events only |
| stderr | Error messages | Error messages |
| Log file | Same as stdout | Traditional logs only |
| Progress bars | Yes (ANSI) | No (use `progress` events) |
| Headers/banners | Yes | No |
| Colors | Yes | No |

## Troubleshooting

**No output received:**
- Ensure `--json-mode` flag is present
- Check that stdout is not being buffered (use line buffering)

**Malformed JSON:**
- Update to latest version
- Check for stderr output indicating errors
- Verify the subprocess is running the correct binary

**Missing events:**
- Some events are conditional (e.g., `error` only on failures)
- `statistics` is only emitted if files were processed

**Performance:**
- JSON mode has minimal overhead (~1-2%)
- Events are emitted in real-time, no batching
