package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/provider"
)

// maxPickerRows caps how many model rows are visible at once; the list scrolls
// beyond this so the header is never pushed off-screen. With single-line items
// this is a generous cap for tall terminals.
const maxPickerRows = 20

// taggedModel pairs a provider.Model with the key of the provider instance that
// offers it, so a model is identified by (ProviderKey, Model.ID) across the
// picker, pricing, state, and generation routing.
type taggedModel struct {
	ProviderKey string
	Model       provider.Model
}

// modelItem adapts a taggedModel to a list.Item (and DefaultItem). It is
// filtered/searched on the model name and id, and shows a classified price in
// its description once pricing has loaded.
type modelItem struct {
	t       taggedModel
	price   provider.Price
	priced  bool // pricing resolved (from cache or fetch)
	loading bool // a pricing fetch is in flight
}

// Title returns the model name. (The custom modelDelegate composes the full
// rendered line — name plus provider tag and muted price/caps — so this is just
// the name.)
func (i modelItem) Title() string { return i.t.Model.Name }

func (i modelItem) FilterValue() string {
	return i.t.Model.Name + " " + i.t.Model.ID + " " + i.t.ProviderKey
}

// Description is unused: the single-line modelDelegate renders everything.
func (i modelItem) Description() string { return "" }

// phase is the current screen of the app.
type phase int

const (
	phaseOnboarding phase = iota // first-run: capture API-key provider choice
	phasePicker                  // choosing a model
	phaseCompose                 // entering a prompt
	phaseAspect                  // choosing an output aspect ratio (modal dialog)
	phaseReview                  // confirming the prompt
	phaseGenerating              // request in flight
	phaseResult                  // showing outcome, offering open
)

// Deps are the collaborators the TUI needs, injected so it stays testable.
type Deps struct {
	// Providers is the registry of configured provider instances keyed by their
	// configured key. ProviderKeys is the same set in configured order, used for
	// stable picker grouping and deterministic startup fan-out.
	Providers    map[string]provider.Provider
	ProviderKeys []string

	Config      config.Config
	StatePath   string
	OutputDir   string
	OpenCommand string
	Version     string

	// FirstRun is true when no usable config.yaml exists yet (absent or a
	// legacy file that was backed up); it triggers the onboarding flow before
	// the picker. ConfigPath is where onboarding writes the newly created
	// config.
	FirstRun   bool
	ConfigPath string

	// Paster reads images off the OS clipboard for reference-image paste. When
	// nil, New installs the default clipboard-backed paster; tests inject a
	// fake. It is read lazily (never at startup) so a missing clipboard never
	// blocks launch.
	Paster imagePaster
}

// Model is the root Bubble Tea model.
type Model struct {
	deps Deps

	width, height int
	phase         phase

	// model picker
	models      []taggedModel
	picker      list.Model
	selected    int
	modelsErr   error
	modelsReady bool
	// offline is the set of provider keys that could not be reached while
	// populating the picker; shown as offline, their models omitted.
	offline map[string]bool

	// pricing
	prices       map[string]provider.Price // "providerKey/modelID" -> resolved price
	pricingPend  int                       // outstanding pricing fetches
	pricingDirty bool                      // fetched this session; needs cache write

	// compose
	prompt textarea.Model

	// aspectRatio is the currently selected output aspect ratio for the active
	// model. Empty when the model offers no aspect ratios. Recomputed whenever
	// the active model changes.
	aspectRatio string

	// aspectCursor is the highlighted row within the aspect-ratio dialog while
	// it is open (phaseAspect). It indexes the active model's offered ratios.
	aspectCursor int

	// reference images attached for the next generation (paste or file path).
	refImages []provider.ReferenceImage

	// in-flight
	spinner spinner.Model
	cancel  context.CancelFunc

	// result
	outcome gen.Outcome

	// onboarding (first run only)
	onboard onboardState

	status string
	quit   bool
}

