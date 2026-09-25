package family

import (
	"context"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	"github.com/nnc/family-manager/sdk/go/family/v1/familyv1connect"
	"github.com/nnc/family-manager/services/telegram/internal/bot"
	"github.com/nnc/family-manager/services/telegram/internal/i18n"
)

const intro = string(i18n.FamilyIntro)

type client struct {
	rpc familyv1connect.FamilyServiceClient
}

func Bot(httpClient *http.Client, addr string) bot.Options {
	c := &client{rpc: familyv1connect.NewFamilyServiceClient(httpClient, addr)}

	return bot.Options{
		Intro: intro,
		Home:  c.home,
		Commands: []bot.Command{
			{Name: "family", Help: string(i18n.FamilyOverview), Run: c.overview},
			{Name: "members", Help: string(i18n.FamilyMembers), Run: c.members},
			{Name: "invite", Args: "<email>", Help: string(i18n.FamilyInvites), Run: c.invite},
			{Name: "invitations", Help: string(i18n.FamilyInvites), Run: c.invitations},
		},
		Callbacks: []bot.Callback{
			{Prefix: "fam", Run: c.overview},
			{Prefix: "members", Run: c.members},
			{Prefix: "invites", Run: c.invitations},
		},
	}
}

func (c *client) home(ctx context.Context, cc *bot.Context) (string, bot.Keyboard, error) {
	text := bot.Lines(
		bot.Bold(cc.T(i18n.FamilyTitle)),
		"",
		bot.Italic(cc.T(i18n.FamilyHint)),
	)
	keyboard := bot.Keyboard{
		bot.Row(bot.Data(cc.T(i18n.FamilyOverview), "fam"), bot.Data(cc.T(i18n.FamilyMembers), "members")),
		bot.Row(bot.Data(cc.T(i18n.FamilyInvites), "invites")),
	}
	return text, keyboard, nil
}

func (c *client) overview(ctx context.Context, cc *bot.Context) error {
	req := connect.NewRequest(&familyv1.GetFamilyRequest{})
	cc.Authorize(req)

	res, err := c.rpc.GetFamily(ctx, req)
	if err != nil {
		return err
	}

	fam := res.Msg.GetFamily()
	return cc.Show(ctx, bot.Lines(
		bot.Bold("👪 "+bot.Esc(fam.GetName())),
		bot.Italic(cc.T(i18n.FamilyMemberCount, len(res.Msg.GetMembers()))),
		"",
		memberList(cc, res.Msg.GetMembers()),
	), backOnly(cc))
}

func (c *client) members(ctx context.Context, cc *bot.Context) error {
	req := connect.NewRequest(&familyv1.ListMembersRequest{})
	cc.Authorize(req)

	res, err := c.rpc.ListMembers(ctx, req)
	if err != nil {
		return err
	}
	return cc.Show(ctx, memberList(cc, res.Msg.GetMembers()), backOnly(cc))
}

func (c *client) invite(ctx context.Context, cc *bot.Context) error {
	email := strings.TrimSpace(cc.Args)
	if email == "" || !strings.Contains(email, "@") {
		return bot.Invalid("%s", cc.T(i18n.FamilyNeedEmail, "person@example.com"))
	}

	req := connect.NewRequest(&familyv1.InviteMemberRequest{
		Email: email,
		Role:  familyv1.Role_ROLE_MEMBER,
	})
	cc.Authorize(req)

	res, err := c.rpc.InviteMember(ctx, req)
	if err != nil {
		return err
	}

	invitation := res.Msg.GetInvitation()
	lines := []string{bot.Bold(cc.T(i18n.FamilyInvited)), "", bot.Esc(invitation.GetEmail())}
	if expires := invitation.GetExpiresAt(); expires != nil {
		lines = append(lines, bot.Italic(cc.T(i18n.FamilyExpires,
			expires.AsTime().Format("2 Jan 15:04"))))
	}
	return cc.Send(ctx, strings.Join(lines, "\n"), backOnly(cc))
}

func (c *client) invitations(ctx context.Context, cc *bot.Context) error {
	req := connect.NewRequest(&familyv1.ListInvitationsRequest{})
	cc.Authorize(req)

	res, err := c.rpc.ListInvitations(ctx, req)
	if err != nil {
		return err
	}

	var sb strings.Builder
	for _, in := range res.Msg.GetInvitations() {
		if in.GetStatus() != familyv1.InvitationStatus_INVITATION_STATUS_PENDING {
			continue
		}
		sb.WriteString("• " + in.GetEmail() + "\n")
	}
	if sb.Len() == 0 {
		return cc.Show(ctx, bot.Lines(bot.Bold(cc.T(i18n.FamilyInvites)), "",
			bot.Italic(cc.T(i18n.FamilyNoInvites))), backOnly(cc))
	}
	return cc.Show(ctx, bot.Lines(bot.Bold(cc.T(i18n.FamilyOpenInvites)), "",
		strings.TrimRight(sb.String(), "\n")), backOnly(cc))
}

func memberList(cc *bot.Context, members []*familyv1.Member) string {
	if len(members) == 0 {
		return cc.T(i18n.FamilyNobody)
	}

	var sb strings.Builder
	sb.WriteString(bot.Bold(cc.T(i18n.FamilyMembers)) + "\n\n")
	for _, m := range members {
		badge := "👤"
		if m.GetRole() == familyv1.Role_ROLE_ADMIN {
			badge = "👑"
		}
		sb.WriteString(badge + " " + bot.Bold(bot.Esc(m.GetDisplayName())) + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func backOnly(cc *bot.Context) bot.Keyboard {
	return bot.Keyboard{bot.Row(bot.Data(cc.T(i18n.Menu), "home"))}
}
