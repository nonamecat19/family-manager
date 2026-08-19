// Package family is a thin client over services/family's internal listener, used to resolve
// the family_id claim while minting a token.
//
// This is the one place services/auth depends on a sibling, and it does so through the
// contract in libs/proto — never by reading family's tables. The call happens at login and
// refresh, not per request.
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

// Client resolves a user's household.
type Client struct {
	client familyv1connect.FamilyServiceClient
}

// New dials the internal listener, e.g. http://family:9090. The timeout is deliberately
// short: this call sits in the login path, and a slow household lookup must not become a slow
// login — the caller treats a failure as "no family yet".
func New(baseURL string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	return &Client{
		client: familyv1connect.NewFamilyServiceClient(
			&http.Client{Timeout: timeout}, baseURL,
			// The household lookup happens inside a login. Forwarding the id makes the two
			// services' logs for that login findable as one thing.
			connect.WithInterceptors(rpc.ForwardRequestID()),
		),
	}
}

// FamilyOf returns the user's family id, or "" when they are in none.
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
