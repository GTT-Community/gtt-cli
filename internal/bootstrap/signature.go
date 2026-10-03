package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GTT-Community/gtt-cli/internal/core"
	"github.com/GTT-Community/gtt-cli/internal/fsx"
	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// SignatureFile is the detached release signature a Bootstrap package may
// carry. It signs the content checksum of every other file of the package.
const SignatureFile = ".gtt/contract/release.sig"

// signatureAlgorithm is the only scheme accepted.
const signatureAlgorithm = "ed25519"

var keyID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

type signatureDoc struct {
	Schema    int    `json:"schema"`
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"key_id"`
	Version   string `json:"bootstrap_version"`
	Checksum  string `json:"checksum"`
	Signature string `json:"signature"`
}

// signedMessage binds the signature to the release identity and content.
func signedMessage(version, checksum string) []byte {
	return []byte("gtt-bootstrap-release\n" + version + "\n" + checksum + "\n")
}

// contentFiles are the files covered by the content checksum: all of them
// except the signature itself.
func contentFiles(files []string) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		if f != SignatureFile {
			out = append(out, f)
		}
	}
	return out
}

// TrustedKeys reads the trusted release keys: every <key-id>.pub file in dir
// holds one base64 ed25519 public key. An unreadable or malformed key is an
// integrity failure, never skipped.
func TrustedKeys(dir string) (map[string]ed25519.PublicKey, error) {
	keys := map[string]ed25519.PublicKey{}
	if dir == "" {
		return keys, nil
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return keys, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".pub")
		if e.IsDir() || !ok {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		raw, derr := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
		if err != nil || derr != nil || len(raw) != ed25519.PublicKeySize || !keyID.MatchString(id) {
			return nil, &core.Error{Code: core.ExitIntegrity, Unmodified: true,
				What: "Trusted key " + filepath.Join(dir, e.Name()) + " is not a valid ed25519 public key.",
				Next: "Fix or remove the file."}
		}
		keys[id] = ed25519.PublicKey(raw)
	}
	return keys, nil
}

// checkSignature applies the signature policy: with no trusted key the
// signature is reported, not required; with trusted keys it is required and
// must verify against one of them for this release and this content.
func checkSignature(root, version, checksum string, keys map[string]ed25519.PublicKey, keysDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(SignatureFile)))
	present := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if len(keys) == 0 {
		if present {
			return "present, not verified (no trusted key in " + keysDir + ")", nil
		}
		return "not available (the package carries no signature; no trusted key is configured)", nil
	}
	if !present {
		return "", fmt.Errorf("signature required: trusted keys are configured in %s, but the package has no %s", keysDir, SignatureFile)
	}
	var doc signatureDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("unreadable signature: %v", err)
	}
	key, known := keys[doc.KeyID]
	sig, derr := base64.StdEncoding.DecodeString(doc.Signature)
	switch {
	case doc.Algorithm != signatureAlgorithm:
		return "", fmt.Errorf("unsupported signature algorithm %q", doc.Algorithm)
	case !known:
		return "", fmt.Errorf("signed with key %q, which is not trusted", doc.KeyID)
	case doc.Version != version:
		return "", fmt.Errorf("the signature is for Bootstrap %s, the package is %s", doc.Version, version)
	case doc.Checksum != checksum:
		return "", fmt.Errorf("signature mismatch: the package content differs from what was signed")
	case derr != nil || !ed25519.Verify(key, signedMessage(doc.Version, doc.Checksum), sig):
		return "", fmt.Errorf("signature mismatch: the signature does not verify with key %q", doc.KeyID)
	}
	return "verified (" + signatureAlgorithm + ", key " + doc.KeyID + ")", nil
}

// Signer creates release keys and signs Bootstrap packages. It is the tool
// of whoever publishes a Bootstrap release, never part of verification.
type Signer struct {
	Factory ports.BootstrapFactory
}

// Keygen writes <id>.key (private, 0600) and <id>.pub in dir. It never
// overwrites an existing key.
func (Signer) Keygen(dir, id string) (ports.KeyPair, error) {
	if !keyID.MatchString(id) {
		return ports.KeyPair{}, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Invalid key id: " + id,
			Why: "Use letters, digits, '.', '_' or '-' (at most 64)."}
	}
	pair := ports.KeyPair{ID: id, Private: filepath.Join(dir, id+".key"), Public: filepath.Join(dir, id+".pub")}
	for _, p := range []string{pair.Private, pair.Public} {
		if fsx.Exists(p) {
			return ports.KeyPair{}, &core.Error{Code: core.ExitConflict, Unmodified: true, What: "Key file already exists: " + p}
		}
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return ports.KeyPair{}, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return ports.KeyPair{}, err
	}
	if err := os.WriteFile(pair.Private, []byte(base64.StdEncoding.EncodeToString(priv.Seed())+"\n"), 0o600); err != nil {
		return ports.KeyPair{}, err
	}
	return pair, os.WriteFile(pair.Public, []byte(base64.StdEncoding.EncodeToString(pub)+"\n"), 0o644)
}

// Sign writes the release signature into a Bootstrap package directory.
func (s Signer) Sign(ctx context.Context, root, keyFile string) (ports.SignResult, error) {
	id, ok := strings.CutSuffix(filepath.Base(keyFile), ".key")
	data, err := os.ReadFile(keyFile)
	if err != nil || !ok || !keyID.MatchString(id) {
		return ports.SignResult{}, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Not a release key file: " + keyFile,
			Why: "Expected <key-id>.key as written by: gtt release keygen"}
	}
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return ports.SignResult{}, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Unreadable release key: " + keyFile}
	}
	if !s.Factory.Installed(root) {
		return ports.SignResult{}, &core.Error{Code: core.ExitUsage, Unmodified: true, What: "Not a GTT Bootstrap package: " + root}
	}
	rel, err := s.Factory.At(root).Release(ctx)
	if err != nil {
		return ports.SignResult{}, err
	}
	files, err := fsx.ListFiles(root)
	if err != nil {
		return ports.SignResult{}, &core.Error{Code: core.ExitIntegrity, Unmodified: true, What: "The package is unsafe to sign.", Why: err.Error()}
	}
	sum, err := fsx.HashTree(root, contentFiles(files))
	if err != nil {
		return ports.SignResult{}, err
	}
	version := rel.Bootstrap.Version
	doc := signatureDoc{Schema: 1, Algorithm: signatureAlgorithm, KeyID: id, Version: version, Checksum: sum,
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed(seed), signedMessage(version, sum)))}
	if err := fsx.WriteJSON(filepath.Join(root, filepath.FromSlash(SignatureFile)), doc); err != nil {
		return ports.SignResult{}, err
	}
	return ports.SignResult{Path: SignatureFile, KeyID: id, Version: version, Checksum: sum}, nil
}
