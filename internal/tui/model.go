package tui

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/preview"
	"github.com/rewdy/genifer/internal/provider"
)

// maxPickerRows caps how many model rows are visible at once; the list scrolls
// beyond this so the header is never pushed off-screen. With single-line items
// this is a generous cap for tall terminals.
const maxPickerRows = 20

// modelItem adapts a provider.Model to a list.Item (and DefaultItem). It is
// filtered/searched on the model name and id, and shows a classified price in
// its description once pricing has loaded.
type modelItem struct {
	m       provider.Model
	price   provider.Price
	priced  bool // pricing resolved (from cache or fetch)
	loading bool // a pricing fetch is in flight
}

// Title returns the model name. (The custom modelDelegate composes the full
// rendered line — name plus muted price/caps — so this is just the name.)
func (i modelItem) Title() string { return i.m.Name }

func (i modelItem) FilterValue() string { return i.m.Name + " " + i.m.ID }

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
	phaseHistory                 // browsing past prompts
	phaseConfig                  // editing settings (save location, auto-open, history)
)

// historyLoadLimit bounds how many of the most-recent history entries the view
// loads into the list. The store itself retains every entry; the full record
// lives in the on-disk history file.
const historyLoadLimit = 25

// recallInactive is the recallIndex sentinel meaning the prompt holds the
// user's own draft, not a recalled history entry.
const recallInactive = -1

// Deps are the collaborators the TUI needs, injected so it stays testable.
type Deps struct {
	Provider    provider.Provider
	Config      config.Config
	StatePath   string
	HistoryPath string
	OutputDir   string
	OpenCommand string
	Version     string

	// FirstRun is true when no config.yaml exists yet; it triggers the
	// onboarding flow before the picker. ConfigPath is where onboarding writes
	// the newly created config.
	FirstRun   bool
	ConfigPath string

	// Paster reads images off the OS clipboard for reference-image paste. When
	// nil, New installs the default clipboard-backed paster; tests inject a
	// fake. It is read lazily (never at startup) so a missing clipboard never
	// blocks launch.
	Paster imagePaster
	// Open launches a saved file; nil uses the system viewer through gen.Open.
	Open func(context.Context, string, string) error
	// OpenFolder opens the containing directory; nil uses the system file manager.
	OpenFolder func(context.Context, string) error
	// Preview is nil when native terminal image support is unavailable.
	Preview *PreviewDeps
}

