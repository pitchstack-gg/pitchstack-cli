package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"

	"github.com/urfave/cli/v3"
)

func newDecksCommand() *cli.Command {
	return &cli.Command{
		Name:  "decks",
		Usage: "Manage decks",
		Commands: []*cli.Command{
			newDecksListCommand(),
			newDecksGetCommand(),
			newDecksCreateCommand(),
			newDecksUpdateCommand(),
			newDecksDeleteCommand(),
			newDecksCloneCommand(),
			newDecksSearchCommand(),
			newDecksBatchGetCommand(),
			newDecksExportCommand(),
			newDecksImportCommand(),
			newDecksStarCommand(),
			newDecksUnstarCommand(),
			newDecksStarredCommand(),
			newDecksPermissionsCommand(),
			newDecksVersionsCommand(),
			newDecksMatchesCommand(),
		},
	}
}

func newDecksListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List decks",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "scope", Usage: "Scope (owned|shared|accessible)"},
			&cli.StringFlag{Name: "user-id", Usage: "Filter by user ID"},
			&cli.StringFlag{Name: "hero-id", Usage: "Filter by hero ID"},
			&cli.StringFlag{Name: "format", Usage: "Filter by format"},
			&cli.StringFlag{Name: "name", Usage: "Filter by name"},
			&cli.StringFlag{Name: "subject-id", Usage: "Shared subject (user/group) ID"},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.ListDecksRequest{
				Scope:     parseDeckListScope(cmd.String("scope")),
				UserID:    strings.TrimSpace(cmd.String("user-id")),
				HeroID:    strings.TrimSpace(cmd.String("hero-id")),
				Format:    strings.TrimSpace(cmd.String("format")),
				Name:      strings.TrimSpace(cmd.String("name")),
				NextToken: strings.TrimSpace(cmd.String("next-token")),
				SubjectID: strings.TrimSpace(cmd.String("subject-id")),
			}
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				req.PageSize = &ps
			}

			resp, err := st.Service.ListDecks(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a deck",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Deck ID", Required: true},
			yesFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(cmd.String("id"))
			resp, err := st.Service.GetDeck(ctx, &clientv1.GetDeckRequest{DeckID: id})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a deck",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Deck name", Required: true},
			&cli.StringFlag{Name: "hero-id", Usage: "Hero ID", Required: true},
			&cli.StringFlag{Name: "format", Usage: "Format", Required: true},
			&cli.StringFlag{Name: "author", Usage: "Author"},
			&cli.StringFlag{Name: "visibility", Usage: "Visibility (private|shared|public)"},
			&cli.StringFlag{Name: "deck-id", Usage: "Optional deck ID (idempotency)"},
			&cli.BoolFlag{Name: "create-initial-version", Usage: "Create an initial version (default: server)"},
			&cli.StringFlag{Name: "initial-version-name", Usage: "Initial version name"},
			&cli.StringFlag{Name: "initial-version-id", Usage: "Initial version ID (idempotency)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.CreateDeckRequest{
				Name:   strings.TrimSpace(cmd.String("name")),
				HeroID: strings.TrimSpace(cmd.String("hero-id")),
				Format: strings.TrimSpace(cmd.String("format")),
				Author: strings.TrimSpace(cmd.String("author")),
				DeckID: strings.TrimSpace(cmd.String("deck-id")),
			}
			if cmd.IsSet("visibility") {
				v := parseVisibility(cmd.String("visibility"))
				if v == clientv1.VisibilityLevelUnspecified {
					return cli.Exit("--visibility must be private|shared|public", 2)
				}
				req.Visibility = v
			}
			req.CreateInitialVersion = boolPtr(cmd, "create-initial-version")

			initialName := strings.TrimSpace(cmd.String("initial-version-name"))
			initialID := strings.TrimSpace(cmd.String("initial-version-id"))
			if initialName != "" || initialID != "" {
				req.InitialVersion = &clientv1.CreateDeckInitialVersion{
					Name:          initialName,
					DeckVersionID: initialID,
				}
			}

			resp, err := st.Service.CreateDeck(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a deck",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Deck ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "New name"},
			&cli.StringFlag{Name: "author", Usage: "New author"},
			&cli.StringFlag{Name: "active-deck-version-id", Usage: "Active deck version ID"},
			&cli.StringFlag{Name: "visibility", Usage: "Visibility (private|shared|public)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			id := strings.TrimSpace(cmd.String("id"))
			var anyUpdate bool

			var updateReq clientv1.UpdateDeckRequest
			updateReq.DeckID = id
			if cmd.IsSet("name") {
				v := strings.TrimSpace(cmd.String("name"))
				updateReq.Name = &v
				anyUpdate = true
			}
			if cmd.IsSet("author") {
				v := strings.TrimSpace(cmd.String("author"))
				updateReq.Author = &v
				anyUpdate = true
			}
			if cmd.IsSet("active-deck-version-id") {
				v := strings.TrimSpace(cmd.String("active-deck-version-id"))
				updateReq.ActiveDeckVersionID = &v
				anyUpdate = true
			}

			if anyUpdate {
				if _, err := st.Service.UpdateDeck(ctx, &updateReq); err != nil {
					return err
				}
			}

			if cmd.IsSet("visibility") {
				v := parseVisibility(cmd.String("visibility"))
				if v == clientv1.VisibilityLevelUnspecified {
					return cli.Exit("--visibility must be private|shared|public", 2)
				}
				if _, err := st.Service.UpdateDeckVisibility(ctx, &clientv1.UpdateDeckVisibilityRequest{
					DeckID:     id,
					Visibility: &v,
				}); err != nil {
					return err
				}
				anyUpdate = true
			}

			if !anyUpdate {
				return cli.Exit("no updates provided", 2)
			}

			resp, err := st.Service.GetDeck(ctx, &clientv1.GetDeckRequest{DeckID: id})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a deck",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Deck ID", Required: true},
			yesFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(cmd.String("id"))
			if err := confirmAction(cmd, "Delete", "deck", id); err != nil {
				return err
			}
			if _, err := st.Service.DeleteDeck(ctx, &clientv1.DeleteDeckRequest{DeckID: id}); err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"deleted": id})
		},
	}
}

