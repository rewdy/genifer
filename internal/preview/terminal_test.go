package preview

import "testing"

func TestDetect(t *testing.T) {
	for _, tc := range []struct {
		name string
		tty  bool
		env  map[string]string
		want Protocol
	}{
		{"ghostty", true, map[string]string{"TERM_PROGRAM": "ghostty"}, Kitty},
		{"iterm", true, map[string]string{"TERM_PROGRAM": "iTerm.app"}, ITerm2},
		{"redirected", false, map[string]string{"TERM_PROGRAM": "ghostty"}, None},
		{"unknown", true, map[string]string{"TERM_PROGRAM": "other"}, None},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Detect(tc.tty, func(k string) string { return tc.env[k] }); got != tc.want {
				t.Fatal(got)
			}
		})
	}
	for _, key := range []string{"TMUX", "STY", "SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY", "TERM"} {
		env := map[string]string{"TERM_PROGRAM": "ghostty", key: "set"}
		if key == "TERM" {
			env[key] = "dumb"
		}
		if got := Detect(true, func(k string) string { return env[k] }); got != None {
			t.Fatalf("%s: %v", key, got)
		}
	}
	for _, term := range []string{"screen-256color", "tmux-256color"} {
		if Detect(true, func(k string) string {
			if k == "TERM" {
				return term
			}
			if k == "TERM_PROGRAM" {
				return "ghostty"
			}
			return ""
		}) != None {
			t.Fatal(term)
		}
	}
}
func TestGeometry(t *testing.T) {
	for _, g := range []Geometry{{}, {80, 24, 0, 0}, {80, 24, 800, 480}, {79, 24, 800, 480}, {80, 23, 800, 480}} {
		want := g == (Geometry{80, 24, 800, 480})
		if g.Valid(80, 24) != want {
			t.Fatal(g)
		}
	}
	if _, err := ReadGeometry(^uintptr(0)); err == nil {
		t.Fatal("invalid fd accepted")
	}
}
