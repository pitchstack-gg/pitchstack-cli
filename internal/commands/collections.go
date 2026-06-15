package commands

import (
	"context"
	"strings"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"

	"github.com/urfave/cli/v3"
)

func newCollectionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "collections",
		Usage: "Manage collections",
		Commands: []*cli.Command{
			newCollectionsListCommand(),
			newCollectionsCountsCommand(),
			newCollectionsGetCommand(),
			newCollectionsHistoryCommand(),
			newCollectionsBatchGetCommand(),
			newCollectionsCreateCommand(),
			newCollectionsUpdateCommand(),
			newCollectionsArtCommand(),
			newCollectionsDeleteCommand(),
			newCollectionsExportCommand(),
			newCollectionsImportCommand(),
			newCollectionsValuationCommand(),
			newCollectionsTradeItemsCommand(),
			newCollectionsPermissionsCommand(),
			newCollectionItemsCommand(),
		},
	}
}

func newCollectionsPermissionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "permissions",
		Usage: "Manage collection access permissions",
		Commands: []*cli.Command{
			newCollectionsPermissionsGetCommand(),
			newCollectionsPermissionsListCommand(),
			newCollectionsPermissionsGrantCommand(),
			newCollectionsPermissionsRevokeCommand(),
			newCollectionsPermissionsStopShareCommand(),
		},
	}
}

func newCollectionsPermissionsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get your effective permission for a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "collection-id", Usage: "Collection ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			resp, err := st.Service.GetCollectionAccess(ctx, &clientv1.GetCollectionAccessRequest{
				CollectionID: strings.TrimSpace(cmd.String("collection-id")),
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionsPermissionsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List explicit access grants for a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "collection-id", Usage: "Collection ID", Required: true},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.ListCollectionAccessGrantsRequest{
				CollectionID: strings.TrimSpace(cmd.String("collection-id")),
				NextToken:    strings.TrimSpace(cmd.String("next-token")),
			}
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				req.PageSize = &ps
			}

			resp, err := st.Service.ListCollectionAccessGrants(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionsPermissionsGrantCommand() *cli.Command {
	return &cli.Command{
		Name:  "grant",
		Usage: "Grant a user access to a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "collection-id", Usage: "Collection ID", Required: true},
			&cli.StringFlag{Name: "subject-id", Usage: "User ID to grant access to", Required: true},
			&cli.StringFlag{Name: "permission", Usage: "Permission (reader|writer)", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			perm := parseCollectionPermission(cmd.String("permission"))
			if perm == clientv1.CollectionPermissionUnspecified {
				return cli.Exit("--permission must be reader|writer", 2)
			}

			collectionID := strings.TrimSpace(cmd.String("collection-id"))
			subjectID := strings.TrimSpace(cmd.String("subject-id"))

			if _, err := st.Service.GrantCollectionAccess(ctx, &clientv1.GrantCollectionAccessRequest{
				CollectionID: collectionID,
				SubjectID:    subjectID,
				Permission:   perm,
			}); err != nil {
				return err
			}

			return writeJSON(cmd.Writer, map[string]any{
				"collectionId": collectionID,
				"subjectId":    subjectID,
				"permission":   perm,
				"granted":      true,
			})
		},
	}
}

func newCollectionsPermissionsRevokeCommand() *cli.Command {
	return &cli.Command{
		Name:  "revoke",
		Usage: "Revoke a user's access to a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "collection-id", Usage: "Collection ID", Required: true},
			&cli.StringFlag{Name: "subject-id", Usage: "User ID to revoke access from", Required: true},
			&cli.StringFlag{Name: "permission", Usage: "Permission (reader|writer)", Required: true},
			yesFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			perm := parseCollectionPermission(cmd.String("permission"))
			if perm == clientv1.CollectionPermissionUnspecified {
				return cli.Exit("--permission must be reader|writer", 2)
			}

			collectionID := strings.TrimSpace(cmd.String("collection-id"))
			subjectID := strings.TrimSpace(cmd.String("subject-id"))
			if err := confirmAction(cmd, "Revoke", "collection access", collectionID); err != nil {
				return err
			}

			if _, err := st.Service.RevokeCollectionAccess(ctx, &clientv1.RevokeCollectionAccessRequest{
				CollectionID: collectionID,
				SubjectID:    subjectID,
				Permission:   perm,
			}); err != nil {
				return err
			}

			return writeJSON(cmd.Writer, map[string]any{
				"collectionId": collectionID,
				"subjectId":    subjectID,
				"permission":   perm,
				"revoked":      true,
			})
		},
	}
}

