//go:build linux

package overlay

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/system"
)

func scanForMountProgramIndicators(home string) (detected bool, err error) {
	err = filepath.WalkDir(home, mountProgramIndicatorWalker(home, &detected))
	return detected, err
}

func mountProgramIndicatorWalker(home string, detected *bool) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if *detected {
			return fs.SkipDir
		}
		if err != nil {
			if path != home && d != nil && d.IsDir() && errors.Is(err, syscall.ENOTDIR) {
				return nil
			}
			return err
		}
		basename := filepath.Base(path)
		if strings.HasPrefix(basename, archive.WhiteoutPrefix) {
			*detected = true
			return fs.SkipDir
		}
		if d.IsDir() {
			xattrs, err := system.Llistxattr(path)
			if err != nil && !errors.Is(err, unix.EOPNOTSUPP) {
				return err
			}
			for _, xattr := range xattrs {
				if strings.HasPrefix(xattr, "user.fuseoverlayfs.") || strings.HasPrefix(xattr, "user.containers.") {
					*detected = true
					return fs.SkipDir
				}
			}
		}
		return nil
	}
}
