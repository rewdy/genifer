package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rewdy/genifer/internal/provider"
)

// modelErrorText distinguishes a missing key from a rejected one.
func TestModelErrorTextDistinguishesKeyCases(t *testing.T) {
	noKey := modelErrorText(fmt.Errorf("%w: resolved key is empty", provider.ErrNoAPIKey))
	if !strings.Contains(noKey, "No API key configured") {
		t.Errorf("missing-key message = %q, want a 'No API key configured' hint", noKey)
	}

	rejected := modelErrorText(fmt.Errorf("%w: bad key", provider.ErrAuth))
	if !strings.Contains(rejected, "rejected") {
		t.Errorf("rejected-key message = %q, want a 'rejected' hint", rejected)
	}

	if noKey == rejected {
		t.Error("missing-key and rejected-key messages must differ")
	}
}
