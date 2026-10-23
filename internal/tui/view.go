package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/preview"
	"github.com/rewdy/genifer/internal/provider"
)

var (
	styleFaint = lipgloss.NewStyle().Foreground(lipgloss.Color("#8b85a0"))
	styleKey   = lipgloss.NewStyle().Foreground(lipgloss.Color("#b9a6ff")).Bold(true)
	styleErr   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b6b"))
	styleOK    = lipgloss.NewStyle().Foreground(lipgloss.Color("#7ee787"))
)

// modelErrorText produces an actionable message for a model-list failure.
func modelErrorText(err error) string {
	switch {
	case errors.Is(err, provider.ErrNoAPIKey):
		return "No API key configured — set your OpenRouter API key (see `genifer config`)"
	case errors.Is(err, provider.ErrAuth):
		return "Could not load models: authentication failed — your API key was rejected"
	default:
		return "Could not load models: " + err.Error()
	}
}

// pasteErrorText produces a transient, non-fatal status for a failed image
// attach (clipboard paste or file path). Every failure mode collapses to a
// short, user-facing hint; none is an application error.
func pasteErrorText(err error) string {
	switch {
	case errors.Is(err, ErrNoClipboardImage):
		return "No image in clipboard"
	case errors.Is(err, errRefImageTooLarge):
		return "Image is too large to attach"
	case errors.Is(err, errRefImageUndecodable):
		return "Clipboard data is not a supported image"
	default:
		return "Could not attach image: " + err.Error()
	}
}

func (m Model) View() string {
	if m.quit {
		return ""
	}
	header := RenderHeader(m.width, m.deps.Version)
	body := ansi.Wrap(m.bodyView(), m.contentWidth(), "")
	footer := m.footerView()
	// Match the header's horizontal padding so the content is not butted right
	// against the terminal edge.
	region := lipgloss.JoinVertical(lipgloss.Left, body, "", footer)
	padded := lipgloss.NewStyle().Background(headerBG).Padding(0, horizontalPadding(m.width)).Render(region)
	content := lipgloss.JoinVertical(lipgloss.Left, header, "", padded)
	// Re-assert the app background across every line so no embedded widget can
	// leave the terminal's default background showing through.
	content = paintBackground(content, m.width)

	// Fill the whole terminal with the app background so the user's terminal
	// background never shows through / clashes with the TUI.
	canvas := lipgloss.NewStyle().Background(headerBG).Foreground(lipgloss.Color("#e6e1f2"))
	if m.width > 0 {
		canvas = canvas.Width(m.width)
	}
	if m.height > 0 {
		canvas = canvas.Height(m.height)
	}
	rendered := canvas.Render(content)
	if m.deps.Preview != nil {
		marker := preview.Marker(m.previewPlacement())
		// Repeat at the end so a clear marker survives Bubble Tea clipping an
		// oversized text-only frame from the top on very small terminals.
		return marker + rendered + marker
	}
	return rendered
}

func (m Model) bodyView() string {
	switch m.phase {
	case phaseOnboarding:
		return m.onboardingView()
	case phasePicker:
		return m.pickerView()
	case phaseCompose:
		return m.composeView()
	case phaseAspect:
		return m.aspectView()
	case phaseReview:
		return m.reviewView()
	case phaseGenerating:
		return m.spinner.View() + " Generating... (esc to cancel)"
	case phaseResult:
		return m.resultView()
	case phaseHistory:
		return m.historyView()
	case phaseConfig:
		return m.configView()
	}
	return ""
}

// welcomeBanner is a bold "WELCOME" wordmark shown on first run, tinted with
// the app's pink→purple palette.
var welcomeBanner = []string{
	"╦ ╦ ╔═╗ ╦   ╔═╗ ╔═╗ ╔╦╗ ╔═╗",
	"║║║ ║╣  ║   ║   ║ ║ ║║║ ║╣ ",
	"╚╩╝ ╚═╝ ╩═╝ ╚═╝ ╚═╝ ╩ ╩ ╚═╝",
}

