//go:build linux

package overlay

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMountProgramIndicatorWalkerStaleDirectory(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "subdir")
	require.NoError(t, os.Mkdir(dir, 0o700))
	entries, err := os.ReadDir(home)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	entry := entries[0]
	require.True(t, entry.IsDir())
	require.NoError(t, os.Remove(dir))
	require.NoError(t, os.WriteFile(dir, nil, 0o600))
	_, readErr := os.ReadDir(dir)
	require.ErrorIs(t, readErr, syscall.ENOTDIR)

	detected := false
	walk := mountProgramIndicatorWalker(home, &detected)
	require.NoError(t, walk(dir, entry, readErr))
	require.False(t, detected)

	whiteout := filepath.Join(home, ".wh.deleted")
	require.NoError(t, os.WriteFile(whiteout, nil, 0o600))
	info, err := os.Lstat(whiteout)
	require.NoError(t, err)
	require.ErrorIs(t, walk(whiteout, fs.FileInfoToDirEntry(info), nil), fs.SkipDir)
	require.True(t, detected)
}

func TestMountProgramIndicatorWalkerErrors(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "subdir")
	require.NoError(t, os.Mkdir(dir, 0o700))
	info, err := os.Lstat(dir)
	require.NoError(t, err)
	directory := fs.FileInfoToDirEntry(info)
	file := filepath.Join(home, "file")
	require.NoError(t, os.WriteFile(file, nil, 0o600))
	info, err = os.Lstat(file)
	require.NoError(t, err)

	for _, test := range []struct {
		name  string
		path  string
		entry fs.DirEntry
		err   error
	}{
		{"root", home, directory, syscall.ENOTDIR},
		{"missing entry", dir, nil, syscall.ENOTDIR},
		{"file", file, fs.FileInfoToDirEntry(info), syscall.ENOTDIR},
		{"permission", dir, directory, syscall.EACCES},
		{"IO", dir, directory, syscall.EIO},
	} {
		t.Run(test.name, func(t *testing.T) {
			detected := false
			walk := mountProgramIndicatorWalker(home, &detected)
			pathErr := &os.PathError{Op: "open", Path: test.path, Err: test.err}
			require.ErrorIs(t, walk(test.path, test.entry, pathErr), pathErr)
			require.False(t, detected)
		})
	}
}

func TestScanForMountProgramIndicators(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(home, "subdir"), 0o700))
	detected, err := scanForMountProgramIndicators(home)
	require.NoError(t, err)
	require.False(t, detected)
	require.NoError(t, os.WriteFile(filepath.Join(home, "subdir", ".wh.deleted"), nil, 0o600))
	detected, err = scanForMountProgramIndicators(home)
	require.NoError(t, err)
	require.True(t, detected)
}
