package postgres

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/eduardolat/pgbackweb/internal/util/strutil"
	"github.com/orsinium-labs/enum"
)

/*
	Important:
	Versions supported by PG Back Web must be supported in PostgreSQL Version Policy
	https://www.postgresql.org/support/versioning/

	Backing up a database from an old unsupported version should not be allowed.
*/

type version struct {
	Version string
	PGDump  string
	PSQL    string
}

type PGVersion enum.Member[version]

var (
	PG13 = PGVersion{version{
		Version: "13",
		PGDump:  "/usr/lib/postgresql/13/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/13/bin/psql",
	}}
	PG14 = PGVersion{version{
		Version: "14",
		PGDump:  "/usr/lib/postgresql/14/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/14/bin/psql",
	}}
	PG15 = PGVersion{version{
		Version: "15",
		PGDump:  "/usr/lib/postgresql/15/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/15/bin/psql",
	}}
	PG16 = PGVersion{version{
		Version: "16",
		PGDump:  "/usr/lib/postgresql/16/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/16/bin/psql",
	}}
	PG17 = PGVersion{version{
		Version: "17",
		PGDump:  "/usr/lib/postgresql/17/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/17/bin/psql",
	}}
	PG18 = PGVersion{version{
		Version: "18",
		PGDump:  "/usr/lib/postgresql/18/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/18/bin/psql",
	}}

	PGVersions     = []PGVersion{PG13, PG14, PG15, PG16, PG17, PG18}
	PGVersionsDesc = []PGVersion{PG18, PG17, PG16, PG15, PG14, PG13}
)

type Client struct{}

func New() *Client {
	return &Client{}
}

// ParseVersion returns the PGVersion enum member for the given PostgreSQL
// version as a string.
func (Client) ParseVersion(version string) (PGVersion, error) {
	switch version {
	case "13":
		return PG13, nil
	case "14":
		return PG14, nil
	case "15":
		return PG15, nil
	case "16":
		return PG16, nil
	case "17":
		return PG17, nil
	case "18":
		return PG18, nil
	default:
		return PGVersion{}, fmt.Errorf("pg version not allowed: %s", version)
	}
}

// dbnameArg passes the connection string as --dbname=... so a value that
// starts with "-" can never be read as a command line option.
func dbnameArg(connString string) string {
	return "--dbname=" + connString
}

// Test tests the connection to the PostgreSQL database
func (Client) Test(version PGVersion, connString string) error {
	cmd := exec.Command(version.Value.PSQL, dbnameArg(connString), "-c", "SELECT 1;")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"error running psql test v%s: %s",
			version.Value.Version, output,
		)
	}

	return nil
}

// CheckDumpVersion returns an error when the server major version is newer
// than the selected pg_dump version. pg_dump refuses to dump a newer server,
// and the plain connection test (SELECT 1) does not catch this.
func (Client) CheckDumpVersion(version PGVersion, connString string) error {
	cmd := exec.Command(
		version.Value.PSQL, dbnameArg(connString), "-X", "-A", "-t",
		"-c", "SHOW server_version_num;",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"error reading server version with psql v%s: %s",
			version.Value.Version, output,
		)
	}

	serverNum, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return fmt.Errorf("error parsing server version %q: %w", output, err)
	}
	clientMajor, err := strconv.Atoi(version.Value.Version)
	if err != nil {
		return fmt.Errorf("error parsing client version: %w", err)
	}

	if serverMajor := serverNum / 10000; serverMajor > clientMajor {
		return fmt.Errorf(
			"server is PostgreSQL %d but PostgreSQL %d is selected for this "+
				"database; pg_dump cannot dump a newer server, select version %d "+
				"or newer", serverMajor, clientMajor, serverMajor,
		)
	}

	return nil
}

// DumpParams contains the parameters for the pg_dump command
type DumpParams struct {
	// DataOnly (--data-only): Dump only the data, not the schema (data definitions).
	// Table data, large objects, and sequence values are dumped.
	DataOnly bool

	// SchemaOnly (--schema-only): Dump only the object definitions (schema), not data.
	SchemaOnly bool

	// Clean (--clean): Output commands to DROP all the dumped database objects
	// prior to outputting the commands for creating them. This option is useful
	// when the restore is to overwrite an existing database. If any of the
	// objects do not exist in the destination database, ignorable error messages
	// will be reported during restore, unless --if-exists is also specified.
	Clean bool

	// IfExists (--if-exists): Use DROP ... IF EXISTS commands to drop objects in
	// --clean mode. This suppresses “does not exist” errors that might otherwise
	// be reported. This option is not valid unless --clean is also specified.
	IfExists bool

	// Create (--create): Begin the output with a command to create the database
	// itself and reconnect to the created database. (With a script of this form,
	// it doesn't matter which database in the destination installation you
	// connect to before running the script.) If --clean is also specified, the
	// script drops and recreates the target database before reconnecting to it.
	Create bool

	// NoComments (--no-comments): Do not dump comments.
	NoComments bool
}

