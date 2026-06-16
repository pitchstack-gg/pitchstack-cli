package commands

import (
	"context"
	"fmt"
	"os"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newGroupsCommand() *cli.Command {
	return &cli.Command{
		Name:  "groups",
		Usage: "Manage groups",
		Commands: []*cli.Command{
			newSDKCommand("create", "Create a group", []cli.Flag{
				&cli.StringFlag{Name: "slug", Usage: "Group slug"},
				&cli.StringFlag{Name: "name", Usage: "Group name"},
				&cli.StringFlag{Name: "description", Usage: "Group description"},
				&cli.StringFlag{Name: "visibility", Usage: "Visibility"},
			}, true, func(cmd *cli.Command, req *clientv1.CreateGroupRequest) error {
				setStringFlag(cmd, "slug", &req.Slug)
				setStringFlag(cmd, "name", &req.Name)
				setStringFlag(cmd, "description", &req.Description)
				if cmd.IsSet("visibility") {
					req.Visibility = parseVisibility(cmd.String("visibility"))
				}
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CreateGroupRequest) (any, error) {
				return c.CreateGroup(ctx, req)
			}),
			newSDKCommand("update", "Update a group", []cli.Flag{
				&cli.StringFlag{Name: "id", Usage: "Group ID"},
				&cli.StringFlag{Name: "name", Usage: "Group name"},
				&cli.StringFlag{Name: "description", Usage: "Group description"},
				&cli.StringFlag{Name: "visibility", Usage: "Visibility"},
			}, true, func(cmd *cli.Command, req *clientv1.UpdateGroupRequest) error {
				setStringFlag(cmd, "id", &req.GroupID)
				setStringPtrFlag(cmd, "name", &req.Name)
				setStringPtrFlag(cmd, "description", &req.Description)
				if cmd.IsSet("visibility") {
					v := parseVisibility(cmd.String("visibility"))
					req.Visibility = &v
				}
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UpdateGroupRequest) (any, error) {
				return c.UpdateGroup(ctx, req)
			}),
			newSDKCommand("delete", "Delete a group", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Group ID"}, yesFlag()}, true, func(cmd *cli.Command, req *clientv1.DeleteGroupRequest) error {
				setStringFlag(cmd, "id", &req.GroupID)
				return confirmAction(cmd, "Delete", "group", req.GroupID)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.DeleteGroupRequest) (any, error) {
				return c.DeleteGroup(ctx, req)
			}),
			newSDKCommand("get", "Get a group", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Group ID"}}, true, func(cmd *cli.Command, req *clientv1.GetGroupRequest) error {
				setStringFlag(cmd, "id", &req.GroupID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetGroupRequest) (any, error) {
				return c.GetGroup(ctx, req)
			}),
			newSDKCommand("list", "List groups", pageFlags(), true, func(cmd *cli.Command, req *clientv1.ListGroupsRequest) error {
				setPageFlags(cmd, &req.PageSize, &req.NextToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListGroupsRequest) (any, error) {
				return c.ListGroups(ctx, req)
			}),
			newSDKCommand("search", "Search groups", append(pageFlags(), &cli.StringFlag{Name: "q", Usage: "Search query"}), true, func(cmd *cli.Command, req *clientv1.SearchGroupsRequest) error {
				setStringFlag(cmd, "q", &req.SearchTerm)
				setPageFlags(cmd, &req.PageSize, &req.NextToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.SearchGroupsRequest) (any, error) {
				return c.SearchGroups(ctx, req)
			}),
			newSDKCommand("mine", "List joined groups", pageFlags(), true, func(cmd *cli.Command, req *clientv1.ListMyGroupsRequest) error {
				setPageFlags(cmd, &req.PageSize, &req.NextToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListMyGroupsRequest) (any, error) {
				return c.ListMyGroups(ctx, req)
			}),
			newGroupMembersCommand(),
			newGroupInvitesCommand(),
			newGroupAvatarCommand(),
		},
	}
}

func newGroupMembersCommand() *cli.Command {
	return &cli.Command{
		Name:  "members",
		Usage: "Manage group members",
		Commands: []*cli.Command{
			newSDKCommand("list", "List group members", append(pageFlags(), &cli.StringFlag{Name: "group-id", Usage: "Group ID"}), true, func(cmd *cli.Command, req *clientv1.ListGroupMembersRequest) error {
				setStringFlag(cmd, "group-id", &req.GroupID)
				setPageFlags(cmd, &req.PageSize, &req.NextToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListGroupMembersRequest) (any, error) {
				return c.ListGroupMembers(ctx, req)
			}),
			newSDKCommand("add", "Add a group member", []cli.Flag{
				&cli.StringFlag{Name: "group-id", Usage: "Group ID"},
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "role", Usage: "Member role"},
			}, true, func(cmd *cli.Command, req *clientv1.AddGroupMemberRequest) error {
				setStringFlag(cmd, "group-id", &req.GroupID)
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "role", &req.Role)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.AddGroupMemberRequest) (any, error) {
				return c.AddGroupMember(ctx, req)
			}),
			newSDKCommand("remove", "Remove a group member", []cli.Flag{
				&cli.StringFlag{Name: "group-id", Usage: "Group ID"},
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				yesFlag(),
			}, true, func(cmd *cli.Command, req *clientv1.RemoveGroupMemberRequest) error {
				setStringFlag(cmd, "group-id", &req.GroupID)
				setStringFlag(cmd, "user-id", &req.UserID)
				return confirmAction(cmd, "Remove", "group member", req.UserID)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.RemoveGroupMemberRequest) (any, error) {
				return c.RemoveGroupMember(ctx, req)
			}),
			newSDKCommand("role", "Update a group member role", []cli.Flag{
				&cli.StringFlag{Name: "group-id", Usage: "Group ID"},
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "role", Usage: "Member role"},
			}, true, func(cmd *cli.Command, req *clientv1.UpdateGroupMemberRoleRequest) error {
				setStringFlag(cmd, "group-id", &req.GroupID)
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "role", &req.Role)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UpdateGroupMemberRoleRequest) (any, error) {
				return c.UpdateGroupMemberRole(ctx, req)
			}),
		},
	}
}

