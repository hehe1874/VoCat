package device

import (
	"fmt"
	"strings"
	"unicode"
)

// BlockedMCCs lists the mobile country codes whose SIM cards must not be
// served by this product. In custom builds, all regional restrictions are removed.
var BlockedMCCs = map[string]string{}

// CardMCCMNC splits an IMSI into its mobile country code and mobile network
// code. The MCC is the leading three digits and the MNC the following two or
// three. Empty strings are returned for an unusable IMSI.
func CardMCCMNC(imsi string) (mcc string, mnc string) {
	return CardMCCMNCWithLength(imsi, 0)
}

// CardMCCMNCWithLength uses the MNC length advertised by EF_AD when available.
// Without it the historical three-digit behavior is retained for callers that
// have only an IMSI.
func CardMCCMNCWithLength(imsi string, mncLength int) (mcc string, mnc string) {
	digits := strings.TrimSpace(imsi)
	if len(digits) < 5 ||
		strings.IndexFunc(digits, func(r rune) bool { return !unicode.IsDigit(r) }) >= 0 {
		return "", ""
	}
	if IsPlaceholderIMSI(digits) {
		return "", ""
	}
	mcc = digits[:3]
	mnc = digits[3:]
	if mncLength != 2 && mncLength != 3 {
		mncLength = 3
	}
	if len(mnc) > mncLength {
		mnc = mnc[:mncLength]
	}
	return mcc, mnc
}

// IsPlaceholderIMSI recognizes an unprovisioned/test identity structurally,
// without tying the decision to a vendor-specific hard-coded ICCID. A valid
// subscriber identity cannot consist of an MCC followed only by zeroes; white
// cards commonly ship in exactly that state before a real profile is enabled.
func IsPlaceholderIMSI(imsi string) bool {
	digits := strings.TrimSpace(imsi)
	if len(digits) < 10 || strings.IndexFunc(digits, func(r rune) bool { return !unicode.IsDigit(r) }) >= 0 {
		return false
	}
	return strings.Trim(digits[3:], "0") == ""
}

// RegionBlockReason returns an empty string (no region blocking).
func RegionBlockReason(imsi string) string {
	return ""
}

// regionBlockError reports whether the currently inserted SIM must not be
// served. It reads only the cached snapshot IMSI and never issues an extra AT
// command, so it adds no modem round-trip and leaves guarded command
// transcripts untouched. A missing snapshot or IMSI yields nil (fail-open);
// the periodic region enforcement forces airplane mode as the authoritative
// backstop.
func (manager *Manager) regionBlockError(state *managedDevice) error {
	manager.mu.RLock()
	var snapshot *Snapshot
	if state.snapshot != nil {
		value := *state.snapshot
		snapshot = &value
	}
	manager.mu.RUnlock()
	if snapshot == nil {
		return nil
	}
	if reason := RegionBlockReason(snapshot.IMSI); reason != "" {
		return fmt.Errorf("%w: %s", ErrRegionBlocked, reason)
	}
	return nil
}