// Dump runs the pg_dump command with the given parameters. It returns the SQL
// dump as an io.ReadCloser. Closing the reader before the end stops pg_dump.
// Cancelling ctx kills pg_dump.
func (Client) Dump(
	ctx context.Context, version PGVersion, connString string,
	params ...DumpParams,
) io.ReadCloser {
	pickedParams := DumpParams{}
	if len(params) > 0 {
		pickedParams = params[0]
	}

	args := []string{dbnameArg(connString)}
	if pickedParams.DataOnly {
		args = append(args, "--data-only")
	}
	if pickedParams.SchemaOnly {
		args = append(args, "--schema-only")
	}
	if pickedParams.Clean {
		args = append(args, "--clean")
	}
	if pickedParams.IfExists {
		args = append(args, "--if-exists")
	}
	if pickedParams.Create {
		args = append(args, "--create")
	}
	if pickedParams.NoComments {
		args = append(args, "--no-comments")
	}

	errorBuffer := &bytes.Buffer{}
	reader, writer := io.Pipe()
	cmd := exec.CommandContext(ctx, version.Value.PGDump, args...)
	cmd.Stdout = writer
	cmd.Stderr = errorBuffer

	go func() {
		if err := cmd.Run(); err != nil {
			msg := errorBuffer.String()
			if ctxErr := ctx.Err(); ctxErr != nil {
				msg = fmt.Sprintf("%s (%v)", msg, ctxErr)
			}
			writer.CloseWithError(fmt.Errorf(
				"error running pg_dump v%s: %s", version.Value.Version, msg,
			))
			return
		}
		writer.Close()
	}()

	return reader
}

// DumpZip runs the pg_dump command with the given parameters and returns the
// ZIP-compressed SQL dump as an io.ReadCloser.
//
// The caller must Close the reader, also when the upload fails. Close stops
// the ZIP goroutine and pg_dump, so no process keeps a connection (and a
// snapshot) open on the source database.
func (c *Client) DumpZip(
	ctx context.Context, version PGVersion, connString string,
	params ...DumpParams,
) io.ReadCloser {
	dumpReader := c.Dump(ctx, version, connString, params...)
	reader, writer := io.Pipe()

	go func() {
		defer dumpReader.Close()

		zipWriter := zip.NewWriter(writer)

		fileWriter, err := zipWriter.Create("dump.sql")
		if err != nil {
			writer.CloseWithError(fmt.Errorf("error creating zip file: %w", err))
			return
		}

		if _, err := io.Copy(fileWriter, dumpReader); err != nil {
			writer.CloseWithError(fmt.Errorf("error writing to zip file: %w", err))
			return
		}

		// Close writes the ZIP central directory. Without it the file is
		// not a valid ZIP, so the error must not be ignored.
		if err := zipWriter.Close(); err != nil {
			writer.CloseWithError(fmt.Errorf("error finalizing zip file: %w", err))
			return
		}

		writer.Close()
	}()

	return reader
}

// RestoreZip downloads or copies the ZIP from the given url or path, unzips it,
// and runs the psql command to restore the database.
//
// The ZIP file must contain a dump.sql file with the SQL dump to restore.
//
//   - version: PostgreSQL version to use for the restore
//   - connString: connection string to the database
//   - isLocal: whether the ZIP file is local or a URL
//   - zipURLOrPath: URL or path to the ZIP file
func (Client) RestoreZip(
	version PGVersion, connString string, isLocal bool, zipURLOrPath string,
) error {
	workDir, err := os.MkdirTemp("", "pbw-restore-*")
	if err != nil {
		return fmt.Errorf("error creating temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)
	zipPath := strutil.CreatePath(true, workDir, "dump.zip")
	dumpPath := strutil.CreatePath(true, workDir, "dump.sql")

	if isLocal {
		cmd := exec.Command("cp", zipURLOrPath, zipPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error copying ZIP file to temp dir: %s", output)
		}
	}

	if !isLocal {
		cmd := exec.Command("wget", "--no-verbose", "-O", zipPath, zipURLOrPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error downloading ZIP file: %s", output)
		}
	}

	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		return fmt.Errorf("zip file not found: %s", zipPath)
	}

	cmd := exec.Command("unzip", "-o", zipPath, "dump.sql", "-d", workDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error unzipping ZIP file: %s", output)
	}

	if _, err := os.Stat(dumpPath); os.IsNotExist(err) {
		return fmt.Errorf("dump.sql file not found in ZIP file: %s", zipPath)
	}

	// ON_ERROR_STOP makes psql exit non-zero on the first SQL error, so a
	// partial restore is reported as failed and not as success.
	cmd = exec.Command(
		version.Value.PSQL, dbnameArg(connString), "-X",
		"-v", "ON_ERROR_STOP=1", "-f", dumpPath,
	)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"error running psql v%s command: %s",
			version.Value.Version, output,
		)
	}

	return nil
}