func newGroupInvitesCommand() *cli.Command {
	return &cli.Command{
		Name:  "invites",
		Usage: "Manage group invites",
		Commands: []*cli.Command{
			newSDKCommand("create", "Create a group invite", []cli.Flag{
				&cli.StringFlag{Name: "group-id", Usage: "Group ID"},
				&cli.StringFlag{Name: "role", Usage: "Invite role"},
				&cli.StringFlag{Name: "expires-at", Usage: "Expiration time (RFC3339)"},
			}, true, func(cmd *cli.Command, req *clientv1.CreateGroupInviteRequest) error {
				setStringFlag(cmd, "group-id", &req.GroupID)
				setStringFlag(cmd, "role", &req.Role)
				return setTimeFlag(cmd, "expires-at", &req.ExpiresAt)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CreateGroupInviteRequest) (any, error) {
				return c.CreateGroupInvite(ctx, req)
			}),
			newSDKCommand("accept", "Accept a group invite", []cli.Flag{
				&cli.StringFlag{Name: "token", Usage: "Invite token"},
				&cli.StringFlag{Name: "group-id", Usage: "Group ID"},
			}, true, func(cmd *cli.Command, req *clientv1.AcceptGroupInviteRequest) error {
				setStringFlag(cmd, "token", &req.Token)
				setStringFlag(cmd, "group-id", &req.GroupID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.AcceptGroupInviteRequest) (any, error) {
				return c.AcceptGroupInvite(ctx, req)
			}),
			newSDKCommand("revoke", "Revoke a group invite", []cli.Flag{
				&cli.StringFlag{Name: "group-id", Usage: "Group ID"},
				&cli.StringFlag{Name: "invite-id", Usage: "Invite ID"},
				yesFlag(),
			}, true, func(cmd *cli.Command, req *clientv1.RevokeGroupInviteRequest) error {
				setStringFlag(cmd, "group-id", &req.GroupID)
				setStringFlag(cmd, "invite-id", &req.InviteID)
				return confirmAction(cmd, "Revoke", "group invite", req.InviteID)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.RevokeGroupInviteRequest) (any, error) {
				return c.RevokeGroupInvite(ctx, req)
			}),
		},
	}
}

func newGroupAvatarCommand() *cli.Command {
	return &cli.Command{
		Name:  "avatar",
		Usage: "Manage group avatar",
		Commands: []*cli.Command{
			newSDKCommand("begin", "Begin group avatar upload", []cli.Flag{
				&cli.StringFlag{Name: "group-id", Usage: "Group ID"},
				&cli.StringFlag{Name: "content-type", Usage: "Content type"},
				&cli.IntFlag{Name: "content-length", Usage: "Content length"},
			}, true, func(cmd *cli.Command, req *clientv1.BeginGroupAvatarUploadRequest) error {
				setStringFlag(cmd, "group-id", &req.GroupID)
				setStringFlag(cmd, "content-type", &req.ContentType)
				setInt64Flag(cmd, "content-length", &req.ContentLength)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.BeginGroupAvatarUploadRequest) (any, error) {
				return c.BeginGroupAvatarUpload(ctx, req)
			}),
			newSDKCommand("complete", "Complete group avatar upload", []cli.Flag{
				&cli.StringFlag{Name: "group-id", Usage: "Group ID"},
				&cli.StringFlag{Name: "upload-id", Usage: "Upload ID"},
			}, true, func(cmd *cli.Command, req *clientv1.CompleteGroupAvatarUploadRequest) error {
				setStringFlag(cmd, "group-id", &req.GroupID)
				setStringFlag(cmd, "upload-id", &req.UploadID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CompleteGroupAvatarUploadRequest) (any, error) {
				return c.CompleteGroupAvatarUpload(ctx, req)
			}),
			newGroupAvatarUploadCommand(),
		},
	}
}

func newGroupAvatarUploadCommand() *cli.Command {
	return &cli.Command{
		Name:  "upload",
		Usage: "Upload and apply a group avatar file",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "group-id", Usage: "Group ID", Required: true},
			&cli.StringFlag{Name: "file", Usage: "File to upload", Required: true},
			&cli.StringFlag{Name: "content-type", Usage: "Content type", Value: "application/octet-stream"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			c, err := st.Service.AuthenticatedClient()
			if err != nil {
				return err
			}
			filePath := cmd.String("file")
			info, err := os.Stat(filePath)
			if err != nil {
				return err
			}
			contentLength := info.Size()
			contentType := cmd.String("content-type")
			begin, err := c.BeginGroupAvatarUpload(ctx, &clientv1.BeginGroupAvatarUploadRequest{
				GroupID:       cmd.String("group-id"),
				ContentType:   contentType,
				ContentLength: &contentLength,
			})
			if err != nil {
				return err
			}
			if err := uploadFileToSignedURL(ctx, st.Service.HTTPClient(), begin.UploadURL, begin.RequiredHeaders, filePath, contentType); err != nil {
				return err
			}
			resp, err := c.CompleteGroupAvatarUpload(ctx, &clientv1.CompleteGroupAvatarUploadRequest{
				GroupID:  cmd.String("group-id"),
				UploadID: begin.UploadID,
			})
			if err != nil {
				return err
			}
			if resp == nil {
				return fmt.Errorf("complete upload returned nil response")
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}
