//go:build linux

package common

import (
	"fmt"
	iofs "io/fs"
	"testing"
	"time"

	"github.com/google/cadvisor/fs"
	"github.com/stretchr/testify/assert"
)

type stubFsInfo struct {
	fs.FsInfo
	errs map[string]error
}

func (s stubFsInfo) GetDirUsage(dir string) (fs.UsageInfo, error) {
	return fs.UsageInfo{Bytes: 1, Inodes: 1}, s.errs[dir]
}

func TestFsHandlerDropsUnwalkableDirs(t *testing.T) {
	gone := fmt.Errorf("could not stat %q: %w", "/rootfs", iofs.ErrNotExist)
	denied := fmt.Errorf("unable to count inodes: %w", iofs.ErrPermission)
	busy := fmt.Errorf("device busy")

	for _, tc := range []struct {
		name         string
		rootErr      error
		extraErr     error
		wantRootfs   string
		wantExtraDir string
		wantErr      bool
	}{
		{name: "missing rootfs dropped", rootErr: gone, wantExtraDir: "/extra"},
		{name: "denied extra dropped", extraErr: denied, wantRootfs: "/rootfs"},
		{name: "both unwalkable", rootErr: gone, extraErr: denied},
		{name: "transient error kept", rootErr: busy, wantRootfs: "/rootfs", wantExtraDir: "/extra", wantErr: true},
		{name: "no error", wantRootfs: "/rootfs", wantExtraDir: "/extra"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := stubFsInfo{errs: map[string]error{"/rootfs": tc.rootErr, "/extra": tc.extraErr}}
			fh := NewFsHandler(time.Minute, "/rootfs", "/extra", info).(*realFsHandler)

			err := fh.update()

			assert.Equal(t, tc.wantErr, err != nil)
			assert.Equal(t, tc.wantRootfs, fh.rootfs)
			assert.Equal(t, tc.wantExtraDir, fh.extraDir)
			assert.Equal(t, tc.wantRootfs != "" || tc.wantExtraDir != "", fh.hasDirs())
		})
	}
}
