package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/pitchstack-gg/pitchstack-cli/internal/config"
	"github.com/pitchstack-gg/pitchstack-cli/internal/paths"
	"github.com/pitchstack-gg/pitchstack-cli/internal/pitchstack"
	"github.com/pitchstack-gg/pitchstack-cli/internal/session"

	"github.com/urfave/cli/v3"
)

type state struct {
	ConfigPath  string
	ProfileName string
	Config      *config.Config
	Profile     config.Profile
	Sessions    *session.Store
	Service     *pitchstack.Service
}

type stateKey struct{}

func loadState(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	cfgPath := cmd.String("config")
	profileName := cmd.String("profile")

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return ctx, cli.Exit(fmt.Sprintf("load config: %s", err.Error()), 1)
	}

	if profileName == "" {
		profileName = cfg.CurrentProfile
	}
	prof, ok := cfg.Profile(profileName)
	if !ok {
		return ctx, cli.Exit(fmt.Sprintf("unknown profile %q", profileName), 2)
	}

	store := session.NewStore(paths.SessionPath(profileName))
	if cur, err := store.Load(); err == nil && cur == nil {
		legacyStore := session.NewStore(paths.DefaultSessionPath())
		if legacySess, legacyErr := legacyStore.Load(); legacyErr == nil && legacySess != nil {
			if strings.TrimSpace(legacySess.BaseURL) == strings.TrimSpace(prof.BaseURL) {
				_ = store.Save(legacySess)
			}
		}
	}
	svc := pitchstack.NewService(pitchstack.ServiceDeps{
		BaseURL:        prof.BaseURL,
		TimeoutSeconds: prof.TimeoutSeconds,
		Sessions:       store,
	})

	st := &state{
		ConfigPath:  cfgPath,
		ProfileName: profileName,
		Config:      cfg,
		Profile:     prof,
		Sessions:    store,
		Service:     svc,
	}
	return context.WithValue(ctx, stateKey{}, st), nil
}

func getState(ctx context.Context) (*state, error) {
	st, _ := ctx.Value(stateKey{}).(*state)
	if st == nil {
		return nil, fmt.Errorf("internal error: missing app state")
	}
	return st, nil
}
