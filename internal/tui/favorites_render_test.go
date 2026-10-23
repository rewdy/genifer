package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/provider"
)

// 3.3: a favorited row shows a yellow star before the name; a non-favorited row
// shows no star, and both keep the name in the same column.
func TestModelDelegateRendersYellowStar(t *testing.T) {
	fav := modelItem{m: provider.Model{Name: "Fav Model"}, favorite: true}
	plainItem := modelItem{m: provider.Model{Name: "Plain Model"}}

	l := list.New([]list.Item{fav, plainItem}, modelDelegate{}, 60, 4)
	l.Select(1) // highlight the non-favorite so selection styling can't be mistaken for the star

	var favBuf, plainBuf bytes.Buffer
	modelDelegate{}.Render(&favBuf, l, 0, fav)
	modelDelegate{}.Render(&plainBuf, l, 1, plainItem)
	favRow, plainRow := favBuf.String(), plainBuf.String()

	if !strings.Contains(ansi.Strip(favRow), "★") {
		t.Errorf("favorited row missing star; got %q", ansi.Strip(favRow))
	}
	if strings.Contains(ansi.Strip(plainRow), "★") {
		t.Errorf("non-favorited row must not show a star; got %q", ansi.Strip(plainRow))
	}
	// Yellow #f5c542 -> 245;197;65 (truecolor profile rounds down).
	const starRGB = "245;197;65"
	if !strings.Contains(favRow, starRGB) {
		t.Errorf("star should be yellow %q; got %q", starRGB, favRow)
	}
	if strings.Contains(plainRow, starRGB) {
		t.Errorf("non-favorited row must not use the star color; got %q", plainRow)
	}

	// Both names start in the same column: the star prefix and its blank
	// counterpart are the same width.
	favPlain := ansi.Strip(favRow)
	plain := ansi.Strip(plainRow)
	favName := ansi.StringWidth(favPlain[:strings.Index(favPlain, "Fav Model")])
	plainName := ansi.StringWidth(plain[:strings.Index(plain, "Plain Model")])
	if favName != plainName {
		t.Errorf("name column differs: favorite at %d, plain at %d\nfav:  %q\nplain:%q", favName, plainName, favPlain, plain)
	}
	if strings.Contains(favRow, "\n") || strings.Contains(plainRow, "\n") {
		t.Error("delegate rows must stay single-line")
	}
}

// 3.3: the picker footer names the favorite key.
func TestPickerFooterMentionsFavoriteKey(t *testing.T) {
	m := New(Deps{})
	m.phase = phasePicker
	footer := m.footerView()
	if !strings.Contains(ansi.Strip(footer), "favorite") {
		t.Errorf("picker footer missing favorite hint; got %q", ansi.Strip(footer))
	}
}
