// Package cmd is the whatsapp command line. Every command is a read: this CLI has no
// way to send a message, change a template or alter an account.
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/piyush-gambhir/whatsapp-cli/cli-go/internal/graph"
)

// version is set at build time.
var version = "dev"

// clock is the current time; tests fix it.
var clock = time.Now

// osArgs is the command line without the program name; tests replace it.
var osArgs = os.Args[1:]

func executeWith(out, errOut io.Writer) int { return Execute(strings.NewReader(""), out, errOut) }

type app struct {
	out, errOut io.Writer
	format      string
	account     string
	client      *graph.Client
	now         func() time.Time
}

// Execute runs the CLI and returns the process exit code: 0 ok, 1 usage or local
// error, 3 an error from the Graph API, 4 the API could not be reached.
func Execute(in io.Reader, out, errOut io.Writer) int {
	root := NewRoot(in, out, errOut)
	root.SetArgs(osArgs)
	if err := root.Execute(); err != nil {
		var apiErr *graph.Error
		if errors.As(err, &apiErr) {
			fmt.Fprintln(errOut, "Error:", err)
			return 3
		}
		var netErr *transportError
		if errors.As(err, &netErr) {
			fmt.Fprintln(errOut, "Error:", err)
			return 4
		}
		fmt.Fprintln(errOut, "Error:", err)
		return 1
	}
	return 0
}

// NewRoot builds the command tree. Settings come from the environment only:
// WHATSAPP_ACCESS_TOKEN, WHATSAPP_BUSINESS_ACCOUNT_ID, WHATSAPP_API_VERSION and,
// for tests, WHATSAPP_GRAPH_URL.
func NewRoot(in io.Reader, out, errOut io.Writer) *cobra.Command {
	a := &app{out: out, errOut: errOut, now: clock, client: &graph.Client{
		Base: os.Getenv("WHATSAPP_GRAPH_URL"), Version: os.Getenv("WHATSAPP_API_VERSION"), Token: os.Getenv("WHATSAPP_ACCESS_TOKEN")}}
	a.account = os.Getenv("WHATSAPP_BUSINESS_ACCOUNT_ID")
	root := &cobra.Command{Use: "whatsapp", Short: "Read a WhatsApp Business Account: numbers, templates and analytics",
		Long: "Read-only access to the WhatsApp Business Management API.\n\nSettings: WHATSAPP_ACCESS_TOKEN (system user token),\n" +
			"WHATSAPP_BUSINESS_ACCOUNT_ID (the WABA), WHATSAPP_API_VERSION (default " + graph.DefaultVersion + ").",
		Version: version, SilenceUsage: true, SilenceErrors: true}
	root.SetIn(in)
	root.SetOut(out)
	root.SetErr(errOut)
	root.PersistentFlags().StringVarP(&a.format, "output", "o", "json", "output format: json")
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if a.format != "json" {
			return fmt.Errorf("unsupported --output %q; use json", a.format)
		}
		if v := a.client.Version; v != "" && !supportedVersion(v) {
			return fmt.Errorf("WHATSAPP_API_VERSION must look like v24.0 or later (the number fields need it)")
		}
		return nil
	}
	root.AddCommand(a.accountCmd(), a.numbersCmd(), a.templatesCmd(), a.analyticsCmd(), a.commandsCmd())
	// A group given an unknown subcommand is an error, not a help page with exit 0.
	for _, group := range root.Commands() {
		if group.HasSubCommands() && group.RunE == nil {
			group.Args = cobra.NoArgs
			group.RunE = func(cmd *cobra.Command, args []string) error { return cmd.Help() }
		}
	}
	return root
}