func newCollectionsPermissionsStopShareCommand() *cli.Command {
	return newSDKCommand("stop-share", "Remove all explicit collection shares", []cli.Flag{&cli.StringFlag{Name: "collection-id", Usage: "Collection ID"}, yesFlag()}, true, func(cmd *cli.Command, req *clientv1.StopCollectionShareRequest) error {
		setStringFlag(cmd, "collection-id", &req.CollectionID)
		return confirmAction(cmd, "Remove", "collection shares", req.CollectionID)
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.StopCollectionShareRequest) (any, error) {
		return c.StopCollectionShare(ctx, req)
	})
}

func newCollectionsListCommand() *cli.Command {
	return newSDKCommand("list", "List collections", []cli.Flag{
		&cli.StringFlag{Name: "scope", Usage: "Scope (owned|shared|accessible)"},
		&cli.StringFlag{Name: "user-id", Usage: "Filter by user ID"},
		&cli.IntFlag{Name: "page-size", Usage: "Page size"},
		&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
	}, true, func(cmd *cli.Command, req *clientv1.ListCollectionsRequest) error {
		if scope := parseCollectionScope(cmd.String("scope")); scope != clientv1.CollectionListScopeUnspecified {
			req.Scope = scope
		}
		setStringFlag(cmd, "user-id", &req.UserID)
		setPageFlags(cmd, &req.PageSize, &req.NextToken)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListCollectionsRequest) (any, error) {
		return c.ListCollections(ctx, req)
	})
}

func newCollectionsGetCommand() *cli.Command {
	return newSDKCommand("get", "Get a collection", []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true},
	}, true, func(cmd *cli.Command, req *clientv1.GetCollectionRequest) error {
		setStringFlag(cmd, "id", &req.CollectionID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetCollectionRequest) (any, error) {
		return c.GetCollection(ctx, req)
	})
}

func newCollectionsHistoryCommand() *cli.Command {
	return newSDKCommand("history", "Get collection history", []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true},
	}, true, func(cmd *cli.Command, req *clientv1.GetCollectionHistoryRequest) error {
		setStringFlag(cmd, "id", &req.CollectionID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetCollectionHistoryRequest) (any, error) {
		return c.GetCollectionHistory(ctx, req)
	})
}

func newCollectionsBatchGetCommand() *cli.Command {
	return newSDKCommand("batch-get", "Batch get collections", []cli.Flag{
		repeatedIDsFlag("id", "Collection ID (repeatable or comma-separated)"),
		&cli.BoolFlag{Name: "allow-partial", Usage: "Allow partial results"},
	}, true, func(cmd *cli.Command, req *clientv1.BatchGetCollectionsRequest) error {
		if cmd.IsSet("id") {
			req.CollectionIDs = splitCSV(cmd.StringSlice("id"))
		}
		if cmd.IsSet("allow-partial") {
			req.AllowPartial = cmd.Bool("allow-partial")
		}
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.BatchGetCollectionsRequest) (any, error) {
		return c.BatchGetCollections(ctx, req)
	})
}

