//go:build darwin || linux

package preview

import "golang.org/x/sys/unix"

// ReadGeometry reads the output TTY without querying the terminal input stream.
func ReadGeometry(fd uintptr) (Geometry, error) {
	size, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	if err != nil {
		return Geometry{}, err
	}
	return Geometry{int(size.Col), int(size.Row), int(size.Xpixel), int(size.Ypixel)}, nil
}
