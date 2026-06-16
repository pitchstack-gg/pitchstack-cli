package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pitchstack-gg/pitchstack-cli/internal/cardsdb"
	"github.com/pitchstack-gg/pitchstack-cli/internal/paths"
	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newCardsCommand() *cli.Command {
	return &cli.Command{
		Name:  "cards",
		Usage: "Search cards and metadata",
		Commands: []*cli.Command{
			newCardsSearchCommand(),
			newCardsGetCommand(),
			newCardsBatchGetCommand(),
			newCardsPrintingsCommand(),
			newCardsPrintingCommand(),
			newCardsPrintingsBatchCommand(),
			newCardsPrintingsSetCommand(),
			newCardsProductCommand(),
			newCardsProductsCommand(),
			newCardsProductsBatchCommand(),
			newCardsSetCommand(),
			newCardsSetsCommand(),
			newCardsSetsBatchCommand(),
			newCardsSnapshotCommand(),
			newCardsPricesCommand(),
			newResourceTrendingCommand("cards", clientv1.TrackableResourceTypeCard),
		},
	}
}

func localCardsFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "refresh", Usage: "Force refresh the local card database before running"},
		&cli.BoolFlag{Name: "offline", Usage: "Use only the cached local card database"},
		&cli.StringFlag{Name: "cards-db-url", Usage: "Remote gzipped SQLite card database URL"},
		&cli.StringFlag{Name: "cards-db-last-updated-url", Usage: "Remote LAST_PUBLISHED freshness URL"},
	}
}

func appendLocalCardsFlags(flags ...cli.Flag) []cli.Flag {
	out := make([]cli.Flag, 0, len(localCardsFlags())+len(flags))
	out = append(out, localCardsFlags()...)
	out = append(out, flags...)
	return out
}

func withLocalCardsRepo(ctx context.Context, cmd *cli.Command, fn func(*cardsdb.Repository, string, *cardsdb.Metadata) (any, error)) error {
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
	refreshInterval := cardsdb.DefaultRefreshInterval
	if raw := strings.TrimSpace(st.Profile.CardsDBRefreshInterval); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return cli.Exit(fmt.Sprintf("cardsDbRefreshInterval must be a duration: %s", err.Error()), 2)
		}
		refreshInterval = parsed
	}
	autoRefresh := true
	if st.Profile.CardsDBAutoRefresh != nil {
		autoRefresh = *st.Profile.CardsDBAutoRefresh
	}

	manager := &cardsdb.Manager{
		DBPath:         paths.CardsDBPath(st.ProfileName),
		MetaPath:       paths.CardsDBMetaPath(st.ProfileName),
		DBURL:          dbURL,
		LastUpdatedURL: lastUpdatedURL,
		OnStatus: func(status cardsdb.Status) {
			switch status.Phase {
			case "checking", "downloading", "installing", "outdated":
				_, _ = fmt.Fprintln(cmd.ErrWriter, status.Message)
			}
		},
	}
	ensure, err := manager.Ensure(ctx, cardsdb.EnsureOptions{
		Force:           cmd.Bool("refresh"),
		Offline:         cmd.Bool("offline"),
		AutoRefresh:     &autoRefresh,
		RefreshInterval: refreshInterval,
	})
	if err != nil {
		return err
	}
	if ensure.Outdated {
		_, _ = fmt.Fprintf(cmd.ErrWriter, "warning: local card database is out of date; run this command with --refresh to update it\n")
	}
	repo, err := cardsdb.OpenRepository(ensure.DBPath)
	if err != nil {
		return err
	}
	defer repo.Close()
	resp, err := fn(repo, ensure.DBPath, ensure.Meta)
	if err != nil {
		return err
	}
	return writeJSON(cmd.Writer, resp)
}

