package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeJoinRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skip("symlinks unavailable")
	}
	for _, bad := range []string{"../x", "a/../../x", "/etc/passwd", "", "~/x", "link/secret", "..",
		`\Windows\win.ini`, `C:\Windows\win.ini`, `C:relative`, `\\server\share\x`} {
		if _, err := SafeJoin(root, bad); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
	for _, good := range []string{"a.md", "docs/a.md", "./docs/../a.md", "new/dir/file"} {
		if _, err := SafeJoin(root, good); err != nil {
			t.Errorf("%q must be accepted: %v", good, err)
		}
	}
}

func TestListFilesRejectsEscapingSymlinkAndIgnoresVCS(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".git"), 0o755)
	os.MkdirAll(filepath.Join(root, "a", "__pycache__"), 0o755)
	os.WriteFile(filepath.Join(root, ".git", "config"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(root, "a", "__pycache__", "m.pyc"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(root, "a", "f.txt"), []byte("x"), 0o644)
	files, err := ListFiles(root)
	if err != nil || len(files) != 1 || files[0] != "a/f.txt" {
		t.Fatalf("got %v %v", files, err)
	}
	if err := os.Symlink("/etc/hosts", filepath.Join(root, "a", "escape")); err != nil {
		t.Skip("symlinks unavailable")
	}
	if _, err := ListFiles(root); err == nil {
		t.Error("a symlink leaving the tree must be an error")
	}
}

func TestHashTreeDependsOnContentAndPath(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a"), []byte("1"), 0o644)
	h1, _ := HashTree(root, []string{"a"})
	os.WriteFile(filepath.Join(root, "a"), []byte("2"), 0o644)
	h2, _ := HashTree(root, []string{"a"})
	if h1 == h2 || h1 == "" {
		t.Error("content change must change the checksum")
	}
}

func TestCopyFileRefusesOverwrite(t *testing.T) {
	root := t.TempDir()
	src, dst := filepath.Join(root, "s"), filepath.Join(root, "d")
	os.WriteFile(src, []byte("new"), 0o644)
	os.WriteFile(dst, []byte("mine"), 0o644)
	if err := CopyFile(src, dst, false); err == nil {
		t.Fatal("must not overwrite silently")
	}
	if data, _ := os.ReadFile(dst); string(data) != "mine" {
		t.Error("the existing file changed")
	}
}

func TestDanglingSymlinkPointingOutsideIsRejected(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "f.txt"), []byte("x"), 0o644)
	for name, target := range map[string]string{
		"absolute, missing": filepath.Join(t.TempDir(), "gone", "secret"),
		"relative, missing": filepath.Join("..", "..", "nowhere"),
	} {
		link := filepath.Join(root, "escape")
		os.Remove(link)
		if err := os.Symlink(target, link); err != nil {
			t.Skip("symlinks unavailable")
		}
		if _, err := ListFiles(root); err == nil {
			t.Errorf("%s: a link whose target leaves the tree is unsafe even when the target does not exist", name)
		}
	}
	os.Remove(filepath.Join(root, "escape"))
	if err := os.Symlink("f.txt", filepath.Join(root, "inside")); err != nil {
		t.Skip("symlinks unavailable")
	}
	if files, err := ListFiles(root); err != nil || len(files) != 2 {
		t.Errorf("a link inside the tree is fine: %v %v", files, err)
	}
}
