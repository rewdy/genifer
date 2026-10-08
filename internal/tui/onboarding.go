package tui

import (
	"fmt"

	"github.com/rewdy/genifer/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// keyMethod is how the user chooses to provide the OpenRouter API key during
// first-run onboarding. Each maps to a config.Value directive form.
type keyMethod int

const (
	methodEnv   keyMethod = iota // {env:NAME}
	methodCmd                    // {cmd:...}
	methodPaste                  // literal (stored plaintext)
)

// onboardStep is the sub-state within the onboarding phase.
type onboardStep int

const (
	stepChoose  onboardStep = iota // selecting a key method
	stepInput                      // typing the detail (env name / command / key)
	stepConfirm                    // paste only: confirm plaintext storage
)

// methodChoice describes one selectable key-provider option.
type methodChoice struct {
	method keyMethod
	label  string
	hint   string
}

var methodChoices = []methodChoice{
	{methodEnv, "Environment variable", "read the key from an env var, e.g. OPENROUTER_API_KEY"},
	{methodCmd, "Command", "run a command to fetch the key, e.g. a secrets manager"},
	{methodPaste, "Paste a key", "store the key directly (saved as plain text)"},
}

// onboardState holds first-run onboarding UI state.
type onboardState struct {
	step     onboardStep
	cursor   int // selected method in stepChoose
	input    textinput.Model
	writeErr error // surfaced if the config write failed
}

func newOnboardState() onboardState {
	ti := textinput.New()
	ti.Prompt = "› "
	return onboardState{step: stepChoose, input: ti}
}

// selectedMethod returns the method currently under the cursor.
func (o onboardState) selectedMethod() keyMethod {
	return methodChoices[o.cursor].method
}

// onboardWrittenMsg reports the result of writing the onboarding config.
type onboardWrittenMsg struct{ err error }

// handleOnboardingKey processes key presses during the onboarding phase.
func (m Model) handleOnboardingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.onboard.step {
	case stepChoose:
		return m.handleOnboardChoose(msg)
	case stepInput:
		return m.handleOnboardInput(msg)
	case stepConfirm:
		return m.handleOnboardConfirm(msg)
	}
	return m, nil
}

func (m Model) handleOnboardChoose(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.quit = true
		return m, tea.Quit
	case "up", "k":
		if m.onboard.cursor > 0 {
			m.onboard.cursor--
		}
	case "down", "j":
		if m.onboard.cursor < len(methodChoices)-1 {
			m.onboard.cursor++
		}
	case "enter":
		m.onboard.step = stepInput
		m.onboard.input.SetValue("")
		m.onboard.input.Placeholder = placeholderFor(m.onboard.selectedMethod())
		m.onboard.input.Focus()
		m.status = promptFor(m.onboard.selectedMethod())
	}
	return m, nil
}

func (m Model) handleOnboardInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.onboard.step = stepChoose
		m.onboard.input.Blur()
		m.status = "Welcome to genifer"
		return m, nil
	case "enter":
		if m.onboard.input.Value() == "" {
			m.status = "Enter a value, or press esc to go back"
			return m, nil
		}
		// Paste requires an explicit plaintext-storage confirmation.
		if m.onboard.selectedMethod() == methodPaste {
			m.onboard.step = stepConfirm
			m.onboard.input.Blur()
			m.status = "This key will be stored as plain text. Save it? (y/n)"
			return m, nil
		}
		return m, m.writeOnboardConfig()
	}
	var cmd tea.Cmd
	m.onboard.input, cmd = m.onboard.input.Update(msg)
	return m, cmd
}

func (m Model) handleOnboardConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m, m.writeOnboardConfig()
	case "n", "esc":
		m.onboard.step = stepInput
		m.onboard.input.Focus()
		m.status = promptFor(methodPaste)
	}
	return m, nil
}

// updateOnboarding forwards non-key messages (e.g. cursor blink) to the input.
func (m Model) updateOnboarding(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.onboard.step == stepInput {
		var cmd tea.Cmd
		m.onboard.input, cmd = m.onboard.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

// writeOnboardConfig builds the chosen api_key directive and writes a starter
// config as a tea.Cmd (side effects stay out of Update). A write failure is
// reported via onboardWrittenMsg so the session can continue on defaults.
func (m Model) writeOnboardConfig() tea.Cmd {
	apiKey := apiKeyValue(m.onboard.selectedMethod(), m.onboard.input.Value())
	path := m.deps.ConfigPath
	return func() tea.Msg {
		_, err := config.WriteStarter(path, apiKey)
		return onboardWrittenMsg{err: err}
	}
}

// apiKeyValue maps a method + raw input to the config.Value directive written
// into config.yaml.
func apiKeyValue(method keyMethod, raw string) config.Value {
	switch method {
	case methodEnv:
		return config.Value(fmt.Sprintf("{env:%s}", raw))
	case methodCmd:
		return config.Value(fmt.Sprintf("{cmd:%s}", raw))
	default: // methodPaste
		return config.Value(raw)
	}
}

func placeholderFor(method keyMethod) string {
	switch method {
	case methodEnv:
		return "OPENROUTER_API_KEY"
	case methodCmd:
		return "op read op://vault/openrouter/key"
	default:
		return "sk-or-..."
	}
}

func promptFor(method keyMethod) string {
	switch method {
	case methodEnv:
		return "Environment variable name"
	case methodCmd:
		return "Command to run"
	default:
		return "Paste your API key"
	}
}
