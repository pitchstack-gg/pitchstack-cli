package commands

import (
	"context"
	"strings"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"

	"github.com/urfave/cli/v3"
)

func newDecksCommand() *cli.Command {
	return &cli.Command{
		Name:  "decks",
		Usage: "Manage decks",
		Commands: []*cli.Command{
			newDecksPermissionsCommand(),
		},
	}
}

func newDecksPermissionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "permissions",
		Usage: "Manage deck access permissions",
		Commands: []*cli.Command{
			newDecksPermissionsGetCommand(),
			newDecksPermissionsListCommand(),
		},
	}
}

func newDecksPermissionsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get your effective permission for a deck",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-id", Usage: "Deck ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			resp, err := st.Service.GetDeckAccess(ctx, &clientv1.GetDeckAccessRequest{
				DeckID: strings.TrimSpace(cmd.String("deck-id")),
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksPermissionsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List explicit access grants for a deck",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-id", Usage: "Deck ID", Required: true},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.ListDeckAccessGrantsRequest{
				DeckID:    strings.TrimSpace(cmd.String("deck-id")),
				NextToken: strings.TrimSpace(cmd.String("next-token")),
			}
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				req.PageSize = &ps
			}

			resp, err := st.Service.ListDeckAccessGrants(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}
