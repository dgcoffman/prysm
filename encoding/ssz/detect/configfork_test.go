package detect

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/time/slots"

	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestSlotFromBlock(t *testing.T) {
	b := util.NewBeaconBlock()
	var slot types.Slot = 3
	b.Block.Slot = slot
	bb, err := b.MarshalSSZ()
	require.NoError(t, err)
	sfb, err := slotFromBlock(bb)
	require.NoError(t, err)
	require.Equal(t, slot, sfb)

	ba := util.NewBeaconBlockAltair()
	ba.Block.Slot = slot
	bab, err := ba.MarshalSSZ()
	require.NoError(t, err)
	sfba, err := slotFromBlock(bab)
	require.NoError(t, err)
	require.Equal(t, slot, sfba)

	bm := util.NewBeaconBlockBellatrix()
	bm.Block.Slot = slot
	bmb, err := ba.MarshalSSZ()
	require.NoError(t, err)
	sfbm, err := slotFromBlock(bmb)
	require.NoError(t, err)
	require.Equal(t, slot, sfbm)
}

func TestByState(t *testing.T) {
	bc, cleanup := hackBellatrixMaxuint()
	defer cleanup()
	altairSlot, err := slots.EpochStart(bc.AltairForkEpoch)
	bellaSlot, err := slots.EpochStart(bc.BellatrixForkEpoch)
	require.NoError(t, err)
	cases := []struct {
		name        string
		version     int
		slot        types.Slot
		forkversion [4]byte
	}{
		{
			name:        "genesis",
			version:     version.Phase0,
			slot:        0,
			forkversion: bytesutil.ToBytes4(bc.GenesisForkVersion),
		},
		{
			name:        "altair",
			version:     version.Altair,
			slot:        altairSlot,
			forkversion: bytesutil.ToBytes4(bc.AltairForkVersion),
		},
		{
			name:        "bellatrix",
			version:     version.Bellatrix,
			slot:        bellaSlot,
			forkversion: bytesutil.ToBytes4(bc.BellatrixForkVersion),
		},
	}
	for _, c := range cases {
		st, err := stateForVersion(c.version)
		require.NoError(t, err)
		require.NoError(t, st.SetFork(&ethpb.Fork{
			PreviousVersion: make([]byte, 4),
			CurrentVersion:  c.forkversion[:],
			Epoch:           0,
		}))
		require.NoError(t, st.SetSlot(c.slot))
		m, err := st.MarshalSSZ()
		require.NoError(t, err)
		cf, err := FromState(m)
		require.NoError(t, err)
		require.Equal(t, c.version, cf.Fork)
		require.Equal(t, c.forkversion, cf.Version)
		require.Equal(t, bc.ConfigName, cf.Config.ConfigName)
	}
}

func stateForVersion(v int) (state.BeaconState, error) {
	switch v {
	case version.Phase0:
		return util.NewBeaconState()
	case version.Altair:
		return util.NewBeaconStateAltair()
	case version.Bellatrix:
		return util.NewBeaconStateBellatrix()
	default:
		return nil, fmt.Errorf("unrecognoized version %d", v)
	}
}

func TestUnmarshalState(t *testing.T) {
	ctx := context.Background()
	bc, cleanup := hackBellatrixMaxuint()
	defer cleanup()
	altairSlot, err := slots.EpochStart(bc.AltairForkEpoch)
	bellaSlot, err := slots.EpochStart(bc.BellatrixForkEpoch)
	require.NoError(t, err)
	cases := []struct {
		name        string
		version     int
		slot        types.Slot
		forkversion [4]byte
	}{
		{
			name:        "genesis",
			version:     version.Phase0,
			slot:        0,
			forkversion: bytesutil.ToBytes4(bc.GenesisForkVersion),
		},
		{
			name:        "altair",
			version:     version.Altair,
			slot:        altairSlot,
			forkversion: bytesutil.ToBytes4(bc.AltairForkVersion),
		},
		{
			name:        "bellatrix",
			version:     version.Bellatrix,
			slot:        bellaSlot,
			forkversion: bytesutil.ToBytes4(bc.BellatrixForkVersion),
		},
	}
	for _, c := range cases {
		st, err := stateForVersion(c.version)
		require.NoError(t, err)
		require.NoError(t, st.SetFork(&ethpb.Fork{
			PreviousVersion: make([]byte, 4),
			CurrentVersion:  c.forkversion[:],
			Epoch:           0,
		}))
		require.NoError(t, st.SetSlot(c.slot))
		m, err := st.MarshalSSZ()
		require.NoError(t, err)
		cf, err := FromState(m)
		require.NoError(t, err)
		s, err := cf.UnmarshalBeaconState(m)
		require.NoError(t, err)
		expected, err := st.HashTreeRoot(ctx)
		require.NoError(t, err)
		actual, err := s.HashTreeRoot(ctx)
		require.NoError(t, err)
		require.DeepEqual(t, expected, actual)
	}
}