func newCollectionsCreateCommand() *cli.Command {
	return newSDKCommand("create", "Create a collection", []cli.Flag{
		&cli.StringFlag{Name: "name", Usage: "Collection name", Required: true},
		&cli.StringFlag{Name: "type", Usage: "Collection type (binder|wantlist|tradelist|list)", Required: true},
		&cli.StringFlag{Name: "description", Usage: "Collection description"},
		&cli.StringFlag{Name: "visibility", Usage: "Visibility (private|shared|public)", Value: "private"},
		&cli.StringFlag{Name: "collection-id", Usage: "Optional collection ID"},
	}, true, func(cmd *cli.Command, req *clientv1.CreateCollectionRequest) error {
		setStringFlag(cmd, "name", &req.Name)
		if cmd.IsSet("type") {
			collectionType := parseCollectionType(cmd.String("type"))
			if collectionType == clientv1.CollectionTypeUnspecified {
				return cli.Exit("--type must be binder|wantlist|tradelist|list", 2)
			}
			req.CollectionType = collectionType
		}
		setStringFlag(cmd, "description", &req.Description)
		if cmd.IsSet("visibility") || req.Visibility == clientv1.VisibilityLevelUnspecified {
			visibility := parseVisibility(cmd.String("visibility"))
			if visibility == clientv1.VisibilityLevelUnspecified {
				return cli.Exit("--visibility must be private|shared|public", 2)
			}
			req.Visibility = visibility
		}
		setStringFlag(cmd, "collection-id", &req.CollectionID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CreateCollectionRequest) (any, error) {
		return c.CreateCollection(ctx, req)
	})
}

func newCollectionsUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "New name"},
			&cli.StringFlag{Name: "description", Usage: "New description"},
			&cli.StringFlag{Name: "visibility", Usage: "New visibility (private|shared|public)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id := strings.TrimSpace(cmd.String("id"))
			var last any
			var anyUpdate bool

			return withSDKClient(ctx, cmd, true, func(c *clientv1.Client) (any, error) {
				updateReq := &clientv1.UpdateCollectionRequest{CollectionID: id}
				if cmd.IsSet("name") {
					name := strings.TrimSpace(cmd.String("name"))
					updateReq.Name = &name
					anyUpdate = true
				}
				if cmd.IsSet("description") {
					description := strings.TrimSpace(cmd.String("description"))
					updateReq.Description = &description
					anyUpdate = true
				}
				if updateReq.Name != nil || updateReq.Description != nil {
					resp, err := c.UpdateCollection(ctx, updateReq)
					if err != nil {
						return nil, err
					}
					last = resp
				}
				if cmd.IsSet("visibility") {
					v := parseVisibility(cmd.String("visibility"))
					if v == clientv1.VisibilityLevelUnspecified {
						return nil, cli.Exit("--visibility must be private|shared|public", 2)
					}
					resp, err := c.UpdateCollectionVisibility(ctx, &clientv1.UpdateCollectionVisibilityRequest{
						CollectionID: id,
						Visibility:   &v,
					})
					if err != nil {
						return nil, err
					}
					last = resp
					anyUpdate = true
				}
				if !anyUpdate {
					return nil, cli.Exit("no updates provided", 2)
				}
				return last, nil
			})
		},
	}
}

func newCollectionsArtCommand() *cli.Command {
	return newSDKCommand("art", "Update collection artwork selection", []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true},
		&cli.StringFlag{Name: "selected-art-printing-id", Usage: "Selected art printing ID"},
		&cli.BoolFlag{Name: "clear-selected-art", Usage: "Clear selected artwork"},
	}, true, func(cmd *cli.Command, req *clientv1.UpdateCollectionArtRequest) error {
		setStringFlag(cmd, "id", &req.CollectionID)
		setStringFlag(cmd, "selected-art-printing-id", &req.SelectedArtPrintingID)
		if cmd.IsSet("clear-selected-art") {
			req.ClearSelectedArt = cmd.Bool("clear-selected-art")
		}
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UpdateCollectionArtRequest) (any, error) {
		return c.UpdateCollectionArt(ctx, req)
	})
}

func newCollectionsDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true},
			yesFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(cmd.String("id"))
			if err := confirmAction(cmd, "Delete", "collection", id); err != nil {
				return err
			}
			if _, err := st.Service.DeleteCollection(ctx, &clientv1.DeleteCollectionRequest{CollectionID: id}); err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"deleted": id})
		},
	}
}