func (m Model) onboardingView() string {
	banner := make([]string, len(welcomeBanner))
	style := lipgloss.NewStyle().Foreground(colorGen).Background(headerBG).Bold(true)
	for i, line := range welcomeBanner {
		banner[i] = style.Render(line)
	}
	title := lipgloss.JoinVertical(lipgloss.Left, banner...)

	intro := styleFaint.Render("Let's set up genifer. First, how should it get your OpenRouter API key?")

	var body string
	switch m.onboard.step {
	case stepChoose:
		var rows []string
		for i, c := range methodChoices {
			marker := "  "
			label := c.label
			if i == m.onboard.cursor {
				marker = styleKey.Render("› ")
				label = styleKey.Render(label)
			}
			rows = append(rows, marker+label+"  "+styleFaint.Render(c.hint))
		}
		body = strings.Join(rows, "\n")
	case stepInput:
		body = styleFaint.Render(promptFor(m.onboard.selectedMethod())) + "\n\n" + m.onboard.input.View()
	case stepConfirm:
		body = styleErr.Render("⚠ The key will be stored as plain text in config.yaml.") +
			"\n" + styleFaint.Render("Press y to save it, or n to go back.")
	}

	parts := []string{title, "", intro, "", body}
	if m.onboard.writeErr != nil {
		parts = append(parts, "", styleErr.Render("Could not write config: "+m.onboard.writeErr.Error()))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) pickerView() string {
	if !m.modelsReady {
		return m.spinner.View() + " Loading models..."
	}
	if m.modelsErr != nil {
		return styleErr.Render(modelErrorText(m.modelsErr))
	}
	if len(m.models) == 0 {
		return styleFaint.Render("No image models available.")
	}
	return "Select a model:\n\n" + m.picker.View()
}

func capHint(c provider.Capabilities) string {
	var parts []string
	if c.AcceptsReferenceImages {
		if c.ReferenceMin != nil && *c.ReferenceMin > 0 {
			parts = append(parts, "refs required")
		} else {
			parts = append(parts, "img2img")
		}
	}
	if len(c.OutputFormats) == 1 && c.OutputFormats[0] == "svg" {
		parts = append(parts, "SVG")
	}
	if len(c.AspectRatios) > 0 {
		parts = append(parts, "aspect")
	}
	if c.SupportsSeed {
		parts = append(parts, "seed")
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func (m Model) composeView() string {
	caps := m.currentCaps()
	ctrls := controlsFor(caps)
	var settings []string
	if len(caps.OutputFormats) == 1 && caps.OutputFormats[0] == "svg" {
		settings = append(settings, "Output: SVG")
	}
	if ctrls.ShowAspectRatio {
		settings = append(settings, "Aspect: "+m.aspectRatio)
	}
	if caps.SupportsTransparentBackground {
		state := "off"
		if m.transparentBackground {
			state = "on"
		}
		settings = append(settings, "Transparent: "+state)
	}
	if ctrls.ShowSeed {
		settings = append(settings, "Seed supported")
	}
	var details []string
	if len(settings) > 0 {
		details = append(details, strings.Join(settings, " · "))
	}
	if guidance := m.referenceGuidance(); guidance != "" {
		details = append(details, guidance, "Reference file: @/path/to/image")
	}
	// Paint the prompt over the input background so the whole input box reads as
	// one area. The textarea's own placeholder path leaves its padding unstyled,
	// so this is repaired here rather than left to the terminal default.
	prompt := paintOver(m.prompt.View(), m.prompt.Width(), inputBG)
	out := fmt.Sprintf("Model: %s\n\n%s", m.currentModelID(), prompt)
	if len(details) > 0 {
		// Wrap plain text first, then style each physical row. Wrapping an already
		// styled string can leave continuation rows without their foreground.
		rows := strings.Split(ansi.Wrap(strings.Join(details, "\n"), m.contentWidth(), ""), "\n")
		for i, row := range rows {
			rows[i] = styleFaint.Render(row)
		}
		out += "\n\n" + strings.Join(rows, "\n")
	}
	if ctrls.ShowReference && len(m.refImages) > 0 {
		out += "\n" + refImagePills(len(m.refImages))
	}
	return out
}

// styleAspectSel highlights the cursor row in the aspect-ratio dialog.
var styleAspectSel = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#1b1726")).
	Background(colorGen).
	Bold(true)

// aspectView renders the modal aspect-ratio dialog: every offered ratio on its
// own row, the highlighted row marked, inside a bordered box.
func (m Model) aspectView() string {
	ratios := m.currentCaps().AspectRatios
	rows := make([]string, len(ratios))
	for i, r := range ratios {
		if i == m.aspectCursor {
			rows[i] = styleAspectSel.Render("› " + r)
		} else {
			rows[i] = "  " + r
		}
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6d5bd0")).
		Padding(0, 1).
		Render(strings.Join(rows, "\n"))
	return "Output aspect ratio:\n\n" + box
}

// stylePill renders an attached-reference-image chip.
var stylePill = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#1b1726")).
	Background(colorGen).
	Padding(0, 1).
	Bold(true)

// refImagePills renders one chip per attached reference image, with a count.
func refImagePills(n int) string {
	chips := make([]string, n)
	for i := 0; i < n; i++ {
		chips[i] = stylePill.Render(fmt.Sprintf("Image %d", i+1))
	}
	count := styleFaint.Render(fmt.Sprintf(" %d attached", n))
	return strings.Join(chips, " ") + count
}

func (m Model) reviewView() string {
	// Bound the box to the terminal width so a long prompt wraps inside the
	// border instead of overflowing it. lipgloss treats Width as the content
	// box and draws the border + padding (2 cols each) outside it, so subtract
	// those from the available width. Fall back to a minimum before the first
	// WindowSizeMsg (m.width == 0).
	outer := m.contentWidth()
	inner := max(1, outer-4)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6d5bd0")).
		Padding(0, 1).
		Width(inner).
		Render(gen.Draft{Prompt: m.prompt.Value()}.Review())
	out := "Review:\n\n" + box
	if n := len(m.refImages); n > 0 {
		out += "\n" + styleFaint.Render(fmt.Sprintf("%d reference image(s) attached", n))
	}
	if guidance := m.referenceGuidance(); guidance != "" {
		out += "\n" + styleFaint.Render(guidance)
	}
	if formats := m.currentCaps().OutputFormats; len(formats) == 1 && formats[0] == "svg" {
		out += "\n" + styleFaint.Render("Output: SVG")
	}
	return ansi.Wrap(out, m.contentWidth(), "")
}

// referenceGuidance describes known requirements without inferring limits.
func (m Model) referenceGuidance() string {
	c := m.currentCaps()
	if !c.AcceptsReferenceImages {
		return ""
	}
	rule := "References optional"
	if c.ReferenceMin != nil && *c.ReferenceMin > 0 {
		rule = fmt.Sprintf("References required: at least %d", *c.ReferenceMin)
	}
	if c.ReferenceMax != nil {
		rule += fmt.Sprintf(" · maximum %d", *c.ReferenceMax)
	}
	rule += fmt.Sprintf(" · %d attached", len(m.refImages))
	if len(c.ReferenceMediaTypes) > 0 {
		var formats []string
		for _, mt := range c.ReferenceMediaTypes {
			formats = append(formats, strings.TrimPrefix(mt, "image/"))
		}
		rule += "\nReference formats: " + strings.Join(formats, ", ")
	}
	if c.ReferenceMinWidth > 0 || c.ReferenceMinHeight > 0 {
		rule += fmt.Sprintf(" · minimum %d x %d pixels", c.ReferenceMinWidth, c.ReferenceMinHeight)
	}
	return rule
}

func (m Model) resultTextView() string {
	if m.outcome.Failure == gen.FailureNone {
		lines := styleOK.Render("✓ " + m.outcome.Message())
		if m.outcome.CostUSD > 0 {
			lines += "\n" + styleFaint.Render(fmt.Sprintf("cost: $%.4f", m.outcome.CostUSD))
		}
		if m.outcome.Seed != nil {
			lines += "\n" + styleFaint.Render(fmt.Sprintf("seed: %d", *m.outcome.Seed))
		}
		return lines + "\n"
	}
	return styleErr.Render("✗ " + m.outcome.Message())
}

func (m Model) currentCaps() provider.Capabilities {
	id := m.currentModelID()
	for _, mdl := range m.models {
		if mdl.ID == id {
			return mdl.Capabilities
		}
	}
	return provider.Capabilities{}
}

func (m Model) footerView() string {
	var keys []string
	switch m.phase {
	case phaseOnboarding:
		switch m.onboard.step {
		case stepChoose:
			keys = []string{k("↑/↓", "move"), k("enter", "select"), k("q", "quit")}
		case stepInput:
			keys = []string{k("enter", "confirm"), k("esc", "back")}
		case stepConfirm:
			keys = []string{k("y", "save"), k("n", "back")}
		}
	case phasePicker:
		keys = []string{k("↑/↓", "move"), k("/", "filter"), k("enter", "select"), k("f", "favorite"), k("h", "history"), k("c", "settings"), k("q", "quit")}
	case phaseCompose:
		// Prompt-shaping commands lead on their own row so they are not lost
		// among the navigation keys that follow.
		caps := m.currentCaps()
		var edit []string
		if len(caps.AspectRatios) > 0 {
			edit = append(edit, k("ctrl+a", "aspect ratio"))
		}
		if caps.SupportsTransparentBackground {
			edit = append(edit, k("ctrl+t", "transparent bg"))
		}
		if caps.AcceptsReferenceImages {
			edit = append(edit, k("ctrl+v", "paste image"))
			if len(m.refImages) > 0 {
				edit = append(edit, k("ctrl+r", "remove image"))
			}
		}
		nav := []string{k("ctrl+s", "review"), k("esc", "back"), k("↑/↓", "history"), k("ctrl+p", "history list")}
		return m.footerRows(edit, nav)
	case phaseAspect:
		keys = []string{k("↑/↓", "move"), k("enter", "select"), k("esc", "cancel")}
	case phaseReview:
		keys = []string{k("enter", "generate"), k("e", "edit")}
	case phaseGenerating:
		keys = []string{k("esc", "cancel")}
	case phaseHistory:
		keys = []string{k("↑/↓", "move"), k("/", "filter"), k("enter", "reuse"), k("esc", "back")}
	case phaseConfig:
		keys = []string{k("↑/↓", "move"), k("space", "toggle"), k("enter", "save"), k("esc", "cancel")}
	case phaseResult:
		if m.outcome.Failure == gen.FailureNone {
			keys = []string{k("o", "open"), k("f", "open folder"), k("p", "edit prompt"), k("m", "try new model"), k("n", "new image")}
			if m.outcome.Seed != nil {
				keys = append(keys, k("g", "re-gen (same seed)"), k("v", "re-gen (vary seed)"))
			} else {
				keys = append(keys, k("g", "re-generate"))
			}
			keys = append(keys, k("q", "quit"))
		} else if m.outcome.Retryable() {
			keys = []string{k("r", "retry"), k("enter", "new"), k("q", "quit")}
		} else {
			keys = []string{k("enter", "new"), k("q", "quit")}
		}
	}
	return m.footerRows(keys)
}

// footerRows renders the status line above one or more key-hint groups, each on
// its own row (and each still width-wrapped). Empty groups are skipped.
func (m Model) footerRows(groups ...[]string) string {
	status := ansi.Wrap(styleFaint.Render(m.status), m.contentWidth(), "")
	sep := styleFaint.Render("  ·  ")
	out := status
	rendered := 0
	for _, g := range groups {
		if len(g) == 0 {
			continue
		}
		if rendered > 0 {
			// Blank line between groups to visually separate them.
			out += "\n"
		}
		out += "\n" + wrapItems(g, sep, m.contentWidth())
		rendered++
	}
	return out
}

func k(key, desc string) string {
	return styleKey.Render(key) + " " + styleFaint.Render(desc)
}

// contentWidth is shared by widgets and their surrounding layout.
func (m Model) contentWidth() int {
	if m.width <= 0 {
		return 76
	}
	return max(1, m.width-2*horizontalPadding(m.width))
}

func horizontalPadding(width int) int {
	if width > 0 && width < 24 {
		return 0
	}
	return 2
}

// wrapItems keeps each shortcut together whenever it fits on a line.
func wrapItems(items []string, separator string, width int) string {
	var rows []string
	row := ""
	for _, item := range items {
		if row != "" && ansi.StringWidth(row+separator+item) > width {
			rows = append(rows, row)
			row = ""
		}
		if row != "" {
			row += separator
		}
		row += ansi.Wrap(item, width, "")
	}
	if row != "" {
		rows = append(rows, row)
	}
	return strings.Join(rows, "\n")
}