// New builds the root model.
func New(d Deps) Model {
	ta := textarea.New()
	ta.Placeholder = "Describe the image you want..."
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.FocusedStyle, ta.BlurredStyle = promptTextareaStyles()

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	picker := list.New(nil, modelDelegate{}, 40, maxPickerRows)
	picker.Styles = pickerStyles()
	picker.FilterInput = filterInputStyles(picker.FilterInput)
	picker.SetShowTitle(false)
	picker.SetShowStatusBar(false)
	picker.SetShowHelp(false)
	picker.SetShowPagination(true)
	picker.DisableQuitKeybindings() // we own quit; the list must not exit the app

	if d.Paster == nil {
		d.Paster = newClipboardPaster()
	}

	start := phasePicker
	status := "Loading models..."
	var onboard onboardState
	if d.FirstRun {
		start = phaseOnboarding
		onboard = newOnboardState()
		status = "Welcome to genifer"
	}

	return Model{
		deps:     d,
		phase:    start,
		prompt:   ta,
		spinner:  sp,
		picker:   picker,
		selected: -1,
		status:   status,
		onboard:  onboard,
	}
}

func (m Model) Init() tea.Cmd {
	// On first run, hold off loading models until onboarding writes the config
	// (the provider needs the chosen API key). Otherwise load immediately.
	if m.phase == phaseOnboarding {
		return m.spinner.Tick
	}
	return tea.Batch(loadModels(m.deps.Providers, m.deps.ProviderKeys), m.spinner.Tick)
}

// --- messages --------------------------------------------------------------

type modelsLoadedMsg struct {
	models  []taggedModel
	offline map[string]bool
	err     error // set only when no provider could be reached at all
}

type generatedMsg struct{ outcome gen.Outcome }

// pastedImageMsg carries the outcome of a clipboard image-paste attempt. On
// success Ref holds the validated image; on any failure Err is set and the UI
// shows a transient, non-fatal status.
type pastedImageMsg struct {
	ref provider.ReferenceImage
	err error
}

// pasteImageCmd reads the clipboard off the UI loop, validates the bytes, and
// returns a pastedImageMsg. Clipboard access and validation are best-effort;
// failures become a non-fatal message, never an app error.
func pasteImageCmd(p imagePaster) tea.Cmd {
	return func() tea.Msg {
		data, err := p.ReadImage(context.Background())
		if err != nil {
			return pastedImageMsg{err: err}
		}
		ref, err := validateRefImage(data)
		if err != nil {
			return pastedImageMsg{err: err}
		}
		return pastedImageMsg{ref: ref}
	}
}

// loadModels fans out Models across every configured provider instance
// concurrently, tags each returned model with its provider key, merges them in
// configured key order, and records a per-instance offline marker for any
// instance that fails. It returns err only when the registry is empty or every
// instance failed, so one unreachable provider never blanks the picker.
func loadModels(registry map[string]provider.Provider, keys []string) tea.Cmd {
	return func() tea.Msg {
		type result struct {
			models []provider.Model
			err    error
		}
		results := make([]result, len(keys))
		var wg sync.WaitGroup
		for i, key := range keys {
			wg.Add(1)
			go func(i int, p provider.Provider) {
				defer wg.Done()
				// Bounded timeout so an unreachable instance fails fast and
				// never stalls the picker behind a slow dial.
				ctx, cancel := context.WithTimeout(context.Background(), modelLoadTimeout)
				defer cancel()
				models, err := p.Models(ctx)
				results[i] = result{models: models, err: err}
			}(i, registry[key])
		}
		wg.Wait()

		var merged []taggedModel
		offline := map[string]bool{}
		var reachable int
		var lastErr error
		for i, key := range keys {
			r := results[i]
			if r.err != nil {
				offline[key] = true
				lastErr = r.err
				continue
			}
			reachable++
			for _, mdl := range r.models {
				merged = append(merged, taggedModel{ProviderKey: key, Model: mdl})
			}
		}

		msg := modelsLoadedMsg{models: merged, offline: offline}
		// Only a true all-failed outcome is an app-level error; otherwise the
		// reachable providers' models stand and failures are isolated as offline.
		if reachable == 0 {
			if lastErr != nil {
				msg.err = lastErr
			} else {
				msg.err = errNoProviders
			}
		}
		return msg
	}
}

