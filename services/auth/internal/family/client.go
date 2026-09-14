package family

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/nnc/family-manager/libs/go/rpc"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	"github.com/nnc/family-manager/sdk/go/family/v1/familyv1connect"
)

type Client struct {
	client familyv1connect.FamilyServiceClient
}

func New(baseURL string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	return &Client{
		client: familyv1connect.NewFamilyServiceClient(
			&http.Client{Timeout: timeout}, baseURL,
			connect.WithInterceptors(rpc.ForwardRequestID()),
		),
	}
}

func (c *Client) FamilyOf(ctx context.Context, userID string) (string, error) {
	res, err := c.client.GetUserMembership(ctx,
		connect.NewRequest(&familyv1.GetUserMembershipRequest{UserId: userID}))
	if err != nil {
		return "", err
	}
	if !res.Msg.GetInFamily() {
		return "", nil
	}
	return res.Msg.GetFamilyId(), nil
}