func newDecksCloneCommand() *cli.Command {
	return &cli.Command{
		Name:  "clone",
		Usage: "Clone a deck version",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "source-deck-version-id", Usage: "Source deck version ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "New deck name", Required: true},
			&cli.StringFlag{Name: "visibility", Usage: "Visibility (private|shared|public)"},
			&cli.StringFlag{Name: "deck-id", Usage: "Optional deck ID (idempotency)"},
			&cli.StringFlag{Name: "initial-version-name", Usage: "Initial version name"},
			&cli.StringFlag{Name: "initial-deck-version-id", Usage: "Initial deck version ID (idempotency)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.CloneDeckRequest{
				SourceDeckVersionID:  strings.TrimSpace(cmd.String("source-deck-version-id")),
				Name:                 strings.TrimSpace(cmd.String("name")),
				DeckID:               strings.TrimSpace(cmd.String("deck-id")),
				InitialVersionName:   strings.TrimSpace(cmd.String("initial-version-name")),
				InitialDeckVersionID: strings.TrimSpace(cmd.String("initial-deck-version-id")),
			}
			if cmd.IsSet("visibility") {
				v := parseVisibility(cmd.String("visibility"))
				if v == clientv1.VisibilityLevelUnspecified {
					return cli.Exit("--visibility must be private|shared|public", 2)
				}
				req.Visibility = &v
			}

			resp, err := st.Service.CloneDeck(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksSearchCommand() *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "Search decks",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "q", Usage: "Search term"},
			&cli.StringFlag{Name: "hero-id", Usage: "Hero ID"},
			&cli.StringFlag{Name: "format", Usage: "Format"},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.SearchDecksRequest{
				SearchTerm: strings.TrimSpace(cmd.String("q")),
				HeroID:     strings.TrimSpace(cmd.String("hero-id")),
				Format:     strings.TrimSpace(cmd.String("format")),
				NextToken:  strings.TrimSpace(cmd.String("next-token")),
			}
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				req.PageSize = &ps
			}

			resp, err := st.Service.SearchDecks(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksBatchGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "batch-get",
		Usage: "Get decks in bulk",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "deck-id", Usage: "Deck ID (repeatable)"},
			&cli.BoolFlag{Name: "allow-partial", Usage: "Allow partial results"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			var deckIDs []string
			for _, raw := range cmd.StringSlice("deck-id") {
				for _, part := range strings.Split(raw, ",") {
					id := strings.TrimSpace(part)
					if id != "" {
						deckIDs = append(deckIDs, id)
					}
				}
			}
			if len(deckIDs) == 0 {
				return cli.Exit("--deck-id is required (repeatable)", 2)
			}

			resp, err := st.Service.BatchGetDecks(ctx, &clientv1.BatchGetDecksRequest{
				DeckIDs:      deckIDs,
				AllowPartial: cmd.Bool("allow-partial"),
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksExportCommand() *cli.Command {
	return &cli.Command{
		Name:  "export",
		Usage: "Export a deck",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Deck ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			id := strings.TrimSpace(cmd.String("id"))
			resp, err := st.Service.ExportDeck(ctx, &clientv1.ExportDeckRequest{DeckID: id})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksImportCommand() *cli.Command {
	return &cli.Command{
		Name:  "import",
		Usage: "Import a deck from JSON",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "file", Usage: "Path to JSON file (ImportDeckRequest)", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req, err := readImportDeckRequest(cmd.String("file"))
			if err != nil {
				return err
			}

			resp, err := st.Service.ImportDeck(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksPermissionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "permissions",
		Usage: "Manage deck access",
		Commands: []*cli.Command{
			newDecksPermissionsGetCommand(),
			newDecksPermissionsListCommand(),
			newDecksPermissionsGrantCommand(),
			newDecksPermissionsRevokeCommand(),
			newDecksPermissionsStopShareCommand(),
		},
	}
}

func newDecksStarCommand() *cli.Command {
	return newSDKCommand("star", "Star a deck", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Deck ID"}}, true, func(cmd *cli.Command, req *clientv1.StarDeckRequest) error {
		setStringFlag(cmd, "id", &req.DeckID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.StarDeckRequest) (any, error) {
		return c.StarDeck(ctx, req)
	})
}

func newDecksUnstarCommand() *cli.Command {
	return newSDKCommand("unstar", "Unstar a deck", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Deck ID"}}, true, func(cmd *cli.Command, req *clientv1.UnstarDeckRequest) error {
		setStringFlag(cmd, "id", &req.DeckID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UnstarDeckRequest) (any, error) {
		return c.UnstarDeck(ctx, req)
	})
}

func newDecksStarredCommand() *cli.Command {
	return newSDKCommand("starred", "List starred decks", []cli.Flag{
		&cli.StringFlag{Name: "user-id", Usage: "User ID"},
		&cli.IntFlag{Name: "limit", Usage: "Limit"},
		&cli.IntFlag{Name: "offset", Usage: "Offset"},
	}, true, func(cmd *cli.Command, req *clientv1.ListStarredDecksRequest) error {
		setStringFlag(cmd, "user-id", &req.UserID)
		setInt32Flag(cmd, "limit", &req.Limit)
		setInt32Flag(cmd, "offset", &req.Offset)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListStarredDecksRequest) (any, error) {
		return c.ListStarredDecks(ctx, req)
	})
}

func newDecksPermissionsStopShareCommand() *cli.Command {
	return newSDKCommand("stop-share", "Remove deck shares", []cli.Flag{&cli.StringFlag{Name: "deck-id", Usage: "Deck ID"}, yesFlag()}, true, func(cmd *cli.Command, req *clientv1.StopDeckShareRequest) error {
		setStringFlag(cmd, "deck-id", &req.DeckID)
		return confirmAction(cmd, "Remove", "deck shares", req.DeckID)
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.StopDeckShareRequest) (any, error) {
		return c.StopDeckShare(ctx, req)
	})
}

func newDecksMatchesCommand() *cli.Command {
	return &cli.Command{
		Name:  "matches",
		Usage: "Manage deck matches",
		Commands: []*cli.Command{
			newSDKCommand("create", "Create a deck version match", []cli.Flag{
				&cli.StringFlag{Name: "deck-version-id", Usage: "Deck version ID"},
				&cli.StringFlag{Name: "match-id", Usage: "Match ID"},
				&cli.StringFlag{Name: "played-at", Usage: "Played time (RFC3339)"},
				&cli.StringFlag{Name: "result", Usage: "Result"},
				&cli.StringFlag{Name: "opponent-hero-id", Usage: "Opponent hero ID"},
				&cli.StringFlag{Name: "opponent", Usage: "Opponent"},
				&cli.BoolFlag{Name: "went-first", Usage: "Went first"},
				&cli.StringFlag{Name: "event-type", Usage: "Event type"},
				&cli.StringFlag{Name: "notes", Usage: "Notes"},
				&cli.StringFlag{Name: "location", Usage: "Location"},
			}, true, func(cmd *cli.Command, req *clientv1.CreateDeckVersionMatchRequest) error {
				setStringFlag(cmd, "deck-version-id", &req.DeckVersionID)
				setStringFlag(cmd, "match-id", &req.MatchID)
				if err := setTimeFlag(cmd, "played-at", &req.PlayedAt); err != nil {
					return err
				}
				if cmd.IsSet("result") {
					req.Result = clientv1.DeckVersionMatchResult(cmd.String("result"))
				}
				setStringFlag(cmd, "opponent-hero-id", &req.OpponentHeroID)
				setStringFlag(cmd, "opponent", &req.Opponent)
				setBoolFlag(cmd, "went-first", &req.WentFirst)
				setStringFlag(cmd, "event-type", &req.EventType)
				setStringFlag(cmd, "notes", &req.Notes)
				setStringFlag(cmd, "location", &req.Location)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CreateDeckVersionMatchRequest) (any, error) {
				return c.CreateDeckVersionMatch(ctx, req)
			}),
			newSDKCommand("list", "List deck version matches", append(pageFlags(), &cli.StringFlag{Name: "deck-version-id", Usage: "Deck version ID"}), true, func(cmd *cli.Command, req *clientv1.ListDeckVersionMatchesRequest) error {
				setStringFlag(cmd, "deck-version-id", &req.DeckVersionID)
				setPageFlags(cmd, &req.PageSize, &req.NextToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListDeckVersionMatchesRequest) (any, error) {
				return c.ListDeckVersionMatches(ctx, req)
			}),
			newSDKCommand("get", "Get a deck version match", deckMatchIDFlags(), true, applyDeckMatchIDFlags, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetDeckVersionMatchRequest) (any, error) {
				return c.GetDeckVersionMatch(ctx, req)
			}),
			newSDKCommand("delete", "Delete a deck version match", append(deckMatchIDFlags(), yesFlag()), true, func(cmd *cli.Command, req *clientv1.DeleteDeckVersionMatchRequest) error {
				setStringFlag(cmd, "deck-version-id", &req.DeckVersionID)
				setStringFlag(cmd, "match-id", &req.MatchID)
				return confirmAction(cmd, "Delete", "deck match", req.MatchID)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.DeleteDeckVersionMatchRequest) (any, error) {
				return c.DeleteDeckVersionMatch(ctx, req)
			}),
		},
	}
}

func deckMatchIDFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "deck-version-id", Usage: "Deck version ID"},
		&cli.StringFlag{Name: "match-id", Usage: "Match ID"},
	}
}

func applyDeckMatchIDFlags(cmd *cli.Command, req *clientv1.GetDeckVersionMatchRequest) error {
	setStringFlag(cmd, "deck-version-id", &req.DeckVersionID)
	setStringFlag(cmd, "match-id", &req.MatchID)
	return nil
}

func newDecksPermissionsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Show deck access",
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
		Usage: "List deck access grants",
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

func newDecksPermissionsGrantCommand() *cli.Command {
	return &cli.Command{
		Name:  "grant",
		Usage: "Grant a user access to a deck",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-id", Usage: "Deck ID", Required: true},
			&cli.StringFlag{Name: "subject-id", Usage: "User ID to grant access to", Required: true},
			&cli.StringFlag{Name: "permission", Usage: "Permission (reader|writer)", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			perm := parseDeckPermission(cmd.String("permission"))
			if perm == clientv1.DeckPermissionUnspecified {
				return cli.Exit("--permission must be reader|writer", 2)
			}

			deckID := strings.TrimSpace(cmd.String("deck-id"))
			subjectID := strings.TrimSpace(cmd.String("subject-id"))

			if _, err := st.Service.GrantDeckAccess(ctx, &clientv1.GrantDeckAccessRequest{
				DeckID:     deckID,
				SubjectID:  subjectID,
				Permission: perm,
			}); err != nil {
				return err
			}

			return writeJSON(cmd.Writer, map[string]any{
				"deckId":     deckID,
				"subjectId":  subjectID,
				"permission": perm,
				"granted":    true,
			})
		},
	}
}

func newDecksPermissionsRevokeCommand() *cli.Command {
	return &cli.Command{
		Name:  "revoke",
		Usage: "Revoke a user's access to a deck",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-id", Usage: "Deck ID", Required: true},
			&cli.StringFlag{Name: "subject-id", Usage: "User ID to revoke access from", Required: true},
			&cli.StringFlag{Name: "permission", Usage: "Permission (reader|writer)", Required: true},
			yesFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			perm := parseDeckPermission(cmd.String("permission"))
			if perm == clientv1.DeckPermissionUnspecified {
				return cli.Exit("--permission must be reader|writer", 2)
			}

			deckID := strings.TrimSpace(cmd.String("deck-id"))
			subjectID := strings.TrimSpace(cmd.String("subject-id"))
			if err := confirmAction(cmd, "Revoke", "deck access", deckID); err != nil {
				return err
			}

			if _, err := st.Service.RevokeDeckAccess(ctx, &clientv1.RevokeDeckAccessRequest{
				DeckID:     deckID,
				SubjectID:  subjectID,
				Permission: perm,
			}); err != nil {
				return err
			}

			return writeJSON(cmd.Writer, map[string]any{
				"deckId":     deckID,
				"subjectId":  subjectID,
				"permission": perm,
				"revoked":    true,
			})
		},
	}
}

func newDecksVersionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "versions",
		Usage: "Manage deck versions",
		Commands: []*cli.Command{
			newDecksVersionsListCommand(),
			newDecksVersionsCreateCommand(),
			newDecksVersionsGetCommand(),
			newDecksVersionsDeleteCommand(),
			newDecksVersionsHistoryCommand(),
			newDecksVersionsNotesCommand(),
			newDecksVersionsCardsCommand(),
			newDecksVersionsSideboardGuidesCommand(),
		},
	}
}

func newDecksVersionsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List deck versions",
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

			req := &clientv1.ListDeckVersionsRequest{
				DeckID:    strings.TrimSpace(cmd.String("deck-id")),
				NextToken: strings.TrimSpace(cmd.String("next-token")),
			}
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				req.PageSize = &ps
			}

			resp, err := st.Service.ListDeckVersions(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksVersionsCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a deck version",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-id", Usage: "Deck ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "Version name", Required: true},
			&cli.StringFlag{Name: "deck-version-id", Usage: "Optional version ID (idempotency)"},
			&cli.StringFlag{Name: "source-deck-version-id", Usage: "Source version to clone"},
			&cli.BoolFlag{Name: "set-active", Usage: "Set new version as active"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.CreateDeckVersionRequest{
				DeckID:              strings.TrimSpace(cmd.String("deck-id")),
				Name:                strings.TrimSpace(cmd.String("name")),
				DeckVersionID:       strings.TrimSpace(cmd.String("deck-version-id")),
				SourceDeckVersionID: strings.TrimSpace(cmd.String("source-deck-version-id")),
				SetActive:           boolPtr(cmd, "set-active"),
			}

			resp, err := st.Service.CreateDeckVersion(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksVersionsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a deck version",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-version-id", Usage: "Deck version ID", Required: true},
			yesFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			resp, err := st.Service.GetDeckVersion(ctx, &clientv1.GetDeckVersionRequest{
				DeckVersionID: strings.TrimSpace(cmd.String("deck-version-id")),
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksVersionsDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a deck version",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-version-id", Usage: "Deck version ID", Required: true},
			yesFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(cmd.String("deck-version-id"))
			if err := confirmAction(cmd, "Delete", "deck version", id); err != nil {
				return err
			}
			if _, err := st.Service.DeleteDeckVersion(ctx, &clientv1.DeleteDeckVersionRequest{DeckVersionID: id}); err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"deleted": id})
		},
	}
}

