package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/nnc/family-manager/sdk/go/auth/v1/authv1connect"
	"github.com/nnc/family-manager/sdk/go/notes/v1/notesv1connect"

	"github.com/nnc/family-manager/apps/tui/internal/config"
	"github.com/nnc/family-manager/apps/tui/internal/credentials"
	"github.com/nnc/family-manager/apps/tui/internal/session"
	"github.com/nnc/family-manager/apps/tui/internal/ui"
)

const usage = `usage: fm [flags] [command]

commands:
  (none)   open the terminal UI
  login    sign in by approving a code in another family-manager app
  logout   revoke this device's session and delete stored credentials
  whoami   print the signed-in user and family

flags:
`

type App struct {
	Session *session.Session
	Notes   notesv1connect.NotesServiceClient
	Store   *credentials.Store
	Config  config.Config
	Out     io.Writer
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fm", flag.ContinueOnError)
	fs.SetOutput(stderr)
	env := fs.String("env", "", "endpoint preset: production or local (env FM_ENV)")
	authURL := fs.String("auth-url", "", "auth service base URL (env FM_AUTH_URL)")
	notesURL := fs.String("notes-url", "", "notes service base URL (env FM_NOTES_URL)")
	credsPath := fs.String("credentials", "", "credentials file (default <user config dir>/family-manager/credentials.json)")
	fs.Usage = func() {
		fmt.Fprint(stderr, usage)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	cfg, err := config.FromEnv(*env)
	if err != nil {
		fmt.Fprintf(stderr, "fm: %v\n", err)
		return 2
	}
	if v := strings.TrimRight(*authURL, "/"); v != "" {
		cfg.Endpoints.Auth = v
	}
	if v := strings.TrimRight(*notesURL, "/"); v != "" {
		cfg.Endpoints.Notes = v
	}

	store := &credentials.Store{Path: *credsPath}
	if store.Path == "" {
		if store, err = credentials.Default(); err != nil {
			fmt.Fprintf(stderr, "fm: %v\n", err)
			return 1
		}
	}

	app := New(cfg, store, stdout)

	cmd := fs.Arg(0)
	if fs.NArg() > 1 {
		fs.Usage()
		return 2
	}
	switch cmd {
	case "":
		err = ui.Run(ctx, app.Session, app.Notes)
	case "login":
		err = app.Login(ctx)
	case "logout":
		err = app.Logout(ctx)
	case "whoami":
		err = app.WhoAmI(ctx)
	case "help":
		fs.SetOutput(stdout)
		fs.Usage()
		return 0
	default:
		fmt.Fprintf(stderr, "fm: unknown command %q\n", cmd)
		fs.Usage()
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "fm: %v\n", err)
		return 1
	}
	return 0
}

func New(cfg config.Config, store *credentials.Store, out io.Writer) *App {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	authClient := authv1connect.NewAuthServiceClient(httpClient, cfg.Endpoints.Auth, connect.WithProtoJSON())
	sess := session.New(authClient, store)
	notes := notesv1connect.NewNotesServiceClient(httpClient, cfg.Endpoints.Notes,
		connect.WithProtoJSON(),
		connect.WithInterceptors(session.Interceptor(sess)),
	)
	return &App{Session: sess, Notes: notes, Store: store, Config: cfg, Out: out}
}

func (a *App) Login(ctx context.Context) error {
	g, err := a.Session.StartDeviceLogin(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Your code: %s\n", g.UserCode)
	fmt.Fprintf(a.Out, "%s.\n", session.ApproveHint)
	if !g.ExpiresAt.IsZero() {
		fmt.Fprintf(a.Out, "The code expires at %s.\n", g.ExpiresAt.Local().Format(time.Kitchen))
	}
	fmt.Fprintln(a.Out, "Waiting for approval…")
	if err := a.Session.WaitForApproval(ctx, &g, session.Sleep); err != nil {
		return err
	}
	c, err := a.Session.Claims(ctx)
	if err != nil {
		fmt.Fprintln(a.Out, "Signed in.")
		return nil
	}
	fmt.Fprintf(a.Out, "Signed in as %s.\n", displayName(c))
	return nil
}

func (a *App) Logout(ctx context.Context) error {
	err := a.Session.Logout(ctx)
	if errors.Is(err, session.ErrLoggedOut) {
		fmt.Fprintln(a.Out, "Not signed in.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("signed out locally, but the server did not confirm: %w", err)
	}
	fmt.Fprintln(a.Out, "Signed out.")
	return nil
}

func (a *App) WhoAmI(ctx context.Context) error {
	c, err := a.Session.Claims(ctx)
	if errors.Is(err, session.ErrLoggedOut) {
		return errors.New("not signed in; run `fm login`")
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "user:   %s\n", c.UserID)
	if c.Email != "" {
		fmt.Fprintf(a.Out, "email:  %s\n", c.Email)
	}
	family := c.FamilyID
	if family == "" {
		family = "(none)"
	}
	fmt.Fprintf(a.Out, "family: %s\n", family)
	fmt.Fprintf(a.Out, "server: %s\n", a.Config.Endpoints.Auth)
	return nil
}

func displayName(c session.Claims) string {
	if c.Email != "" {
		return c.Email
	}
	return c.UserID
}

func Main() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
}
