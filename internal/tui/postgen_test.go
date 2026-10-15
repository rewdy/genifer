package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/provider"
)

// capModel builds a Model with a single selected model whose capabilities are
// supplied, sitting in the compose phase.
func capModel(caps provider.Capabilities) Model {
	m := New(Deps{})
	m.models = []provider.Model{{
		ID:           "test/model",
		Name:         "Test Model",
		Capabilities: caps,
	}}
	m.selected = 0
	m.phase = phaseCompose
	m.rebuildItems()
	m.picker.Select(0)
	return m
}

// resultModel builds a Model sitting on a successful result screen for a model
// with the given capabilities. lastDraft is captured as if the generation had
// just fired, and the outcome's seed mirrors the draft's.
func resultModel(caps provider.Capabilities) Model {
	m := capModel(caps)
	m.prompt.SetValue("a red bird")
	d := m.composeDraft()
	m.lastDraft = d
	m.phase = phaseResult
	m.outcome = gen.Outcome{Path: "out.png", Seed: d.Seed}
	return m
}

// 1.1: a seed is minted only for seed-capable models.
func TestComposeDraftMintsSeedOnlyWhenSupported(t *testing.T) {
	cases := []struct {
		name     string
		caps     provider.Capabilities
		wantSeed bool
	}{
		{"seed-capable", provider.Capabilities{SupportsSeed: true}, true},
		{"no seed", provider.Capabilities{}, false},
		{"ref-only", provider.Capabilities{AcceptsReferenceImages: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := capModel(tc.caps)
			m.prompt.SetValue("a red bird")
			d := m.composeDraft()
			if tc.wantSeed && d.Seed == nil {
				t.Error("expected a minted seed for a seed-capable model")
			}
			if !tc.wantSeed && d.Seed != nil {
				t.Errorf("did not expect a seed, got %d", *d.Seed)
			}
		})
	}
}

// Minted seeds fit the non-negative signed 32-bit range and are present for a
// seed-capable model.
func TestComposeDraftSeedValue(t *testing.T) {
	m := capModel(provider.Capabilities{SupportsSeed: true})
	m.prompt.SetValue("x")
	d := m.composeDraft()
	if d.Seed == nil {
		t.Fatal("expected a seed")
	}
	if *d.Seed < 0 || *d.Seed > 1<<31-1 {
		t.Errorf("seed = %d, want 0 through 2147483647", *d.Seed)
	}
}

// 2.1: the result view shows the seed only when one was used.
func TestResultViewShowsSeedWhenUsed(t *testing.T) {
	seeded := resultModel(provider.Capabilities{SupportsSeed: true})
	out := ansi.Strip(seeded.resultView())
	if !strings.Contains(out, "seed:") {
		t.Errorf("result view missing seed line for seed-capable model; got %q", out)
	}

	plain := resultModel(provider.Capabilities{})
	out = ansi.Strip(plain.resultView())
	if strings.Contains(out, "seed:") {
		t.Errorf("result view should not show a seed for a non-seed model; got %q", out)
	}
}

// 3.1: lastDraft equals the submitted draft when generation fires.
func TestLastDraftCapturedOnGenerate(t *testing.T) {
	m := capModel(provider.Capabilities{SupportsSeed: true})
	m.prompt.SetValue("a red bird")
	want := m.composeDraft()
	// generate() captures composeDraft() into lastDraft. Because the seed is
	// minted per call, compare the stable fields and assert a seed is present.
	updated, _ := m.generate()
	um := updated
	if um.lastDraft.Prompt != want.Prompt || um.lastDraft.Model != want.Model {
		t.Errorf("lastDraft = %+v, want prompt/model to match %+v", um.lastDraft, want)
	}
	if um.lastDraft.Seed == nil {
		t.Error("lastDraft should carry the minted seed")
	}
}

