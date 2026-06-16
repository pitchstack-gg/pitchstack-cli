package commands

import (
	"context"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newPricingCommand() *cli.Command {
	return &cli.Command{
		Name:  "pricing",
		Usage: "Manage product pricing",
		Commands: []*cli.Command{
			newSDKCommand("get", "Get current product price", []cli.Flag{
				&cli.StringFlag{Name: "product-id", Usage: "Product ID"},
				&cli.StringFlag{Name: "source", Usage: "Price source"},
			}, true, func(cmd *cli.Command, req *clientv1.GetProductPriceRequest) error {
				setStringFlag(cmd, "product-id", &req.ProductID)
				setStringFlag(cmd, "source", &req.Source)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetProductPriceRequest) (any, error) {
				return c.GetProductPrice(ctx, req)
			}),
			newSDKCommand("history", "Get product price history", []cli.Flag{
				&cli.StringFlag{Name: "product-id", Usage: "Product ID"},
				&cli.StringFlag{Name: "source", Usage: "Price source"},
				&cli.StringFlag{Name: "start-date", Usage: "Start date"},
				&cli.StringFlag{Name: "end-date", Usage: "End date"},
				&cli.IntFlag{Name: "limit", Usage: "Limit"},
			}, true, func(cmd *cli.Command, req *clientv1.GetProductPriceHistoryRequest) error {
				setStringFlag(cmd, "product-id", &req.ProductID)
				setStringFlag(cmd, "source", &req.Source)
				setStringFlag(cmd, "start-date", &req.StartDate)
				setStringFlag(cmd, "end-date", &req.EndDate)
				setInt32Flag(cmd, "limit", &req.Limit)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetProductPriceHistoryRequest) (any, error) {
				return c.GetProductPriceHistory(ctx, req)
			}),
			newSDKCommand("batch", "Get product prices in bulk", []cli.Flag{
				repeatedIDsFlag("product-id", "Product ID (repeatable or comma-separated)"),
				&cli.StringFlag{Name: "source", Usage: "Price source"},
			}, true, func(cmd *cli.Command, req *clientv1.BatchGetProductPricesRequest) error {
				if cmd.IsSet("product-id") {
					req.ProductIDs = splitCSV(cmd.StringSlice("product-id"))
				}
				setStringFlag(cmd, "source", &req.Source)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.BatchGetProductPricesRequest) (any, error) {
				return c.BatchGetProductPrices(ctx, req)
			}),
			newPriceWatchesCommand(),
		},
	}
}

func newPriceWatchesCommand() *cli.Command {
	return &cli.Command{
		Name:  "watches",
		Usage: "Manage product price watches",
		Commands: []*cli.Command{
			newSDKCommand("list", "List product price watches", []cli.Flag{
				&cli.BoolFlag{Name: "active-only", Usage: "Only active watches"},
				repeatedIDsFlag("product-id", "Product ID (repeatable or comma-separated)"),
			}, true, func(cmd *cli.Command, req *clientv1.ListProductPriceWatchesRequest) error {
				setBoolFlag(cmd, "active-only", &req.ActiveOnly)
				if cmd.IsSet("product-id") {
					req.ProductIDs = splitCSV(cmd.StringSlice("product-id"))
				}
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListProductPriceWatchesRequest) (any, error) {
				return c.ListProductPriceWatches(ctx, req)
			}),
			newSDKCommand("create", "Create a product price watch", priceWatchFlags(false), true, applyCreatePriceWatchFlags, func(ctx context.Context, c *clientv1.Client, req *clientv1.CreateProductPriceWatchRequest) (any, error) {
				return c.CreateProductPriceWatch(ctx, req)
			}),
			newSDKCommand("update", "Update a product price watch", priceWatchFlags(true), true, applyUpdatePriceWatchFlags, func(ctx context.Context, c *clientv1.Client, req *clientv1.UpdateProductPriceWatchRequest) (any, error) {
				return c.UpdateProductPriceWatch(ctx, req)
			}),
			newSDKCommand("delete", "Delete a product price watch", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Watch ID"}, yesFlag()}, true, func(cmd *cli.Command, req *clientv1.DeleteProductPriceWatchRequest) error {
				setStringFlag(cmd, "id", &req.WatchID)
				return confirmAction(cmd, "Delete", "price watch", req.WatchID)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.DeleteProductPriceWatchRequest) (any, error) {
				return c.DeleteProductPriceWatch(ctx, req)
			}),
		},
	}
}

func priceWatchFlags(withID bool) []cli.Flag {
	flags := []cli.Flag{}
	if withID {
		flags = append(flags, &cli.StringFlag{Name: "id", Usage: "Watch ID"})
	} else {
		flags = append(flags, &cli.StringFlag{Name: "product-id", Usage: "Product ID"})
	}
	return append(flags,
		&cli.StringFlag{Name: "source", Usage: "Price source"},
		&cli.StringFlag{Name: "direction", Usage: "Direction"},
		&cli.Float64Flag{Name: "absolute-change", Usage: "Absolute change threshold"},
		&cli.Float64Flag{Name: "percent-change", Usage: "Percent change threshold"},
		&cli.StringFlag{Name: "period", Usage: "Watch period"},
		&cli.BoolFlag{Name: "active", Usage: "Active"},
	)
}

func applyCreatePriceWatchFlags(cmd *cli.Command, req *clientv1.CreateProductPriceWatchRequest) error {
	setStringFlag(cmd, "product-id", &req.ProductID)
	setStringFlag(cmd, "source", &req.Source)
	setStringFlag(cmd, "direction", &req.Direction)
	setFloat64Flag(cmd, "absolute-change", &req.AbsoluteChange)
	setFloat64Flag(cmd, "percent-change", &req.PercentChange)
	setStringFlag(cmd, "period", &req.Period)
	return nil
}

func applyUpdatePriceWatchFlags(cmd *cli.Command, req *clientv1.UpdateProductPriceWatchRequest) error {
	setStringFlag(cmd, "id", &req.WatchID)
	setStringPtrFlag(cmd, "direction", &req.Direction)
	setFloat64Flag(cmd, "absolute-change", &req.AbsoluteChange)
	setFloat64Flag(cmd, "percent-change", &req.PercentChange)
	setStringPtrFlag(cmd, "period", &req.Period)
	setBoolFlag(cmd, "active", &req.Active)
	return nil
}