func newCardsSearchCommand() *cli.Command {
	return &cli.Command{
		Name:      "search",
		Usage:     "Search cards",
		ArgsUsage: "[query]",
		Flags: appendLocalCardsFlags(
			&cli.StringFlag{Name: "q", Aliases: []string{"query"}, Usage: "Search query using Pitchstack card syntax"},
			&cli.StringFlag{Name: "class", Aliases: []string{"c"}, Usage: "Deprecated: use class:<value> in --q"},
			&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Usage: "Deprecated: use type:<value> in --q"},
			&cli.StringFlag{Name: "subtype", Aliases: []string{"st"}, Usage: "Deprecated: use subtype:<value> in --q"},
			&cli.StringFlag{Name: "talent", Aliases: []string{"tal"}, Usage: "Deprecated: use talent:<value> in --q"},
			&cli.StringFlag{Name: "keyword", Aliases: []string{"kw"}, Usage: "Deprecated: use keyword:<value> in --q"},
			&cli.StringFlag{Name: "artist", Aliases: []string{"art"}, Usage: "Deprecated: use artist:<value> in --q"},
			&cli.StringFlag{Name: "set", Aliases: []string{"s"}, Usage: "Deprecated: use set:<value> in --q"},
			&cli.StringFlag{Name: "rarity", Aliases: []string{"r"}, Usage: "Deprecated: use rarity:<value> in --q"},
			&cli.StringFlag{Name: "language", Aliases: []string{"lang"}, Usage: "Deprecated: use lang:<value> in --q"},
			&cli.StringFlag{Name: "cost", Aliases: []string{"co"}, Usage: "Deprecated: use cost:<value> in --q"},
			&cli.StringFlag{Name: "defense", Aliases: []string{"def", "d", "block", "b"}, Usage: "Deprecated: use defense:<value> in --q"},
			&cli.StringFlag{Name: "pitch", Aliases: []string{"p"}, Usage: "Deprecated: use pitch:<value> in --q"},
			&cli.StringFlag{Name: "power", Aliases: []string{"pow", "pwr", "attack"}, Usage: "Deprecated: use power:<value> in --q"},
			&cli.StringFlag{Name: "health", Aliases: []string{"life", "li", "hp"}, Usage: "Deprecated: use life:<value> in --q"},
			&cli.StringFlag{Name: "intelligence", Aliases: []string{"intellect", "i"}, Usage: "Deprecated: use intellect:<value> in --q"},
			&cli.StringFlag{Name: "arcane", Usage: "Deprecated: use arcane:<value> in --q"},
			&cli.StringFlag{Name: "color-identity", Aliases: []string{"color", "colour"}, Usage: "Deprecated: use color:<red|yellow|blue|none> in --q"},
			&cli.BoolFlag{Name: "is-double-faced", Aliases: []string{"double", "double-faced"}, Usage: "Deprecated: use double:<true|false> in --q"},
			&cli.BoolFlag{Name: "blitz-legal", Usage: "Deprecated: use legal:blitz in --q"},
			&cli.BoolFlag{Name: "blitz-banned", Usage: "Deprecated: use banned:blitz in --q"},
			&cli.BoolFlag{Name: "blitz-suspended", Usage: "Deprecated: use suspended:blitz in --q"},
			&cli.BoolFlag{Name: "blitz-living-legend", Usage: "Deprecated: use livinglegend:blitz in --q"},
			&cli.BoolFlag{Name: "cc-legal", Usage: "Deprecated: use legal:cc in --q"},
			&cli.BoolFlag{Name: "cc-banned", Usage: "Deprecated: use banned:cc in --q"},
			&cli.BoolFlag{Name: "cc-suspended", Usage: "Deprecated: use suspended:cc in --q"},
			&cli.BoolFlag{Name: "cc-living-legend", Usage: "Deprecated: use livinglegend:cc in --q"},
			&cli.BoolFlag{Name: "commoner-legal", Usage: "Deprecated: use legal:commoner in --q"},
			&cli.BoolFlag{Name: "commoner-banned", Usage: "Deprecated: use banned:commoner in --q"},
			&cli.BoolFlag{Name: "commoner-suspended", Usage: "Deprecated: use suspended:commoner in --q"},
			&cli.BoolFlag{Name: "upf-banned", Usage: "Deprecated: use banned:upf in --q"},
			&cli.BoolFlag{Name: "ll-banned", Usage: "Deprecated: use banned:ll in --q"},
			&cli.BoolFlag{Name: "ll-restricted", Usage: "Deprecated: use restricted:ll in --q"},
			&cli.BoolFlag{Name: "project-blue-legal", Usage: "Deprecated: use legal:projectblue in --q"},
			&cli.BoolFlag{Name: "project-blue-banned", Usage: "Deprecated: use banned:projectblue in --q"},
			&cli.BoolFlag{Name: "project-blue-suspended", Usage: "Deprecated: use suspended:projectblue in --q"},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			query := strings.TrimSpace(cmd.String("q"))
			if query == "" {
				query = strings.TrimSpace(cmd.Args().First())
			}
			params := cardsdb.ParseCardSearchQuery(query)
			applySearchFlagOverrides(cmd, &params)
			if !hasCardSearchCriteria(params) {
				return fmt.Errorf("cards search requires a query or at least one filter")
			}
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, _ string, _ *cardsdb.Metadata) (any, error) {
				return repo.SearchCards(ctx, params)
			})
		},
	}
}

