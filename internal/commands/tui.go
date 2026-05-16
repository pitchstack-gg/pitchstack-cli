package commands

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/pitchstack-gg/pitchstack-cli/internal/cardsdb"
	"github.com/pitchstack-gg/pitchstack-cli/internal/paths"
	"github.com/pitchstack-gg/pitchstack-cli/internal/powersync"
	cardstui "github.com/pitchstack-gg/pitchstack-cli/internal/tui/cards"
	deckstui "github.com/pitchstack-gg/pitchstack-cli/internal/tui/decks"
	"github.com/pitchstack-gg/pitchstack-cli/internal/tui/shell"
	"github.com/urfave/cli/v3"
)

func newTUICommand() *cli.Command {
	return &cli.Command{
		Name:  "tui",
		Usage: "Open the interactive TUI",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "refresh", Usage: "Force refresh the local card database before opening"},
			&cli.BoolFlag{Name: "offline", Usage: "Use only the cached local card database"},
			&cli.StringFlag{Name: "tab", Usage: "Initial tab (cards|decks)", Value: "cards"},
			&cli.StringFlag{Name: "cards-db-url", Usage: "Remote gzipped SQLite card database URL"},
			&cli.StringFlag{Name: "cards-db-last-updated-url", Usage: "Remote LAST_PUBLISHED freshness URL"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			initialTab, err := shell.ParseTab(cmd.String("tab"))
			if err != nil {
				return cli.Exit(err.Error(), 2)
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
			var decksCardsRepo *cardsdb.Repository
			if _, err := os.Stat(paths.CardsDBPath(st.ProfileName)); err == nil {
				if repo, err := cardsdb.OpenRepository(paths.CardsDBPath(st.ProfileName)); err == nil {
					decksCardsRepo = repo
					defer repo.Close()
				}
			}
			var pricingClient cardstui.PricingClient
			if !cmd.Bool("offline") {
				pricingClient, _ = st.Service.AuthenticatedClient()
			}
			cardsModel := cardstui.New(cardstui.Options{
				Manager:       manager,
				ImageCacheDir: paths.CardsImageCacheDir(st.ProfileName),
				PricingClient: pricingClient,
				ForceRefresh:  cmd.Bool("refresh"),
				Offline:       cmd.Bool("offline"),
			})
			syncPath := paths.SyncDBPath(st.ProfileName)
			_, statErr := os.Stat(syncPath)
			missingSyncDB := os.IsNotExist(statErr)
			syncStore, err := powersync.OpenStore(syncPath)
			if err != nil {
				return err
			}
			defer syncStore.Close()
			viewerUserID := ""
			viewerName := ""
			autoSync := false
			if sess, err := st.Sessions.Load(); err == nil && sess != nil {
				viewerUserID = strings.TrimSpace(sess.UserID)
				viewerName = strings.TrimSpace(sess.Username)
				autoSync = hasUsableLocalCredentials(sess.AccessToken, sess.RefreshToken, sess.AccessTokenExpiresAt)
			}
			var syncClient *powersync.Client
			var profileClient deckstui.ProfileClient
			if autoSync {
				apiClient, err := st.Service.AuthenticatedClient()
				if err != nil {
					return err
				}
				if !cmd.Bool("offline") {
					profileClient = apiClient
				}
				syncClient = &powersync.Client{
					Store: syncStore,
					Connector: &powersync.APIConnector{
						Client:           apiClient,
						EndpointOverride: strings.TrimSpace(st.Profile.PowerSyncURL),
						TokenProvider:    st.Service.BearerToken,
					},
					HTTPClient: &http.Client{},
				}
			}
			decksModel := deckstui.New(deckstui.Options{
				Store:         syncStore,
				Client:        syncClient,
				ProfileClient: profileClient,
				ViewerUserID:  viewerUserID,
				ViewerName:    viewerName,
				MissingDB:     missingSyncDB,
				AutoSync:      autoSync,
				InitialScope:  powersync.DeckListScopeAccessible,
				Limit:         500,
				CardsRepo:     decksCardsRepo,
				ImageCacheDir: paths.CardsImageCacheDir(st.ProfileName),
			})
			model := shell.New(shell.Options{
				Cards:      cardsModel,
				Decks:      decksModel,
				InitialTab: initialTab,
			})
			_, err = tea.NewProgram(model).Run()
			return err
		},
	}
}

func hasUsableLocalCredentials(accessToken, refreshToken string, accessTokenExpiresAt time.Time) bool {
	if strings.TrimSpace(refreshToken) != "" {
		return true
	}
	return strings.TrimSpace(accessToken) != "" && (accessTokenExpiresAt.IsZero() || time.Until(accessTokenExpiresAt) > time.Minute)
}
