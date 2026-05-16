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
			newCollectionsDeleteCommand(),
			newCollectionsExportCommand(),
			newCollectionsImportCommand(),
			newCollectionsValuationCommand(),
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
	return newSDKCommand("stop-share", "Remove all explicit collection shares", []cli.Flag{&cli.StringFlag{Name: "collection-id", Usage: "Collection ID"}}, true, func(cmd *cli.Command, req *clientv1.StopCollectionShareRequest) error {
		setStringFlag(cmd, "collection-id", &req.CollectionID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.StopCollectionShareRequest) (any, error) {
		return c.StopCollectionShare(ctx, req)
	})
}

func newCollectionsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List collections",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "scope", Usage: "Scope (owned|shared|accessible)"},
			&cli.StringFlag{Name: "user-id", Usage: "Filter by user ID"},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.ListCollectionsRequest{
				Scope:     parseCollectionScope(cmd.String("scope")),
				UserID:    strings.TrimSpace(cmd.String("user-id")),
				NextToken: strings.TrimSpace(cmd.String("next-token")),
			}
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				req.PageSize = &ps
			}

			resp, err := st.Service.ListCollections(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(cmd.String("id"))
			resp, err := st.Service.GetCollection(ctx, &clientv1.GetCollectionRequest{CollectionID: id})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionsHistoryCommand() *cli.Command {
	return newSDKCommand("history", "Get collection history", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Collection ID"}}, true, func(cmd *cli.Command, req *clientv1.GetCollectionHistoryRequest) error {
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
	return &cli.Command{
		Name:  "create",
		Usage: "Create a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Collection name", Required: true},
			&cli.StringFlag{Name: "type", Usage: "Collection type (binder|wantlist|tradelist|list)", Required: true},
			&cli.StringFlag{Name: "description", Usage: "Collection description"},
			&cli.StringFlag{Name: "visibility", Usage: "Visibility (private|shared|public)", Value: "private"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			collectionType := parseCollectionType(cmd.String("type"))
			if collectionType == clientv1.CollectionTypeUnspecified {
				return cli.Exit("--type must be binder|wantlist|tradelist|list", 2)
			}

			req := &clientv1.CreateCollectionRequest{
				Name:           strings.TrimSpace(cmd.String("name")),
				CollectionType: collectionType,
				Description:    strings.TrimSpace(cmd.String("description")),
				Visibility:     parseVisibility(cmd.String("visibility")),
			}

			resp, err := st.Service.CreateCollection(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
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
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			id := strings.TrimSpace(cmd.String("id"))
			var anyUpdate bool

			var updateReq clientv1.UpdateCollectionRequest
			updateReq.CollectionID = id
			if cmd.IsSet("name") {
				v := strings.TrimSpace(cmd.String("name"))
				updateReq.Name = &v
				anyUpdate = true
			}
			if cmd.IsSet("description") {
				v := strings.TrimSpace(cmd.String("description"))
				updateReq.Description = &v
				anyUpdate = true
			}

			if anyUpdate {
				if _, err := st.Service.UpdateCollection(ctx, &updateReq); err != nil {
					return err
				}
			}

			if cmd.IsSet("visibility") {
				v := parseVisibility(cmd.String("visibility"))
				if v == clientv1.VisibilityLevelUnspecified {
					return cli.Exit("--visibility must be private|shared|public", 2)
				}
				if _, err := st.Service.UpdateCollectionVisibility(ctx, &clientv1.UpdateCollectionVisibilityRequest{
					CollectionID: id,
					Visibility:   &v,
				}); err != nil {
					return err
				}
				anyUpdate = true
			}

			if !anyUpdate {
				return cli.Exit("no updates provided", 2)
			}

			resp, err := st.Service.GetCollection(ctx, &clientv1.GetCollectionRequest{CollectionID: id})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionsDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(cmd.String("id"))
			if _, err := st.Service.DeleteCollection(ctx, &clientv1.DeleteCollectionRequest{CollectionID: id}); err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"deleted": id})
		},
	}
}