func hasCardSearchCriteria(params cardsdb.SearchCardsParams) bool {
	if strings.TrimSpace(params.SearchTerm) != "" ||
		strings.TrimSpace(params.Class) != "" ||
		strings.TrimSpace(params.Type) != "" ||
		strings.TrimSpace(params.Subtype) != "" ||
		strings.TrimSpace(params.Talent) != "" ||
		strings.TrimSpace(params.Keyword) != "" ||
		strings.TrimSpace(params.Cost) != "" ||
		strings.TrimSpace(params.Defense) != "" ||
		strings.TrimSpace(params.Pitch) != "" ||
		strings.TrimSpace(params.Power) != "" ||
		strings.TrimSpace(params.Health) != "" ||
		strings.TrimSpace(params.Intelligence) != "" ||
		strings.TrimSpace(params.Arcane) != "" ||
		strings.TrimSpace(params.ColorIdentity) != "" ||
		strings.TrimSpace(params.Artist) != "" ||
		strings.TrimSpace(params.SetCode) != "" ||
		strings.TrimSpace(params.Rarity) != "" ||
		strings.TrimSpace(params.Language) != "" {
		return true
	}
	return params.BlitzLegal != nil ||
		params.BlitzBanned != nil ||
		params.BlitzSuspended != nil ||
		params.BlitzLivingLegend != nil ||
		params.CCLegal != nil ||
		params.CCBanned != nil ||
		params.CCSuspended != nil ||
		params.CCLivingLegend != nil ||
		params.CommonerLegal != nil ||
		params.CommonerBanned != nil ||
		params.CommonerSuspended != nil ||
		params.UPFBanned != nil ||
		params.LLBanned != nil ||
		params.LLRestricted != nil ||
		params.ProjectBlueLegal != nil ||
		params.ProjectBlueBanned != nil ||
		params.ProjectBlueSuspended != nil ||
		params.IsDoubleFaced != nil
}

func applySearchFlagOverrides(cmd *cli.Command, params *cardsdb.SearchCardsParams) {
	setString := func(flag string, dst *string) {
		if cmd.IsSet(flag) {
			*dst = strings.TrimSpace(cmd.String(flag))
		}
	}
	setString("class", &params.Class)
	setString("type", &params.Type)
	setString("subtype", &params.Subtype)
	setString("talent", &params.Talent)
	setString("keyword", &params.Keyword)
	setString("artist", &params.Artist)
	setString("set", &params.SetCode)
	setString("rarity", &params.Rarity)
	setString("language", &params.Language)
	setString("cost", &params.Cost)
	setString("defense", &params.Defense)
	setString("pitch", &params.Pitch)
	setString("power", &params.Power)
	setString("health", &params.Health)
	setString("intelligence", &params.Intelligence)
	setString("arcane", &params.Arcane)
	setString("color-identity", &params.ColorIdentity)
	if cmd.IsSet("page-size") {
		params.PageSize = cmd.Int("page-size")
	}
	if cmd.IsSet("next-token") {
		params.NextToken = strings.TrimSpace(cmd.String("next-token"))
	}
	params.BlitzLegal = boolPtr(cmd, "blitz-legal")
	params.BlitzBanned = boolPtr(cmd, "blitz-banned")
	params.BlitzSuspended = boolPtr(cmd, "blitz-suspended")
	params.BlitzLivingLegend = boolPtr(cmd, "blitz-living-legend")
	params.CCLegal = boolPtr(cmd, "cc-legal")
	params.CCBanned = boolPtr(cmd, "cc-banned")
	params.CCSuspended = boolPtr(cmd, "cc-suspended")
	params.CCLivingLegend = boolPtr(cmd, "cc-living-legend")
	params.CommonerLegal = boolPtr(cmd, "commoner-legal")
	params.CommonerBanned = boolPtr(cmd, "commoner-banned")
	params.CommonerSuspended = boolPtr(cmd, "commoner-suspended")
	params.UPFBanned = boolPtr(cmd, "upf-banned")
	params.LLBanned = boolPtr(cmd, "ll-banned")
	params.LLRestricted = boolPtr(cmd, "ll-restricted")
	params.ProjectBlueLegal = boolPtr(cmd, "project-blue-legal")
	params.ProjectBlueBanned = boolPtr(cmd, "project-blue-banned")
	params.ProjectBlueSuspended = boolPtr(cmd, "project-blue-suspended")
	params.IsDoubleFaced = boolPtr(cmd, "is-double-faced")
}

func newCardsBatchGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "batch-get",
		Usage: "Get cards in bulk",
		Flags: appendLocalCardsFlags(
			repeatedIDsFlag("id", "Card ID (repeatable or comma-separated)"),
			&cli.BoolFlag{Name: "allow-partial", Usage: "Allow partial results"},
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			ids := splitCSV(cmd.StringSlice("id"))
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, _ string, _ *cardsdb.Metadata) (any, error) {
				return repo.BatchGetCardSummaries(ctx, ids)
			})
		},
	}
}

func newCardsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a card",
		Flags: appendLocalCardsFlags(&cli.StringFlag{Name: "id", Usage: "Card ID", Required: true}),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, _ string, _ *cardsdb.Metadata) (any, error) {
				return repo.GetCardSummary(ctx, strings.TrimSpace(cmd.String("id")))
			})
		},
	}
}

func newCardsPrintingsCommand() *cli.Command {
	return &cli.Command{
		Name:  "printings",
		Usage: "List printings for a card",
		Flags: appendLocalCardsFlags(
			&cli.StringFlag{Name: "card-id", Usage: "Card ID", Required: true},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, _ string, _ *cardsdb.Metadata) (any, error) {
				return repo.ListPrintingSummaries(ctx, strings.TrimSpace(cmd.String("card-id")), cmd.Int("page-size"), strings.TrimSpace(cmd.String("next-token")))
			})
		},
	}
}

func newCardsPrintingCommand() *cli.Command {
	return &cli.Command{
		Name:  "printing",
		Usage: "Get a printing",
		Flags: appendLocalCardsFlags(&cli.StringFlag{Name: "id", Usage: "Printing ID", Required: true}),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, _ string, _ *cardsdb.Metadata) (any, error) {
				return repo.GetPrintingSummary(ctx, strings.TrimSpace(cmd.String("id")))
			})
		},
	}
}

func newCardsPrintingsBatchCommand() *cli.Command {
	return &cli.Command{
		Name:  "printings-batch",
		Usage: "Get printings in bulk",
		Flags: appendLocalCardsFlags(
			repeatedIDsFlag("id", "Printing ID (repeatable or comma-separated)"),
			&cli.BoolFlag{Name: "allow-partial", Usage: "Allow partial results"},
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			ids := splitCSV(cmd.StringSlice("id"))
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, _ string, _ *cardsdb.Metadata) (any, error) {
				return repo.BatchGetPrintingSummaries(ctx, ids)
			})
		},
	}
}

func newCardsPrintingsSetCommand() *cli.Command {
	return &cli.Command{
		Name:  "printings-set",
		Usage: "List printings for a set number",
		Flags: appendLocalCardsFlags(&cli.StringFlag{Name: "set-number", Usage: "Set number", Required: true}),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, _ string, _ *cardsdb.Metadata) (any, error) {
				return repo.ListPrintingsForSetNumber(ctx, strings.TrimSpace(cmd.String("set-number")))
			})
		},
	}
}