// 3.2: re-generate reuse sends the same seed; vary mints a new one and shows it.
func TestRegenerateReuseAndVarySeed(t *testing.T) {
	m := resultModel(provider.Capabilities{SupportsSeed: true})
	origSeed := *m.lastDraft.Seed

	// Reuse (g): same seed.
	reuse, _ := m.handleResultKey(key("g"))
	rm := reuse.(Model)
	if rm.lastDraft.Seed == nil || *rm.lastDraft.Seed != origSeed {
		t.Errorf("reuse seed = %v, want %d", rm.lastDraft.Seed, origSeed)
	}
	if rm.phase != phaseGenerating {
		t.Errorf("phase = %v, want generating", rm.phase)
	}

	// Vary (v): a new seed, shown in status.
	vary, _ := m.handleResultKey(key("v"))
	vm := vary.(Model)
	if vm.lastDraft.Seed == nil {
		t.Fatal("vary should set a seed")
	}
	if *vm.lastDraft.Seed == origSeed {
		t.Error("vary should mint a different seed")
	}
	if !strings.Contains(vm.status, "new seed") {
		t.Errorf("vary status should mention a new seed; got %q", vm.status)
	}
}

// 3.2: for a non-seed model, re-generate repeats lastDraft unchanged.
func TestRegenerateNonSeedRepeats(t *testing.T) {
	m := resultModel(provider.Capabilities{})
	before := m.lastDraft
	updated, _ := m.handleResultKey(key("g"))
	um := updated.(Model)
	if um.lastDraft.Prompt != before.Prompt || um.lastDraft.Seed != nil {
		t.Errorf("non-seed re-generate changed the draft: %+v", um.lastDraft)
	}
	if um.phase != phaseGenerating {
		t.Errorf("phase = %v, want generating", um.phase)
	}
}

// 3.3: try-again-with-new-model preserves prompt, aspect ratio, and ref images.
func TestTryNewModelPreservesParameters(t *testing.T) {
	m := resultModel(provider.Capabilities{AcceptsReferenceImages: true})
	m.aspectRatio = "16:9"
	m.addRefImage(provider.ReferenceImage{MediaType: "image/png"})

	updated, _ := m.handleResultKey(key("m"))
	um := updated.(Model)
	if um.phase != phasePicker {
		t.Errorf("phase = %v, want picker", um.phase)
	}
	if um.prompt.Value() != "a red bird" {
		t.Errorf("prompt not preserved; got %q", um.prompt.Value())
	}
	if um.aspectRatio != "16:9" {
		t.Errorf("aspect ratio not preserved; got %q", um.aspectRatio)
	}
	if len(um.refImages) != 1 {
		t.Errorf("reference images not preserved; got %d", len(um.refImages))
	}
}

// 3.3: reference images are dropped when the newly selected model rejects them,
// and the user is informed.
func TestTryNewModelDropsRefImages(t *testing.T) {
	m := resultModel(provider.Capabilities{AcceptsReferenceImages: true})
	m.addRefImage(provider.ReferenceImage{MediaType: "image/png"})
	// Carry over to the picker.
	toPicker, _ := m.handleResultKey(key("m"))
	pm := toPicker.(Model)

	// The picker now offers a model that does NOT accept reference images.
	pm.models = []provider.Model{{ID: "plain/model", Name: "Plain", Capabilities: provider.Capabilities{}}}
	pm.rebuildItems()
	pm.picker.Select(0)

	selected, _ := pm.handlePickerKey(key("enter"))
	sm := selected.(Model)
	if len(sm.refImages) != 0 {
		t.Errorf("reference images should be dropped; got %d", len(sm.refImages))
	}
	if !strings.Contains(sm.status, "Dropped") {
		t.Errorf("user should be informed of dropped images; status = %q", sm.status)
	}
}