func (a *app) print(v any) error {
	if raw, ok := v.(json.RawMessage); ok {
		// Keep the response as Graph sent it (number precision, key order), except
		// paging.next and paging.previous, whose URLs can carry the access token.
		var envelope map[string]json.RawMessage
		if json.Unmarshal(raw, &envelope) == nil {
			if paging, ok := envelope["paging"]; ok {
				var p map[string]json.RawMessage
				if json.Unmarshal(paging, &p) == nil {
					delete(p, "next")
					delete(p, "previous")
					raw = replaceKey(raw, envelope, "paging", p)
				}
			}
		}
		var buf bytes.Buffer
		if err := json.Indent(&buf, raw, "", "  "); err != nil {
			return err
		}
		buf.WriteByte('\n')
		_, err := a.out.Write(buf.Bytes())
		return err
	}
	enc := json.NewEncoder(a.out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// replaceKey re-encodes the envelope with one key replaced; other values stay raw.
func replaceKey(raw json.RawMessage, envelope map[string]json.RawMessage, key string, value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	envelope[key] = encoded
	out, err := json.Marshal(envelope)
	if err != nil {
		return raw
	}
	return out
}

// transportError marks a request that never got an answer from Graph.
type transportError struct{ err error }

func (e *transportError) Error() string { return "could not reach the Graph API: " + e.err.Error() }
func (e *transportError) Unwrap() error { return e.err }

var versionPattern = regexp.MustCompile(`^v(\d+)\.\d+$`)

func supportedVersion(v string) bool {
	m := versionPattern.FindStringSubmatch(v)
	if m == nil {
		return false
	}
	major, _ := strconv.Atoi(m[1])
	return major >= 24
}

func (a *app) waba() (string, error) {
	if a.account == "" {
		return "", fmt.Errorf("set WHATSAPP_BUSINESS_ACCOUNT_ID to the WhatsApp Business Account id")
	}
	if _, err := strconv.ParseUint(a.account, 10, 64); err != nil {
		return "", fmt.Errorf("WHATSAPP_BUSINESS_ACCOUNT_ID must be numeric")
	}
	return a.account, nil
}

func (a *app) get(cmd *cobra.Command, path string, query url.Values) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 45*time.Second)
	defer cancel()
	body, err := a.client.Get(ctx, path, query)
	if err != nil {
		var apiErr *graph.Error
		if !errors.As(err, &apiErr) && !strings.HasPrefix(err.Error(), "no access token") && !strings.HasPrefix(err.Error(), "graph API returned") {
			return &transportError{err}
		}
		return err
	}
	return a.print(body)
}

const numberFields = "id,display_phone_number,verified_name,quality_rating,status,name_status,code_verification_status," +
	"platform_type,throughput,whatsapp_business_manager_messaging_limit"

func (a *app) accountCmd() *cobra.Command {
	c := &cobra.Command{Use: "account", Short: "The WhatsApp Business Account"}
	c.AddCommand(&cobra.Command{Use: "get", Short: "Name, currency, timezone, review status", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := a.waba()
			if err != nil {
				return err
			}
			return a.get(cmd, "/"+id, url.Values{"fields": {"id,name,currency,timezone_id,message_template_namespace,account_review_status"}})
		}})
	return c
}

func (a *app) numbersCmd() *cobra.Command {
	var after string
	c := &cobra.Command{Use: "numbers", Aliases: []string{"phone-numbers"}, Short: "Business phone numbers, quality ratings and limits"}
	list := &cobra.Command{Use: "list", Short: "Numbers in the account with quality rating and messaging limit (page with --after)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := a.waba()
			if err != nil {
				return err
			}
			q := url.Values{"fields": {numberFields}}
			if err := cursor(q, after); err != nil {
				return err
			}
			return a.get(cmd, "/"+id+"/phone_numbers", q)
		}}
	list.Flags().StringVar(&after, "after", "", "paging.cursors.after from the previous page")
	c.AddCommand(list)
	c.AddCommand(&cobra.Command{Use: "get <phone-number-id>", Short: "One number by its id", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := strconv.ParseUint(args[0], 10, 64); err != nil {
				return fmt.Errorf("phone number id must be numeric (the id, not the phone number)")
			}
			return a.get(cmd, "/"+args[0], url.Values{"fields": {numberFields}})
		}})
	return c
}

