package family

import (
	"context"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/nnc/family-manager/libs/go/rpc"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	"github.com/nnc/family-manager/sdk/go/family/v1/familyv1connect"
)

type Member struct {
	UserID      string
	DisplayName string
	Email       string
}

type Client struct {
	public   familyv1connect.FamilyServiceClient
	internal familyv1connect.FamilyServiceClient
}

func New(publicURL, internalURL string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	hc := &http.Client{Timeout: timeout}
	opts := connect.WithInterceptors(rpc.ForwardRequestID())
	return &Client{
		public:   familyv1connect.NewFamilyServiceClient(hc, strings.TrimSuffix(publicURL, "/"), opts),
		internal: familyv1connect.NewFamilyServiceClient(hc, strings.TrimSuffix(internalURL, "/"), opts),
	}
}

func (c *Client) ListMembers(ctx context.Context, bearer, familyID string) ([]Member, error) {
	req := connect.NewRequest(&familyv1.ListMembersRequest{FamilyId: familyID})
	req.Header().Set("Authorization", "Bearer "+bearer)
	res, err := c.public.ListMembers(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make([]Member, 0, len(res.Msg.GetMembers()))
	for _, m := range res.Msg.GetMembers() {
		out = append(out, Member{UserID: m.GetUserId(), DisplayName: m.GetDisplayName(), Email: m.GetEmail()})
	}
	return out, nil
}

func (c *Client) IsMember(ctx context.Context, familyID, userID string) (bool, error) {
	res, err := c.internal.CheckMembership(ctx, connect.NewRequest(&familyv1.CheckMembershipRequest{
		UserId: userID, FamilyId: familyID,
	}))
	if err != nil {
		return false, err
	}
	return res.Msg.GetIsMember(), nil
}

func (c *Client) FamilyOf(ctx context.Context, userID string) (string, error) {
	res, err := c.internal.GetUserMembership(ctx, connect.NewRequest(&familyv1.GetUserMembershipRequest{UserId: userID}))
	if err != nil {
		return "", err
	}
	if !res.Msg.GetInFamily() {
		return "", nil
	}
	return res.Msg.GetFamilyId(), nil
}
