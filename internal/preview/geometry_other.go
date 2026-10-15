//go:build !darwin && !linux

package preview

import "fmt"

func ReadGeometry(fd uintptr) (Geometry, error) {
	return Geometry{}, fmt.Errorf("preview: terminal pixel geometry unavailable")
}
