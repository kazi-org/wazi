package observatory

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// lockFile uses an advisory kernel lock: concurrent processes serialize and
// process exit releases it without leaving a stale ownership marker.
func lockFile(path string) (func(), error) {
	if st, err := os.Lstat(path); err == nil && (st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular()) {
		return nil, errors.New("lock path must be a regular private file")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	if err = f.Chmod(0600); err != nil {
		_ = f.Close()
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("lock is held by another Wazi process: %w", err)
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
}