func newCollectionsExportCommand() *cli.Command {
	return newSDKCommand("export", "Export a collection", append(pageFlags(),
		&cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true},
	), true, func(cmd *cli.Command, req *clientv1.ExportCollectionRequest) error {
		setStringFlag(cmd, "id", &req.CollectionID)
		setPageFlags(cmd, &req.PageSize, &req.NextToken)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ExportCollectionRequest) (any, error) {
		return c.ExportCollection(ctx, req)
	})
}

func newCollectionsImportCommand() *cli.Command {
	return newSDKCommand("import", "Import a collection from JSON", []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: "Collection ID"},
		&cli.StringFlag{Name: "name", Usage: "Collection name"},
		&cli.StringFlag{Name: "type", Usage: "Collection type"},
		&cli.StringFlag{Name: "description", Usage: "Description"},
		&cli.StringFlag{Name: "visibility", Usage: "Visibility"},
	}, true, func(cmd *cli.Command, req *clientv1.ImportCollectionRequest) error {
		setStringFlag(cmd, "id", &req.CollectionID)
		setStringFlag(cmd, "name", &req.Name)
		if cmd.IsSet("type") {
			collectionType := parseCollectionType(cmd.String("type"))
			if collectionType == clientv1.CollectionTypeUnspecified {
				return cli.Exit("--type must be binder|wantlist|tradelist|list", 2)
			}
			req.CollectionType = collectionType
		}
		setStringFlag(cmd, "description", &req.Description)
		if cmd.IsSet("visibility") {
			visibility := parseVisibility(cmd.String("visibility"))
			if visibility == clientv1.VisibilityLevelUnspecified {
				return cli.Exit("--visibility must be private|shared|public", 2)
			}
			req.Visibility = visibility
		}
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ImportCollectionRequest) (any, error) {
		return c.ImportCollection(ctx, req)
	})
}

func newCollectionsValuationCommand() *cli.Command {
	return newSDKCommand("valuation", "Get collection valuation", []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: "Collection ID"},
		&cli.StringFlag{Name: "source", Usage: "Price source"},
	}, true, func(cmd *cli.Command, req *clientv1.GetCollectionValuationRequest) error {
		setStringFlag(cmd, "id", &req.CollectionID)
		setStringFlag(cmd, "source", &req.Source)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetCollectionValuationRequest) (any, error) {
		return c.GetCollectionValuation(ctx, req)
	})
}

func newCollectionsTradeItemsCommand() *cli.Command {
	return newSDKCommand("trade-items", "List tradable collection items", append(pageFlags(),
		&cli.StringFlag{Name: "user-id", Usage: "Filter by user ID"},
	), true, func(cmd *cli.Command, req *clientv1.ListTradeItemsRequest) error {
		setStringFlag(cmd, "user-id", &req.UserID)
		setPageFlags(cmd, &req.PageSize, &req.NextToken)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListTradeItemsRequest) (any, error) {
		return c.ListTradeItems(ctx, req)
	})
}

func parseCollectionScope(v string) clientv1.CollectionListScope {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "unspecified":
		return clientv1.CollectionListScopeUnspecified
	case "owned":
		return clientv1.CollectionListScopeOwned
	case "shared":
		return clientv1.CollectionListScopeShared
	case "accessible":
		return clientv1.CollectionListScopeAccessible
	default:
		return clientv1.CollectionListScope(v)
	}
}

func parseCollectionType(v string) clientv1.CollectionType {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "unspecified":
		return clientv1.CollectionTypeUnspecified
	case "binder":
		return clientv1.CollectionTypeBinder
	case "wantlist":
		return clientv1.CollectionTypeWantlist
	case "tradelist":
		return clientv1.CollectionTypeTradelist
	case "list":
		return clientv1.CollectionTypeList
	default:
		return clientv1.CollectionType(v)
	}
}

