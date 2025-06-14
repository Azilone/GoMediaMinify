package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/kevindurb/media-converter/internal/config"
	"github.com/kevindurb/media-converter/internal/converter"
	"github.com/kevindurb/media-converter/internal/logger"
	"github.com/kevindurb/media-converter/internal/utils"
)

var (
	cfg     *config.Config
	log     *logger.Logger
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use:   "media-converter [source] [destination]",
	Short: "Secure parallel media converter for images and videos",
	Long: `A secure, parallel media converter that converts images to modern formats (AVIF, WebP) 
and videos to efficient codecs (H.265, AV1) with built-in safety checks and file organization.`,
	Args: cobra.ExactArgs(2),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize configuration
		cfg = config.NewConfig()
		cfg.SourceDir = args[0]
		cfg.DestDir = args[1]

		// Validate directories
		if _, err := os.Stat(cfg.SourceDir); os.IsNotExist(err) {
			return fmt.Errorf("source directory does not exist: %s", cfg.SourceDir)
		}

		// Create destination directory if it doesn't exist
		if err := os.MkdirAll(cfg.DestDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}

		// Initialize logger
		logPath := filepath.Join(cfg.DestDir, "conversion.log")
		var err error
		log, err = logger.NewLogger(logPath)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		// Check dependencies
		if err := utils.CheckDependencies(); err != nil {
			return fmt.Errorf("dependency check failed: %w", err)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		defer log.Close()

		// Show header
		log.ShowHeader(cfg.KeepOriginals)

		// Initialize converter
		conv := converter.NewConverter(cfg, log)

		// Run conversion
		return conv.Convert()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Configuration file flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.media-converter.yaml)")

	// Core flags
	rootCmd.Flags().BoolP("dry-run", "n", false, "Show what would be converted without actually converting")
	rootCmd.Flags().BoolP("keep-originals", "k", true, "Keep original files after conversion")
	rootCmd.Flags().IntP("jobs", "j", 0, "Number of parallel jobs (default: CPU cores - 1)")

	// Image conversion flags
	rootCmd.Flags().String("photo-format", "avif", "Output format for photos (avif, webp)")
	rootCmd.Flags().Int("photo-quality-avif", 80, "Quality for AVIF images (1-100)")
	rootCmd.Flags().Int("photo-quality-webp", 85, "Quality for WebP images (1-100)")

	// Video conversion flags
	rootCmd.Flags().String("video-codec", "h265", "Video codec (h265, h264, av1)")
	rootCmd.Flags().Int("video-crf", 28, "Video CRF value (lower = better quality)")

	// Organization flags
	rootCmd.Flags().BoolP("organize-by-date", "o", true, "Organize files by date")
	rootCmd.Flags().String("language", "fr", "Language for month names (en, fr, es, de)")

	// Security flags
	rootCmd.Flags().Int("timeout-photo", 300, "Timeout for photo conversion in seconds")
	rootCmd.Flags().Int("timeout-video", 1800, "Timeout for video conversion in seconds")
	rootCmd.Flags().Float64("min-output-ratio", 0.0, "Minimum output size ratio (0.0 uses format-specific defaults)")

	// Bind flags to viper
	viper.BindPFlag("dry_run", rootCmd.Flags().Lookup("dry-run"))
	viper.BindPFlag("keep_originals", rootCmd.Flags().Lookup("keep-originals"))
	viper.BindPFlag("max_jobs", rootCmd.Flags().Lookup("jobs"))
	viper.BindPFlag("photo_format", rootCmd.Flags().Lookup("photo-format"))
	viper.BindPFlag("photo_quality_avif", rootCmd.Flags().Lookup("photo-quality-avif"))
	viper.BindPFlag("photo_quality_webp", rootCmd.Flags().Lookup("photo-quality-webp"))
	viper.BindPFlag("video_codec", rootCmd.Flags().Lookup("video-codec"))
	viper.BindPFlag("video_crf", rootCmd.Flags().Lookup("video-crf"))
	viper.BindPFlag("organize_by_date", rootCmd.Flags().Lookup("organize-by-date"))
	viper.BindPFlag("language", rootCmd.Flags().Lookup("language"))
	viper.BindPFlag("timeout_photo", rootCmd.Flags().Lookup("timeout-photo"))
	viper.BindPFlag("timeout_video", rootCmd.Flags().Lookup("timeout-video"))
	viper.BindPFlag("min_output_size_ratio", rootCmd.Flags().Lookup("min-output-ratio"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".media-converter")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