// modelLoadTimeout bounds each provider's startup Models call so an unreachable
// instance (e.g. a local WebUI that is not running) fails fast.
const modelLoadTimeout = 10 * time.Second

// errNoProviders is the model-list error when the registry is empty.
var errNoProviders = errors.New("no providers configured")

// priceLoadedMsg carries one model's resolved price (or a fetch error), keyed
// by the composite "providerKey/modelID".
type priceLoadedMsg struct {
	key   string // config.PricingCacheKey(providerKey, modelID)
	price provider.Price
	err   error
}

// initPricing seeds m.prices from the on-disk cache when it is fresh. When the
// cache is stale/missing, prices stay empty and pricingCmds will fetch them.
func (m Model) initPricing() Model {
	m.prices = map[string]provider.Price{}
	path, err := config.PricingCachePath()
	if err != nil {
		return m
	}
	cache := config.LoadPricingCache(path)
	if cache.Fresh(timeNow()) {
		for key, p := range cache.Prices {
			m.prices[key] = p
		}
	}
	return m
}

// pricingCmds returns background fetches for every model missing a cached
// price, each resolved against its owning provider instance. Each returns a
// priceLoadedMsg; the picker fills in live.
func (m *Model) pricingCmds() tea.Cmd {
	var cmds []tea.Cmd
	for _, tm := range m.models {
		key := config.PricingCacheKey(tm.ProviderKey, tm.Model.ID)
		if _, ok := m.prices[key]; ok {
			continue // already have a fresh cached price
		}
		p := m.deps.Providers[tm.ProviderKey]
		if p == nil {
			continue
		}
		modelID := tm.Model.ID
		m.pricingPend++
		cmds = append(cmds, func() tea.Msg {
			price, err := p.Pricing(context.Background(), modelID)
			return priceLoadedMsg{key: key, price: price, err: err}
		})
	}
	if len(cmds) == 0 {
		return nil
	}
	// Keep the spinner animating while prices stream in.
	cmds = append(cmds, m.spinner.Tick)
	return tea.Batch(cmds...)
}

// savePricingCmd persists the current prices to the cache with a fresh
// timestamp. Runs off the UI loop; failures are ignored (cache is best-effort).
func (m Model) savePricingCmd() tea.Cmd {
	snapshot := make(map[string]provider.Price, len(m.prices))
	for k, v := range m.prices {
		snapshot[k] = v
	}
	return func() tea.Msg {
		if path, err := config.PricingCachePath(); err == nil {
			_ = config.SavePricingCache(path, config.PricingCache{
				FetchedAt: timeNow(),
				Prices:    snapshot,
			})
		}
		return nil
	}
}

// rebuildItems refreshes the picker items from models + current prices,
// preserving the selected index.
func (m *Model) rebuildItems() {
	sel := m.picker.Index()
	items := make([]list.Item, len(m.models))
	for i, tm := range m.models {
		it := modelItem{t: tm}
		key := config.PricingCacheKey(tm.ProviderKey, tm.Model.ID)
		if p, ok := m.prices[key]; ok {
			it.price, it.priced = p, true
		} else {
			it.loading = m.pricingPend > 0
		}
		items[i] = it
	}
	m.picker.SetItems(items)
	if sel >= 0 && sel < len(items) {
		m.picker.Select(sel)
	}
}

// addRefImage appends a validated reference image to the attachment set.
func (m *Model) addRefImage(ref provider.ReferenceImage) {
	m.refImages = append(m.refImages, ref)
}

// removeLastRefImage drops the most-recently-added reference image, if any. It
// returns true when an image was removed.
func (m *Model) removeLastRefImage() bool {
	if len(m.refImages) == 0 {
		return false
	}
	m.refImages = m.refImages[:len(m.refImages)-1]
	return true
}

