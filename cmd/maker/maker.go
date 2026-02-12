package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/baoswarm/baobun/internal/core"
)

func main() {
	trackers := []string{"bao.06b77fc89d2b9785433dd37a9b98a3c8fa37f03db2b2cc0e79be76f87b223d21"}

	inputPath, outputPath, err := resolvePaths()
	if err != nil {
		log.Fatal(err)
	}

	file, err := core.CreateFromFile(inputPath, trackers)
	if err != nil {
		log.Fatalf("failed to create .bao from %s: %v", inputPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		log.Fatalf("failed to create output directory: %v", err)
	}

	if err := file.Save(outputPath); err != nil {
		log.Fatalf("failed to save .bao file to %s: %v", outputPath, err)
	}

	log.Printf("Saved .bao file: %s", outputPath)
}

func resolvePaths() (string, string, error) {
	candidates := []struct {
		input  string
		output string
	}{
		{
			input:  filepath.Clean("./downloads/BigBuckBunny_320x180.mp4"),
			output: filepath.Clean("./test.bao"),
		},
		{
			input:  filepath.Clean("./BaoBun/downloads/BigBuckBunny_320x180.mp4"),
			output: filepath.Clean("./BaoBun/test.bao"),
		},
	}

	for _, c := range candidates {
		if _, err := os.Stat(c.input); err == nil {
			return c.input, c.output, nil
		}
	}

	return "", "", fmt.Errorf(
		"source file not found. looked for %q and %q",
		candidates[0].input,
		candidates[1].input,
	)
}
