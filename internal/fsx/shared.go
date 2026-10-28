package fsx

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// SharedFile is the one core file a host project may already have: AGENTS.md
// is an open convention other tools write too. GTT never takes it over and
// never refuses to install because of it (Canon 3.0 §28): it keeps its
// contract inside a delimited block of the host's file and touches only that
// block - on install, update, clean, export and rollback alike.
const SharedFile = "AGENTS.md"

const (
	blockBegin = "<!-- gtt:begin - GTT agent contract, maintained by GTT; edit outside this block -->"
	blockEnd   = "<!-- gtt:end -->"
)

// IsShared reports whether a core path is the shared instruction file.
func IsShared(rel string) bool { return filepath.ToSlash(rel) == SharedFile }

// splitBlock returns the host text around the GTT block, the block's content
// (verbatim, without its markers), and whether a block was found.
func splitBlock(text string) (host, block string, found bool) {
	// Markers count only on a line of their own: the contract may name them in its prose.
	start := strings.Index("\n"+text, "\n"+blockBegin)
	if start < 0 {
		return text, "", false
	}
	rest := text[start+len(blockBegin):]
	end := strings.LastIndex("\n"+rest, "\n"+blockEnd+"\n")
	if end < 0 && strings.HasSuffix(rest, "\n"+blockEnd) {
		end = len(rest) - len(blockEnd)
	}
	if end < 0 {
		return text, "", false
	}
	block = strings.TrimPrefix(rest[:end], "\n")
	block = strings.TrimSuffix(block, "\n")
	before := strings.TrimRight(text[:start], "\n")
	after := strings.TrimLeft(rest[end+len(blockEnd):], "\n")
	host = before
	if after != "" {
		if host != "" {
			host += "\n\n"
		}
		host += after
	}
	if host != "" && !strings.HasSuffix(host, "\n") {
		host += "\n"
	}
	return host, block, true
}

// Merge places GTT's contract in the host's file as one delimited block,
// replacing a previous block and keeping every host line.
func Merge(hostText, gttText string) string {
	host, _, _ := splitBlock(hostText)
	host = strings.TrimRight(host, "\n")
	out := ""
	if host != "" {
		out = host + "\n\n"
	}
	return out + blockBegin + "\n" + strings.TrimSuffix(gttText, "\n") + "\n" + blockEnd + "\n"
}

// HostText is the file without GTT's block, and whether anything of the host
// remains (false: the file is entirely GTT's and may be removed whole).
func HostText(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	host, _, found := splitBlock(string(data))
	if !found {
		return "", false
	}
	return host, strings.TrimSpace(host) != ""
}

// BlockHash is the hash of GTT's block in a shared file, comparable with the
// ledger's hash of the package file; ok is false when the file has no block.
func BlockHash(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	_, block, found := splitBlock(string(data))
	if !found {
		return "", false
	}
	sum := sha256.Sum256([]byte(block + "\n"))
	return hex.EncodeToString(sum[:]), true
}

// StripOrRemove takes GTT out of a shared file: the block goes, the host's
// lines stay; a file that is only GTT's contract is removed. A file with no
// GTT block that is not GTT's contract is the host's alone and is left as is
// (an init interrupted before the merge must not cost the host its file).
func StripOrRemove(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	host, _, found := splitBlock(string(data))
	switch {
	case found && strings.TrimSpace(host) != "":
		return writeAtomic(path, host)
	case found || strings.Contains(string(data), contractTitle):
		return os.RemoveAll(path)
	}
	return nil
}

// contractTitle identifies a file that is entirely GTT's agent contract.
const contractTitle = "GTT Bootstrap Agent Contract"

// WriteMerged writes GTT's contract into the host's existing file.
func WriteMerged(path, gttText string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return writeAtomic(path, Merge(string(data), gttText))
}

func writeAtomic(path, text string) error {
	tmp := path + ".gtt-tmp"
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	if err := os.WriteFile(tmp, []byte(text), mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
