package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/pepabo/xpoint-cli/internal/xpoint"
	"github.com/spf13/cobra"
)

var (
	formListOutput string
	formListJQ     string
	formShowOutput string
	formShowJQ     string
)

var formCmd = &cobra.Command{
	Use:   "form",
	Short: "Manage X-point forms",
}

var formListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available forms",
	Long:  "List available forms via GET /api/v1/forms.",
	RunE:  runFormList,
}

var formShowCmd = &cobra.Command{
	Use:   "show <form_code|form_id>",
	Short: "Show a form definition (fields, routes, pages)",
	Long: `Fetch field definitions via GET /api/v1/forms/{fid}.

The argument may be a form_code (e.g. "TORIHIKISAKI_a") or a numeric
form_id. When a form_code is given, the CLI first calls /api/v1/forms
to resolve the id.`,
	Args: cobra.ExactArgs(1),
	RunE: runFormShow,
}

func init() {
	rootCmd.AddCommand(formCmd)
	formCmd.AddCommand(formListCmd)
	formListCmd.Flags().StringVarP(&formListOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	formListCmd.Flags().StringVar(&formListJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")

	formCmd.AddCommand(formShowCmd)
	formShowCmd.Flags().StringVarP(&formShowOutput, "output", "o", "", "output format: table|json (default: table on TTY, json otherwise)")
	formShowCmd.Flags().StringVar(&formShowJQ, "jq", "", "apply a gojq filter to the JSON response (forces JSON output)")
}

func runFormList(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}
	res, err := client.ListAvailableForms(cmd.Context())
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(formListOutput), formListJQ, func() error {
		w := newTable(os.Stdout, "GROUP_ID", "GROUP_NAME", "FORM_ID", "FORM_CODE", "FORM_NAME")
		for _, g := range res.FormGroup {
			if len(g.Form) == 0 {
				w.AddRow(g.ID, g.Name, "-", "-", "-")
				continue
			}
			for _, f := range g.Form {
				w.AddRow(g.ID, g.Name, f.ID, f.Code, f.Name)
			}
		}
		w.Print()
		return nil
	})
}

func runFormShow(cmd *cobra.Command, args []string) error {
	client, err := newClientFromFlags(cmd.Context())
	if err != nil {
		return err
	}

	formID, err := resolveFormID(cmd.Context(), client, args[0])
	if err != nil {
		return err
	}

	res, err := client.GetFormDetail(cmd.Context(), formID)
	if err != nil {
		return err
	}

	return render(res, resolveOutputFormat(formShowOutput), formShowJQ, func() error {
		form := res.Form
		fmt.Fprintf(os.Stdout, "FORM: %s  %s  MAX_STEP: %d\n", form.Code, form.Name, form.MaxStep)
		w := newTable(os.Stdout, "PAGE", "FIELD_ID", "TYPE", "REQUIRED", "UNIQUE", "ARRAYSIZE", "LABEL")
		for _, p := range form.Pages {
			for _, f := range p.Fields {
				w.AddRow(p.PageNo, f.FieldID, f.FieldType, f.Required, f.Unique, f.ArraySize, f.Label)
			}
		}
		w.Print()
		return nil
	})
}

type formLister interface {
	ListAvailableForms(ctx context.Context) (*xpoint.FormsListResponse, error)
}

// resolveFormID returns the form_id for a form_code or a numeric id string.
// When arg is numeric, lister is not called.
func resolveFormID(ctx context.Context, lister formLister, arg string) (int, error) {
	if id, err := strconv.Atoi(arg); err == nil {
		return id, nil
	}
	forms, err := lister.ListAvailableForms(ctx)
	if err != nil {
		return 0, fmt.Errorf("resolve form code: %w", err)
	}
	for _, g := range forms.FormGroup {
		for _, f := range g.Form {
			if f.Code == arg {
				return f.ID, nil
			}
		}
	}
	return 0, fmt.Errorf("form code %q not found", arg)
}

// newClientFromFlags resolves the credentials in priority order:
//  1. explicit --xpoint-* flags
//  2. XPOINT_* environment variables
//  3. OAuth token saved in the system keyring by `xp auth login`
//
// The keyring fallback automatically refreshes the access token when it has
// expired and writes the new token back, so callers do not need to think
// about expiry.
func newClientFromFlags(ctx context.Context) (*xpoint.Client, error) {
	sub, err := resolveSubdomain()
	if err != nil {
		return nil, err
	}

	if auth, ok, err := authFromFlags(); err != nil {
		return nil, err
	} else if ok {
		return xpoint.NewClient(sub, auth), nil
	}

	if auth, ok, err := authFromEnv(); err != nil {
		return nil, err
	} else if ok {
		return xpoint.NewClient(sub, auth), nil
	}

	auth, err := loadStoredTokenAuth(ctx, sub)
	if err != nil {
		return nil, err
	}
	return xpoint.NewClient(sub, auth), nil
}

// loadStoredTokenAuth reads the token saved by `xp auth login`, refreshing
// it via the X-point token endpoint when expired. The refreshed token is
// written back to the keyring so the next call has an up-to-date refresh
// token (the X-point server invalidates the old refresh token once a
// refresh succeeds). subdomain is the target of the outgoing request and
// must match the stored login; mismatch means the caller asked for a
// subdomain we have no credentials for.
func loadStoredTokenAuth(ctx context.Context, subdomain string) (xpoint.Auth, error) {
	stored, err := xpoint.LoadToken()
	if err != nil {
		if errors.Is(err, xpoint.ErrTokenNotFound) {
			return xpoint.Auth{}, errors.New("authentication is required: set XPOINT_GENERIC_API_TOKEN (with XPOINT_DOMAIN_CODE and XPOINT_USER) or XPOINT_API_ACCESS_TOKEN, or run 'xp auth login'")
		}
		return xpoint.Auth{}, err
	}
	if stored.Subdomain != subdomain {
		return xpoint.Auth{}, fmt.Errorf("stored token is for subdomain %q but request targets %q: run 'xp auth login' to re-authenticate", stored.Subdomain, subdomain)
	}
	if stored.AccessToken == "" {
		return xpoint.Auth{}, errors.New("stored token has no access token: run 'xp auth login'")
	}

	if stored.Expired() && stored.RefreshToken != "" && stored.ClientID != "" {
		cfg := &xpoint.OAuthConfig{
			Subdomain:  subdomain,
			DomainCode: stored.DomainCode,
			ClientID:   stored.ClientID,
		}
		fresh, err := cfg.RefreshToken(ctx, stored.RefreshToken)
		if err != nil {
			return xpoint.Auth{}, fmt.Errorf("refresh stored token: %w", err)
		}
		stored.Token = *fresh
		if err := xpoint.SaveToken(stored); err != nil {
			return xpoint.Auth{}, fmt.Errorf("save refreshed token: %w", err)
		}
	}

	return xpoint.Auth{AccessToken: stored.AccessToken}, nil
}
