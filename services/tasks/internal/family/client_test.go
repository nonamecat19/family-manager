package family

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	"github.com/nnc/family-manager/sdk/go/family/v1/familyv1connect"
)

type verifier struct{}

func (verifier) Verify(_ context.Context, token string) (*fmauth.Claims, error) {
	if token != "good" {
		return nil, fmauth.ErrInvalidToken
	}
	return &fmauth.Claims{UserID: "u1", FamilyID: "fam-1"}, nil
}

type fakeFamily struct {
	familyv1connect.UnimplementedFamilyServiceHandler
}

func (fakeFamily) ListMembers(ctx context.Context, req *connect.Request[familyv1.ListMembersRequest]) (*connect.Response[familyv1.ListMembersResponse], error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return nil, err
	}
	if claims.FamilyID != req.Msg.GetFamilyId() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("not your family"))
	}
	return connect.NewResponse(&familyv1.ListMembersResponse{Members: []*familyv1.Member{
		{UserId: "u1", DisplayName: "Ann", Email: "ann@example.com"},
		{UserId: "u2", Email: "bob@example.com"},
	}}), nil
}

func (fakeFamily) CheckMembership(_ context.Context, req *connect.Request[familyv1.CheckMembershipRequest]) (*connect.Response[familyv1.CheckMembershipResponse], error) {
	return connect.NewResponse(&familyv1.CheckMembershipResponse{IsMember: req.Msg.GetUserId() == "u1" && req.Msg.GetFamilyId() == "fam-1"}), nil
}

func (fakeFamily) GetUserMembership(_ context.Context, req *connect.Request[familyv1.GetUserMembershipRequest]) (*connect.Response[familyv1.GetUserMembershipResponse], error) {
	if req.Msg.GetUserId() != "u1" {
		return connect.NewResponse(&familyv1.GetUserMembershipResponse{}), nil
	}
	return connect.NewResponse(&familyv1.GetUserMembershipResponse{InFamily: true, FamilyId: "fam-1"}), nil
}

func servers(t *testing.T) *Client {
	public := http.NewServeMux()
	p, ph := familyv1connect.NewFamilyServiceHandler(fakeFamily{}, connect.WithInterceptors(fmauth.Interceptor(verifier{})))
	public.Handle(p, ph)
	internal := http.NewServeMux()
	i, ih := familyv1connect.NewFamilyServiceHandler(fakeFamily{})
	internal.Handle(i, ih)
	pub := httptest.NewServer(public)
	in := httptest.NewServer(internal)
	t.Cleanup(pub.Close)
	t.Cleanup(in.Close)
	return New(pub.URL+"/", in.URL, 0)
}

func TestListMembersGoesThroughThePublicAuthInterceptor(t *testing.T) {
	c := servers(t)
	members, err := c.ListMembers(context.Background(), "good", "fam-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 || members[0].DisplayName != "Ann" || members[1].Email != "bob@example.com" {
		t.Fatalf("%+v", members)
	}
	if _, err := c.ListMembers(context.Background(), "bad", "fam-1"); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("bad token: %v", err)
	}
	if _, err := c.ListMembers(context.Background(), "good", "fam-2"); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("other family: %v", err)
	}
}

func TestInternalMembershipChecksNeedNoToken(t *testing.T) {
	c := servers(t)
	ok, err := c.IsMember(context.Background(), "fam-1", "u1")
	if err != nil || !ok {
		t.Fatalf("%v %v", ok, err)
	}
	ok, _ = c.IsMember(context.Background(), "fam-1", "gone")
	if ok {
		t.Fatal("a removed member must not check out")
	}
	fam, err := c.FamilyOf(context.Background(), "u1")
	if err != nil || fam != "fam-1" {
		t.Fatalf("%q %v", fam, err)
	}
	fam, _ = c.FamilyOf(context.Background(), "nobody")
	if fam != "" {
		t.Fatalf("%q", fam)
	}
}
