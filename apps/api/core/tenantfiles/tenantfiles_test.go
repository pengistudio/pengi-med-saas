package tenantfiles_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"pengi-med-saas/core/tenantfiles"
)

// Both adapters must behave the same: they are interchangeable at the seam.
func stores(t *testing.T) map[string]tenantfiles.Store {
	return map[string]tenantfiles.Store{
		"disk":   tenantfiles.Disk(t.TempDir()),
		"memory": tenantfiles.Memory(),
	}
}

func TestStore_WriteThenReadBack(t *testing.T) {
	for name, s := range stores(t) {
		t.Run(name, func(t *testing.T) {
			if err := s.Write(1, "invoices/0101.xml", []byte("<xml/>")); err != nil {
				t.Fatalf("write: %v", err)
			}
			got, err := s.Read(1, "invoices/0101.xml")
			if err != nil || string(got) != "<xml/>" {
				t.Fatalf("read = %q, %v", got, err)
			}
			if !s.Exists(1, "invoices/0101.xml") {
				t.Fatalf("Exists = false after write")
			}
		})
	}
}

func TestStore_TenantsDoNotSeeEachOthersFiles(t *testing.T) {
	for name, s := range stores(t) {
		t.Run(name, func(t *testing.T) {
			_ = s.Write(1, "logo.png", []byte("one"))
			if s.Exists(2, "logo.png") {
				t.Fatalf("tenant 2 sees tenant 1's logo")
			}
			if _, err := s.Read(2, "logo.png"); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("err = %v, want fs.ErrNotExist", err)
			}
		})
	}
}

func TestStore_RemoveIsIdempotent(t *testing.T) {
	for name, s := range stores(t) {
		t.Run(name, func(t *testing.T) {
			_ = s.Write(1, "prescription_template.html", []byte("x"))
			if err := s.Remove(1, "prescription_template.html"); err != nil {
				t.Fatalf("remove: %v", err)
			}
			if err := s.Remove(1, "prescription_template.html"); err != nil {
				t.Fatalf("second remove: %v, want nil", err)
			}
			if s.Exists(1, "prescription_template.html") {
				t.Fatalf("file still exists")
			}
		})
	}
}

func TestStore_RejectsNamesEscapingTheTenant(t *testing.T) {
	for name, s := range stores(t) {
		t.Run(name, func(t *testing.T) {
			for _, bad := range []string{"../2/logo.png", "/etc/passwd", "a/../../x", "", "."} {
				if err := s.Write(1, bad, []byte("x")); !errors.Is(err, tenantfiles.ErrInvalidName) {
					t.Fatalf("Write(%q) err = %v, want ErrInvalidName", bad, err)
				}
				if _, err := s.Read(1, bad); !errors.Is(err, tenantfiles.ErrInvalidName) {
					t.Fatalf("Read(%q) err = %v, want ErrInvalidName", bad, err)
				}
			}
		})
	}
}

// The disk layout is unchanged (no data migration): storage/tenants/<id>/<name>,
// the same path already persisted for P12 certificates and logos.
func TestDisk_KeepsTheExistingLayout(t *testing.T) {
	root := t.TempDir()
	s := tenantfiles.Disk(root)
	_ = s.Write(7, "invoices/k.pdf", []byte("pdf"))

	if _, err := os.Stat(filepath.Join(root, "7", "invoices", "k.pdf")); err != nil {
		t.Fatalf("file not at <root>/7/invoices/k.pdf: %v", err)
	}
	if got, want := s.Path(7, "signature.p12"), filepath.Join(root, "7", "signature.p12"); got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}
	if got := tenantfiles.Disk(tenantfiles.DefaultRoot).Path(7, "logo.png"); got != filepath.Join("storage", "tenants", "7", "logo.png") {
		t.Fatalf("default Path = %q, want storage/tenants/7/logo.png", got)
	}
}

// A reader racing a rewrite sees the old or the new content, never a partial
// file, and no temp files are left behind.
func TestDiskStore_WriteIsAtomicAndLeavesNoTempFiles(t *testing.T) {
	root := t.TempDir()
	s := tenantfiles.Disk(root)
	oldData := bytes.Repeat([]byte("a"), 256<<10)
	newData := bytes.Repeat([]byte("b"), 256<<10)
	if err := s.Write(1, "x/doc.xml", oldData); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			if err := s.Write(1, "x/doc.xml", newData); err != nil {
				t.Errorf("write: %v", err)
				return
			}
		}
	}()
	for i := 0; i < 2000; i++ {
		got, err := s.Read(1, "x/doc.xml")
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if !bytes.Equal(got, oldData) && !bytes.Equal(got, newData) {
			t.Fatalf("read a partial file of %d bytes", len(got))
		}
	}
	<-done

	entries, err := os.ReadDir(filepath.Join(root, "1", "x"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected only doc.xml, found %d entries", len(entries))
	}
}
