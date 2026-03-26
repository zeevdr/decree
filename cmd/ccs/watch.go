package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	pb "github.com/zeevdr/central-config-service/api/centralconfig/v1"
	"google.golang.org/grpc/metadata"
)

var watchCmd = &cobra.Command{
	Use:   "watch <tenant-id> [field-paths...]",
	Short: "Stream live config changes (like tail -f)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tenantID := args[0]
		fieldPaths := args[1:]

		conn, err := dialServer()
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close() }()

		ctx := cmd.Context()
		// Inject auth metadata.
		pairs := make([]string, 0, 6)
		if flagSubject != "" {
			pairs = append(pairs, "x-subject", flagSubject)
		}
		if flagRole != "" {
			pairs = append(pairs, "x-role", flagRole)
		}
		if flagTenantID != "" {
			pairs = append(pairs, "x-tenant-id", flagTenantID)
		}
		if flagToken != "" {
			pairs = append(pairs, "authorization", "Bearer "+flagToken)
		}
		if len(pairs) > 0 {
			ctx = metadata.AppendToOutgoingContext(ctx, pairs...)
		}

		rpc := pb.NewConfigServiceClient(conn)
		stream, err := rpc.Subscribe(ctx, &pb.SubscribeRequest{
			TenantId:   tenantID,
			FieldPaths: fieldPaths,
		})
		if err != nil {
			return err
		}

		for {
			resp, err := stream.Recv()
			if err != nil {
				return err
			}
			c := resp.Change
			ts := time.Now().Format("15:04:05")
			old, new_ := "<null>", "<null>"
			if c.OldValue != nil {
				old = *c.OldValue
			}
			if c.NewValue != nil {
				new_ = *c.NewValue
			}
			fmt.Printf("[%s] v%d %s: %q → %q (by %s)\n", ts, c.Version, c.FieldPath, old, new_, c.ChangedBy)
		}
	},
}
