package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/piyush-gambhir/whatsapp-cli/cli-go/internal/build"
)

func init() {
	clock = func() time.Time { return time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) }
}

type seen struct {
	path, query, auth string
}

// fake answers like the Graph API and records each request.
func fake(t *testing.T, status int, body string) (*httptest.Server, *[]seen) {
	var got []seen
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method %s: this CLI only reads", r.Method)
		}
		got = append(got, seen{r.URL.Path, r.URL.Query().Get("fields") + "|" + r.URL.RawQuery, r.Header.Get("Authorization")})
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

func run(t *testing.T, srv *httptest.Server, args ...string) (int, string, string) {
	t.Setenv("WHATSAPP_GRAPH_URL", srv.URL)
	t.Setenv("WHATSAPP_ACCESS_TOKEN", "EAAG-secret")
	t.Setenv("WHATSAPP_BUSINESS_ACCOUNT_ID", "1029")
	t.Setenv("WHATSAPP_API_VERSION", "")
	var out, errOut bytes.Buffer
	root := NewRoot(strings.NewReader(""), &out, &errOut)
	root.SetArgs(args)
	code := 0
	if err := root.Execute(); err != nil {
		errOut.WriteString(err.Error())
		code = 1
	}
	return code, out.String(), errOut.String()
}

func TestReadsUseTheAccountAndHeaderToken(t *testing.T) {
	srv, got := fake(t, 200, `{"data":[{"id":"55","quality_rating":"GREEN"}]}`)
	if code, out, errOut := run(t, srv, "numbers", "list"); code != 0 || !strings.Contains(out, "GREEN") {
		t.Fatalf("%d %s %s", code, out, errOut)
	}
	r := (*got)[0]
	if r.path != "/v25.0/1029/phone_numbers" || r.auth != "Bearer EAAG-secret" || strings.Contains(r.query, "secret") {
		t.Fatalf("%+v", r)
	}
	run(t, srv, "templates", "list", "--status", "approved", "--limit", "20")
	if q := (*got)[1].query; !strings.Contains(q, "status=APPROVED") || !strings.Contains(q, "limit=20") {
		t.Fatalf("templates query %s", q)
	}
}

func TestAnalyticsExpressions(t *testing.T) {
	srv, got := fake(t, 200, `{"id":"1029"}`)
	code, _, errOut := run(t, srv, "analytics", "pricing", "--from", "2026-10-01", "--to", "2026-10-08",
		"--dimension", "pricing_category", "--dimension", "COUNTRY", "--country", "in")
	if code != 0 {
		t.Fatal(errOut)
	}
	want := `pricing_analytics.start(1790812800).end(1791417600).granularity(DAILY).country_codes(["IN"]).dimensions(["PRICING_CATEGORY","COUNTRY"])`
	if fields := strings.SplitN((*got)[0].query, "|", 2)[0]; fields != want {
		t.Fatalf("got  %s\nwant %s", fields, want)
	}
	if code, _, errOut := run(t, srv, "analytics", "messages", "--granularity", "DAILY", "--from", "2026-10-01", "--to", "2026-10-02"); code == 0 ||
		!strings.Contains(errOut, "HALF_HOUR,DAY,MONTH") {
		t.Fatalf("messages accepted DAILY: %s", errOut)
	}
	for _, bad := range [][]string{
		{"analytics", "pricing", "--from", "2020-01-01", "--to", "2020-01-02"},  // older than a year
		{"analytics", "pricing", "--from", "2026-12-01"},                        // in the future
		{"analytics", "pricing", "--from", "1790812800000"},                     // milliseconds
		{"analytics", "pricing", "--country", "u.s"},                            // not a country code
		{"analytics", "pricing", "--dimension", "COUNTRY]).metric_types([COST"}, // not a dimension
		{"analytics", "pricing", "--phone", "1).granularity(HALF_HOUR"},         // not a phone number
		{"analytics", "conversations"},                                          // gone: no data left
		{"templates", "list", "--after", "x&access_token=y"},                    // not a cursor
	} {
		if code, _, _ := run(t, srv, bad...); code == 0 {
			t.Errorf("accepted %v", bad)
		}
	}
	run(t, srv, "analytics", "pricing", "--phone", "+91 98765-43210", "--from", "2026-10-01", "--to", "2026-10-02")
	if q := (*got)[len(*got)-1].query; !strings.Contains(q, `phone_numbers(["919876543210"])`) {
		t.Fatalf("phone not normalised: %s", q)
	}
}

func TestGraphErrorsKeepTheirCode(t *testing.T) {
	srv, _ := fake(t, 400, `{"error":{"message":"Invalid OAuth access token","type":"OAuthException","code":190,"fbtrace_id":"AbC"}}`)
	t.Setenv("WHATSAPP_GRAPH_URL", srv.URL)
	t.Setenv("WHATSAPP_ACCESS_TOKEN", "EAAG-secret")
	t.Setenv("WHATSAPP_BUSINESS_ACCOUNT_ID", "1029")
	var out, errOut bytes.Buffer
	old := osArgs
	defer func() { osArgs = old }()
	osArgs = []string{"account", "get"}
	if code := executeWith(&out, &errOut); code != 3 || !strings.Contains(errOut.String(), "code 190") || strings.Contains(errOut.String(), "secret") {
		t.Fatalf("%d %s", code, errOut.String())
	}
}