// 3.4: new-image clears the prompt and reference images but keeps the model.
func TestNewImageClearsAndKeepsModel(t *testing.T) {
	m := resultModel(provider.Capabilities{AcceptsReferenceImages: true})
	m.addRefImage(provider.ReferenceImage{MediaType: "image/png"})
	wantModel := m.currentModelID()

	updated, _ := m.handleResultKey(key("n"))
	um := updated.(Model)
	if um.prompt.Value() != "" {
		t.Errorf("prompt not cleared; got %q", um.prompt.Value())
	}
	if len(um.refImages) != 0 {
		t.Errorf("reference images not cleared; got %d", len(um.refImages))
	}
	if um.currentModelID() != wantModel {
		t.Errorf("model changed; got %q, want %q", um.currentModelID(), wantModel)
	}
	if um.phase != phaseCompose {
		t.Errorf("phase = %v, want compose", um.phase)
	}
}

// 3.5: the success footer shows all post-generation hints (with the reuse/vary
// affordance for seed-capable models), and the failure footer is unchanged.
func TestResultFooterHints(t *testing.T) {
	seeded := ansi.Strip(resultModel(provider.Capabilities{SupportsSeed: true}).footerView())
	for _, want := range []string{"open", "try new model", "new image", "same seed", "vary seed", "quit"} {
		if !strings.Contains(seeded, want) {
			t.Errorf("seed-capable success footer missing %q; got %q", want, seeded)
		}
	}

	plain := ansi.Strip(resultModel(provider.Capabilities{}).footerView())
	if !strings.Contains(plain, "re-generate") {
		t.Errorf("non-seed success footer missing re-generate; got %q", plain)
	}
	if strings.Contains(plain, "vary seed") {
		t.Errorf("non-seed success footer should not offer vary; got %q", plain)
	}

	// Failure path is unchanged: retryable shows retry/new/quit only.
	fm := capModel(provider.Capabilities{})
	fm.phase = phaseResult
	fm.outcome = gen.Outcome{Failure: gen.FailureGeneration}
	fail := ansi.Strip(fm.footerView())
	for _, want := range []string{"retry", "new", "quit"} {
		if !strings.Contains(fail, want) {
			t.Errorf("failure footer missing %q; got %q", want, fail)
		}
	}
	if strings.Contains(fail, "try new model") || strings.Contains(fail, "new image") {
		t.Errorf("failure footer should not show success-only actions; got %q", fail)
	}
}

func TestMintSeedProviderCompatibleRange(t *testing.T) {
	for i := 0; i < 100; i++ {
		seed := mintSeed()
		if seed < 0 || seed > 1<<31-1 {
			t.Fatalf("seed = %d, want 0 through 2147483647", seed)
		}
	}
}

func TestResultEditPromptPreservesDraft(t *testing.T) {
	m := resultModel(provider.Capabilities{AcceptsReferenceImages: true})
	m.aspectRatio = "16:9"
	m.addRefImage(provider.ReferenceImage{Data: []byte("reference"), MediaType: "image/png"})
	updated, cmd := m.handleResultKey(key("p"))
	got := updated.(Model)
	if cmd != nil || got.phase != phaseCompose || !got.prompt.Focused() || got.prompt.Value() != m.prompt.Value() || got.currentModelID() != m.currentModelID() || got.aspectRatio != m.aspectRatio || len(got.refImages) != 1 {
		t.Fatal("edit prompt must preserve the draft without generating")
	}
	footer := ansi.Strip(m.footerView())
	if !strings.Contains(footer, "p edit prompt") || !strings.Contains(footer, "m try new model") || strings.Contains(footer, "t try new model") {
		t.Fatal(footer)
	}
	unchanged, cmd := m.handleResultKey(key("t"))
	if cmd != nil || unchanged.(Model).phase != phaseResult {
		t.Fatal("old model shortcut still active")
	}
	m.outcome.Failure = gen.FailureGeneration
	updated, cmd = m.handleResultKey(key("p"))
	if cmd != nil || updated.(Model).phase != phaseResult || strings.Contains(ansi.Strip(m.footerView()), "edit prompt") {
		t.Fatal("success-only action available on failure")
	}
}
