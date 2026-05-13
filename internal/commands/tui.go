package commands

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/pitchstack-gg/pitchstack-cli/internal/cardsdb"
	"github.com/pitchstack-gg/pitchstack-cli/internal/paths"
	cardstui "github.com/pitchstack-gg/pitchstack-cli/internal/tui/cards"
	"github.com/urfave/cli/v3"
)

func newTUICommand() *cli.Command {
	return &cli.Command{
		Name:  "tui",
		Usage: "Open the interactive card browser",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "refresh", Usage: "Force refresh the local card database before opening"},
			&cli.BoolFlag{Name: "offline", Usage: "Use only the cached local card database"},
			&cli.StringFlag{Name: "cards-db-url", Usage: "Remote gzipped SQLite card database URL"},
			&cli.StringFlag{Name: "cards-db-last-updated-url", Usage: "Remote LAST_PUBLISHED freshness URL"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			dbURL := strings.TrimSpace(cmd.String("cards-db-url"))
			if dbURL == "" {
				dbURL = strings.TrimSpace(st.Profile.CardsDBURL)
			}
			if dbURL == "" {
				dbURL = cardsdb.DefaultCardsDBURL
			}
			lastUpdatedURL := strings.TrimSpace(cmd.String("cards-db-last-updated-url"))
			if lastUpdatedURL == "" {
				lastUpdatedURL = strings.TrimSpace(st.Profile.CardsDBLastUpdatedURL)
			}
			if lastUpdatedURL == "" {
				lastUpdatedURL = cardsdb.DefaultCardsDBLastUpdatedURL
			}

			manager := &cardsdb.Manager{
				DBPath:         paths.CardsDBPath(st.ProfileName),
				MetaPath:       paths.CardsDBMetaPath(st.ProfileName),
				DBURL:          dbURL,
				LastUpdatedURL: lastUpdatedURL,
			}
			var pricingClient cardstui.PricingClient
			if !cmd.Bool("offline") {
				pricingClient, _ = st.Service.AuthenticatedClient()
			}
			model := cardstui.New(cardstui.Options{
				Manager:       manager,
				ImageCacheDir: paths.CardsImageCacheDir(st.ProfileName),
				PricingClient: pricingClient,
				ForceRefresh:  cmd.Bool("refresh"),
				Offline:       cmd.Bool("offline"),
			})
			_, err = tea.NewProgram(model).Run()
			return err
		},
	}
}
