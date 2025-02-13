package state_native

import (
	"github.com/prysmaticlabs/prysm/runtime/version"
)

// CurrentEpochParticipation corresponding to participation bits on the beacon chain.
func (b *BeaconState) CurrentEpochParticipation() ([]byte, error) {
	if b.version == version.Phase0 {
		return nil, errNotSupported("CurrentEpochParticipation", b.version)
	}

	if b.currentEpochParticipation == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.currentEpochParticipationVal(), nil
}

// PreviousEpochParticipation corresponding to participation bits on the beacon chain.
func (b *BeaconState) PreviousEpochParticipation() ([]byte, error) {
	if b.version == version.Phase0 {
		return nil, errNotSupported("PreviousEpochParticipation", b.version)
	}

	if b.previousEpochParticipation == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.previousEpochParticipationVal(), nil
}

// currentEpochParticipationVal corresponding to participation bits on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) currentEpochParticipationVal() []byte {
	tmp := make([]byte, len(b.currentEpochParticipation))
	copy(tmp, b.currentEpochParticipation)
	return tmp
}

// previousEpochParticipationVal corresponding to participation bits on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) previousEpochParticipationVal() []byte {
	tmp := make([]byte, len(b.previousEpochParticipation))
	copy(tmp, b.previousEpochParticipation)
	return tmp
}
