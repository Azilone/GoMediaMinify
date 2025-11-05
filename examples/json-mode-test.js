#!/usr/bin/env node

/**
 * Example: Testing JSON mode output from media-converter
 *
 * This script demonstrates how to integrate media-converter with a Node.js application
 * using the --json-mode flag for structured output.
 *
 * Usage:
 *   node json-mode-test.js /path/to/source /path/to/dest
 */

const { spawn } = require('child_process');
const path = require('path');

// ANSI color codes for terminal output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  red: '\x1b[31m',
  cyan: '\x1b[36m',
};

function log(message, color = colors.reset) {
  console.log(`${color}${message}${colors.reset}`);
}

function formatBytes(bytes) {
  const mb = bytes / (1024 * 1024);
  return `${mb.toFixed(1)} MB`;
}

function formatDuration(duration) {
  // duration is like "5m30s" or "45.2s"
  return duration;
}

// Parse command line arguments
const args = process.argv.slice(2);
if (args.length < 2) {
  console.error('Usage: node json-mode-test.js <source> <destination>');
  console.error('Example: node json-mode-test.js ./photos ./photos-converted');
  process.exit(1);
}

const [sourceDir, destDir] = args;

// Path to media-converter binary (adjust as needed)
const binaryPath = path.join(__dirname, '..', 'media-converter');

log(`\n${'='.repeat(60)}`, colors.bright);
log('Media Converter - JSON Mode Test', colors.bright);
log('='.repeat(60), colors.bright);

// Spawn the converter process
const converter = spawn(binaryPath, [
  '--json-mode',
  '--dry-run', // Safe mode for testing
  sourceDir,
  destDir
]);

let totalFiles = 0;
let processedFiles = 0;
let startTime = Date.now();

// Process stdout (JSON events)
converter.stdout.on('data', (data) => {
  // Split by newlines in case multiple events arrive together
  const lines = data.toString().split('\n').filter(line => line.trim());

  lines.forEach(line => {
    try {
      const event = JSON.parse(line);
      const timestamp = new Date(event.timestamp).toLocaleTimeString();

      switch (event.type) {
        case 'started':
          log(`\n[${timestamp}] Conversion Started`, colors.cyan);
          log(`  Source: ${event.data.source_dir}`, colors.blue);
          log(`  Destination: ${event.data.dest_dir}`, colors.blue);
          log(`  Total files: ${event.data.total_files}`, colors.blue);
          log(`  Mode: ${event.data.mode}`, colors.blue);
          log(`  Dry run: ${event.data.dry_run}`, colors.yellow);
          totalFiles = event.data.total_files;
          break;

        case 'progress':
          const percent = event.data.progress_percent.toFixed(1);
          const eta = event.data.eta || 'calculating...';
          process.stdout.write(`\r${colors.green}Progress: ${percent}% (${event.data.processed_files}/${event.data.total_files}) ETA: ${eta}${colors.reset}`);
          break;

        case 'file_start':
          const fileType = event.data.file_type === 'image' ? '📸' : '🎬';
          const operation = event.data.operation;
          log(`\n  ${fileType} ${operation}: ${event.data.file_name} (${formatBytes(event.data.file_size)})`, colors.blue);
          break;

        case 'file_end':
          if (event.data.success) {
            const savings = event.data.compression_ratio
              ? ` (${(event.data.compression_ratio * 100).toFixed(1)}% saved)`
              : '';
            log(`    ✓ Completed in ${event.data.duration}${savings}`, colors.green);
            processedFiles++;
          } else {
            log(`    ✗ Failed: ${event.data.error_message}`, colors.red);
          }
          break;

        case 'log':
          const levelColors = {
            info: colors.cyan,
            warn: colors.yellow,
            error: colors.red,
            debug: colors.blue,
          };
          const levelColor = levelColors[event.data.level] || colors.reset;
          log(`\n[${event.data.level.toUpperCase()}] ${event.data.message}`, levelColor);
          break;

        case 'error':
          log(`\n[ERROR] ${event.data.message}`, colors.red);
          if (event.data.file_path) {
            log(`  File: ${event.data.file_path}`, colors.red);
          }
          if (event.data.fatal) {
            log('  This is a fatal error!', colors.red);
          }
          break;

        case 'complete':
          const duration = (Date.now() - startTime) / 1000;
          log(`\n\n${'='.repeat(60)}`, colors.bright);
          log('Conversion Complete!', colors.green + colors.bright);
          log('='.repeat(60), colors.bright);
          log(`  Success: ${event.data.success ? '✓' : '✗'}`, event.data.success ? colors.green : colors.red);
          log(`  Total files: ${event.data.total_files}`, colors.cyan);
          log(`  Processed: ${event.data.processed_files}`, colors.green);
          if (event.data.failed_files > 0) {
            log(`  Failed: ${event.data.failed_files}`, colors.red);
          }
          if (event.data.skipped_files > 0) {
            log(`  Skipped: ${event.data.skipped_files}`, colors.yellow);
          }
          log(`  Duration: ${event.data.total_duration}`, colors.cyan);
          break;

        case 'statistics':
          log(`\n📊 Statistics:`, colors.bright);
          if (event.data.images_converted > 0) {
            log(`  Images converted: ${event.data.images_converted}`, colors.cyan);
          }
          if (event.data.videos_converted > 0) {
            log(`  Videos converted: ${event.data.videos_converted}`, colors.cyan);
          }
          if (event.data.files_copied > 0) {
            log(`  Files copied: ${event.data.files_copied}`, colors.cyan);
          }
          if (event.data.duplicates_found > 0) {
            log(`  Duplicates found: ${event.data.duplicates_found}`, colors.yellow);
          }
          log(`  Input size: ${formatBytes(event.data.total_input_size)}`, colors.blue);
          log(`  Output size: ${formatBytes(event.data.total_output_size)}`, colors.blue);
          log(`  Space saved: ${formatBytes(event.data.space_saved)} (${(event.data.compression_ratio * 100).toFixed(1)}%)`, colors.green);
          log(`  Duration: ${event.data.total_duration}`, colors.cyan);
          break;

        default:
          log(`\nUnknown event type: ${event.type}`, colors.yellow);
      }
    } catch (e) {
      console.error('Failed to parse JSON line:', e.message);
      console.error('Line was:', line);
    }
  });
});

// Process stderr
converter.stderr.on('data', (data) => {
  log(`\n[STDERR] ${data.toString().trim()}`, colors.red);
});

// Process exit
converter.on('close', (code) => {
  log(`\n\n${'='.repeat(60)}`, colors.bright);
  log(`Process exited with code ${code}`, code === 0 ? colors.green : colors.red);
  log('='.repeat(60) + '\n', colors.bright);
  process.exit(code);
});

// Handle termination signals
process.on('SIGINT', () => {
  log('\n\nReceived SIGINT, terminating converter...', colors.yellow);
  converter.kill('SIGINT');
});

process.on('SIGTERM', () => {
  log('\n\nReceived SIGTERM, terminating converter...', colors.yellow);
  converter.kill('SIGTERM');
});
