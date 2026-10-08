package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/provider"
	"github.com/charmbracelet/lipgloss"
)

var (
	styleFaint = lipgloss.NewStyle().Foreground(lipgloss.Color("#8b85a0"))
	styleKey   = lipgloss.NewStyle().Foreground(lipgloss.Color("#b9a6ff")).Bold(true)
	styleErr   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b6b"))
	styleOK    = lipgloss.NewStyle().Foreground(lipgloss.Color("#7ee787"))
)

// modelErrorText produces an actionable message for a model-list failure.
func modelErrorText(err error) string {
	if errors.Is(err, provider.ErrAuth) {
		return "Could not load models: authentication failed — check your API key"
	}
	return "Could not load models: " + err.Error()
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
	body := m.bodyView()
	footer := m.footerView()
	content := lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", footer)

	// Fill the whole terminal with the app background so the user's terminal
	// background never shows through / clashes with the TUI.
	canvas := lipgloss.NewStyle().Background(headerBG).Foreground(lipgloss.Color("#e6e1f2"))
	if m.width > 0 {
		canvas = canvas.Width(m.width)
	}
	if m.height > 0 {
		canvas = canvas.Height(m.height)
	}
	return canvas.Render(content)
}

func (m Model) bodyView() string {
	switch m.phase {
	case phaseOnboarding:
		return m.onboardingView()
	case phasePicker:
		return m.pickerView()
	case phaseCompose:
		return m.composeView()
	case phaseReview:
		return m.reviewView()
	case phaseGenerating:
		return m.spinner.View() + " Generating... (esc to cancel)"
	case phaseResult:
		return m.resultView()
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
		parts = append(parts, "img2img")
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
	ctrls := controlsFor(m.currentCaps())
	var extras []string
	if ctrls.ShowReference {
		extras = append(extras, "paste image: ctrl+v")
	}
	if ctrls.ShowAspectRatio {
		extras = append(extras, "aspect ratios: "+strings.Join(ctrls.AspectRatios, " "))
	}
	if ctrls.ShowSeed {
		extras = append(extras, "seed supported")
	}
	hint := ""
	if len(extras) > 0 {
		hint = "\n" + styleFaint.Render(strings.Join(extras, " · "))
	}
	pills := ""
	if ctrls.ShowReference && len(m.refImages) > 0 {
		pills = "\n" + refImagePills(len(m.refImages))
	}
	return fmt.Sprintf("Model: %s%s%s\n\n%s", m.currentModelID(), hint, pills, m.prompt.View())
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
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6d5bd0")).
		Padding(0, 1).
		Render(gen.Draft{Prompt: m.prompt.Value()}.Review())
	out := "Review:\n\n" + box
	if n := len(m.refImages); n > 0 {
		out += "\n" + styleFaint.Render(fmt.Sprintf("%d reference image(s) attached", n))
	}
	return out
}

func (m Model) resultView() string {
	if m.outcome.Failure == gen.FailureNone {
		lines := styleOK.Render("✓ " + m.outcome.Message())
		if m.outcome.CostUSD > 0 {
			lines += "\n" + styleFaint.Render(fmt.Sprintf("cost: $%.4f", m.outcome.CostUSD))
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
		keys = []string{k("↑/↓", "move"), k("/", "filter"), k("enter", "select"), k("q", "quit")}
	case phaseCompose:
		keys = []string{k("ctrl+s", "review"), k("esc", "back")}
		if m.currentCaps().AcceptsReferenceImages {
			keys = append(keys, k("ctrl+v", "paste image"))
			if len(m.refImages) > 0 {
				keys = append(keys, k("ctrl+r", "remove image"))
			}
		}
	case phaseReview:
		keys = []string{k("enter", "generate"), k("e", "edit")}
	case phaseGenerating:
		keys = []string{k("esc", "cancel")}
	case phaseResult:
		if m.outcome.Failure == gen.FailureNone {
			keys = []string{k("o", "open"), k("enter", "new"), k("q", "quit")}
		} else if m.outcome.Retryable() {
			keys = []string{k("r", "retry"), k("enter", "new"), k("q", "quit")}
		} else {
			keys = []string{k("enter", "new"), k("q", "quit")}
		}
	}
	line := strings.Join(keys, styleFaint.Render("  ·  "))
	status := styleFaint.Render(m.status)
	return status + "\n" + line
}

func k(key, desc string) string {
	return styleKey.Render(key) + " " + styleFaint.Render(desc)
}
