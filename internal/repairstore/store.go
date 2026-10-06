// Package repairstore stores immutable, source-bound repair proposals and
// applies them with digest and inode compare-and-swap checks.
package repairstore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"
)

const maxSourceSize = 4 << 20

var (
	ErrInvalidSource  = errors.New("invalid repair source")
	ErrInvalidStore   = errors.New("invalid repair store")
	ErrStale          = errors.New("repair source is stale")
	ErrCorrupt        = errors.New("repair candidate is corrupt")
	ErrApplyUncertain = errors.New("repair apply completed but directory sync failed; reconcile source and backup hashes")
)

type Source struct {
	Path  string
	Bytes []byte
	Mode  os.FileMode
}

type Manifest struct {
	ID              string `json:"id"`
	SourcePath      string `json:"sourcePath"`
	SourceDigest    string `json:"sourceDigest"`
	CandidateDigest string `json:"candidateDigest"`
	Profile         string `json:"profile"`
}

type fileIdentity struct {
	dev, ino uint64
	mode     os.FileMode
	size     int64
	mtime    int64
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// ReadSource reads one canonical, regular Markdown file without following a
// symlink in any path component.
func ReadSource(path string) (Source, error) {
	p, err := canonicalNoSymlinks(path)
	if err != nil {
		return Source{}, fmt.Errorf("resolve repair source: %w", err)
	}
	f, err := openNoFollow(p)
	if err != nil {
		return Source{}, fmt.Errorf("open repair source: %w", err)
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return Source{}, fmt.Errorf("stat repair source: %w", err)
	}
	if !before.Mode().IsRegular() || !strings.EqualFold(filepath.Ext(p), ".md") || before.Size() > maxSourceSize {
		return Source{}, fmt.Errorf("%w: expected a regular .md file no larger than 4 MiB", ErrInvalidSource)
	}
	b, err := io.ReadAll(io.LimitReader(f, maxSourceSize+1))
	if err != nil {
		return Source{}, fmt.Errorf("read repair source: %w", err)
	}
	if len(b) > maxSourceSize || !utf8.Valid(b) || bytes.IndexByte(b, 0) >= 0 {
		return Source{}, fmt.Errorf("%w: source must be valid UTF-8 without NUL and no larger than 4 MiB", ErrInvalidSource)
	}
	after, err := f.Stat()
	if err != nil {
		return Source{}, fmt.Errorf("restat repair source: %w", err)
	}
	if !sameIdentity(identity(before), identity(after)) {
		return Source{}, fmt.Errorf("%w: source changed while reading", ErrStale)
	}
	return Source{Path: p, Bytes: b, Mode: before.Mode().Perm()}, nil
}

func Save(dir string, source Source, candidate []byte, profile string) (Manifest, error) {
	return saveWithSync(dir, source, candidate, profile, syncDir)
}

func saveWithSync(dir string, source Source, candidate []byte, profile string, syncDirectory func(string) error) (Manifest, error) {
	p, err := canonicalNoSymlinks(source.Path)
	if err != nil || p != source.Path {
		return Manifest{}, fmt.Errorf("%w: source path must be canonical", ErrInvalidSource)
	}
	if !strings.EqualFold(filepath.Ext(source.Path), ".md") || len(source.Bytes) > maxSourceSize || len(candidate) > maxSourceSize || !utf8.Valid(source.Bytes) || bytes.IndexByte(source.Bytes, 0) >= 0 || !utf8.Valid(candidate) || bytes.IndexByte(candidate, 0) >= 0 {
		return Manifest{}, fmt.Errorf("%w: source and candidate must be bounded UTF-8 without NUL", ErrInvalidSource)
	}
	if profile == "" || len(profile) > 256 || !utf8.ValidString(profile) || strings.IndexByte(profile, 0) >= 0 {
		return Manifest{}, fmt.Errorf("%w: invalid profile", ErrInvalidSource)
	}
	if err := validateSourceSnapshot(source); err != nil {
		return Manifest{}, err
	}
	root, err := prepareStoreWithSync(dir, source.Path, syncDirectory)
	if err != nil {
		return Manifest{}, err
	}
	sd, cd := digest(source.Bytes), digest(candidate)
	id := digest([]byte(source.Path + "\x00" + sd + "\x00" + cd + "\x00" + profile))
	m := Manifest{ID: id, SourcePath: source.Path, SourceDigest: sd, CandidateDigest: cd, Profile: profile}
	entry := filepath.Join(root, id)
	if err := makePrivateDir(root, entry, syncDirectory); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return Manifest{}, fmt.Errorf("create candidate directory: %w", err)
		}
		old, e := readManifest(root, id)
		if e != nil || old != m {
			return Manifest{}, fmt.Errorf("%w: existing candidate identity conflicts", ErrCorrupt)
		}
		existingCandidate, e := readPrivate(filepath.Join(entry, "candidate.md"))
		if e != nil || !bytes.Equal(existingCandidate, candidate) {
			return Manifest{}, fmt.Errorf("%w: existing candidate bytes conflict", ErrCorrupt)
		}
		existingSource, e := readPrivate(filepath.Join(entry, "source.original"))
		if e != nil || !bytes.Equal(existingSource, source.Bytes) {
			return Manifest{}, fmt.Errorf("%w: existing source bytes conflict", ErrCorrupt)
		}
		return m, nil
	}
	if err := writeExclusive(filepath.Join(entry, "manifest.json"), mustJSON(m), 0600); err != nil {
		return Manifest{}, err
	}
	if err := writeExclusive(filepath.Join(entry, "candidate.md"), candidate, 0600); err != nil {
		return Manifest{}, err
	}
	if err := writeExclusive(filepath.Join(entry, "source.original"), source.Bytes, 0600); err != nil {
		return Manifest{}, err
	}
	if err := syncDir(entry); err != nil {
		return Manifest{}, err
	}
	if err := syncDir(root); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func Apply(dir, id string) (string, error) {
	return applyWithSync(dir, id, syncDir)
}