func TestPagingKeepsCursorsButNotTokenURLs(t *testing.T) {
	srv, got := fake(t, 200, `{"data":[{"id":12345678901234567890}],"paging":{"cursors":{"after":"QVFI"},"next":"https://graph.facebook.com/v25.0/x?access_token=EAAG-secret&after=QVFI"}}`)
	code, out, errOut := run(t, srv, "templates", "list", "--after", "QVFH")
	if code != 0 || strings.Contains(out, "access_token") || !strings.Contains(out, `"after": "QVFI"`) || !strings.Contains(out, "12345678901234567890") {
		t.Fatalf("%d %s %s", code, out, errOut)
	}
	if !strings.Contains((*got)[0].query, "after=QVFH") {
		t.Fatalf("cursor not sent: %s", (*got)[0].query)
	}
}

func TestOutputAndVersionAreCheckedBeforeAnyRequest(t *testing.T) {
	srv, got := fake(t, 200, `{}`)
	if code, _, _ := run(t, srv, "-o", "yaml", "numbers", "list"); code == 0 {
		t.Fatal("yaml accepted")
	}
	t.Setenv("WHATSAPP_API_VERSION", "v19.0")
	var out, errOut bytes.Buffer
	root := NewRoot(strings.NewReader(""), &out, &errOut)
	root.SetArgs([]string{"numbers", "list"})
	if root.Execute() == nil || len(*got) != 0 {
		t.Fatal("an old API version reached the API")
	}
}

func TestUnreachableAPIHasItsOwnExitCode(t *testing.T) {
	t.Setenv("WHATSAPP_GRAPH_URL", "http://127.0.0.1:1")
	t.Setenv("WHATSAPP_ACCESS_TOKEN", "EAAG-secret")
	t.Setenv("WHATSAPP_BUSINESS_ACCOUNT_ID", "1029")
	var out, errOut bytes.Buffer
	old := osArgs
	defer func() { osArgs = old }()
	osArgs = []string{"numbers", "list"}
	if code := executeWith(&out, &errOut); code != 4 || strings.Contains(errOut.String(), "secret") {
		t.Fatalf("%d %s", code, errOut.String())
	}
}

func TestSettingsAreValidated(t *testing.T) {
	srv, got := fake(t, 200, `{}`)
	t.Setenv("WHATSAPP_GRAPH_URL", srv.URL)
	t.Setenv("WHATSAPP_ACCESS_TOKEN", "x")
	t.Setenv("WHATSAPP_BUSINESS_ACCOUNT_ID", "../../me")
	var out, errOut bytes.Buffer
	root := NewRoot(strings.NewReader(""), &out, &errOut)
	root.SetArgs([]string{"numbers", "list"})
	if root.Execute() == nil || len(*got) != 0 {
		t.Fatal("a non-numeric account id reached the API")
	}
	if code, _, _ := run(t, srv, "numbers", "get", "+15551234567"); code == 0 {
		t.Fatal("a phone number was accepted as an id")
	}
}

func TestCommandsResolve(t *testing.T) {
	for _, args := range [][]string{{"numbers", "list"}, {"-o", "json", "analytics", "pricing", "--from", "2026-10-01"}, {"phone-numbers", "get", "5"},
		{"numbers", "list", "--help"}, {"--version"}, {"version"}, {"help", "numbers"}} {
		var out bytes.Buffer
		root := NewRoot(strings.NewReader(""), &out, &bytes.Buffer{})
		root.SetArgs(append([]string{"commands", "resolve", "--"}, args...))
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		var got resolution
		if err := json.Unmarshal(out.Bytes(), &got); err != nil || got.Effect != "read" || got.Secret {
			t.Errorf("%v: %+v %v", args, got, err)
		}
	}
	root := NewRoot(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	root.SetArgs([]string{"commands", "resolve", "--", "messages", "send"})
	if root.Execute() == nil {
		t.Fatal("an unknown command resolved")
	}
	root = NewRoot(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	root.SetArgs([]string{"commands", "resolve", "--", "numbers", "get"})
	if root.Execute() == nil {
		t.Fatal("a command missing its argument resolved")
	}
}

func TestVersionCommandPrintsBuildInfo(t *testing.T) {
	old := build.Version
	build.Version = "1.2.3"
	defer func() { build.Version = old }()
	var out bytes.Buffer
	root := NewRoot(strings.NewReader(""), &out, &bytes.Buffer{})
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got["version"] != "1.2.3" || got["commit"] == "" || got["date"] == "" {
		t.Fatalf("%v %s", err, out.String())
	}
}