func newCollectionsExportCommand() *cli.Command {
	return newSDKCommand("export", "Export a collection", append(pageFlags(), &cli.StringFlag{Name: "id", Usage: "Collection ID"}), true, func(cmd *cli.Command, req *clientv1.ExportCollectionRequest) error {
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
			req.CollectionType = parseCollectionType(cmd.String("type"))
		}
		setStringFlag(cmd, "description", &req.Description)
		if cmd.IsSet("visibility") {
			req.Visibility = parseVisibility(cmd.String("visibility"))
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
	return &cli.Command{
		Name:  "list",
		Usage: "List collection items",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "collection-id", Usage: "Collection ID", Required: true},
			&cli.StringFlag{Name: "card-id", Usage: "Filter by card ID"},
			&cli.StringFlag{Name: "printing-id", Usage: "Filter by printing ID"},
			&cli.StringFlag{Name: "product-id", Usage: "Filter by product ID"},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			req := &clientv1.ListCollectionItemsRequest{
				CollectionID: strings.TrimSpace(cmd.String("collection-id")),
				CardID:       strings.TrimSpace(cmd.String("card-id")),
				PrintingID:   strings.TrimSpace(cmd.String("printing-id")),
				ProductID:    strings.TrimSpace(cmd.String("product-id")),
				NextToken:    strings.TrimSpace(cmd.String("next-token")),
			}
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				req.PageSize = &ps
			}
			resp, err := st.Service.ListCollectionItems(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionItemsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a collection item",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Item ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(cmd.String("id"))
			resp, err := st.Service.GetCollectionItem(ctx, &clientv1.GetCollectionItemRequest{ItemID: id})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionItemsAddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add an item to a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "collection-id", Usage: "Collection ID", Required: true},
			&cli.StringFlag{Name: "product-id", Usage: "Product ID", Required: true},
			&cli.IntFlag{Name: "quantity", Usage: "Quantity (>= 1)", Value: 1},
			&cli.StringFlag{Name: "condition", Usage: "Condition (near_mint|lightly_played|heavily_played|damaged)", Value: "near_mint"},
			&cli.Float64Flag{Name: "value", Usage: "Optional value"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			qty := cmd.Int("quantity")
			if qty < 1 {
				return cli.Exit("--quantity must be >= 1", 2)
			}
			cond, ok := parseCondition(cmd.String("condition"))
			if !ok {
				return cli.Exit("--condition must be near_mint|lightly_played|heavily_played|damaged", 2)
			}

			req := &clientv1.CreateCollectionItemRequest{
				CollectionID: strings.TrimSpace(cmd.String("collection-id")),
				ProductID:    strings.TrimSpace(cmd.String("product-id")),
				Quantity:     int32(qty),
				Condition:    cond,
				Value:        float64Ptr(cmd, "value"),
			}

			resp, err := st.Service.CreateCollectionItem(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionItemsUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a collection item",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Item ID", Required: true},
			&cli.IntFlag{Name: "quantity", Usage: "Quantity (>= 1)"},
			&cli.StringFlag{Name: "condition", Usage: "Condition (near_mint|lightly_played|heavily_played|damaged)"},
			&cli.Float64Flag{Name: "value", Usage: "Value"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.UpdateCollectionItemRequest{
				ItemID: strings.TrimSpace(cmd.String("id")),
			}
			var anyUpdate bool

			if cmd.IsSet("quantity") {
				qty := cmd.Int("quantity")
				if qty < 1 {
					return cli.Exit("--quantity must be >= 1", 2)
				}
				q := int32(qty)
				req.Quantity = &q
				anyUpdate = true
			}
			if cmd.IsSet("condition") {
				cond, ok := parseCondition(cmd.String("condition"))
				if !ok {
					return cli.Exit("--condition must be near_mint|lightly_played|heavily_played|damaged", 2)
				}
				req.Condition = &cond
				anyUpdate = true
			}
			if cmd.IsSet("value") {
				req.Value = float64Ptr(cmd, "value")
				anyUpdate = true
			}

			if !anyUpdate {
				return cli.Exit("no updates provided", 2)
			}

			resp, err := st.Service.UpdateCollectionItem(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
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
		&cli.StringFlag{Name: "id", Usage: "Item ID"},
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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(cmd.String("id"))
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