func parseCollectionPermission(v string) clientv1.CollectionPermission {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "unspecified":
		return clientv1.CollectionPermissionUnspecified
	case "reader", "read", "r":
		return clientv1.CollectionPermissionReader
	case "writer", "write", "w":
		return clientv1.CollectionPermissionWriter
	default:
		return clientv1.CollectionPermission(v)
	}
}

func parseVisibility(v string) clientv1.VisibilityLevel {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "unspecified":
		return clientv1.VisibilityLevelUnspecified
	case "private":
		return clientv1.VisibilityLevelPrivate
	case "shared":
		return clientv1.VisibilityLevelShared
	case "public":
		return clientv1.VisibilityLevelPublic
	default:
		return clientv1.VisibilityLevel(v)
	}
}

func newCollectionItemsCommand() *cli.Command {
	return &cli.Command{
		Name:  "items",
		Usage: "Manage collection items",
		Commands: []*cli.Command{
			newCollectionItemsListCommand(),
			newCollectionItemsGetCommand(),
			newCollectionItemsAddCommand(),
			newCollectionItemsUpdateCommand(),
			newCollectionItemsAdjustCommand(),
			newCollectionItemsTransferCommand(),
			newCollectionItemsBatchGetCommand(),
			newCollectionItemsDeleteCommand(),
		},
	}
}

func newCollectionItemsListCommand() *cli.Command {
	return newSDKCommand("list", "List collection items", []cli.Flag{
		&cli.StringFlag{Name: "collection-id", Usage: "Collection ID"},
		&cli.StringFlag{Name: "card-id", Usage: "Filter by card ID"},
		&cli.StringFlag{Name: "printing-id", Usage: "Filter by printing ID"},
		&cli.StringFlag{Name: "product-id", Usage: "Filter by product ID"},
		&cli.IntFlag{Name: "page-size", Usage: "Page size"},
		&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
	}, true, func(cmd *cli.Command, req *clientv1.ListCollectionItemsRequest) error {
		setStringFlag(cmd, "collection-id", &req.CollectionID)
		setStringFlag(cmd, "card-id", &req.CardID)
		setStringFlag(cmd, "printing-id", &req.PrintingID)
		setStringFlag(cmd, "product-id", &req.ProductID)
		setPageFlags(cmd, &req.PageSize, &req.NextToken)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListCollectionItemsRequest) (any, error) {
		return c.ListCollectionItems(ctx, req)
	})
}

func newCollectionItemsGetCommand() *cli.Command {
	return newSDKCommand("get", "Get a collection item", []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: "Item ID", Required: true},
	}, true, func(cmd *cli.Command, req *clientv1.GetCollectionItemRequest) error {
		setStringFlag(cmd, "id", &req.ItemID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetCollectionItemRequest) (any, error) {
		return c.GetCollectionItem(ctx, req)
	})
}

func newCollectionItemsAddCommand() *cli.Command {
	return newSDKCommand("add", "Add an item to a collection", []cli.Flag{
		&cli.StringFlag{Name: "collection-id", Usage: "Collection ID", Required: true},
		&cli.StringFlag{Name: "product-id", Usage: "Product ID", Required: true},
		&cli.IntFlag{Name: "quantity", Usage: "Quantity (>= 1)", Value: 1},
		&cli.StringFlag{Name: "condition", Usage: "Condition (near_mint|lightly_played|heavily_played|damaged)", Value: "near_mint"},
		&cli.Float64Flag{Name: "value", Usage: "Optional value"},
		&cli.StringFlag{Name: "item-id", Usage: "Optional item ID (idempotency)"},
		&cli.IntFlag{Name: "trade-quantity", Usage: "Trade quantity"},
		&cli.StringFlag{Name: "notes", Usage: "Notes"},
	}, true, func(cmd *cli.Command, req *clientv1.CreateCollectionItemRequest) error {
		setStringFlag(cmd, "collection-id", &req.CollectionID)
		setStringFlag(cmd, "product-id", &req.ProductID)
		if cmd.IsSet("quantity") || req.Quantity == 0 {
			qty := cmd.Int("quantity")
			if qty < 1 {
				return cli.Exit("--quantity must be >= 1", 2)
			}
			req.Quantity = int32(qty)
		}
		if req.Quantity < 1 {
			return cli.Exit("--quantity must be >= 1", 2)
		}
		if cmd.IsSet("condition") || req.Condition == clientv1.ConditionUnspecified || req.Condition == "" {
			cond, ok := parseCondition(cmd.String("condition"))
			if !ok {
				return cli.Exit("--condition must be near_mint|lightly_played|heavily_played|damaged", 2)
			}
			req.Condition = cond
		}
		setFloat64Flag(cmd, "value", &req.Value)
		setStringFlag(cmd, "item-id", &req.ItemID)
		if cmd.IsSet("trade-quantity") {
			req.TradeQuantity = int32(cmd.Int("trade-quantity"))
		}
		setStringFlag(cmd, "notes", &req.Notes)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CreateCollectionItemRequest) (any, error) {
		return c.CreateCollectionItem(ctx, req)
	})
}

