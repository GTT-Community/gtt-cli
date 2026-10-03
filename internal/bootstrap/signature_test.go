package bootstrap

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/fsx"
)

func key(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

func writeSignature(t *testing.T, root string, doc signatureDoc) {
	if err := fsx.WriteJSON(filepath.Join(root, filepath.FromSlash(SignatureFile)), doc); err != nil {
		t.Fatal(err)
	}
}

func signed(priv ed25519.PrivateKey, id, version, sum string) signatureDoc {
	return signatureDoc{Schema: 1, Algorithm: "ed25519", KeyID: id, Version: version, Checksum: sum,
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, signedMessage(version, sum)))}
}

func TestSignaturePolicy(t *testing.T) {
	pub, priv := key(t)
	_, other := key(t)
	trusted := map[string]ed25519.PublicKey{"release": pub}
	for name, c := range map[string]struct {
		doc     *signatureDoc
		keys    map[string]ed25519.PublicKey
		want    string // prefix of the reported status, or of the error
		failure bool
	}{
		"no key, no signature":      {nil, nil, "not available", false},
		"no key, signature present": {&signatureDoc{Algorithm: "ed25519"}, nil, "present, not verified", false},
		"key, no signature":         {nil, trusted, "signature required", true},
		"valid":                     {ptr(signed(priv, "release", "1.2.0", "abc")), trusted, "verified (ed25519, key release)", false},
		"content changed":           {ptr(signed(priv, "release", "1.2.0", "old")), trusted, "signature mismatch: the package content", true},
		"other version":             {ptr(signed(priv, "release", "1.1.0", "abc")), trusted, "the signature is for Bootstrap 1.1.0", true},
		"untrusted key":             {ptr(signed(other, "someone", "1.2.0", "abc")), trusted, `signed with key "someone"`, true},
		"forged with a trusted id":  {ptr(signed(other, "release", "1.2.0", "abc")), trusted, "signature mismatch: the signature does not verify", true},
		"unsupported algorithm":     {&signatureDoc{Algorithm: "rsa", KeyID: "release", Version: "1.2.0", Checksum: "abc"}, trusted, "unsupported signature algorithm", true},
	} {
		root := t.TempDir()
		if c.doc != nil {
			writeSignature(t, root, *c.doc)
		}
		got, err := checkSignature(root, "1.2.0", "abc", c.keys, "/keys")
		if c.failure {
			if err == nil || !strings.HasPrefix(err.Error(), c.want) {
				t.Errorf("%s: got %q, %v; want failure %q", name, got, err, c.want)
			}
		} else if err != nil || !strings.HasPrefix(got, c.want) {
			t.Errorf("%s: got %q, %v; want %q", name, got, err, c.want)
		}
	}
}

func ptr(d signatureDoc) *signatureDoc { return &d }

func TestTrustedKeys(t *testing.T) {
	if keys, err := TrustedKeys(filepath.Join(t.TempDir(), "absent")); err != nil || len(keys) != 0 {
		t.Fatalf("an absent key directory means no trusted key: %v, %v", keys, err)
	}
	dir := t.TempDir()
	pub, _ := key(t)
	os.WriteFile(filepath.Join(dir, "release.pub"), []byte(base64.StdEncoding.EncodeToString(pub)+"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "README"), []byte("ignored"), 0o644)
	keys, err := TrustedKeys(dir)
	if err != nil || len(keys) != 1 || !keys["release"].Equal(pub) {
		t.Fatalf("keys: %v, %v", keys, err)
	}
	os.WriteFile(filepath.Join(dir, "broken.pub"), []byte("not a key"), 0o644)
	if _, err := TrustedKeys(dir); core.CodeOf(err) != core.ExitIntegrity {
		t.Errorf("a malformed trusted key is an integrity failure, never skipped: %v", err)
	}
}

func TestContentFilesExcludeOnlyTheSignature(t *testing.T) {
	got := contentFiles([]string{".gtt/contract/release.json", SignatureFile, "AGENTS.md"})
	if len(got) != 2 || got[0] != ".gtt/contract/release.json" || got[1] != "AGENTS.md" {
		t.Errorf("got %v", got)
	}
}

func TestKeygenNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	pair, err := Signer{}.Keygen(dir, "release")
	if err != nil {
		t.Fatal(err)
	}
	// Windows has no Unix permission bits; elsewhere the key is owner-only.
	if info, err := os.Stat(pair.Private); err != nil || (runtime.GOOS != "windows" && info.Mode().Perm() != 0o600) {
		t.Errorf("the private key must be readable by its owner only: %v %v", info.Mode(), err)
	}
	if _, err := (Signer{}).Keygen(dir, "release"); core.CodeOf(err) != core.ExitConflict {
		t.Errorf("an existing key is never overwritten: %v", err)
	}
	if _, err := (Signer{}).Keygen(dir, "../escape"); core.CodeOf(err) != core.ExitUsage {
		t.Errorf("a key id is a plain name: %v", err)
	}
}