func applyWithSync(dir, id string, syncDirectory func(string) error) (string, error) {
	return applyWithSourceLockHook(dir, id, syncDirectory, nil)
}

// applyWithSourceLockHook exposes a synchronization point to filesystem tests
// after the original source descriptor is opened and before its flock is
// acquired. Production calls use no hook.
func applyWithSourceLockHook(dir, id string, syncDirectory func(string) error, beforeSourceLock func()) (string, error) {
	if !validID(id) {
		return "", fmt.Errorf("%w: invalid candidate id", ErrInvalidStore)
	}
	root, err := prepareStore(dir, "")
	if err != nil {
		return "", err
	}
	initialManifest, err := readManifest(root, id)
	if err != nil {
		return "", err
	}
	if _, err := prepareStore(root, initialManifest.SourcePath); err != nil {
		return "", err
	}
	// Candidate-specific locks would permit two different proposals for one
	// plan to pass CAS at once. Serialize every apply to a canonical path.
	lockPath := filepath.Join(root, ".lock-source-"+digest([]byte(initialManifest.SourcePath)))
	lockFD, err := syscall.Open(lockPath, syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return "", fmt.Errorf("open candidate lock: %w", err)
	}
	lock := os.NewFile(uintptr(lockFD), lockPath)
	lockInfo, err := lock.Stat()
	if err != nil || !lockInfo.Mode().IsRegular() || lockInfo.Mode().Perm() != 0600 {
		_ = lock.Close()
		return "", fmt.Errorf("%w: unsafe candidate lock", ErrInvalidStore)
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return "", fmt.Errorf("lock candidate: %w", err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	m, err := readManifest(root, id)
	if err != nil {
		return "", err
	}
	if m != initialManifest {
		return "", fmt.Errorf("%w: manifest changed while acquiring lock", ErrCorrupt)
	}
	entry := filepath.Join(root, id)
	candidate, err := readPrivate(filepath.Join(entry, "candidate.md"))
	if err != nil {
		return "", err
	}
	original, err := readPrivate(filepath.Join(entry, "source.original"))
	if err != nil {
		return "", err
	}
	if digest(candidate) != m.CandidateDigest || digest(original) != m.SourceDigest {
		return "", fmt.Errorf("%w: stored bytes do not match manifest", ErrCorrupt)
	}
	if m.CandidateDigest == m.SourceDigest {
		return "", fmt.Errorf("%w: candidate makes no source change", ErrInvalidSource)
	}
	p, err := canonicalNoSymlinks(m.SourcePath)
	if err != nil || p != m.SourcePath {
		return "", fmt.Errorf("%w: source path changed or contains symlink", ErrInvalidSource)
	}
	f, err := openNoFollow(p)
	if err != nil {
		return "", fmt.Errorf("reopen source: %w", err)
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return "", err
	}
	if beforeSourceLock != nil {
		beforeSourceLock()
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return "", fmt.Errorf("lock repair source: %w", err)
	}
	// The descriptor may have waited on the old inode while another process
	// atomically replaced the canonical path. In that case its flock no longer
	// protects the file currently named by the source path.
	canonicalAfterLock, canonicalErr := canonicalNoSymlinks(m.SourcePath)
	pathAfterLock, pathErr := os.Lstat(p)
	if canonicalErr != nil || canonicalAfterLock != m.SourcePath || pathErr != nil || !sameInode(identity(opened), identity(pathAfterLock)) {
		return "", ErrStale
	}
	initial, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !sameInode(identity(opened), identity(initial)) {
		return "", ErrStale
	}
	if !initial.Mode().IsRegular() || initial.Mode()&os.ModeSetuid != 0 || initial.Mode()&os.ModeSetgid != 0 || initial.Mode()&os.ModeSticky != 0 {
		return "", fmt.Errorf("%w: special source mode refused", ErrInvalidSource)
	}
	if initial.Size() > maxSourceSize {
		return "", fmt.Errorf("%w: source exceeds size limit", ErrInvalidSource)
	}
	current, err := io.ReadAll(io.LimitReader(f, maxSourceSize+1))
	if err != nil {
		return "", err
	}
	if len(current) > maxSourceSize || digest(current) != m.SourceDigest || !bytes.Equal(current, original) {
		return "", ErrStale
	}
	readStat, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !sameIdentity(identity(initial), identity(readStat)) {
		return "", ErrStale
	}
	// Recheck the path after reading. This detects path replacement before the
	// backup and narrows the unavoidable race with non-cooperating editors.
	pathStat, err := os.Lstat(p)
	if err != nil || !sameIdentity(identity(initial), identity(pathStat)) {
		return "", ErrStale
	}
	backup := filepath.Join(entry, "backup-"+m.SourceDigest+".md")
	if err := writeExactOrVerify(backup, original, 0600); err != nil {
		return "", err
	}
	if err := syncDirectory(entry); err != nil {
		return "", err
	}
	if err := replaceAtomically(p, candidate, initial.Mode().Perm(), identity(initial), m.SourceDigest, syncDirectory); err != nil {
		if errors.Is(err, ErrApplyUncertain) {
			return backup, err
		}
		return "", err
	}
	return backup, nil
}

func validateSourceSnapshot(s Source) error {
	f, err := openNoFollow(s.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	b, err := io.ReadAll(io.LimitReader(f, maxSourceSize+1))
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || len(b) > maxSourceSize || !bytes.Equal(b, s.Bytes) || st.Mode().Perm() != s.Mode.Perm() {
		return ErrStale
	}
	return nil
}

// Prepare qualifies and creates private repair storage outside the selected source
// repository. Callers use it before persisting explicit AI request state.
func Prepare(dir, source string) (string, error) {
	return prepareStore(dir, source)
}

func prepareStore(dir, source string) (string, error) {
	return prepareStoreWithSync(dir, source, syncDir)
}

func prepareStoreWithSync(dir, source string, syncDirectory func(string) error) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("%w: empty store path", ErrInvalidStore)
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	root = filepath.Clean(root)
	if source != "" {
		gitRoot := findGitRoot(source)
		if gitRoot == "" {
			gitRoot = filepath.Dir(source)
		}
		if within(root, gitRoot) {
			return "", fmt.Errorf("%w: store must be outside source repository", ErrInvalidStore)
		}
	}
	root, err = canonicalPathCreateWithSync(root, syncDirectory)
	if err != nil {
		return "", err
	}
	st, err := os.Lstat(root)
	if err != nil {
		return "", err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 || !ownedByCurrentUser(st) {
		return "", fmt.Errorf("%w: store root must be an owned private directory (0700)", ErrInvalidStore)
	}
	return root, nil
}

func readManifest(root, id string) (Manifest, error) {
	if !validID(id) {
		return Manifest{}, fmt.Errorf("%w: invalid id", ErrInvalidStore)
	}
	e := filepath.Join(root, id)
	st, err := os.Lstat(e)
	if err != nil {
		return Manifest{}, err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 || !ownedByCurrentUser(st) {
		return Manifest{}, ErrCorrupt
	}
	b, err := readPrivate(filepath.Join(e, "manifest.json"))
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("%w: manifest parse: %v", ErrCorrupt, err)
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return Manifest{}, ErrCorrupt
	}
	if m.ID != id || !validID(m.ID) || !validDigest(m.SourceDigest) || !validDigest(m.CandidateDigest) || m.SourcePath == "" || m.Profile == "" {
		return Manifest{}, ErrCorrupt
	}
	want := digest([]byte(m.SourcePath + "\x00" + m.SourceDigest + "\x00" + m.CandidateDigest + "\x00" + m.Profile))
	if want != id {
		return Manifest{}, ErrCorrupt
	}
	return m, nil
}

func readPrivate(path string) ([]byte, error) {
	f, err := openNoFollow(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || !ownedByCurrentUser(st) {
		return nil, fmt.Errorf("%w: unsafe private file %s", ErrCorrupt, filepath.Base(path))
	}
	b, err := io.ReadAll(io.LimitReader(f, maxSourceSize+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxSourceSize {
		return nil, fmt.Errorf("%w: private file too large", ErrCorrupt)
	}
	return b, nil
}
func mustJSON(v any) []byte { b, _ := json.MarshalIndent(v, "", "  "); return append(b, '\n') }
func writeExclusive(path string, b []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("create private candidate file: %w", err)
	}
	_, werr := f.Write(b)
	if werr == nil {
		werr = f.Sync()
	}
	cerr := f.Close()
	if werr != nil {
		return werr
	}
	if cerr != nil {
		return cerr
	}
	return nil
}
func writeExactOrVerify(path string, b []byte, mode os.FileMode) error {
	err := writeExclusive(path, b, mode)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrExist) {
		return err
	}
	old, e := readPrivate(path)
	if e != nil || !bytes.Equal(old, b) {
		return fmt.Errorf("%w: backup conflicts", ErrCorrupt)
	}
	return nil
}
func syncDir(p string) error {
	f, e := os.Open(p)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
func replaceAtomically(path string, b []byte, mode os.FileMode, expected fileIdentity, expectedDigest string, syncParent func(string) error) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".wazi-repair-*")
	if err != nil {
		return fmt.Errorf("create source temp: %w", err)
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(b); err == nil {
		err = f.Chmod(mode)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write source temp: %w", err)
	}
	// Recheck immediately before rename. A non-cooperating editor can still
	// race this last Lstat/rename interval; cooperating apply calls are locked.
	st, e := os.Lstat(path)
	if e != nil || st.Mode()&os.ModeSymlink != 0 || !sameIdentity(expected, identity(st)) {
		return ErrStale
	}
	check, e := openNoFollow(path)
	if e != nil {
		return ErrStale
	}
	opened, statErr := check.Stat()
	if statErr != nil || !sameIdentity(expected, identity(opened)) {
		_ = check.Close()
		return ErrStale
	}
	now, readErr := io.ReadAll(io.LimitReader(check, maxSourceSize+1))
	readStat, restatErr := check.Stat()
	closeErr = check.Close()
	if readErr != nil || restatErr != nil || closeErr != nil || !sameIdentity(expected, identity(readStat)) || digest(now) != expectedDigest {
		return ErrStale
	}
	if e = os.Rename(tmp, path); e != nil {
		return fmt.Errorf("replace repair source: %w", e)
	}
	if e = syncParent(dir); e != nil {
		return fmt.Errorf("%w: %w", ErrApplyUncertain, e)
	}
	return nil
}

func canonicalNoSymlinks(p string) (string, error) {
	if p == "" {
		return "", ErrInvalidSource
	}
	a, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	a = filepath.Clean(a)
	vol := filepath.VolumeName(a)
	rest := strings.TrimPrefix(a, vol)
	cur := vol + string(os.PathSeparator)
	for _, part := range strings.Split(strings.TrimPrefix(rest, string(os.PathSeparator)), string(os.PathSeparator)) {
		if part == "" {
			continue
		}
		cur = filepath.Join(cur, part)
		st, e := os.Lstat(cur)
		if e != nil {
			return "", e
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%w: symlink path component", ErrInvalidSource)
		}
	}
	return a, nil
}
func canonicalPathCreate(p string) (string, error) {
	return canonicalPathCreateWithSync(p, syncDir)
}

// canonicalPathCreateWithSync creates missing path components one at a time
// and makes each directory entry durable before proceeding to its child.
// The sync function is passed explicitly so failures can be exercised in tests.
func canonicalPathCreateWithSync(p string, syncParent func(string) error) (string, error) {
	a, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	a = filepath.Clean(a)
	vol := filepath.VolumeName(a)
	cur := vol + string(os.PathSeparator)
	for _, part := range strings.Split(strings.TrimPrefix(strings.TrimPrefix(a, vol), string(os.PathSeparator)), string(os.PathSeparator)) {
		if part == "" {
			continue
		}
		cur = filepath.Join(cur, part)
		st, e := os.Lstat(cur)
		if errors.Is(e, os.ErrNotExist) {
			if e = makePrivateDir(filepath.Dir(cur), cur, syncParent); e != nil && !errors.Is(e, os.ErrExist) {
				return "", e
			}
			st, e = os.Lstat(cur)
		}
		if e != nil {
			return "", e
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%w: symlink path component", ErrInvalidStore)
		}
	}
	return a, nil
}

// makePrivateDir syncs the parent immediately after creating a private
// directory. A failure is returned before any child entry can be written.
func makePrivateDir(parent, path string, syncParent func(string) error) error {
	if err := os.Mkdir(path, 0700); err != nil {
		return err
	}
	if err := syncParent(parent); err != nil {
		return fmt.Errorf("sync parent of private directory: %w", err)
	}
	return nil
}
func openNoFollow(p string) (*os.File, error) {
	fd, e := syscall.Open(p, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return nil, e
	}
	return os.NewFile(uintptr(fd), p), nil
}
func identity(st os.FileInfo) fileIdentity {
	sys, ok := st.Sys().(*syscall.Stat_t)
	i := fileIdentity{mode: st.Mode(), size: st.Size(), mtime: st.ModTime().UnixNano()}
	if ok {
		i.dev = uint64(sys.Dev)
		i.ino = uint64(sys.Ino)
	}
	return i
}
func sameIdentity(a, b fileIdentity) bool {
	return a.dev == b.dev && a.ino == b.ino && a.mode == b.mode && a.size == b.size && a.mtime == b.mtime
}
func sameInode(a, b fileIdentity) bool { return a.dev == b.dev && a.ino == b.ino }
func validID(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, e := hex.DecodeString(s)
	return e == nil && strings.ToLower(s) == s
}
func validDigest(s string) bool { return validID(s) }
func within(path, root string) bool {
	rel, e := filepath.Rel(root, path)
	return e == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}
func findGitRoot(p string) string {
	d := filepath.Dir(p)
	for {
		if st, e := os.Lstat(filepath.Join(d, ".git")); e == nil && (st.IsDir() || st.Mode().IsRegular()) {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}
func ownedByCurrentUser(st os.FileInfo) bool {
	sys, ok := st.Sys().(*syscall.Stat_t)
	return ok && int(sys.Uid) == os.Getuid()
}
