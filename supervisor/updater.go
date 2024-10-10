package supervisor

import (
	"archive/zip"
	"context"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/zeebo/errs"

	"storj.io/common/sync2"
	"storj.io/common/version"
	"storj.io/storj/private/version/checker"
)

type Updater struct {
	checker *checker.Client
}

// NewUpdater creates a new updater.
func NewUpdater(checker *checker.Client) *Updater {
	return &Updater{
		checker: checker,
	}
}

// Update checks if the process should be updated and updates it if necessary.
func (u *Updater) Update(ctx context.Context, process *Process, currentVersion version.SemVer) (updated bool, err error) {
	all, err := u.checker.All(ctx)
	if err != nil {
		return false, errs.Wrap(err)
	}

	newVersion, reason, err := version.ShouldUpdateVersion(currentVersion, process.nodeID, all.Processes.Storagenode)
	if err != nil {
		return false, errs.Wrap(err)
	}

	if newVersion.IsZero() {
		log.Println(reason)
		return false, nil
	}

	newVersionPath := prependExtension(process.binPath, newVersion.Version)
	if err = downloadBinary(ctx, parseDownloadURL(newVersion.URL), newVersionPath); err != nil {
		return false, errs.Wrap(err)
	}

	downloadedVersion, err := binaryVersion(ctx, newVersionPath)
	if err != nil {
		return false, errs.Combine(errs.Wrap(err), os.Remove(newVersionPath))
	}

	slog.Info("Downloaded version.", slog.String("Version", downloadedVersion.String()))

	newSemVer, err := newVersion.SemVer()
	if err != nil {
		return false, errs.Combine(err, os.Remove(newVersionPath))
	}

	if newSemVer.Compare(downloadedVersion) != 0 {
		err := errs.New("invalid version downloaded: wants %s got %s", newVersion.Version, downloadedVersion)
		return false, errs.Combine(err, os.Remove(newVersionPath))
	}

	if err = copyToStore(ctx, process.storeDir, newVersionPath); err != nil {
		return false, errs.Wrap(err)
	}

	if err = replaceBinary(ctx, currentVersion, newVersionPath, process.binPath); err != nil {
		return false, errs.Wrap(err)
	}

	return true, nil
}

// replaceBinary replaces the binary with the new binary.
func replaceBinary(ctx context.Context, currentVersion version.SemVer, newVersionPath, binaryLocation string) (err error) {
	backupPath := prependExtension(binaryLocation, "old."+currentVersion.String())
	reverseFunc := func() error {
		return nil
	}

	// first backup the current binary
	if !currentVersion.IsZero() {
		if err := os.Rename(binaryLocation, backupPath); err != nil {
			return errs.Wrap(err)
		}

		reverseFunc = func() error {
			return os.Rename(backupPath, binaryLocation)
		}
	}

	// replace the binary with the new binary
	if err := os.Rename(newVersionPath, binaryLocation); err != nil {
		return errs.Combine(err, reverseFunc(), os.Remove(newVersionPath))
	}

	// remove the backup binary
	if err := os.Remove(backupPath); err != nil && !errs.Is(err, os.ErrNotExist) {
		log.Println("Error removing backup binary:", err, "consider removing it manually")
	}

	return nil
}

// copyToStore copies binary to store directory if the storeDir is set and different from the binary location.
func copyToStore(ctx context.Context, storeDir, binaryLocation string) error {
	if storeDir == "" {
		return nil
	}

	_ = os.MkdirAll(storeDir, 0755)

	dir, base := filepath.Split(binaryLocation)
	if dir == storeDir {
		return nil
	}

	storeLocation := filepath.Join(storeDir, base)

	// copy binary to store
	slog.Info("Copying binary to store.", slog.String("From", binaryLocation), slog.String("To", storeLocation))
	src, err := os.Open(binaryLocation)
	if err != nil {
		return errs.Wrap(err)
	}
	defer func() {
		err = errs.Combine(err, src.Close())
	}()

	dest, err := os.OpenFile(storeLocation, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return errs.Wrap(err)
	}

	defer func() {
		err = errs.Combine(err, dest.Close())
	}()

	_, err = sync2.Copy(ctx, dest, src)
	if err != nil {
		return errs.Wrap(err)
	}

	slog.Info("Binary copied to store.", slog.String("From", binaryLocation), slog.String("To", storeLocation))

	return nil
}

func downloadBinary(ctx context.Context, url, target string) error {
	f, err := os.CreateTemp("", createPattern(url))
	if err != nil {
		return errs.New("cannot create temporary archive: %v", err)
	}
	defer func() {
		err = errs.Combine(err,
			f.Close(),
			os.Remove(f.Name()),
		)
	}()

	slog.Info("Download started.", slog.String("From", url), slog.String("To", f.Name()))

	if err = downloadArchive(ctx, f, url); err != nil {
		return errs.Wrap(err)
	}
	slog.Info("Download finished.", slog.String("From", url), slog.String("To", f.Name()))
	slog.Info("Unpacking archive.", slog.String("From", f.Name()), slog.String("To", target))
	if err = unpackBinary(ctx, f.Name(), target); err != nil {
		return errs.Wrap(err)
	}

	return nil
}

func downloadArchive(ctx context.Context, file io.Writer, url string) (err error) {
	resp, err := httpGet(ctx, url)
	if err != nil {
		return err
	}

	defer func() { err = errs.Combine(err, resp.Body.Close()) }()

	if resp.StatusCode != http.StatusOK {
		return errs.New("bad status: %s", resp.Status)
	}

	_, err = sync2.Copy(ctx, file, resp.Body)
	return err
}

// unpackBinary unpack zip compressed binary.
func unpackBinary(ctx context.Context, archive, target string) (err error) {
	zipReader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() { err = errs.Combine(err, zipReader.Close()) }()

	if len(zipReader.File) != 1 {
		return errs.New("archive should contain only one file")
	}

	zipedExec, err := zipReader.File[0].Open()
	if err != nil {
		return err
	}
	defer func() { err = errs.Combine(err, zipedExec.Close()) }()

	newExec, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(0755))
	if err != nil {
		return err
	}
	defer func() { err = errs.Combine(err, newExec.Close()) }()

	_, err = sync2.Copy(ctx, newExec, zipedExec)
	if err != nil {
		return errs.Combine(err, os.Remove(newExec.Name()))
	}
	return nil
}

func httpGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}
