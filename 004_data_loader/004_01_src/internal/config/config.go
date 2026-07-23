package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// Harus sama dengan 005_data_storing/docker/compose.yml.
	DBHost = "127.0.0.1"
	DBPort = "5432"
	DBUser = "px_user"
	DBName = "px_db"
	// Batas waktu koneksi ke Postgres.
	ConnectTimeout = 10 * time.Second
)

// ReadDBPassword membaca POSTGRES_PASSWORD dari .env docker-compose; 004 tidak punya .env sendiri.
func ReadDBPassword(envPath string) (string, error) {
	data, err := os.ReadFile(envPath)
	if err != nil {
		return "", fmt.Errorf("gagal membaca %s: %w", envPath, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if value, ok := strings.CutPrefix(line, "POSTGRES_PASSWORD="); ok {
			return strings.TrimSpace(value), nil
		}
	}
	return "", fmt.Errorf("POSTGRES_PASSWORD tidak ditemukan di %s", envPath)
}

// FindLatestRun mencari folder pxs_<timestamp> terbesar secara leksikografis (timestamp ISO 8601 urut sebagai string).
func FindLatestRun(dataRoot string) (string, error) {
	entries, err := os.ReadDir(dataRoot)
	if err != nil {
		return "", fmt.Errorf("gagal membaca %s: %w", dataRoot, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "pxs_") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("tidak ada folder hasil cleaning ditemukan di %s", dataRoot)
	}
	sort.Strings(names)
	return filepath.Join(dataRoot, names[len(names)-1]), nil
}
