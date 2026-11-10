package storage

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/eduardolat/pgbackweb/internal/util/strutil"
)

const (
	localBackupsDir string = "/backups"
)

// localFullPath joins the relative path to the local backups directory and
// rejects any path that would end up outside of it (for example "../x").
func localFullPath(relativeFilePath string) (string, error) {
	fullPath := filepath.Clean(
		strutil.CreatePath(true, localBackupsDir, relativeFilePath),
	)
	if fullPath != localBackupsDir &&
		!strings.HasPrefix(fullPath, localBackupsDir+string(filepath.Separator)) {
		return "", fmt.Errorf(
			"path %q is outside of %s", relativeFilePath, localBackupsDir,
		)
	}
	return fullPath, nil
}

// LocalUpload Creates a new file using the provided path and reader relative
// to the local backups directory.
//
// Returns the size of the file created, in bytes. On error, the partial file
// is removed.
func (Client) LocalUpload(relativeFilePath string, fileReader io.Reader) (size int64, err error) {
	fullPath, err := localFullPath(relativeFilePath)
	if err != nil {
		return 0, err
	}
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return 0, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create file %s: %w", fullPath, err)
	}
	defer func() {
		if err != nil {
			_ = file.Close()
			_ = os.Remove(fullPath)
		}
	}()

	size, err = io.Copy(file, fileReader)
	if err != nil {
		return 0, fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	// Flush to disk and check Close: a write error can show up only here.
	if err = file.Sync(); err != nil {
		return 0, fmt.Errorf("failed to sync file %s: %w", fullPath, err)
	}
	if err = file.Close(); err != nil {
		return 0, fmt.Errorf("failed to close file %s: %w", fullPath, err)
	}

	return size, nil
}

// LocalDelete Deletes a file using the provided path relative to the local
// backups directory. A file that does not exist is not an error, so one
// missing file cannot stop the retention cleanup.
func (Client) LocalDelete(relativeFilePath string) error {
	fullPath, err := localFullPath(relativeFilePath)
	if err != nil {
		return err
	}

	err = os.Remove(fullPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to delete file %s: %w", fullPath, err)
	}

	return nil
}

// LocalGetFullPath Returns the full path of a file using the provided relative
// file path to the local backups directory.
func (Client) LocalGetFullPath(relativeFilePath string) string {
	fullPath, err := localFullPath(relativeFilePath)
	if err != nil {
		// Never point outside of the backups directory.
		return localBackupsDir + "/invalid-path"
	}
	return fullPath
}