func newCardsProductsCommand() *cli.Command {
	return &cli.Command{
		Name:  "products",
		Usage: "List products",
		Flags: appendLocalCardsFlags(
			&cli.StringFlag{Name: "type", Usage: "Product type"},
			&cli.StringFlag{Name: "set-code", Usage: "Set code"},
			&cli.StringFlag{Name: "product-group-id", Usage: "Product group ID"},
			&cli.StringFlag{Name: "card-id", Usage: "Card ID"},
			&cli.StringFlag{Name: "printing-id", Usage: "Printing ID"},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			params := cardsdb.ListProductsParams{
				Type:           strings.TrimSpace(cmd.String("type")),
				SetCode:        strings.TrimSpace(cmd.String("set-code")),
				ProductGroupID: strings.TrimSpace(cmd.String("product-group-id")),
				CardID:         strings.TrimSpace(cmd.String("card-id")),
				PrintingID:     strings.TrimSpace(cmd.String("printing-id")),
				PageSize:       cmd.Int("page-size"),
				NextToken:      strings.TrimSpace(cmd.String("next-token")),
			}
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, _ string, _ *cardsdb.Metadata) (any, error) {
				return repo.ListProductSummaries(ctx, params)
			})
		},
	}
}

func newCardsProductCommand() *cli.Command {
	return &cli.Command{
		Name:  "product",
		Usage: "Get a product",
		Flags: appendLocalCardsFlags(&cli.StringFlag{Name: "id", Usage: "Product ID", Required: true}),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, _ string, _ *cardsdb.Metadata) (any, error) {
				return repo.GetProductSummary(ctx, strings.TrimSpace(cmd.String("id")))
			})
		},
	}
}

func newCardsProductsBatchCommand() *cli.Command {
	return &cli.Command{
		Name:  "products-batch",
		Usage: "Get products in bulk",
		Flags: appendLocalCardsFlags(
			repeatedIDsFlag("id", "Product ID (repeatable or comma-separated)"),
			&cli.BoolFlag{Name: "allow-partial", Usage: "Allow partial results"},
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			ids := splitCSV(cmd.StringSlice("id"))
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, _ string, _ *cardsdb.Metadata) (any, error) {
				return repo.BatchGetProductSummaries(ctx, ids)
			})
		},
	}
}

func newCardsSetCommand() *cli.Command {
	return &cli.Command{
		Name:  "set",
		Usage: "Get a set",
		Flags: appendLocalCardsFlags(&cli.StringFlag{Name: "code", Usage: "Set code", Required: true}),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, _ string, _ *cardsdb.Metadata) (any, error) {
				return repo.GetSetSummary(ctx, strings.TrimSpace(cmd.String("code")))
			})
		},
	}
}

func newCardsSetsCommand() *cli.Command {
	return &cli.Command{
		Name:  "sets",
		Usage: "List sets",
		Flags: appendLocalCardsFlags(
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, _ string, _ *cardsdb.Metadata) (any, error) {
				return repo.ListSetSummaries(ctx, cmd.Int("page-size"), strings.TrimSpace(cmd.String("next-token")))
			})
		},
	}
}

func newCardsSetsBatchCommand() *cli.Command {
	return &cli.Command{
		Name:  "sets-batch",
		Usage: "Get sets in bulk",
		Flags: appendLocalCardsFlags(
			repeatedIDsFlag("code", "Set code (repeatable or comma-separated)"),
			&cli.BoolFlag{Name: "allow-partial", Usage: "Allow partial results"},
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			codes := splitCSV(cmd.StringSlice("code"))
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, _ string, _ *cardsdb.Metadata) (any, error) {
				return repo.BatchGetSetSummaries(ctx, codes)
			})
		},
	}
}

func newCardsSnapshotCommand() *cli.Command {
	return &cli.Command{
		Name:  "snapshot",
		Usage: "Show local card database metadata",
		Flags: appendLocalCardsFlags(
			&cli.IntFlag{Name: "schema-version", Usage: "Ignored compatibility flag"},
			&cli.StringFlag{Name: "version", Usage: "Ignored compatibility flag"},
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withLocalCardsRepo(ctx, cmd, func(repo *cardsdb.Repository, dbPath string, meta *cardsdb.Metadata) (any, error) {
				return repo.Snapshot(dbPath, meta), nil
			})
		},
	}
}
