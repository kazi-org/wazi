// Package repairrequests stores private, operation-scoped AI request receipts
// and completed responses. It never persists request or input bodies.
package repairrequests

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const (
	maxResponse = 4 << 20
	maxOccupied = 16
	maxReserved = 64 << 20
	reservation = maxResponse
	ttl         = 24 * time.Hour
)

var (
	ErrUnknown = errors.New("AI request outcome is unknown; do not retry automatically")
	ErrFailed  = errors.New("AI request failed definitively; do not retry automatically")
	ErrFull    = errors.New("AI request cache capacity is full")
	ErrUnsafe  = errors.New("unsafe AI request cache")
	ErrCorrupt = errors.New("AI request receipt or response is corrupt")
	ErrBadKey  = errors.New("AI request key must be 64 lowercase hexadecimal characters")
	keyPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Failure marks a callback error known to be definitive, such as a local
// rejection before dispatch or a response that failed proposal validation.
type Failure struct{ Cause error }

func (f Failure) Error() string {
	if f.Cause == nil {
		return "definitive AI request failure"
	}
	return f.Cause.Error()
}
func (f Failure) Unwrap() error { return f.Cause }

// Failed classifies an error whose request outcome is known and which must not
// be retried. Unclassified callback errors remain uncertain.
func Failed(err error) error { return Failure{Cause: err} }

type receipt struct {
	Key       string    `json:"key"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
	Size      int64     `json:"size,omitempty"`
	Digest    string    `json:"digest,omitempty"`
}

// Run returns an exact-input cached response or invokes generate once. A
// dispatched request that does not reach a durable completed state is unknown
// and cannot be automatically sent again.
func Run(ctx context.Context, dir string, key string, generate func(context.Context) ([]byte, error)) ([]byte, error) {
	if !keyPattern.MatchString(key) {
		return nil, ErrBadKey
	}
	if generate == nil {
		return nil, errors.New("AI request generator is nil")
	}
	root, err := prepareRoot(dir)
	if err != nil {
		return nil, err
	}
	keyUnlock, err := lockNamed(ctx, root, ".key-"+key)
	if err != nil {
		return nil, err
	}
	defer keyUnlock()
	unlock, err := lock(ctx, root)
	if err != nil {
		return nil, err
	}
	p := filepath.Join(root, key+".json")
	r, found, err := readReceipt(p, key)
	if err != nil {
		unlock()
		return nil, err
	}
	if found {
		switch r.State {
		case "complete":
			if time.Now().Before(r.ExpiresAt) {
				b, e := readBody(root, r)
				unlock()
				return b, e
			}
			if err := removeExpired(root, p, key); err != nil {
				unlock()
				return nil, err
			}
		case "failed":
			unlock()
			return nil, fmt.Errorf("%w: receipt %s", ErrFailed, key)
		case "dispatching", "unknown":
			unlock()
			return nil, fmt.Errorf("%w: receipt %s", ErrUnknown, key)
		default:
			unlock()
			return nil, fmt.Errorf("%w: invalid state", ErrCorrupt)
		}
	}
	if err := reserve(root); err != nil {
		unlock()
		return nil, err
	}
	r = receipt{Key: key, State: "dispatching", CreatedAt: time.Now().UTC()}
	if err := writeReceipt(root, p, r); err != nil {
		unlock()
		return nil, err
	}
	unlock()

	body, genErr := generate(ctx)
	if genErr != nil || ctx.Err() != nil {
		var definitive Failure
		if ctx.Err() == nil && errors.As(genErr, &definitive) {
			markFailed(root, p, key)
			return nil, ErrFailed
		}
		markUnknown(root, p, key)
		return nil, ErrUnknown
	}
	if len(body) > maxResponse {
		markFailed(root, p, key)
		return nil, fmt.Errorf("%w: response exceeds 4 MiB", ErrFailed)
	}
	unlock, err = lock(context.Background(), root)
	if err != nil {
		markUnknown(root, p, key)
		return nil, fmt.Errorf("%w: lock completion receipt: %v", ErrUnknown, err)
	}
	defer unlock()
	current, ok, err := readReceipt(p, key)
	if err != nil || !ok || current.State != "dispatching" {
		return nil, fmt.Errorf("%w: dispatched receipt was not available to complete", ErrUnknown)
	}
	digest := sha256.Sum256(body)
	if err := writePrivate(filepath.Join(root, key+".body"), body); err != nil {
		markUnknownLocked(root, p, current)
		return nil, fmt.Errorf("%w: persist response: %v", ErrUnknown, err)
	}
	current.State = "complete"
	current.ExpiresAt = time.Now().UTC().Add(ttl)
	current.Size = int64(len(body))
	current.Digest = hex.EncodeToString(digest[:])
	if err := writeReceipt(root, p, current); err != nil {
		return nil, fmt.Errorf("%w: persist completion receipt: %v", ErrUnknown, err)
	}
	return append([]byte(nil), body...), nil
}

func prepareRoot(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("%w: empty cache path", ErrUnsafe)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve request cache: %w", err)
	}
	cur := string(filepath.Separator)
	for _, part := range strings.Split(strings.TrimPrefix(filepath.Clean(abs), cur), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		cur = filepath.Join(cur, part)
		st, e := os.Lstat(cur)
		if errors.Is(e, os.ErrNotExist) {
			if e = os.Mkdir(cur, 0700); e != nil && !errors.Is(e, os.ErrExist) {
				return "", fmt.Errorf("create request cache directory: %w", e)
			}
			if e == nil {
				if err := syncDirectory(filepath.Dir(cur)); err != nil {
					return "", fmt.Errorf("sync request cache parent: %w", err)
				}
				if err := syncDirectory(cur); err != nil {
					return "", fmt.Errorf("sync new request cache directory: %w", err)
				}
			}
			st, e = os.Lstat(cur)
		}
		if e != nil || st.Mode()&os.ModeSymlink != 0 || !st.IsDir() {
			return "", fmt.Errorf("%w: path component %q is not a real directory", ErrUnsafe, cur)
		}
	}
	st, err := os.Lstat(abs)
	if err != nil || st.Mode().Perm() != 0700 || st.Mode()&os.ModeSymlink != 0 || !st.IsDir() || !ownedByCurrentUser(st) {
		return "", fmt.Errorf("%w: cache root must be an owned private 0700 directory", ErrUnsafe)
	}
	return abs, nil
}

func lock(ctx context.Context, root string) (func(), error) {
	return lockNamed(ctx, root, ".lock")
}

func lockNamed(ctx context.Context, root, name string) (func(), error) {
	path := filepath.Join(root, name)
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, fmt.Errorf("open request cache lock: %w", err)
	}
	f := os.NewFile(uintptr(fd), path)
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || !ownedByCurrentUser(st) {
		_ = f.Close()
		return nil, fmt.Errorf("%w: lock must be an owned private regular file", ErrUnsafe)
	}
	for {
		err = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			_ = f.Close()
			return nil, fmt.Errorf("lock request cache: %w", err)
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
	return func() { _ = syscall.Flock(fd, syscall.LOCK_UN); _ = f.Close() }, nil
}

func readReceipt(path, key string) (receipt, bool, error) {
	b, err := readPrivate(path)
	if errors.Is(err, os.ErrNotExist) {
		return receipt{}, false, nil
	}
	if err != nil {
		return receipt{}, false, err
	}
	var r receipt
	if json.Unmarshal(b, &r) != nil || r.Key != key || !keyPattern.MatchString(r.Key) {
		return receipt{}, false, fmt.Errorf("%w: invalid receipt", ErrCorrupt)
	}
	return r, true, nil
}

func ownedByCurrentUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Getuid())
}

func readPrivate(path string) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || !ownedByCurrentUser(st) {
		return nil, fmt.Errorf("%w: %q must be an owned private regular file", ErrUnsafe, filepath.Base(path))
	}
	b, err := io.ReadAll(io.LimitReader(f, maxResponse+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxResponse {
		return nil, fmt.Errorf("%w: oversized cache file", ErrCorrupt)
	}
	return b, nil
}

func readBody(root string, r receipt) ([]byte, error) {
	b, err := readPrivate(filepath.Join(root, r.Key+".body"))
	if err != nil {
		return nil, fmt.Errorf("%w: cached response unavailable", ErrCorrupt)
	}
	d := sha256.Sum256(b)
	if int64(len(b)) != r.Size || hex.EncodeToString(d[:]) != r.Digest {
		return nil, fmt.Errorf("%w: cached response digest mismatch", ErrCorrupt)
	}
	return b, nil
}

func reserve(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("list request cache: %w", err)
	}
	occupied, used := 0, int64(0)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		if e.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink receipt", ErrUnsafe)
		}
		b, er := readPrivate(filepath.Join(root, name))
		if er != nil {
			return fmt.Errorf("%w: cannot account for occupied capacity", ErrCorrupt)
		}
		var r receipt
		if json.Unmarshal(b, &r) != nil || !keyPattern.MatchString(r.Key) || name != r.Key+".json" {
			return fmt.Errorf("%w: cannot account for receipt", ErrCorrupt)
		}
		if r.State == "complete" && !time.Now().Before(r.ExpiresAt) {
			if err := removeExpired(root, filepath.Join(root, name), r.Key); err != nil {
				return fmt.Errorf("remove expired request cache entry: %w", err)
			}
			continue
		}
		occupied++
		used += reservation
	}
	if occupied >= maxOccupied || used+reservation > maxReserved {
		return ErrFull
	}
	return nil
}

func writeReceipt(root, path string, r receipt) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	tmp := filepath.Join(root, ".receipt-"+r.Key+"-tmp")
	_ = os.Remove(tmp)
	if err := writePrivate(tmp, b); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return syncDirectory(root)
}

func writePrivate(path string, b []byte) error {
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_EXCL|syscall.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return fmt.Errorf("create private cache file: %w", err)
	}
	f := os.NewFile(uintptr(fd), path)
	_, writeErr := f.Write(b)
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	return nil
}

func syncDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func removeExpired(root, path, key string) error {
	if err := os.Remove(filepath.Join(root, key+".body")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return syncDirectory(root)
}

func markFailed(root, path, key string) {
	unlock, err := lock(context.Background(), root)
	if err != nil {
		return
	}
	defer unlock()
	r, ok, err := readReceipt(path, key)
	if err == nil && ok {
		r.State = "failed"
		r.ExpiresAt = time.Time{}
		r.Size = 0
		r.Digest = ""
		_ = writeReceipt(root, path, r)
	}
}

func markUnknown(root, path, key string) {
	ctx := context.Background()
	unlock, err := lock(ctx, root)
	if err != nil {
		return
	}
	defer unlock()
	r, ok, err := readReceipt(path, key)
	if err == nil && ok {
		markUnknownLocked(root, path, r)
	}
}

func markUnknownLocked(root, path string, r receipt) {
	r.State = "unknown"
	r.ExpiresAt = time.Time{}
	r.Size = 0
	r.Digest = ""
	if err := writeReceipt(root, path, r); err != nil {
		return
	}
	if err := os.Remove(filepath.Join(root, r.Key+".body")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}
	_ = syncDirectory(root)
}
