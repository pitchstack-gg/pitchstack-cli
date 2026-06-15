package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/pitchstack-gg/pitchstack-cli/internal/paths"
	"github.com/pitchstack-gg/pitchstack-cli/internal/powersync"
)

func openLocalPowerSyncStore(ctx context.Context) (*powersync.Store, error) {
	st, err := getState(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(st.ProfileName) == "" {
		return nil, errors.New("missing profile name")
	}
	return powersync.OpenStore(paths.SyncDBPath(st.ProfileName))
}
