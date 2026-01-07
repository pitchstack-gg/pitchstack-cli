package commands

import (
	"context"
	"strings"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"

	"github.com/urfave/cli/v3"
)

func newCardsCommand() *cli.Command {
	return &cli.Command{
		Name:  "cards",
		Usage: "Card search and metadata",
		Commands: []*cli.Command{
			newCardsSearchCommand(),
			newCardsGetCommand(),
			newCardsPrintingsCommand(),
			newCardsPrintingCommand(),
			newCardsPrintingsSetCommand(),
			newCardsProductCommand(),
			newCardsSnapshotCommand(),
		},
	}
}

func newCardsSearchCommand() *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "Search cards",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "q", Usage: "Search term"},
			&cli.StringFlag{Name: "class", Usage: "Class filter"},
			&cli.StringFlag{Name: "type", Usage: "Type filter"},
			&cli.StringFlag{Name: "subtype", Usage: "Subtype filter"},
			&cli.StringFlag{Name: "talent", Usage: "Talent filter"},
			&cli.StringFlag{Name: "cost", Usage: "Cost filter"},
			&cli.StringFlag{Name: "defense", Usage: "Defense filter"},
			&cli.StringFlag{Name: "pitch", Usage: "Pitch filter"},
			&cli.StringFlag{Name: "power", Usage: "Power filter"},
			&cli.StringFlag{Name: "health", Usage: "Health filter"},
			&cli.StringFlag{Name: "intelligence", Usage: "Intelligence filter"},
			&cli.StringFlag{Name: "arcane", Usage: "Arcane filter"},
			&cli.StringFlag{Name: "color-identity", Usage: "Color identity filter (e.g. COLOR_IDENTITY_RED)"},
			&cli.BoolFlag{Name: "is-double-faced", Usage: "Filter by double-faced cards"},

			&cli.BoolFlag{Name: "blitz-legal", Usage: "Blitz legality filter"},
			&cli.BoolFlag{Name: "blitz-banned", Usage: "Blitz banned filter"},
			&cli.BoolFlag{Name: "blitz-suspended", Usage: "Blitz suspended filter"},
			&cli.BoolFlag{Name: "blitz-living-legend", Usage: "Blitz living legend filter"},
			&cli.BoolFlag{Name: "cc-legal", Usage: "Classic Constructed legality filter"},
			&cli.BoolFlag{Name: "cc-banned", Usage: "Classic Constructed banned filter"},
			&cli.BoolFlag{Name: "cc-suspended", Usage: "Classic Constructed suspended filter"},
			&cli.BoolFlag{Name: "cc-living-legend", Usage: "Classic Constructed living legend filter"},
			&cli.BoolFlag{Name: "commoner-legal", Usage: "Commoner legality filter"},
			&cli.BoolFlag{Name: "commoner-banned", Usage: "Commoner banned filter"},
			&cli.BoolFlag{Name: "commoner-suspended", Usage: "Commoner suspended filter"},
			&cli.BoolFlag{Name: "upf-banned", Usage: "UPF banned filter"},
			&cli.BoolFlag{Name: "ll-banned", Usage: "Living Legend format banned filter"},
			&cli.BoolFlag{Name: "ll-restricted", Usage: "Living Legend format restricted filter"},
			&cli.BoolFlag{Name: "project-blue-legal", Usage: "Project Blue legality filter"},
			&cli.BoolFlag{Name: "project-blue-banned", Usage: "Project Blue banned filter"},
			&cli.BoolFlag{Name: "project-blue-suspended", Usage: "Project Blue suspended filter"},

			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.SearchCardsRequest{
				SearchTerm:           strings.TrimSpace(cmd.String("q")),
				Class:                strings.TrimSpace(cmd.String("class")),
				Type:                 strings.TrimSpace(cmd.String("type")),
				Subtype:              strings.TrimSpace(cmd.String("subtype")),
				Talent:               strings.TrimSpace(cmd.String("talent")),
				Cost:                 strings.TrimSpace(cmd.String("cost")),
				Defense:              strings.TrimSpace(cmd.String("defense")),
				Pitch:                strings.TrimSpace(cmd.String("pitch")),
				Power:                strings.TrimSpace(cmd.String("power")),
				Health:               strings.TrimSpace(cmd.String("health")),
				Intelligence:         strings.TrimSpace(cmd.String("intelligence")),
				Arcane:               strings.TrimSpace(cmd.String("arcane")),
				ColorIdentity:        strings.TrimSpace(cmd.String("color-identity")),
				BlitzLegal:           boolPtr(cmd, "blitz-legal"),
				BlitzBanned:          boolPtr(cmd, "blitz-banned"),
				BlitzSuspended:       boolPtr(cmd, "blitz-suspended"),
				BlitzLivingLegend:    boolPtr(cmd, "blitz-living-legend"),
				CCLegal:              boolPtr(cmd, "cc-legal"),
				CCBanned:             boolPtr(cmd, "cc-banned"),
				CCSuspended:          boolPtr(cmd, "cc-suspended"),
				CCLivingLegend:       boolPtr(cmd, "cc-living-legend"),
				CommonerLegal:        boolPtr(cmd, "commoner-legal"),
				CommonerBanned:       boolPtr(cmd, "commoner-banned"),
				CommonerSuspended:    boolPtr(cmd, "commoner-suspended"),
				UPFBanned:            boolPtr(cmd, "upf-banned"),
				LLBanned:             boolPtr(cmd, "ll-banned"),
				LLRestricted:         boolPtr(cmd, "ll-restricted"),
				ProjectBlueLegal:     boolPtr(cmd, "project-blue-legal"),
				ProjectBlueBanned:    boolPtr(cmd, "project-blue-banned"),
				ProjectBlueSuspended: boolPtr(cmd, "project-blue-suspended"),
				IsDoubleFaced:        boolPtr(cmd, "is-double-faced"),
				NextToken:            strings.TrimSpace(cmd.String("next-token")),
			}
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				req.PageSize = &ps
			}

			resp, err := st.Service.SearchCards(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCardsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a card",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Card ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.GetCard(ctx, &clientv1.GetCardRequest{CardID: strings.TrimSpace(cmd.String("id"))})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCardsPrintingsCommand() *cli.Command {
	return &cli.Command{
		Name:  "printings",
		Usage: "List printings for a card",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "card-id", Usage: "Card ID", Required: true},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			req := &clientv1.ListPrintingsRequest{
				CardID:    strings.TrimSpace(cmd.String("card-id")),
				NextToken: strings.TrimSpace(cmd.String("next-token")),
			}
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				req.PageSize = &ps
			}
			resp, err := st.Service.ListPrintings(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCardsPrintingCommand() *cli.Command {
	return &cli.Command{
		Name:  "printing",
		Usage: "Get a printing",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Printing ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.GetPrinting(ctx, &clientv1.GetPrintingRequest{PrintingID: strings.TrimSpace(cmd.String("id"))})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCardsPrintingsSetCommand() *cli.Command {
	return &cli.Command{
		Name:  "printings-set",
		Usage: "List printings for a set number",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "set-number", Usage: "Set number", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.ListPrintingsForSetNumber(ctx, &clientv1.ListPrintingsForSetNumberRequest{SetNumber: strings.TrimSpace(cmd.String("set-number"))})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCardsProductCommand() *cli.Command {
	return &cli.Command{
		Name:  "product",
		Usage: "Get a product",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Product ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.GetProduct(ctx, &clientv1.GetProductRequest{ProductID: strings.TrimSpace(cmd.String("id"))})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCardsSnapshotCommand() *cli.Command {
	return &cli.Command{
		Name:  "snapshot",
		Usage: "Get data snapshot metadata",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "schema-version", Usage: "Schema version"},
			&cli.StringFlag{Name: "version", Usage: "Snapshot version override"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			req := &clientv1.GetDataSnapshotRequest{
				Version: strings.TrimSpace(cmd.String("version")),
			}
			if cmd.IsSet("schema-version") {
				v := cmd.Int("schema-version")
				if v < 0 {
					return cli.Exit("--schema-version must be >= 0", 2)
				}
				vs := int32(v)
				req.SchemaVersion = &vs
			}

			resp, err := st.Service.GetDataSnapshot(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}