func (a *app) templatesCmd() *cobra.Command {
	var status, category, name, after string
	var limit int
	c := &cobra.Command{Use: "templates", Short: "Message templates and their review status"}
	list := &cobra.Command{Use: "list", Short: "Templates with status, category, language and quality", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := a.waba()
			if err != nil {
				return err
			}
			if limit < 1 || limit > 500 {
				return fmt.Errorf("--limit must be 1-500")
			}
			q := url.Values{"fields": {"id,name,status,category,language,quality_score,rejected_reason"}, "limit": {strconv.Itoa(limit)}}
			for key, value := range map[string]string{"status": strings.ToUpper(status), "category": strings.ToUpper(category), "name": name} {
				if value != "" {
					q.Set(key, value)
				}
			}
			if err := cursor(q, after); err != nil {
				return err
			}
			return a.get(cmd, "/"+id+"/message_templates", q)
		}}
	list.Flags().StringVar(&status, "status", "", "review status, e.g. APPROVED, PENDING, REJECTED, PAUSED, DISABLED, IN_APPEAL")
	list.Flags().StringVar(&after, "after", "", "paging.cursors.after from the previous page")
	list.Flags().StringVar(&category, "category", "", "MARKETING, UTILITY or AUTHENTICATION")
	list.Flags().StringVar(&name, "name", "", "filter by template name (Graph's name filter)")
	list.Flags().IntVar(&limit, "limit", 100, "most templates to return (1-500)")
	c.AddCommand(list)
	return c
}

// window holds the shared analytics flags.
type window struct {
	from, to, granularity string
	phones, countries     []string
	dimensions            []string
}

func (w *window) bind(c *cobra.Command, granularity string) {
	c.Flags().StringVar(&w.from, "from", "", "start: YYYY-MM-DD (UTC) or a unix time (default 7 days ago)")
	c.Flags().StringVar(&w.to, "to", "", "end: YYYY-MM-DD (UTC, exclusive) or a unix time (default now)")
	c.Flags().StringVar(&w.granularity, "granularity", granularity, "HALF_HOUR, or the day and month values this metric takes")
	c.Flags().StringSliceVar(&w.phones, "phone", nil, "limit to these phone numbers (repeatable)")
	c.Flags().StringSliceVar(&w.countries, "country", nil, "limit to these 2-letter country codes (repeatable)")
}

func (a *app) bounds(w *window) (int64, int64, error) {
	parse := func(text string, fallback time.Time) (int64, error) {
		if text == "" {
			return fallback.Unix(), nil
		}
		if unixSeconds.MatchString(text) {
			return strconv.ParseInt(text, 10, 64)
		}
		day, err := time.Parse("2006-01-02", text)
		if err != nil {
			return 0, fmt.Errorf("%q is not YYYY-MM-DD or a 10-digit unix time in seconds", text)
		}
		return day.Unix(), nil
	}
	now := a.now().UTC()
	start, err := parse(w.from, now.AddDate(0, 0, -7))
	if err != nil {
		return 0, 0, err
	}
	end, err := parse(w.to, now)
	if err != nil {
		return 0, 0, err
	}
	if end <= start {
		return 0, 0, fmt.Errorf("--to must be after --from")
	}
	if start > now.Unix() {
		return 0, 0, fmt.Errorf("--from is in the future")
	}
	if end > now.Unix()+24*3600 {
		end = now.Unix()
	}
	if now.Unix()-start > 365*24*3600 {
		return 0, 0, fmt.Errorf("Meta keeps analytics for one year; move --from later")
	}
	return start, end, nil
}