func newDecksVersionsHistoryCommand() *cli.Command {
	return &cli.Command{
		Name:  "history",
		Usage: "Get deck version history",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-version-id", Usage: "Deck version ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.GetDeckVersionHistory(ctx, &clientv1.GetDeckVersionHistoryRequest{
				DeckVersionID: strings.TrimSpace(cmd.String("deck-version-id")),
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksVersionsNotesCommand() *cli.Command {
	return &cli.Command{
		Name:  "notes",
		Usage: "Manage deck notes",
		Commands: []*cli.Command{
			newDecksVersionsNotesGetCommand(),
			newDecksVersionsNotesUpdateCommand(),
		},
	}
}

func newDecksVersionsNotesGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get deck version notes",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-version-id", Usage: "Deck version ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.GetDeckVersionNotes(ctx, &clientv1.GetDeckVersionNotesRequest{
				DeckVersionID: strings.TrimSpace(cmd.String("deck-version-id")),
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksVersionsNotesUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update deck version notes",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-version-id", Usage: "Deck version ID", Required: true},
			&cli.StringFlag{Name: "notes", Usage: "Notes", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.UpdateDeckVersionNotesRequest{
				DeckVersionID: strings.TrimSpace(cmd.String("deck-version-id")),
				Notes:         strings.TrimSpace(cmd.String("notes")),
			}
			resp, err := st.Service.UpdateDeckVersionNotes(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksVersionsCardsCommand() *cli.Command {
	return &cli.Command{
		Name:  "cards",
		Usage: "Manage deck cards",
		Commands: []*cli.Command{
			newDecksVersionsCardsListCommand(),
			newDecksVersionsCardsModifyCommand(),
		},
	}
}

func newDecksVersionsCardsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List cards for a deck version",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-version-id", Usage: "Deck version ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.ListDeckVersionCards(ctx, &clientv1.ListDeckVersionCardsRequest{
				DeckVersionID: strings.TrimSpace(cmd.String("deck-version-id")),
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksVersionsCardsModifyCommand() *cli.Command {
	return &cli.Command{
		Name:  "modify",
		Usage: "Modify a card in a deck version",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-version-id", Usage: "Deck version ID", Required: true},
			&cli.StringFlag{Name: "card-id", Usage: "Card ID", Required: true},
			&cli.StringFlag{Name: "board", Usage: "Board (mainboard|sideboard|maybeboard)", Required: true},
			&cli.IntFlag{Name: "quantity", Usage: "Quantity (>= 0)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			if !cmd.IsSet("quantity") {
				return cli.Exit("--quantity is required", 2)
			}
			qty := cmd.Int("quantity")
			if qty < 0 {
				return cli.Exit("--quantity must be >= 0", 2)
			}
			board, ok := parseDeckBoardType(cmd.String("board"))
			if !ok {
				return cli.Exit("--board must be mainboard|sideboard|maybeboard", 2)
			}

			req := &clientv1.ModifyDeckVersionCardRequest{
				DeckVersionID: strings.TrimSpace(cmd.String("deck-version-id")),
				CardID:        strings.TrimSpace(cmd.String("card-id")),
				Board:         board,
				Quantity:      int32(qty),
			}
			resp, err := st.Service.ModifyDeckVersionCard(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksVersionsSideboardGuidesCommand() *cli.Command {
	return &cli.Command{
		Name:  "sideboard-guides",
		Usage: "Manage sideboard guides",
		Commands: []*cli.Command{
			newDecksVersionsSideboardGuidesListCommand(),
			newDecksVersionsSideboardGuidesUpsertCommand(),
			newDecksVersionsSideboardGuidesDeleteCommand(),
		},
	}
}

func newDecksVersionsSideboardGuidesListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List sideboard guides for a deck version",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-version-id", Usage: "Deck version ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.ListDeckVersionSideboardGuides(ctx, &clientv1.ListDeckVersionSideboardGuidesRequest{
				DeckVersionID: strings.TrimSpace(cmd.String("deck-version-id")),
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksVersionsSideboardGuidesUpsertCommand() *cli.Command {
	return &cli.Command{
		Name:  "upsert",
		Usage: "Upsert a sideboard guide for a deck version",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-version-id", Usage: "Deck version ID", Required: true},
			&cli.StringFlag{Name: "target-type", Usage: "Target type (hero|class|archetype)", Required: true},
			&cli.StringFlag{Name: "target", Usage: "Target identifier", Required: true},
			&cli.StringFlag{Name: "guide", Usage: "Guide text", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			targetType, ok := parseSideboardGuideTargetType(cmd.String("target-type"))
			if !ok {
				return cli.Exit("--target-type must be hero|class|archetype", 2)
			}

			req := &clientv1.UpsertDeckVersionSideboardGuideRequest{
				DeckVersionID: strings.TrimSpace(cmd.String("deck-version-id")),
				TargetType:    targetType,
				Target:        strings.TrimSpace(cmd.String("target")),
				Guide:         strings.TrimSpace(cmd.String("guide")),
			}
			resp, err := st.Service.UpsertDeckVersionSideboardGuide(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newDecksVersionsSideboardGuidesDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a sideboard guide for a deck version",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deck-version-id", Usage: "Deck version ID", Required: true},
			&cli.StringFlag{Name: "target-type", Usage: "Target type (hero|class|archetype)", Required: true},
			&cli.StringFlag{Name: "target", Usage: "Target identifier", Required: true},
			yesFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			targetType, ok := parseSideboardGuideTargetType(cmd.String("target-type"))
			if !ok {
				return cli.Exit("--target-type must be hero|class|archetype", 2)
			}
			target := strings.TrimSpace(cmd.String("target"))
			if err := confirmAction(cmd, "Delete", "sideboard guide", target); err != nil {
				return err
			}

			if _, err := st.Service.DeleteDeckVersionSideboardGuide(ctx, &clientv1.DeleteDeckVersionSideboardGuideRequest{
				DeckVersionID: strings.TrimSpace(cmd.String("deck-version-id")),
				TargetType:    targetType,
				Target:        target,
			}); err != nil {
				return err
			}

			return writeJSON(cmd.Writer, map[string]any{
				"deleted": true,
			})
		},
	}
}

func parseDeckListScope(v string) clientv1.DeckListScope {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "unspecified":
		return clientv1.DeckListScopeUnspecified
	case "owned":
		return clientv1.DeckListScopeOwned
	case "shared":
		return clientv1.DeckListScopeShared
	case "accessible":
		return clientv1.DeckListScopeAccessible
	default:
		upper := strings.ToUpper(strings.TrimSpace(v))
		if strings.HasPrefix(upper, "DECK_LIST_SCOPE_") {
			return clientv1.DeckListScope(upper)
		}
		return clientv1.DeckListScope(v)
	}
}

func parseDeckPermission(v string) clientv1.DeckPermission {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "unspecified":
		return clientv1.DeckPermissionUnspecified
	case "reader", "read", "r":
		return clientv1.DeckPermissionReader
	case "writer", "write", "w":
		return clientv1.DeckPermissionWriter
	default:
		upper := strings.ToUpper(strings.TrimSpace(v))
		if strings.HasPrefix(upper, "PERMISSION_") {
			return clientv1.DeckPermission(upper)
		}
		return clientv1.DeckPermission(v)
	}
}

func parseDeckBoardType(v string) (clientv1.BoardType, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "mainboard", "main":
		return clientv1.BoardTypeMainboard, true
	case "sideboard", "side":
		return clientv1.BoardTypeSideboard, true
	case "maybeboard", "maybe":
		return clientv1.BoardTypeMaybeboard, true
	case "", "unspecified":
		return clientv1.BoardTypeUnspecified, false
	default:
		upper := strings.ToUpper(strings.TrimSpace(v))
		if strings.HasPrefix(upper, "BOARD_TYPE_") {
			return clientv1.BoardType(upper), true
		}
		return clientv1.BoardTypeUnspecified, false
	}
}

func parseSideboardGuideTargetType(v string) (clientv1.SideboardGuideTargetType, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "hero":
		return clientv1.SideboardGuideTargetTypeHero, true
	case "class":
		return clientv1.SideboardGuideTargetTypeClass, true
	case "archetype", "arch":
		return clientv1.SideboardGuideTargetTypeArchetype, true
	case "", "unspecified":
		return clientv1.SideboardGuideTargetTypeUnspecified, false
	default:
		upper := strings.ToUpper(strings.TrimSpace(v))
		if strings.HasPrefix(upper, "SIDEBOARD_GUIDE_TARGET_TYPE_") {
			return clientv1.SideboardGuideTargetType(upper), true
		}
		return clientv1.SideboardGuideTargetTypeUnspecified, false
	}
}

func readImportDeckRequest(path string) (*clientv1.ImportDeckRequest, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("file path must not be empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var req clientv1.ImportDeckRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &req, nil
}
