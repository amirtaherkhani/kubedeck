package enrollment

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

const maxState = 1 << 20

// Store owns a single-writer process lock for both journals. It never rewrites .env.
type Store struct {
	Dir  string
	lock *os.File
}

func owner(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == uint32(os.Getuid())
}
func OpenStore(dir string) (*Store, error) {
	if !filepath.IsAbs(dir) {
		return nil, errors.New("absolute_state_directory_required")
	}
	info, e := os.Lstat(dir)
	if e != nil || !info.IsDir() || info.Mode().Perm() != 0700 || !owner(info) {
		return nil, errors.New("private_state_directory_required")
	}
	f, e := os.OpenFile(filepath.Join(dir, "controller.lock"), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if e != nil {
		return nil, errors.New("state_lock_unavailable")
	}
	fi, e := f.Stat()
	if e != nil || !fi.Mode().IsRegular() || fi.Mode().Perm() != 0600 || !owner(fi) {
		f.Close()
		return nil, errors.New("invalid_state_lock")
	}
	if syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		f.Close()
		return nil, errors.New("controller_already_running")
	}
	return &Store{Dir: dir, lock: f}, nil
}
func (s *Store) Close() error {
	if s.lock == nil {
		return nil
	}
	_ = syscall.Flock(int(s.lock.Fd()), syscall.LOCK_UN)
	return s.lock.Close()
}
func readPrivate(path string) ([]byte, error) {
	f, e := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if e != nil {
		return nil, errors.New("private_file_unavailable")
	}
	defer f.Close()
	i, e := f.Stat()
	if e != nil || !i.Mode().IsRegular() || i.Mode().Perm() != 0600 || !owner(i) || i.Size() > maxState {
		return nil, errors.New("invalid_private_file")
	}
	b, e := io.ReadAll(io.LimitReader(f, maxState+1))
	if e != nil || len(b) > maxState {
		return nil, errors.New("invalid_private_file")
	}
	return b, nil
}
func (s *Store) Load(name string, out any) error {
	b, e := readPrivate(filepath.Join(s.Dir, name))
	if e != nil {
		return e
	}
	return decode(b, out)
}
func (s *Store) Exists(name string) bool {
	_, e := os.Lstat(filepath.Join(s.Dir, name))
	return !errors.Is(e, os.ErrNotExist)
}
func (s *Store) Save(name string, value any) error {
	if name != "session.json" && name != "ledger.json" {
		return errors.New("invalid_state_name")
	}
	b, e := json.Marshal(value)
	if e != nil || len(b) > maxState {
		return errors.New("invalid_state")
	}
	f, e := os.CreateTemp(s.Dir, ".pending-")
	if e != nil {
		return errors.New("state_write_failed")
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	fail := func() error { f.Close(); return errors.New("state_write_failed") }
	if f.Chmod(0600) != nil {
		return fail()
	}
	if _, e = f.Write(b); e != nil {
		return fail()
	}
	if f.Sync() != nil {
		return fail()
	}
	if f.Close() != nil {
		return errors.New("state_write_failed")
	}
	if os.Rename(tmp, filepath.Join(s.Dir, name)) != nil {
		return errors.New("state_write_failed")
	}
	d, e := os.Open(s.Dir)
	if e != nil {
		return errors.New("state_write_failed")
	}
	defer d.Close()
	if d.Sync() != nil {
		return errors.New("state_write_failed")
	}
	return nil
}