func (a *app) analyticsCmd() *cobra.Command {
	c := &cobra.Command{Use: "analytics", Short: "Messages, conversations and pricing for a time window"}
	type metric struct {
		use, short, field, granularity, allowed, dims string
	}
	// Conversation analytics are left out: conversation-based pricing ended on 2025-07-01,
	// and Meta keeps a year of analytics, so it has no data left to read.
	for _, m := range []metric{
		{"messages", "Messages sent and delivered", "analytics", "DAY", "HALF_HOUR,DAY,MONTH", ""},
		{"pricing", "Delivered messages and cost per pricing category (per-message pricing)", "pricing_analytics", "DAILY", "HALF_HOUR,DAILY,MONTHLY",
			"COUNTRY,PHONE,PRICING_CATEGORY,PRICING_TYPE,TIER"},
	} {
		m := m
		w := &window{}
		sub := &cobra.Command{Use: m.use, Short: m.short, Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				id, err := a.waba()
				if err != nil {
					return err
				}
				start, end, err := a.bounds(w)
				if err != nil {
					return err
				}
				g := strings.ToUpper(w.granularity)
				if !contains(strings.Split(m.allowed, ","), g) {
					return fmt.Errorf("--granularity for %s is one of %s", m.use, m.allowed)
				}
				filters := []graph.Filter{{Name: "start", Value: strconv.FormatInt(start, 10)}, {Name: "end", Value: strconv.FormatInt(end, 10)},
					{Name: "granularity", Value: g}}
				phones, err := digits(w.phones)
				if err != nil {
					return err
				}
				if len(phones) > 0 {
					filters = append(filters, graph.Filter{Name: "phone_numbers", List: phones})
				}
				countries := upper(w.countries)
				for _, c := range countries {
					if !countryCode.MatchString(c) {
						return fmt.Errorf("--country takes 2-letter codes such as IN or US, not %q", c)
					}
				}
				if len(countries) > 0 {
					filters = append(filters, graph.Filter{Name: "country_codes", List: countries})
				}
				dims := upper(w.dimensions)
				for _, d := range dims {
					if !contains(strings.Split(m.dims, ","), d) {
						return fmt.Errorf("--dimension for %s is one of %s", m.use, m.dims)
					}
				}
				if len(dims) > 0 {
					filters = append(filters, graph.Filter{Name: "dimensions", List: dims})
				}
				return a.get(cmd, "/"+id, url.Values{"fields": {graph.Expression(m.field, filters)}})
			}}
		w.bind(sub, m.granularity)
		if m.dims != "" {
			sub.Flags().StringSliceVar(&w.dimensions, "dimension", nil, "break down by: "+m.dims+" (repeatable)")
		}
		c.AddCommand(sub)
	}
	return c
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func upper(values []string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = strings.ToUpper(strings.TrimSpace(v))
	}
	return out
}

var (
	unixSeconds = regexp.MustCompile(`^\d{10}$`)
	countryCode = regexp.MustCompile(`^[A-Z]{2}$`)
	phoneDigits = regexp.MustCompile(`^\d{6,15}$`)
	cursorText  = regexp.MustCompile(`^[A-Za-z0-9_=-]{1,512}$`)
)

// digits normalises phone filters to the digits-only form Graph expects.
func digits(phones []string) ([]string, error) {
	out := make([]string, 0, len(phones))
	for _, p := range phones {
		d := strings.NewReplacer("+", "", " ", "", "-", "", "(", "", ")", "").Replace(p)
		if !phoneDigits.MatchString(d) {
			return nil, fmt.Errorf("--phone takes a phone number such as +91 98765 43210, not %q", p)
		}
		out = append(out, d)
	}
	return out, nil
}

func cursor(q url.Values, after string) error {
	if after == "" {
		return nil
	}
	if !cursorText.MatchString(after) {
		return fmt.Errorf("--after takes the paging.cursors.after value from the previous page")
	}
	q.Set("after", after)
	return nil
}