func newCollectionItemsUpdateCommand() *cli.Command {
	return newSDKCommand("update", "Update a collection item", []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: "Item ID", Required: true},
		&cli.IntFlag{Name: "quantity", Usage: "Quantity (>= 1)"},
		&cli.StringFlag{Name: "condition", Usage: "Condition (near_mint|lightly_played|heavily_played|damaged)"},
		&cli.Float64Flag{Name: "value", Usage: "Value"},
		&cli.BoolFlag{Name: "pinned", Usage: "Pin or unpin item"},
		&cli.StringFlag{Name: "expected-updated-at", Usage: "Expected updated_at precondition (RFC3339)"},
		&cli.StringFlag{Name: "client-mutation-id", Usage: "Client mutation ID"},
		&cli.IntFlag{Name: "trade-quantity", Usage: "Trade quantity"},
		&cli.StringFlag{Name: "notes", Usage: "Notes"},
	}, true, func(cmd *cli.Command, req *clientv1.UpdateCollectionItemRequest) error {
		hasUpdate := strings.TrimSpace(cmd.String("file")) != ""
		setStringFlag(cmd, "id", &req.ItemID)
		if cmd.IsSet("quantity") {
			qty := cmd.Int("quantity")
			if qty < 1 {
				return cli.Exit("--quantity must be >= 1", 2)
			}
			v := int32(qty)
			req.Quantity = &v
			hasUpdate = true
		}
		if cmd.IsSet("condition") {
			cond, ok := parseCondition(cmd.String("condition"))
			if !ok {
				return cli.Exit("--condition must be near_mint|lightly_played|heavily_played|damaged", 2)
			}
			req.Condition = &cond
			hasUpdate = true
		}
		if cmd.IsSet("value") {
			v := cmd.Float64("value")
			req.Value = &v
			hasUpdate = true
		}
		if cmd.IsSet("pinned") {
			v := cmd.Bool("pinned")
			req.Pinned = &v
			hasUpdate = true
		}
		if cmd.IsSet("expected-updated-at") {
			if err := setTimeFlag(cmd, "expected-updated-at", &req.ExpectedUpdatedAt); err != nil {
				return err
			}
			hasUpdate = true
		}
		if cmd.IsSet("client-mutation-id") {
			req.ClientMutationID = strings.TrimSpace(cmd.String("client-mutation-id"))
			hasUpdate = true
		}
		if cmd.IsSet("trade-quantity") {
			v := int32(cmd.Int("trade-quantity"))
			req.TradeQuantity = &v
			hasUpdate = true
		}
		if cmd.IsSet("notes") {
			notes := strings.TrimSpace(cmd.String("notes"))
			req.Notes = &notes
			hasUpdate = true
		}
		if !hasUpdate {
			return cli.Exit("no updates provided", 2)
		}
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UpdateCollectionItemRequest) (any, error) {
		return c.UpdateCollectionItem(ctx, req)
	})
}