// Model is the root Bubble Tea model.
type Model struct {
	deps Deps

	// outputDir is the session's effective image output directory. It starts
	// from Deps.OutputDir and can be changed live via the config view, so a new
	// save location applies to the next generation without a restart.
	outputDir string

	width, height int
	phase         phase

	// model picker
	models      []provider.Model
	picker      list.Model
	selected    int
	modelsErr   error
	modelsReady bool

	// pricing
	prices       map[string]provider.Price // modelID -> resolved price
	pricingPend  int                       // outstanding pricing fetches
	pricingDirty bool                      // fetched this session; needs cache write

	// compose
	prompt textarea.Model

	// aspectRatio is the currently selected output aspect ratio for the active
	// model. Empty when the model offers no aspect ratios. Recomputed whenever
	// the active model changes.
	aspectRatio string
	// transparentBackground is true when the user has enabled transparent
	// background generation.
	transparentBackground bool

	// aspectCursor is the highlighted row within the aspect-ratio dialog while
	// it is open (phaseAspect). It indexes the active model's offered ratios.
	aspectCursor int

	// reference images attached for the next generation (paste or file path).
	refImages     []provider.ReferenceImage
	draftRevision uint64

	// in-flight
	spinner spinner.Model
	cancel  context.CancelFunc

	// result
	outcome                                 gen.Outcome
	previewGeometry                         preview.Geometry
	previewGeometryRevision                 uint64
	previewRevision                         uint64
	previewCancel                           context.CancelFunc
	previewItem                             preview.Item
	previewBudgetWidth, previewBudgetHeight int

	// lastDraft is the exact draft used for the in-flight / just-completed
	// generation (including the minted seed), captured when generation fires so
	// re-generate can repeat it precisely.
	lastDraft gen.Draft

	// history
	history          list.Model            // list of loaded past prompts
	historyEntries   []config.HistoryEntry // the loaded (bounded) entries
	historyMoreExist bool                  // store holds more than were loaded
	historyPrev      phase                 // phase to return to on esc

	// reuseSeed, when set, is a seed carried from a reused history entry; it is
	// applied to the next composed draft for a seed-capable model instead of
	// minting a fresh one. Cleared once consumed.
	reuseSeed *int64

	// recall session: inline Up/Down history recall on the compose prompt.
	// recallIndex is the position in historyEntries, or recallInactive (-1)
	// when the prompt holds the user's own draft rather than a recalled entry.
	// recallDraft snapshots the in-progress prompt when a recall session starts
	// so stepping forward past the newest entry restores it.
	recallIndex int
	recallDraft string

	// onboarding (first run only)
	onboard onboardState

	// config view (phaseConfig)
	cfg        configState
	configPrev phase // phase to return to on esc / confirm

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

	history := list.New(nil, historyDelegate{}, 40, maxPickerRows)
	history.Styles = pickerStyles()
	history.FilterInput = filterInputStyles(history.FilterInput)
	history.SetShowTitle(false)
	history.SetShowStatusBar(false)
	history.SetShowHelp(false)
	history.SetShowPagination(true)
	history.DisableQuitKeybindings()

	if d.Paster == nil {
		d.Paster = newClipboardPaster()
	}
	if d.Open == nil {
		d.Open = gen.Open
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
		deps:        d,
		outputDir:   d.OutputDir,
		phase:       start,
		prompt:      ta,
		spinner:     sp,
		picker:      picker,
		history:     history,
		selected:    -1,
		recallIndex: recallInactive,
		status:      status,
		onboard:     onboard,
	}
}

func (m Model) Init() tea.Cmd {
	// On first run, hold off loading models until onboarding writes the config
	// (the provider needs the chosen API key). Otherwise load immediately.
	if m.phase == phaseOnboarding {
		return m.spinner.Tick
	}
	return tea.Batch(loadModels(m.deps.Provider), m.spinner.Tick)
}

// --- messages --------------------------------------------------------------

type modelsLoadedMsg struct {
	models []provider.Model
	err    error
}

type generatedMsg struct{ outcome gen.Outcome }

// pastedImageMsg carries the outcome of a clipboard image-paste attempt. On
// success Ref holds the validated image; on any failure Err is set and the UI
// shows a transient, non-fatal status.
type pastedImageMsg struct {
	token attachmentToken
	ref   provider.ReferenceImage
	err   error
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

func loadModels(p provider.Provider) tea.Cmd {
	return func() tea.Msg {
		models, err := p.Models(context.Background())
		return modelsLoadedMsg{models: models, err: err}
	}
}

// priceLoadedMsg carries one model's resolved price (or a fetch error).
type priceLoadedMsg struct {
	modelID string
	price   provider.Price
	err     error
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
		for id, p := range cache.Prices {
			m.prices[id] = p
		}
	}
	return m
}

