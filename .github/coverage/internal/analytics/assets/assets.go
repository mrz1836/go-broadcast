// Package assets provides embedded web assets for coverage analytics reports and dashboards.
package assets

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FS embeds all static assets for the coverage analytics system.
// This includes CSS, images, and web manifest files that are deployed
// alongside generated coverage reports.
//
//go:embed css/*.css images/*.ico images/*.svg js/*.js *.webmanifest
var FS embed.FS

// CopyAssetsTo copies all embedded assets to the specified output directory,
// maintaining the directory structure.
func CopyAssetsTo(outputDir string) error {
	// Create the assets directory in the output location
	assetsDir := filepath.Join(outputDir, "assets")
	if err := os.MkdirAll(assetsDir, 0o750); err != nil {
		return fmt.Errorf("creating assets directory: %w", err)
	}

	// Walk through all embedded files and copy them
	return fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == "." {
			return nil
		}

		// Determine the output path
		outputPath := filepath.Join(assetsDir, path)

		// If it's a directory, create it
		if d.IsDir() {
			return os.MkdirAll(outputPath, 0o750)
		}

		// Read the embedded file
		data, err := fs.ReadFile(FS, path)
		if err != nil {
			return fmt.Errorf("reading embedded file %s: %w", path, err)
		}

		// Ensure the parent directory exists
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
			return fmt.Errorf("creating parent directory for %s: %w", outputPath, err)
		}

		// Write the file to the output directory
		if err := os.WriteFile(outputPath, data, 0o600); err != nil {
			return fmt.Errorf("writing file %s: %w", outputPath, err)
		}

		return nil
	})
}

// GetAsset returns the content of a specific embedded asset.
func GetAsset(path string) ([]byte, error) {
	data, err := fs.ReadFile(FS, path)
	if err != nil {
		return nil, fmt.Errorf("reading asset %s: %w", path, err)
	}
	return data, nil
}

// AssetExists checks if an asset exists in the embedded filesystem.
func AssetExists(path string) bool {
	_, err := fs.Stat(FS, path)
	return err == nil
}

// ListAssets returns a list of all embedded asset paths.
func ListAssets() ([]string, error) {
	var assets []string
	err := fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && path != "." {
			assets = append(assets, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking embedded assets: %w", err)
	}
	return assets, nil
}
