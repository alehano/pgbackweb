package storage

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/eduardolat/pgbackweb/internal/logger"
	"github.com/eduardolat/pgbackweb/internal/util/strutil"
)

const (
	localBackupsDir string = "/backups"
)

// localRelPath cleans a path relative to the local backups directory and
// rejects any path that would end up outside of it (for example "../x").
func localRelPath(relativeFilePath string) (string, error) {
	rel := filepath.Clean(strutil.CreatePath(false, relativeFilePath))
	if rel == "." || rel == ".." || filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf(
			"path %q is outside of %s", relativeFilePath, localBackupsDir,
		)
	}
	return rel, nil
}

// openBackupsRoot opens the backups directory as an os.Root. All file
// operations through it stay inside the directory, also through symlinks.
func openBackupsRoot() (*os.Root, error) {
	root, err := os.OpenRoot(localBackupsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", localBackupsDir, err)
	}
	return root, nil
}

// LocalUpload Creates a new file using the provided path and reader relative
// to the local backups directory.
//
// Returns the size of the file created, in bytes. On error, the partial file
// is removed.
func (Client) LocalUpload(relativeFilePath string, fileReader io.Reader) (size int64, err error) {
	rel, err := localRelPath(relativeFilePath)
	if err != nil {
		return 0, err
	}
	fullPath := filepath.Join(localBackupsDir, rel)

	root, err := openBackupsRoot()
	if err != nil {
		return 0, err
	}
	defer root.Close()

	if dir := filepath.Dir(rel); dir != "." {
		if err := root.MkdirAll(dir, 0o755); err != nil {
			return 0, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	file, err := root.Create(rel)
	if err != nil {
		return 0, fmt.Errorf("failed to create file %s: %w", fullPath, err)
	}
	defer func() {
		if err != nil {
			_ = file.Close()
			_ = root.Remove(rel)
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
	rel, err := localRelPath(relativeFilePath)
	if err != nil {
		return err
	}

	root, err := openBackupsRoot()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	defer root.Close()

	err = root.Remove(rel)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to delete file %s/%s: %w", localBackupsDir, rel, err)
	}

	return nil
}

// LocalGetFullPath Returns the full path of a file using the provided relative
// file path to the local backups directory. The path is checked after
// symlinks are resolved, so it never points outside of the directory.
func (Client) LocalGetFullPath(relativeFilePath string) string {
	invalid := localBackupsDir + "/invalid-path"

	rel, err := localRelPath(relativeFilePath)
	if err != nil {
		return invalid
	}
	fullPath := filepath.Join(localBackupsDir, rel)

	resolved, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		return fullPath // does not exist (yet): nothing to follow
	}
	base, err := filepath.EvalSymlinks(localBackupsDir)
	if err != nil {
		return invalid
	}
	if resolved != base &&
		!strings.HasPrefix(resolved, base+string(filepath.Separator)) {
		return invalid
	}
	return resolved
}

// LocalCheckWritable logs a warning when the local backups directory exists
// but the app cannot write to it (for example a volume created by an older
// image that ran as root). Local backups would fail in that case.
func (Client) LocalCheckWritable() {
	if _, err := os.Stat(localBackupsDir); err != nil {
		return
	}
	f, err := os.CreateTemp(localBackupsDir, ".pbw-write-test-*")
	if err != nil {
		logger.Warn(
			"local backups directory is not writable; local backups will fail. "+
				"Fix the owner on the host, e.g. chown -R 10001:10001 ./backups",
			logger.KV{"dir": localBackupsDir, "error": err.Error(), "uid": os.Getuid()},
		)
		return
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
}
