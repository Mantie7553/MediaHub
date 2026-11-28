package notifications

import (
	"testing"

	"github.com/Mantie7553/MediaHub/backend/internal/notify"
)

func TestCheckKinds(t *testing.T) {
	user := notify.KindsFor(false)
	if err := checkKinds(map[string]bool{notify.KindNewParts: false, notify.KindTitleReady: true}, user); err != nil {
		t.Errorf("user kinds: %v", err)
	}
	if err := checkKinds(map[string]bool{notify.KindApprovalNeeded: false}, user); err == nil {
		t.Error("a user can't set an admin kind")
	}
	if err := checkKinds(map[string]bool{"nonsense": true}, notify.KindsFor(true)); err == nil {
		t.Error("an unknown kind must be refused")
	}
	if err := checkKinds(map[string]bool{notify.KindApprovalNeeded: false}, notify.KindsFor(true)); err != nil {
		t.Errorf("an admin kind for an admin: %v", err)
	}
}
