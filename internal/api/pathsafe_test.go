package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRootedPath_Contained(t *testing.T) {
	root := t.TempDir()

	cases := []struct {
		name string
		in   string
	}{
		{"simple", "/foo.txt"},
		{"nested", "/a/b/c.txt"},
		{"no leading slash", "foo.txt"},
		{"empty", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := resolveRootedPath(root, tc.in)
			if !ok {
				t.Fatalf("expected containment for %q, got rejected", tc.in)
			}
			absRoot, _ := filepath.Abs(root)
			if got != absRoot && filepath.Dir(got) == "" {
				t.Fatalf("resolved path %q looks wrong for root %q", got, absRoot)
			}
		})
	}
}

// Lexical ".." segments never leave root: the leading "/" prepended before
// filepath.Clean anchors them, so ".." above the root has nowhere to go and
// the request just clamps to root itself — it's neutralized, not rejected.
func TestResolveRootedPath_LexicalTraversalIsClamped(t *testing.T) {
	root := t.TempDir()
	absRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}

	cases := []string{
		"../../../../etc/passwd",
		"/../../etc/passwd",
		"../outside.txt",
		"a/../../../outside.txt",
	}
	for _, in := range cases {
		got, ok := resolveRootedPath(root, in)
		if !ok {
			t.Errorf("resolveRootedPath(%q) rejected, want clamped-and-accepted", in)
			continue
		}
		if got != absRoot && !strings.HasPrefix(got, absRoot+string(filepath.Separator)) {
			t.Errorf("resolveRootedPath(%q) = %q, escapes root %q", in, got, absRoot)
		}
	}
}

func TestResolveRootedPath_SymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unsupported in this environment: %v", err)
	}

	if _, ok := resolveRootedPath(root, "/escape/secret.txt"); ok {
		t.Error("expected symlink escape to be rejected, but it was allowed through")
	}
}

func TestResolveRootedPath_SymlinkWithinRootIsFine(t *testing.T) {
	root := t.TempDir()

	realDir := filepath.Join(root, "real")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "alias")
	if err := os.Symlink(realDir, link); err != nil {
		t.Skipf("symlinks unsupported in this environment: %v", err)
	}

	if _, ok := resolveRootedPath(root, "/alias/file.txt"); !ok {
		t.Error("expected an in-root symlink to still resolve as contained")
	}
}
