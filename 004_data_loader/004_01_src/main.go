package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"

	"pxdl/internal/config"
	"pxdl/internal/entity"
	"pxdl/internal/load"
)

// Urutan file entity sesuai dependensi FK.
var entityOrder = []string{
	"organization.json", "maintainer.json", "classifier.json",
	"package.json",
	"release.json",
	"release_detail.json", "release_file.json", "project_link.json", "release_keyword.json", "release_extra.json",
	"file_hash.json", "release_file_tag.json", "attestation.json",
	"maintained_by.json",
	"tagged_with.json",
}

func main() {
	exeDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gagal mendapatkan direktori kerja:", err)
		os.Exit(1)
	}
	projectRoot := filepath.Join(exeDir, "..", "..")
	dataRoot := filepath.Join(projectRoot, "003_data_transformer", "003_02_data_cleaned")
	envPath := filepath.Join(projectRoot, "005_data_storing", "005_01_docker", ".env")

	if err := run(dataRoot, envPath); err != nil {
		fmt.Fprintln(os.Stderr, "004_data_loader berhenti karena error:", err)
		os.Exit(1)
	}
}

func run(dataRoot, envPath string) error {
	runDir, err := config.FindLatestRun(dataRoot)
	if err != nil {
		return err
	}
	password, err := config.ReadDBPassword(envPath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	connURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(config.DBUser, password),
		Host:   net.JoinHostPort(config.DBHost, config.DBPort),
		Path:   "/" + config.DBName,
	}
	connString := connURL.String()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return fmt.Errorf("koneksi database gagal: %w", err)
	}
	defer conn.Close(context.Background())

	tx, err := conn.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("memulai transaksi gagal: %w", err)
	}
	// Rollback no-op setelah Commit sukses; menjamin tidak ada data parsial jika return awal.
	defer tx.Rollback(context.Background())

	if err := load.TruncateAll(context.Background(), tx); err != nil {
		return err
	}

	total, err := loadAll(context.Background(), tx, runDir)
	if err != nil {
		return err
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("commit gagal: %w", err)
	}

	fmt.Printf("Selesai. %d baris dimuat ke database.\n", total)
	return nil
}

