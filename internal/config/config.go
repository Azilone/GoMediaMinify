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

	return cfg
}
