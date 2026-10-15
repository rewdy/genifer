package preview

import "strings"

// Geometry reports output TTY dimensions in cells and terminal-reported pixels.
type Geometry struct{ Columns, Rows, PixelWidth, PixelHeight int }

func (g Geometry) Valid(columns, rows int) bool {
	return columns > 0 && rows > 0 && g.Columns == columns && g.Rows == rows && g.PixelWidth >= columns && g.PixelHeight >= rows
}

// Detect conservatively selects a transport for known, direct terminal sessions.
func Detect(tty bool, getenv func(string) string) Protocol {
	if !tty || getenv("TERM") == "dumb" {
		return None
	}
	for _, key := range []string{"TMUX", "STY", "SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		if getenv(key) != "" {
			return None
		}
	}
	if term := getenv("TERM"); strings.HasPrefix(term, "screen") || strings.HasPrefix(term, "tmux") {
		return None
	}
	switch getenv("TERM_PROGRAM") {
	case "ghostty":
		return Kitty
	case "iTerm.app":
		return ITerm2
	default:
		return None
	}
}