// loadAll memuat tiap file entity berurutan sesuai FK; maintained_by/tagged_with dijalankan terakhir.
func loadAll(ctx context.Context, tx pgx.Tx, runDir string) (int, error) {
	r := load.NewResolver()
	total := 0
	step := 0

	// Cetak progres, jalankan fn, akumulasi jumlah baris ke total.
	loadStep := func(name string, fn func() (int, error)) error {
		step++
		fmt.Printf("[%d/%d] Memuat %s...\n", step, len(entityOrder), name)
		count, err := fn()
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		total += count
		return nil
	}

	var organizations []entity.Organization
	if err := readEntity(runDir, "organization.json", &organizations); err != nil {
		return 0, err
	}
	if err := loadStep("organization.json", func() (int, error) { return load.Organizations(ctx, tx, organizations, r) }); err != nil {
		return 0, err
	}

	var maintainers []entity.Maintainer
	if err := readEntity(runDir, "maintainer.json", &maintainers); err != nil {
		return 0, err
	}
	if err := loadStep("maintainer.json", func() (int, error) { return load.Maintainers(ctx, tx, maintainers, r) }); err != nil {
		return 0, err
	}

	var classifiers []entity.Classifier
	if err := readEntity(runDir, "classifier.json", &classifiers); err != nil {
		return 0, err
	}
	if err := loadStep("classifier.json", func() (int, error) { return load.Classifiers(ctx, tx, classifiers, r) }); err != nil {
		return 0, err
	}

	var packages []entity.Package
	if err := readEntity(runDir, "package.json", &packages); err != nil {
		return 0, err
	}
	if err := loadStep("package.json", func() (int, error) { return load.Packages(ctx, tx, packages, r) }); err != nil {
		return 0, err
	}

	var releases []entity.Release
	if err := readEntity(runDir, "release.json", &releases); err != nil {
		return 0, err
	}
	if err := loadStep("release.json", func() (int, error) { return load.Releases(ctx, tx, releases, r) }); err != nil {
		return 0, err
	}

	var releaseDetails []entity.ReleaseDetail
	if err := readEntity(runDir, "release_detail.json", &releaseDetails); err != nil {
		return 0, err
	}
	if err := loadStep("release_detail.json", func() (int, error) { return load.ReleaseDetails(ctx, tx, releaseDetails, r) }); err != nil {
		return 0, err
	}

	var releaseFiles []entity.ReleaseFile
	if err := readEntity(runDir, "release_file.json", &releaseFiles); err != nil {
		return 0, err
	}
	if err := loadStep("release_file.json", func() (int, error) { return load.ReleaseFiles(ctx, tx, releaseFiles, r) }); err != nil {
		return 0, err
	}

	var projectLinks []entity.ProjectLink
	if err := readEntity(runDir, "project_link.json", &projectLinks); err != nil {
		return 0, err
	}
	if err := loadStep("project_link.json", func() (int, error) { return load.ProjectLinks(ctx, tx, projectLinks, r) }); err != nil {
		return 0, err
	}

	var releaseKeywords []entity.ReleaseKeyword
	if err := readEntity(runDir, "release_keyword.json", &releaseKeywords); err != nil {
		return 0, err
	}
	if err := loadStep("release_keyword.json", func() (int, error) { return load.ReleaseKeywords(ctx, tx, releaseKeywords, r) }); err != nil {
		return 0, err
	}

	var releaseExtras []entity.ReleaseExtra
	if err := readEntity(runDir, "release_extra.json", &releaseExtras); err != nil {
		return 0, err
	}
	if err := loadStep("release_extra.json", func() (int, error) { return load.ReleaseExtras(ctx, tx, releaseExtras, r) }); err != nil {
		return 0, err
	}

	var fileHashes []entity.FileHash
	if err := readEntity(runDir, "file_hash.json", &fileHashes); err != nil {
		return 0, err
	}
	if err := loadStep("file_hash.json", func() (int, error) { return load.FileHashes(ctx, tx, fileHashes, r) }); err != nil {
		return 0, err
	}

	var releaseFileTags []entity.ReleaseFileTag
	if err := readEntity(runDir, "release_file_tag.json", &releaseFileTags); err != nil {
		return 0, err
	}
	if err := loadStep("release_file_tag.json", func() (int, error) { return load.ReleaseFileTags(ctx, tx, releaseFileTags, r) }); err != nil {
		return 0, err
	}

	var attestations []entity.Attestation
	if err := readEntity(runDir, "attestation.json", &attestations); err != nil {
		return 0, err
	}
	if err := loadStep("attestation.json", func() (int, error) { return load.Attestations(ctx, tx, attestations, r) }); err != nil {
		return 0, err
	}

	var maintainedBy []entity.MaintainedBy
	if err := readEntity(runDir, "maintained_by.json", &maintainedBy); err != nil {
		return 0, err
	}
	if err := loadStep("maintained_by.json", func() (int, error) { return load.MaintainedBy(ctx, tx, maintainedBy, r) }); err != nil {
		return 0, err
	}

	var taggedWith []entity.TaggedWith
	if err := readEntity(runDir, "tagged_with.json", &taggedWith); err != nil {
		return 0, err
	}
	if err := loadStep("tagged_with.json", func() (int, error) { return load.TaggedWith(ctx, tx, taggedWith, r) }); err != nil {
		return 0, err
	}

	return total, nil
}

func readEntity(runDir, filename string, out interface{}) error {
	path := filepath.Join(runDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("gagal membaca %s: %w", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("gagal parse %s: %w", path, err)
	}
	return nil
}
