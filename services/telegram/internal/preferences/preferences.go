package preferences

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	"github.com/nnc/family-manager/sdk/go/family/v1/familyv1connect"
)

type Client struct {
	rpc familyv1connect.FamilyServiceClient
}

func New(httpClient *http.Client, addr string) *Client {
	return &Client{rpc: familyv1connect.NewFamilyServiceClient(httpClient, addr)}
}

func (c *Client) Locale(ctx context.Context, accessToken string) (string, error) {
	req := connect.NewRequest(&familyv1.GetMySettingsRequest{})
	req.Header().Set("Authorization", "Bearer "+accessToken)

	res, err := c.rpc.GetMySettings(ctx, req)
	if err != nil {
		return "", err
	}
	return res.Msg.GetSettings().GetLocale(), nil
}

func (c *Client) SetLocale(ctx context.Context, accessToken, locale string) (string, error) {
	req := connect.NewRequest(&familyv1.UpdateMySettingsRequest{Locale: locale})
	req.Header().Set("Authorization", "Bearer "+accessToken)

	res, err := c.rpc.UpdateMySettings(ctx, req)
	if err != nil {
		return "", err
	}
	return res.Msg.GetSettings().GetLocale(), nil
}