func hackBellatrixMaxuint() (*params.BeaconChainConfig, func()) {
	// We monkey patch the config to use a smaller value for the bellatrix fork epoch.
	// Upstream configs use MaxUint64, which leads to a multiplication overflow when converting epoch->slot.
	// Unfortunately we have unit tests that assert our config matches the upstream config, so we have to choose between
	// breaking conformance, adding a special case to the conformance unit test, or patch it here.
	previous := params.BeaconConfig()
	bc := params.MainnetConfig().Copy()
	bc.BellatrixForkEpoch = math.MaxUint32
	bc.InitializeForkSchedule()
	params.OverrideBeaconConfig(bc)
	// override the param used for mainnet with the patched version
	params.KnownConfigs[params.MainnetName] = func() *params.BeaconChainConfig {
		return bc
	}
	return bc, func() {
		// put the previous BeaconChainConfig back in place at the end of the test
		params.OverrideBeaconConfig(previous)
		// restore the normal MainnetConfig func in the KnownConfigs mapping
		params.KnownConfigs[params.MainnetName] = params.MainnetConfig
	}
}

func TestUnmarshalBlock(t *testing.T) {
	bc, cleanup := hackBellatrixMaxuint()
	defer cleanup()
	require.Equal(t, types.Epoch(math.MaxUint32), params.KnownConfigs[params.MainnetName]().BellatrixForkEpoch)
	genv := bytesutil.ToBytes4(bc.GenesisForkVersion)
	altairv := bytesutil.ToBytes4(bc.AltairForkVersion)
	bellav := bytesutil.ToBytes4(bc.BellatrixForkVersion)
	altairS, err := slots.EpochStart(bc.AltairForkEpoch)
	bellaS, err := slots.EpochStart(bc.BellatrixForkEpoch)
	require.NoError(t, err)
	cases := []struct {
		b       func(*testing.T, types.Slot) interfaces.SignedBeaconBlock
		name    string
		version [4]byte
		slot    types.Slot
		err     error
	}{
		{
			name:    "genesis - slot 0",
			b:       signedTestBlockGenesis,
			version: genv,
		},
		{
			name:    "last slot of phase 0",
			b:       signedTestBlockGenesis,
			version: genv,
			slot:    altairS - 1,
		},
		{
			name:    "first slot of altair",
			b:       signedTestBlockAltair,
			version: altairv,
			slot:    altairS,
		},
		{
			name:    "last slot of altair",
			b:       signedTestBlockAltair,
			version: altairv,
			slot:    bellaS - 1,
		},
		{
			name:    "first slot of bellatrix",
			b:       signedTestBlockBellatrix,
			version: bellav,
			slot:    bellaS,
		},
		{
			name:    "bellatrix block in altair slot",
			b:       signedTestBlockBellatrix,
			version: bellav,
			slot:    bellaS - 1,
			err:     errBlockForkMismatch,
		},
		{
			name:    "genesis block in altair slot",
			b:       signedTestBlockGenesis,
			version: genv,
			slot:    bellaS - 1,
			err:     errBlockForkMismatch,
		},
		{
			name:    "altair block in genesis slot",
			b:       signedTestBlockAltair,
			version: altairv,
			err:     errBlockForkMismatch,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := c.b(t, c.slot)
			marshaled, err := b.MarshalSSZ()
			require.NoError(t, err)
			cf, err := FromForkVersion(c.version)
			require.NoError(t, err)
			bcf, err := cf.UnmarshalBeaconBlock(marshaled)
			if c.err != nil {
				require.ErrorIs(t, err, c.err)
				return
			}
			require.NoError(t, err)
			expected, err := b.Block().HashTreeRoot()
			require.NoError(t, err)
			actual, err := bcf.Block().HashTreeRoot()
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	}
}

func TestUnmarshalBlindedBlock(t *testing.T) {
	bc, cleanup := hackBellatrixMaxuint()
	defer cleanup()
	require.Equal(t, types.Epoch(math.MaxUint32), params.KnownConfigs[params.MainnetName]().BellatrixForkEpoch)
	genv := bytesutil.ToBytes4(bc.GenesisForkVersion)
	altairv := bytesutil.ToBytes4(bc.AltairForkVersion)
	bellav := bytesutil.ToBytes4(bc.BellatrixForkVersion)
	altairS, err := slots.EpochStart(bc.AltairForkEpoch)
	bellaS, err := slots.EpochStart(bc.BellatrixForkEpoch)
	require.NoError(t, err)
	cases := []struct {
		b       func(*testing.T, types.Slot) interfaces.SignedBeaconBlock
		name    string
		version [4]byte
		slot    types.Slot
		err     error
	}{
		{
			name:    "genesis - slot 0",
			b:       signedTestBlockGenesis,
			version: genv,
		},
		{
			name:    "last slot of phase 0",
			b:       signedTestBlockGenesis,
			version: genv,
			slot:    altairS - 1,
		},
		{
			name:    "first slot of altair",
			b:       signedTestBlockAltair,
			version: altairv,
			slot:    altairS,
		},
		{
			name:    "last slot of altair",
			b:       signedTestBlockAltair,
			version: altairv,
			slot:    bellaS - 1,
		},
		{
			name:    "first slot of bellatrix",
			b:       signedTestBlindedBlockBellatrix,
			version: bellav,
			slot:    bellaS,
		},
		{
			name:    "bellatrix block in altair slot",
			b:       signedTestBlindedBlockBellatrix,
			version: bellav,
			slot:    bellaS - 1,
			err:     errBlockForkMismatch,
		},
		{
			name:    "genesis block in altair slot",
			b:       signedTestBlockGenesis,
			version: genv,
			slot:    bellaS - 1,
			err:     errBlockForkMismatch,
		},
		{
			name:    "altair block in genesis slot",
			b:       signedTestBlockAltair,
			version: altairv,
			err:     errBlockForkMismatch,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := c.b(t, c.slot)
			marshaled, err := b.MarshalSSZ()
			require.NoError(t, err)
			cf, err := FromForkVersion(c.version)
			require.NoError(t, err)
			bcf, err := cf.UnmarshalBlindedBeaconBlock(marshaled)
			if c.err != nil {
				require.ErrorIs(t, err, c.err)
				return
			}
			require.NoError(t, err)
			expected, err := b.Block().HashTreeRoot()
			require.NoError(t, err)
			actual, err := bcf.Block().HashTreeRoot()
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	}
}

func signedTestBlockGenesis(t *testing.T, slot types.Slot) interfaces.SignedBeaconBlock {
	b := util.NewBeaconBlock()
	b.Block.Slot = slot
	s, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	return s
}

func signedTestBlockAltair(t *testing.T, slot types.Slot) interfaces.SignedBeaconBlock {
	b := util.NewBeaconBlockAltair()
	b.Block.Slot = slot
	s, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	return s
}

func signedTestBlockBellatrix(t *testing.T, slot types.Slot) interfaces.SignedBeaconBlock {
	b := util.NewBeaconBlockBellatrix()
	b.Block.Slot = slot
	s, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	return s
}

func signedTestBlindedBlockBellatrix(t *testing.T, slot types.Slot) interfaces.SignedBeaconBlock {
	b := util.NewBlindedBeaconBlockBellatrix()
	b.Block.Slot = slot
	s, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	return s
}
