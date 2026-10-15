package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/provider"
)

// attachmentToken identifies the draft a background read belongs to. Revision
// also detects navigation away and back, even if the text/model are unchanged.
type attachmentToken struct {
	revision      uint64
	model, prompt string
}

func (m Model) attachmentToken() attachmentToken {
	return attachmentToken{m.draftRevision, m.currentModelID(), m.prompt.Value()}
}
func (m Model) acceptsAttachment(token attachmentToken) bool {
	return !m.quit && m.phase == phaseCompose && token == m.attachmentToken()
}

type pathImagesMsg struct {
	token   attachmentToken
	refs    []provider.ReferenceImage
	cleaned string
	err     error
}

func (m Model) loadPathsCmd(paths []string, cleaned string) tea.Cmd {
	token, caps := m.attachmentToken(), m.currentCaps()
	existing := append([]provider.ReferenceImage(nil), m.refImages...)
	return func() tea.Msg {
		msg := pathImagesMsg{token: token, cleaned: cleaned}
		for _, path := range paths {
			ref, err := loadRefImageFile(path)
			if err != nil {
				msg.err = err
				return msg
			}
			msg.refs = append(msg.refs, ref)
		}
		all := append(existing, msg.refs...)
		msg.err = provider.ValidateReferences(context.Background(), caps, all, false)
		return msg
	}
}
func (m Model) pasteCmd() tea.Cmd {
	token := m.attachmentToken()
	cmd := pasteImageCmd(m.deps.Paster)
	return func() tea.Msg {
		msg := cmd().(pastedImageMsg)
		msg.token = token
		return msg
	}
}

// reviewCompose performs pure final draft validation after any path batch.
func (m Model) reviewCompose() (tea.Model, tea.Cmd) {
	if err := provider.ValidateReferences(context.Background(), m.currentCaps(), m.refImages, true); err != nil {
		m.status = err.Error()
		return m, nil
	}
	if len(strings.TrimSpace(m.prompt.Value())) == 0 {
		m.status = "Prompt is empty — enter some text"
		return m, nil
	}
	m.phase = phaseReview
	m.prompt.Blur()
	m.status = "Review your prompt"
	return m, nil
}

// editInvalidDraft restores the exact rejected draft so it can be corrected.
func (m Model) editInvalidDraft(d gen.Draft, err error) Model {
	m.prompt.SetValue(d.Prompt)
	m.refImages = d.ReferenceImages
	m.aspectRatio = d.AspectRatio
	m.reuseSeed = d.Seed
	m.phase = phaseCompose
	m.prompt.Focus()
	m.status = err.Error()
	return m
}