func newCollectionItemsAdjustCommand() *cli.Command {
	return newSDKCommand("adjust", "Adjust collection item quantity", []cli.Flag{
		&cli.StringFlag{Name: "collection-id", Usage: "Collection ID"},
		&cli.StringFlag{Name: "product-id", Usage: "Product ID"},
		&cli.StringFlag{Name: "condition", Usage: "Condition"},
		&cli.IntFlag{Name: "quantity-delta", Usage: "Quantity delta"},
		&cli.StringFlag{Name: "item-id", Usage: "Item ID"},
		&cli.StringFlag{Name: "client-mutation-id", Usage: "Client mutation ID"},
	}, true, func(cmd *cli.Command, req *clientv1.AdjustCollectionItemQuantityRequest) error {
		setStringFlag(cmd, "collection-id", &req.CollectionID)
		setStringFlag(cmd, "product-id", &req.ProductID)
		if cmd.IsSet("condition") {
			cond, ok := parseCondition(cmd.String("condition"))
			if !ok {
				return cli.Exit("--condition must be near_mint|lightly_played|heavily_played|damaged", 2)
			}
			req.Condition = cond
		}
		if cmd.IsSet("quantity-delta") {
			req.QuantityDelta = int32(cmd.Int("quantity-delta"))
		}
		setStringFlag(cmd, "item-id", &req.ItemID)
		setStringFlag(cmd, "client-mutation-id", &req.ClientMutationID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.AdjustCollectionItemQuantityRequest) (any, error) {
		return c.AdjustCollectionItemQuantity(ctx, req)
	})
}

func newCollectionItemsTransferCommand() *cli.Command {
	return newSDKCommand("transfer", "Transfer an item to another collection", []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: "Item ID", Required: true},
		&cli.StringFlag{Name: "destination-collection-id", Usage: "Destination collection ID"},
	}, true, func(cmd *cli.Command, req *clientv1.TransferCollectionItemRequest) error {
		setStringFlag(cmd, "id", &req.ItemID)
		setStringFlag(cmd, "destination-collection-id", &req.DestinationCollectionID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.TransferCollectionItemRequest) (any, error) {
		return c.TransferCollectionItem(ctx, req)
	})
}

func newCollectionItemsBatchGetCommand() *cli.Command {
	return newSDKCommand("batch-get", "Batch get collection items", []cli.Flag{
		repeatedIDsFlag("id", "Item ID (repeatable or comma-separated)"),
		&cli.BoolFlag{Name: "allow-partial", Usage: "Allow partial results"},
	}, true, func(cmd *cli.Command, req *clientv1.BatchGetCollectionItemsRequest) error {
		if cmd.IsSet("id") {
			req.ItemIDs = splitCSV(cmd.StringSlice("id"))
		}
		if cmd.IsSet("allow-partial") {
			req.AllowPartial = cmd.Bool("allow-partial")
		}
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.BatchGetCollectionItemsRequest) (any, error) {
		return c.BatchGetCollectionItems(ctx, req)
	})
}

func newCollectionItemsDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a collection item",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Item ID", Required: true},
			yesFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(cmd.String("id"))
			if err := confirmAction(cmd, "Delete", "collection item", id); err != nil {
				return err
			}
			if _, err := st.Service.DeleteCollectionItem(ctx, &clientv1.DeleteCollectionItemRequest{ItemID: id}); err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"deleted": id})
		},
	}
}

func parseCondition(v string) (clientv1.Condition, bool) {
	n := strings.ToLower(strings.TrimSpace(v))
	n = strings.ReplaceAll(n, "-", "_")
	switch n {
	case "", "unspecified":
		return clientv1.ConditionUnspecified, false
	case "near_mint", "nm":
		return clientv1.ConditionNearMint, true
	case "lightly_played", "lp":
		return clientv1.ConditionLightlyPlayed, true
	case "heavily_played", "hp":
		return clientv1.ConditionHeavilyPlayed, true
	case "damaged", "dmg":
		return clientv1.ConditionDamaged, true
	default:
		return clientv1.Condition(v), true
	}
}