// composeDraft builds the generation draft from current model state. It is
// separated from generate so the draft contents (prompt, model, attached
// reference images) are directly testable.
func (m Model) composeDraft() gen.Draft {
	_, modelID := m.currentSelection()
	return gen.Draft{Prompt: m.prompt.Value(), Model: modelID, AspectRatio: m.aspectRatio, ReferenceImages: m.refImages}
}

// persistAspectRatio records the current aspect-ratio selection as the
// last-used value, preserving any other remembered selection (read-modify-write
// so the remembered model is not clobbered). A no-op when nothing is selected.
func (m Model) persistAspectRatio() {
	if m.aspectRatio == "" {
		return
	}
	s := config.LoadState(m.deps.StatePath)
	s.LastAspectRatio = m.aspectRatio
	_ = config.SaveState(m.deps.StatePath, s)
}

func (m Model) generate() (Model, tea.Cmd) {
	m.persistAspectRatio()
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	d := m.composeDraft()
	// Route to the provider instance that owns the selected model.
	providerKey, _ := m.currentSelection()
	p := m.deps.Providers[providerKey]
	dir := m.deps.OutputDir
	cmd := func() tea.Msg {
		if p == nil {
			return generatedMsg{outcome: gen.Outcome{Failure: gen.FailureOther, Err: errNoProviderForModel}}
		}
		return generatedMsg{outcome: gen.Run(ctx, p, d, dir)}
	}
	return m, tea.Batch(cmd, m.spinner.Tick)
}

// errNoProviderForModel is used when the selected model's provider key does not
// resolve in the registry (should not happen in normal operation).
var errNoProviderForModel = errors.New("no provider for selected model")

// --- update ----------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.prompt.SetWidth(max(20, msg.Width-4))
		m.picker.SetSize(max(20, msg.Width-4), pickerHeight(msg.Height))
		return m, nil

	case modelsLoadedMsg:
		m.modelsReady = true
		m.models, m.modelsErr, m.offline = msg.models, msg.err, msg.offline
		if msg.err != nil {
			m.status = modelErrorText(msg.err)
			return m, nil
		}
		m = m.initPricing()
		m.rebuildItems()
		s := config.LoadState(m.deps.StatePath)
		if idx := preselectModel(m.models, s.LastProviderKey, s.LastModel); idx >= 0 {
			m.picker.Select(idx)
		}
		m.status = modelsStatus(len(m.models), m.offline)
		return m, m.pricingCmds()

	case priceLoadedMsg:
		if m.prices == nil {
			m.prices = map[string]provider.Price{}
		}
		if msg.err == nil {
			m.prices[msg.key] = msg.price
			m.pricingDirty = true
		}
		if m.pricingPend > 0 {
			m.pricingPend--
		}
		m.rebuildItems()
		var cmd tea.Cmd
		if m.pricingPend == 0 && m.pricingDirty {
			cmd = m.savePricingCmd()
		}
		return m, cmd

	case generatedMsg:
		m.cancel = nil
		m.outcome = msg.outcome
		m.phase = phaseResult
		m.status = msg.outcome.Message()
		if m.outcome.Failure == gen.FailureNone && m.deps.Config.AutoOpen {
			return m, m.openCmd()
		}
		return m, nil

	case pastedImageMsg:
		if msg.err != nil {
			m.status = pasteErrorText(msg.err)
			return m, nil
		}
		m.addRefImage(msg.ref)
		m.status = fmt.Sprintf("Attached image — %d reference image(s)", len(m.refImages))
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case onboardWrittenMsg:
		// Onboarding finished writing config; enter the picker and load models.
		m.phase = phasePicker
		if msg.err != nil {
			m.onboard.writeErr = msg.err
		}
		m.status = "Loading models..."
		return m, tea.Batch(loadModels(m.deps.Providers, m.deps.ProviderKeys), m.spinner.Tick)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Route to the active input widget.
	if m.phase == phaseOnboarding {
		return m.updateOnboarding(msg)
	}
	if m.phase == phaseCompose {
		var cmd tea.Cmd
		m.prompt, cmd = m.prompt.Update(msg)
		return m, cmd
	}
	if m.phase == phasePicker {
		// Forward filter-result and other list messages to the picker.
		var cmd tea.Cmd
		m.picker, cmd = m.picker.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit (not while typing a prompt, where ctrl+c still quits).
	switch msg.String() {
	case "ctrl+c":
		if m.cancel != nil {
			m.cancel()
		}
		m.quit = true
		return m, tea.Quit
	}

	switch m.phase {
	case phaseOnboarding:
		return m.handleOnboardingKey(msg)
	case phasePicker:
		return m.handlePickerKey(msg)
	case phaseCompose:
		return m.handleComposeKey(msg)
	case phaseAspect:
		return m.handleAspectKey(msg)
	case phaseReview:
		return m.handleReviewKey(msg)
	case phaseGenerating:
		if msg.String() == "esc" && m.cancel != nil {
			m.cancel()
		}
		return m, nil
	case phaseResult:
		return m.handleResultKey(msg)
	}
	return m, nil
}

