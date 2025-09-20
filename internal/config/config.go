package config

import (
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	// Directories
	SourceDir string
	DestDir   string

	// Processing
	MaxJobs int
	DryRun  bool

	// Image settings
	PhotoFormat      string
	PhotoQualityAVIF int
	PhotoQualityWebP int

	// Video settings
	VideoCodec        string
	VideoCRF          int
	VideoAcceleration bool

	// Organization
	OrganizeByDate bool
	KeepOriginals  bool
	Language       string

	// Security
	ConversionTimeoutPhoto time.Duration
	ConversionTimeoutVideo time.Duration
	MinOutputSizeRatio     float64
	MinOutputSizeRatioAVIF float64
	MinOutputSizeRatioWebP float64

	// Supported formats
	PhotoFormats []string
	VideoFormats []string

	// Adaptive worker management
	AdaptiveWorkers AdaptiveWorkerConfig
}

type AdaptiveWorkerConfig struct {
	Enabled       bool
	MinWorkers    int
	MaxWorkers    int
	CPUHigh       float64
	CPULow        float64
	MemLowPercent float64
	CheckInterval time.Duration
}

func NewConfig() *Config {
	// Set default values for viper
	viper.SetDefault("max_jobs", runtime.NumCPU()-2)
	viper.SetDefault("dry_run", false)
	viper.SetDefault("photo_format", "avif")
	viper.SetDefault("photo_quality_avif", 80)
	viper.SetDefault("photo_quality_webp", 85)
	viper.SetDefault("video_codec", "h265")
	viper.SetDefault("video_crf", 28)
	viper.SetDefault("video_acceleration", true)
	viper.SetDefault("organize_by_date", true)
	viper.SetDefault("keep_originals", true)
	viper.SetDefault("timeout_photo", 300)
	viper.SetDefault("timeout_video", 1800)
	viper.SetDefault("min_output_size_ratio", 0.005)
	viper.SetDefault("min_output_size_ratio_avif", 0.001)
	viper.SetDefault("min_output_size_ratio_webp", 0.003)
	viper.SetDefault("language", "en")
	viper.SetDefault("adaptive_workers.enabled", false)
	viper.SetDefault("adaptive_workers.min", 1)
	viper.SetDefault("adaptive_workers.max", 6)
	viper.SetDefault("adaptive_workers.cpu_high", 80.0)
	viper.SetDefault("adaptive_workers.cpu_low", 50.0)
	viper.SetDefault("adaptive_workers.mem_low_percent", 20.0)
	viper.SetDefault("adaptive_workers.interval_seconds", 3)

	cfg := &Config{
		MaxJobs:                viper.GetInt("max_jobs"),
		DryRun:                 viper.GetBool("dry_run"),
		PhotoFormat:            viper.GetString("photo_format"),
		PhotoQualityAVIF:       viper.GetInt("photo_quality_avif"),
		PhotoQualityWebP:       viper.GetInt("photo_quality_webp"),
		VideoCodec:             viper.GetString("video_codec"),
		VideoCRF:               viper.GetInt("video_crf"),
		VideoAcceleration:      viper.GetBool("video_acceleration"),
		OrganizeByDate:         viper.GetBool("organize_by_date"),
		KeepOriginals:          viper.GetBool("keep_originals"),
		Language:               strings.ToLower(viper.GetString("language")),
		ConversionTimeoutPhoto: time.Duration(viper.GetInt("timeout_photo")) * time.Second,
		ConversionTimeoutVideo: time.Duration(viper.GetInt("timeout_video")) * time.Second,
		MinOutputSizeRatio:     viper.GetFloat64("min_output_size_ratio"),
		MinOutputSizeRatioAVIF: viper.GetFloat64("min_output_size_ratio_avif"),
		MinOutputSizeRatioWebP: viper.GetFloat64("min_output_size_ratio_webp"),
		PhotoFormats: []string{
			"jpg", "jpeg", "heic", "heif", "cr2", "arw", "nef", "dng",
			"tiff", "tif", "png", "raw", "bmp", "gif", "webp",
		},
		VideoFormats: []string{
			"mov", "mp4", "avi", "mkv", "m4v", "mts", "m2ts", "mpg",
			"mpeg", "wmv", "flv", "3gp", "3gpp",
		},
		AdaptiveWorkers: AdaptiveWorkerConfig{
			Enabled:       viper.GetBool("adaptive_workers.enabled"),
			MinWorkers:    viper.GetInt("adaptive_workers.min"),
			MaxWorkers:    viper.GetInt("adaptive_workers.max"),
			CPUHigh:       viper.GetFloat64("adaptive_workers.cpu_high"),
			CPULow:        viper.GetFloat64("adaptive_workers.cpu_low"),
			MemLowPercent: viper.GetFloat64("adaptive_workers.mem_low_percent"),
			CheckInterval: time.Duration(viper.GetInt("adaptive_workers.interval_seconds")) * time.Second,
		},
	}

	// Validate max jobs
	if cfg.MaxJobs < 1 {
		cfg.MaxJobs = 1
	}
	if cfg.MaxJobs > runtime.NumCPU() {
		cfg.MaxJobs = runtime.NumCPU()
	}

	// When the flag is explicitly set to 0, fall back to sensible defaults.
	if cfg.MinOutputSizeRatio <= 0 {
		cfg.MinOutputSizeRatio = 0.005
	}
	if cfg.MinOutputSizeRatioAVIF <= 0 {
		cfg.MinOutputSizeRatioAVIF = 0.001
	}
	if cfg.MinOutputSizeRatioWebP <= 0 {
		cfg.MinOutputSizeRatioWebP = 0.003
	}

	if cfg.Language == "" {
		cfg.Language = "en"
	}

	// Sanitize adaptive worker settings
	if cfg.AdaptiveWorkers.MinWorkers < 1 {
		cfg.AdaptiveWorkers.MinWorkers = 1
	}
	if cfg.AdaptiveWorkers.MaxWorkers < cfg.AdaptiveWorkers.MinWorkers {
		cfg.AdaptiveWorkers.MaxWorkers = cfg.AdaptiveWorkers.MinWorkers
	}
	if cfg.AdaptiveWorkers.CheckInterval <= 0 {
		cfg.AdaptiveWorkers.CheckInterval = 3 * time.Second
	}
	// Respect global job ceiling if provided
	if cfg.AdaptiveWorkers.MaxWorkers > cfg.MaxJobs {
		cfg.AdaptiveWorkers.MaxWorkers = cfg.MaxJobs
	}
	if cfg.AdaptiveWorkers.MinWorkers > cfg.AdaptiveWorkers.MaxWorkers {
		cfg.AdaptiveWorkers.MinWorkers = cfg.AdaptiveWorkers.MaxWorkers
	}
	if cfg.AdaptiveWorkers.CPUHigh <= 0 {
		cfg.AdaptiveWorkers.CPUHigh = 80.0
	}
	if cfg.AdaptiveWorkers.CPULow <= 0 || cfg.AdaptiveWorkers.CPULow >= cfg.AdaptiveWorkers.CPUHigh {
		cfg.AdaptiveWorkers.CPULow = cfg.AdaptiveWorkers.CPUHigh * 0.6
	}
	if cfg.AdaptiveWorkers.MemLowPercent <= 0 {
		cfg.AdaptiveWorkers.MemLowPercent = 20.0
	}

	return cfg
}
