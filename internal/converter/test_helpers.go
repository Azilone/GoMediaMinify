package converter

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
)

// isImageMagickAvailable checks if ImageMagick is installed
func isImageMagickAvailable() bool {
	cmd := exec.Command("magick", "-version")
	err := cmd.Run()
	return err == nil
}

// isFFmpegAvailable checks if FFmpeg is installed
func isFFmpegAvailable() bool {
	cmd := exec.Command("ffmpeg", "-version")
	err := cmd.Run()
	return err == nil
}

// createTestImage creates a simple test JPEG image
func createTestImage(path string) error {
	// Create a simple 100x100 red image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, red)
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
}

// createTestVideo creates a simple test video using FFmpeg
func createTestVideo(path string) error {
	// Create a 1-second test video with FFmpeg
	// This creates a simple colored video without requiring input files
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", "color=c=blue:s=320x240:d=1",
		"-pix_fmt", "yuv420p",
		"-y", // Overwrite output file
		path,
	)

	return cmd.Run()
}