func (m Model) handlePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While the filter input is active, let the list handle everything
	// (typing, esc-to-clear, enter-to-accept) except our global ctrl+c.
	if m.picker.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.picker, cmd = m.picker.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "q":
		m.quit = true
		return m, tea.Quit
	case "enter":
		if it, ok := m.picker.SelectedItem().(modelItem); ok {
			m.selected = indexOfModel(m.models, it.t.ProviderKey, it.t.Model.ID)
			s := config.LoadState(m.deps.StatePath)
			m.aspectRatio = pickAspectRatio(it.t.Model.Capabilities.AspectRatios, s.LastAspectRatio)
			s.LastModel = it.t.Model.ID
			s.LastProviderKey = it.t.ProviderKey
			_ = config.SaveState(m.deps.StatePath, s)
			m.phase = phaseCompose
			m.prompt.Focus()
			m.status = "Compose your prompt"
		}
		return m, nil
	}

	// Everything else (up/down, /, paging) goes to the list.
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

// indexOfModel returns the index of the model identified by (providerKey, id),
// or -1.
func indexOfModel(models []taggedModel, providerKey, id string) int {
	for i, tm := range models {
		if tm.ProviderKey == providerKey && tm.Model.ID == id {
			return i
		}
	}
	return -1
}

// pickerHeight returns the list height: capped at maxPickerRows but shrinking
// on small terminals so it never overflows the viewport.
func pickerHeight(termHeight int) int {
	// Reserve rows for header (~9), spacing, status/footer (~4).
	avail := termHeight - 14
	if avail < 3 {
		avail = 3
	}
	if avail > maxPickerRows {
		return maxPickerRows
	}
	return avail
}

func (m Model) handleComposeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.phase = phasePicker
		m.prompt.Blur()
		return m, nil
	case "ctrl+v", "alt+v":
		// Paste a reference image from the clipboard, but only for a
		// reference-capable model. Otherwise fall through so the keystroke
		// reaches the textarea unchanged.
		if m.currentCaps().AcceptsReferenceImages {
			m.status = "Reading clipboard…"
			return m, pasteImageCmd(m.deps.Paster)
		}
	case "ctrl+r":
		// Remove the most-recently attached reference image.
		if m.currentCaps().AcceptsReferenceImages && m.removeLastRefImage() {
			m.status = fmt.Sprintf("Removed image — %d reference image(s)", len(m.refImages))
			return m, nil
		}
	case "ctrl+a":
		// Open the aspect-ratio dialog, but only for a model that offers a
		// choice. Otherwise fall through so the keystroke reaches the textarea.
		if ratios := m.currentCaps().AspectRatios; len(ratios) > 0 {
			m.phase = phaseAspect
			m.aspectCursor = indexOfString(ratios, m.aspectRatio)
			if m.aspectCursor < 0 {
				m.aspectCursor = 0
			}
			m.status = "Choose an output aspect ratio"
			return m, nil
		}
	case "ctrl+s", "alt+enter":
		return m.submitCompose()
	}
	var cmd tea.Cmd
	m.prompt, cmd = m.prompt.Update(msg)
	return m, cmd
}

