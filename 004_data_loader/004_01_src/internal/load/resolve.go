package load

import "fmt"

type releaseKey struct {
	PackageName string
	Version     string
}

type fileKey struct {
	PackageName string
	Version     string
	Filename    string
}

type classifierKey struct {
	Category string
	Value    string
}

// Peta natural-key -> UUID hasil insert; miss saat lookup dianggap fatal.
type Resolver struct {
	orgID        map[string]string
	maintainerID map[string]string
	classifierID map[classifierKey]string
	packageID    map[string]string
	releaseID    map[releaseKey]string
	fileID       map[fileKey]string
}

func NewResolver() *Resolver {
	return &Resolver{
		orgID:        make(map[string]string),
		maintainerID: make(map[string]string),
		classifierID: make(map[classifierKey]string),
		packageID:    make(map[string]string),
		releaseID:    make(map[releaseKey]string),
		fileID:       make(map[fileKey]string),
	}
}

// Nama kosong berarti "tanpa organisasi", bukan referensi tak terselesaikan.
func (r *Resolver) lookupOrg(name string) (*string, error) {
	if name == "" {
		return nil, nil
	}
	id, ok := r.orgID[name]
	if !ok {
		return nil, fmt.Errorf("organisasi %q tidak ditemukan (referensi natural key gagal)", name)
	}
	return &id, nil
}

func (r *Resolver) lookupPackage(name string) (string, error) {
	id, ok := r.packageID[name]
	if !ok {
		return "", fmt.Errorf("package %q tidak ditemukan (referensi natural key gagal)", name)
	}
	return id, nil
}

func (r *Resolver) lookupRelease(pkg, version string) (string, error) {
	id, ok := r.releaseID[releaseKey{PackageName: pkg, Version: version}]
	if !ok {
		return "", fmt.Errorf("release %s %s tidak ditemukan (referensi natural key gagal)", pkg, version)
	}
	return id, nil
}

func (r *Resolver) lookupFile(pkg, version, filename string) (string, error) {
	id, ok := r.fileID[fileKey{PackageName: pkg, Version: version, Filename: filename}]
	if !ok {
		return "", fmt.Errorf("release_file %s %s %s tidak ditemukan (referensi natural key gagal)", pkg, version, filename)
	}
	return id, nil
}

func (r *Resolver) lookupMaintainer(username string) (string, error) {
	id, ok := r.maintainerID[username]
	if !ok {
		return "", fmt.Errorf("maintainer %q tidak ditemukan (referensi natural key gagal)", username)
	}
	return id, nil
}

func (r *Resolver) lookupClassifier(category, value string) (string, error) {
	id, ok := r.classifierID[classifierKey{Category: category, Value: value}]
	if !ok {
		return "", fmt.Errorf("classifier %s=%s tidak ditemukan (referensi natural key gagal)", category, value)
	}
	return id, nil
}
