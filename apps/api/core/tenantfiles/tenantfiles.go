// Package tenantfiles stores each tenant's files (signed XML, RIDE PDFs, P12
// certificate, logo, custom templates) by tenant and relative name. It is the
// only place that knows where they live; callers never build storage paths.
package tenantfiles

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// DefaultRoot is where tenant folders live, relative to the API's working
// directory (the api_storage volume in production).
const DefaultRoot = "storage/tenants"

// ErrInvalidName: the name is empty, absolute or escapes the tenant's folder.
var ErrInvalidName = errors.New("tenantfiles: invalid file name")

// Store holds files per tenant. Names are slash-separated and relative to the
// tenant, e.g. "invoices/<access key>.xml". A missing file reads as an error
// wrapping fs.ErrNotExist.
type Store interface {
	Write(tenantID uint, name string, data []byte) error
	Read(tenantID uint, name string) ([]byte, error)
	// Remove deletes the file; a missing file is not an error.
	Remove(tenantID uint, name string) error
	Exists(tenantID uint, name string) bool
}

// clean validates a tenant-relative name and returns its canonical form.
func clean(name string) (string, error) {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, `\`) {
		return "", fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	c := path.Clean(name)
	if c == "." || c == ".." || strings.HasPrefix(c, "../") {
		return "", fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	return c, nil
}

// DiskStore keeps files under <root>/<tenant id>/<name>.
type DiskStore struct{ root string }

func Disk(root string) *DiskStore { return &DiskStore{root: root} }

// Path is the on-disk path of a tenant file, for values already persisted as
// paths (tenant P12 certificate and logo).
func (d *DiskStore) Path(tenantID uint, name string) string {
	return filepath.Join(d.root, fmt.Sprint(tenantID), filepath.FromSlash(name))
}

func (d *DiskStore) Write(tenantID uint, name string, data []byte) error {
	c, err := clean(name)
	if err != nil {
		return err
	}
	p := d.Path(tenantID, c)
	if err := os.MkdirAll(filepath.Dir(p), os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

func (d *DiskStore) Read(tenantID uint, name string) ([]byte, error) {
	c, err := clean(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(d.Path(tenantID, c))
}

func (d *DiskStore) Remove(tenantID uint, name string) error {
	c, err := clean(name)
	if err != nil {
		return err
	}
	if err := os.Remove(d.Path(tenantID, c)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (d *DiskStore) Exists(tenantID uint, name string) bool {
	c, err := clean(name)
	if err != nil {
		return false
	}
	_, err = os.Stat(d.Path(tenantID, c))
	return err == nil
}

// MemoryStore is an in-memory Store for tests.
type MemoryStore struct {
	mu    sync.Mutex
	files map[string][]byte
}

func Memory() *MemoryStore { return &MemoryStore{files: map[string][]byte{}} }

func memKey(tenantID uint, name string) string { return fmt.Sprintf("%d/%s", tenantID, name) }

func (m *MemoryStore) Write(tenantID uint, name string, data []byte) error {
	c, err := clean(name)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[memKey(tenantID, c)] = append([]byte(nil), data...)
	return nil
}

func (m *MemoryStore) Read(tenantID uint, name string) ([]byte, error) {
	c, err := clean(name)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.files[memKey(tenantID, c)]
	if !ok {
		return nil, fmt.Errorf("tenantfiles: %s: %w", c, fs.ErrNotExist)
	}
	return append([]byte(nil), data...), nil
}

func (m *MemoryStore) Remove(tenantID uint, name string) error {
	c, err := clean(name)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, memKey(tenantID, c))
	return nil
}

func (m *MemoryStore) Exists(tenantID uint, name string) bool {
	c, err := clean(name)
	if err != nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.files[memKey(tenantID, c)]
	return ok
}