// pricingCmds returns background fetches for every model missing a cached
// price. Each returns a priceLoadedMsg; the picker fills in live.
func (m *Model) pricingCmds() tea.Cmd {
	var cmds []tea.Cmd
	p := m.deps.Provider
	for _, mdl := range m.models {
		if _, ok := m.prices[mdl.ID]; ok {
			continue // already have a fresh cached price
		}
		id := mdl.ID
		m.pricingPend++
		cmds = append(cmds, func() tea.Msg {
			price, err := p.Pricing(context.Background(), id)
			return priceLoadedMsg{modelID: id, price: price, err: err}
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
	for i, mdl := range m.models {
		it := modelItem{m: mdl}
		if p, ok := m.prices[mdl.ID]; ok {
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
// reference images) are directly testable. For a seed-capable model it mints a
// random seed when none is set, so the seed is always known to genifer (shown,
// reused, reproducible); a non-seed model is left with a nil seed.
// The Background field is set if transparentBackground is true and the model
// supports it.
// The OutputFormat is set to "png" if transparentBackground is true since PNG
// supports alpha channels.
func (m Model) composeDraft() gen.Draft {
	d := gen.Draft{Prompt: m.prompt.Value(), Model: m.currentModelID(), AspectRatio: m.aspectRatio, ReferenceImages: m.refImages}

	caps := m.currentCaps()
	if caps.SupportsSeed {
		switch {
		case m.reuseSeed != nil:
			s := *m.reuseSeed
			d.Seed = &s
		case d.Seed == nil:
			s := mintSeed()
			d.Seed = &s
		}
	}
	if m.transparentBackground && caps.SupportsTransparentBackground {
		d.Background = "transparent"
		d.OutputFormat = "png" // PNG supports alpha channel
	}

	return d
}

// mintSeed uses non-negative signed 32-bit values for provider compatibility.
// Keep the int64 request type so existing recorded seeds remain reproducible.
func mintSeed() int64 {
	return int64(rand.Int31())
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
	if err := provider.ValidateReferences(context.Background(), m.currentCaps(), m.refImages, true); err != nil {
		return m.editInvalidDraft(m.composeDraft(), err), nil
	}
	m.persistAspectRatio()
	d := m.composeDraft()
	m.reuseSeed = nil // consumed into the draft
	return m.runDraft(d)
}

// runDraft starts a generation for the exact draft d, capturing it as lastDraft
// so re-generate can repeat it precisely.
func (m Model) runDraft(d gen.Draft) (Model, tea.Cmd) {
	for _, model := range m.models {
		if model.ID == d.Model {
			if err := provider.ValidateReferences(context.Background(), model.Capabilities, d.ReferenceImages, true); err != nil {
				return m.editInvalidDraft(d, err), nil
			}
			break
		}
	}
	m.lastDraft = d
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	p := m.deps.Provider
	dir := m.outputDir
	nameOpts := gen.NameOptions{
		Enabled: m.deps.Config.SmartFilenames,
		Model:   string(m.deps.Config.FilenameModel),
	}
	cmd := func() tea.Msg {
		return generatedMsg{outcome: gen.Run(ctx, p, d, dir, nameOpts)}
	}
	return m, tea.Batch(cmd, m.spinner.Tick)
}

// --- update ----------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (updated tea.Model, command tea.Cmd) {
	// Invalidate pending attachment reads when the draft or active screen changes.
	before := m.attachmentToken()
	beforePhase, beforeRefs := m.phase, len(m.refImages)
	beforeStatus := m.status
	defer func() {
		if next, ok := updated.(Model); ok {
			if next.phase != beforePhase || next.attachmentToken() != before || len(next.refImages) != beforeRefs || next.quit {
				next.draftRevision++
			}
			if next.phase != phaseResult || next.outcome.Failure != gen.FailureNone || next.quit {
				if next.previewCancel != nil || next.previewItem.ID != 0 {
					next.clearPreview()
				}
			} else if beforePhase == phaseResult && next.status != beforeStatus && next.deps.Preview != nil {
				width, height, _ := next.previewBox()
				if width != next.previewBudgetWidth || height != next.previewBudgetHeight {
					if cmd := next.beginPreview(); cmd != nil {
						command = tea.Batch(command, cmd)
					}
				}
			}
			updated = next
		}
	}()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.prompt.SetWidth(m.contentWidth())
		m.picker.SetSize(m.contentWidth(), pickerHeight(msg.Height))
		m.history.SetSize(m.contentWidth(), pickerHeight(msg.Height))
		m.previewGeometryRevision++
		m.previewGeometry = preview.Geometry{}
		m.clearPreview()
		return m, m.geometryCmd()

	case previewGeometryMsg:
		if msg.revision != m.previewGeometryRevision {
			return m, nil
		}
		if msg.err != nil || !msg.geometry.Valid(m.width, m.height) {
			return m, nil
		}
		m.previewGeometry = msg.geometry
		if m.phase == phaseResult && m.outcome.Failure == gen.FailureNone {
			cmd := m.beginPreview()
			return m, cmd
		}
		return m, nil

	case previewLoadedMsg:
		if msg.revision != m.previewRevision || m.phase != phaseResult || m.outcome.Failure != gen.FailureNone || m.quit {
			return m, nil
		}
		if m.previewCancel != nil {
			m.previewCancel()
			m.previewCancel = nil
		}
		if msg.err == nil {
			m.previewItem = msg.item
		}
		return m, nil

	case modelsLoadedMsg:
		m.modelsReady = true
		m.models, m.modelsErr = msg.models, msg.err
		if msg.err != nil {
			m.status = modelErrorText(msg.err)
			return m, nil
		}
		m = m.initPricing()
		m.rebuildItems()
		last := config.LoadState(m.deps.StatePath).LastModel
		if idx := preselectModel(m.models, last); idx >= 0 {
			m.picker.Select(idx)
		}
		m.status = fmt.Sprintf("%d models · / to filter", len(m.models))
		return m, m.pricingCmds()

	case priceLoadedMsg:
		if m.prices == nil {
			m.prices = map[string]provider.Price{}
		}
		if msg.err == nil {
			m.prices[msg.modelID] = msg.price
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
		if msg.outcome.Failure == gen.FailureInput {
			return m.editInvalidDraft(m.lastDraft, msg.outcome.Err), nil
		}
		m.phase = phaseResult
		m.status = msg.outcome.Message()
		var cmds []tea.Cmd
		if m.outcome.Failure == gen.FailureNone {
			if pc := m.beginPreview(); pc != nil {
				cmds = append(cmds, pc)
			}
			if hc := m.recordHistoryCmd(); hc != nil {
				cmds = append(cmds, hc)
			}
			if m.deps.Config.AutoOpen {
				cmds = append(cmds, m.openCmd())
			}
		}
		return m, tea.Batch(cmds...)

	case pastedImageMsg:
		if !m.acceptsAttachment(msg.token) {
			return m, nil
		}
		if msg.err != nil {
			m.status = pasteErrorText(msg.err)
			return m, nil
		}
		all := append(append([]provider.ReferenceImage(nil), m.refImages...), msg.ref)
		if err := provider.ValidateReferences(context.Background(), m.currentCaps(), all, false); err != nil {
			m.status = pasteErrorText(err)
			return m, nil
		}
		m.refImages = all
		m.status = fmt.Sprintf("Attached image — %d reference image(s)", len(m.refImages))
		return m, nil

	case pathImagesMsg:
		if !m.acceptsAttachment(msg.token) {
			return m, nil
		}
		if msg.err != nil {
			m.status = pasteErrorText(msg.err)
			return m, nil
		}
		m.refImages = append(append([]provider.ReferenceImage(nil), m.refImages...), msg.refs...)
		m.prompt.SetValue(msg.cleaned)
		return m.reviewCompose()

	case folderOpenedMsg:
		if msg.err != nil {
			m.status = "Could not open folder: " + msg.err.Error()
		} else {
			m.status = "Opened containing folder"
		}
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
		return m, tea.Batch(loadModels(m.deps.Provider), m.spinner.Tick)

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
	if m.phase == phaseHistory {
		var cmd tea.Cmd
		m.history, cmd = m.history.Update(msg)
		return m, cmd
	}
	if m.phase == phaseConfig {
		return m.updateConfig(msg)
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
	case phaseHistory:
		return m.handleHistoryKey(msg)
	case phaseConfig:
		return m.handleConfigKey(msg)
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
	case "h":
		return m.openHistory(phasePicker), nil
	case "c":
		return m.openConfig(phasePicker), nil
	case "enter":
		if it, ok := m.picker.SelectedItem().(modelItem); ok {
			m.selected = indexOfModel(m.models, it.m.ID)
			s := config.LoadState(m.deps.StatePath)
			m.aspectRatio = pickAspectRatio(it.m.Capabilities.AspectRatios, s.LastAspectRatio)
			m.transparentBackground = it.m.Capabilities.SupportsTransparentBackground && s.LastTransparentBackground
			s.LastModel = it.m.ID
			s.LastTransparentBackground = m.transparentBackground
			_ = config.SaveState(m.deps.StatePath, s)
			m.phase = phaseCompose
			m.prompt.Focus()
			m.status = "Compose your prompt"
			// If reference images are carried over (e.g. try-again-with-new-model)
			// but the newly chosen model cannot accept them, drop them and tell
			// the user rather than erroring at generate time.
			if !it.m.Capabilities.AcceptsReferenceImages && len(m.refImages) > 0 {
				n := len(m.refImages)
				m.refImages = nil
				m.status = fmt.Sprintf("Dropped %d reference image(s) — this model does not accept them", n)
			} else if err := provider.ValidateReferences(context.Background(), it.m.Capabilities, m.refImages, true); err != nil {
				m.status = err.Error()
			}
		}
		return m, nil
	}

	// Everything else (up/down, /, paging) goes to the list.
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

// indexOfModel returns the index of the model with id, or -1.
func indexOfModel(models []provider.Model, id string) int {
	for i, mdl := range models {
		if mdl.ID == id {
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
			return m, m.pasteCmd()
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
	case "ctrl+p":
		// Open the history view to recall and reuse a past prompt.
		return m.openHistory(phaseCompose), nil
	case "ctrl+t":
		// Toggle transparent background. The compose settings line is the
		// canonical on/off indicator, so a successful toggle sets no transient
		// status (which would duplicate it). Only the unsupported case needs a
		// message.
		if m.currentCaps().SupportsTransparentBackground {
			m.transparentBackground = !m.transparentBackground
			return m, nil
		}
		m.status = "Current model does not support transparent backgrounds."
		return m, nil

	case "up":
		// Recall an older prompt only at the first line; otherwise let the
		// textarea move the cursor up.
		if m.prompt.Line() == 0 {
			return m.recallPrevious(), nil
		}
	case "down":
		// Advance toward a newer prompt (or the draft) only at the last line;
		// otherwise let the textarea move the cursor down.
		if m.recallActive() && m.prompt.Line() == m.prompt.LineCount()-1 {
			return m.recallNext(), nil
		}
	}
	// Any other key reaches the textarea. If it edits the text during a recall
	// session, end the session so the next Up starts fresh from the newest.
	if m.recallActive() && isEditingKey(msg) {
		m.endRecallSession()
	}
	var cmd tea.Cmd
	m.prompt, cmd = m.prompt.Update(msg)
	return m, cmd
}

// isEditingKey reports whether a key reaching the textarea should end a recall
// session. Cursor-movement keys do not; everything else (text entry, deletion)
// does. Conservative by design — a false positive merely restarts recall from
// the newest entry on the next Up, which is the expected shell behavior.
func isEditingKey(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyLeft, tea.KeyRight, tea.KeyUp, tea.KeyDown,
		tea.KeyHome, tea.KeyEnd, tea.KeyPgUp, tea.KeyPgDown:
		return false
	default:
		return true
	}
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
			m.status = "Reading reference images…"
			return m, m.loadPathsCmd(paths, cleaned)
		}
	}
	return m.reviewCompose()
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
	case "f":
		if m.outcome.Failure == gen.FailureNone && m.outcome.Path != "" {
			return m, m.openFolderCmd()
		}
	case "r":
		if m.outcome.Retryable() {
			m.phase = phaseGenerating
			m.status = "Generating..."
			return m.generate()
		}
	case "p":
		if m.outcome.Failure == gen.FailureNone {
			m.phase = phaseCompose
			m.prompt.Focus()
			m.status = "Editing prompt"
		}
		return m, nil
	case "m":
		// Try again with a new model: back to the picker, preserving prompt,
		// aspect ratio, and reference images (dropped later if the newly chosen
		// model cannot accept them).
		if m.outcome.Failure == gen.FailureNone {
			return m.tryNewModel()
		}
	case "n":
		if m.outcome.Failure == gen.FailureNone {
			// New image: clear prompt and reference images, keep the model.
			return m.newImage()
		}
		// Failure path keeps the historical "new" behavior.
		m.phase = phaseCompose
		m.prompt.Focus()
		m.status = "Compose your prompt"
	case "g":
		// Re-generate reusing the same parameters (and seed) to reproduce.
		if m.outcome.Failure == gen.FailureNone {
			return m.regenerate(false)
		}
	case "v":
		// Re-generate with a fresh seed to vary (seed-capable models only).
		if m.outcome.Failure == gen.FailureNone && m.lastDraftSeedCapable() {
			return m.regenerate(true)
		}
	case "enter":
		// Historical convenience: on the failure path, enter starts a new
		// prompt. On success it reproduces (same as re-generate).
		if m.outcome.Failure == gen.FailureNone {
			return m.regenerate(false)
		}
		m.phase = phaseCompose
		m.prompt.Focus()
		m.status = "Compose your prompt"
	case "q":
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

// lastDraftSeedCapable reports whether the just-run draft used a seed (i.e. the
// model is seed-capable), so vary makes sense.
func (m Model) lastDraftSeedCapable() bool {
	return m.lastDraft.Seed != nil
}

// tryNewModel returns to the picker preserving the composed parameters. On the
// subsequent model selection, reference images are dropped if the new model
// cannot accept them.
func (m Model) tryNewModel() (tea.Model, tea.Cmd) {
	m.phase = phasePicker
	m.status = "Choose a model to try again"
	return m, nil
}

// newImage clears the prompt and reference images, keeps the selected model,
// and returns to compose. The next generation mints a fresh seed.
func (m Model) newImage() (tea.Model, tea.Cmd) {
	m.prompt.SetValue("")
	m.refImages = nil
	m.reuseSeed = nil
	m.phase = phaseCompose
	m.prompt.Focus()
	m.status = "Compose your prompt"
	return m, nil
}

// regenerate re-runs the just-completed generation. For a seed-capable model,
// vary mints a fresh seed (and the result shows it); reuse repeats the prior
// seed to reproduce. For a non-seed model it simply repeats lastDraft.
func (m Model) regenerate(vary bool) (tea.Model, tea.Cmd) {
	d := m.lastDraft
	switch {
	case !m.lastDraftSeedCapable():
		m.status = "Re-generating…"
	case vary:
		s := mintSeed()
		d.Seed = &s
		m.status = fmt.Sprintf("Re-generating with new seed %d…", s)
	default:
		m.status = fmt.Sprintf("Re-generating with seed %d…", *d.Seed)
	}
	m.phase = phaseGenerating
	return m.runDraft(d)
}

func (m Model) openCmd() tea.Cmd {
	path := m.outcome.Path
	override := m.deps.OpenCommand
	opener := m.deps.Open
	if opener == nil {
		opener = gen.Open
	}
	return func() tea.Msg {
		_ = opener(context.Background(), override, path)
		return nil
	}
}

func (m Model) currentModelID() string {
	if m.selected >= 0 && m.selected < len(m.models) {
		return m.models[m.selected].ID
	}
	if it, ok := m.picker.SelectedItem().(modelItem); ok {
		return it.m.ID
	}
	return ""
}

type folderOpenedMsg struct{ err error }

func (m Model) openFolderCmd() tea.Cmd {
	path := m.outcome.Path
	opener := m.deps.OpenFolder
	if opener == nil {
		opener = gen.OpenFolder
	}
	return func() tea.Msg { return folderOpenedMsg{err: opener(context.Background(), path)} }
}