// handleAspectKey drives the modal aspect-ratio dialog: up/down move the
// highlighted row (clamped), enter confirms the highlighted ratio as the new
// selection, and esc leaves the selection unchanged. All paths return to
// compose except movement.
func (m Model) handleAspectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	ratios := m.currentCaps().AspectRatios
	switch msg.String() {
	case "up", "k":
		if m.aspectCursor > 0 {
			m.aspectCursor--
		}
	case "down", "j":
		if m.aspectCursor < len(ratios)-1 {
			m.aspectCursor++
		}
	case "enter":
		if m.aspectCursor >= 0 && m.aspectCursor < len(ratios) {
			m.aspectRatio = ratios[m.aspectCursor]
		}
		m.phase = phaseCompose
		m.status = "Aspect ratio: " + m.aspectRatio
	case "esc":
		m.phase = phaseCompose
		m.status = "Compose your prompt"
	}
	return m, nil
}

// submitCompose handles a compose submit: it extracts sigil-prefixed reference
// paths from the prompt, attaches each (keeping the user in compose on any
// path failure), strips those lines from the prompt, and advances to review
// only when the remaining prompt is non-empty and all paths loaded.
func (m Model) submitCompose() (tea.Model, tea.Cmd) {
	if m.currentCaps().AcceptsReferenceImages {
		paths, cleaned := extractRefPaths(m.prompt.Value())
		if len(paths) > 0 {
			var loaded []provider.ReferenceImage
			for _, p := range paths {
				ref, err := loadRefImageFile(p)
				if err != nil {
					// Block the submit; keep the sigil lines so the user can fix
					// the path rather than silently sending an unresolved prompt.
					m.status = pasteErrorText(err)
					return m, nil
				}
				loaded = append(loaded, ref)
			}
			for _, ref := range loaded {
				m.addRefImage(ref)
			}
			m.prompt.SetValue(cleaned)
			m.status = fmt.Sprintf("Attached %d image(s) from path — %d reference image(s)", len(loaded), len(m.refImages))
		}
	}
	if strings.TrimSpace(m.prompt.Value()) == "" {
		m.status = "Prompt is empty — enter some text"
		return m, nil
	}
	m.phase = phaseReview
	m.prompt.Blur()
	m.status = "Review your prompt"
	return m, nil
}

func (m Model) handleReviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.phase = phaseGenerating
		m.status = "Generating..."
		return m.generate()
	case "e", "esc":
		m.phase = phaseCompose
		m.prompt.Focus()
		m.status = "Editing prompt"
	}
	return m, nil
}

func (m Model) handleResultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "o":
		if m.outcome.Failure == gen.FailureNone {
			return m, m.openCmd()
		}
	case "r":
		if m.outcome.Retryable() {
			m.phase = phaseGenerating
			m.status = "Generating..."
			return m.generate()
		}
	case "enter", "n":
		m.phase = phaseCompose
		m.prompt.Focus()
		m.status = "Compose your prompt"
	case "q":
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) openCmd() tea.Cmd {
	path := m.outcome.Path
	override := m.deps.OpenCommand
	return func() tea.Msg {
		_ = gen.Open(context.Background(), override, path)
		return nil
	}
}

// currentSelection returns the (providerKey, modelID) of the active model: the
// committed selection when one exists, otherwise the row currently highlighted
// in the picker. Both are empty when nothing is selected.
func (m Model) currentSelection() (providerKey, modelID string) {
	if m.selected >= 0 && m.selected < len(m.models) {
		tm := m.models[m.selected]
		return tm.ProviderKey, tm.Model.ID
	}
	if it, ok := m.picker.SelectedItem().(modelItem); ok {
		return it.t.ProviderKey, it.t.Model.ID
	}
	return "", ""
}

// currentModelID returns just the model id of the active model, for display.
func (m Model) currentModelID() string {
	_, id := m.currentSelection()
	return id
}
