package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
	if code, _, errOut := run(t, srv, "analytics", "conversations", "--from", "2020-01-01", "--to", "2020-01-02"); code == 0 ||
		!strings.Contains(errOut, "one year") {
		t.Fatalf("old window accepted: %s", errOut)
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
	for _, args := range [][]string{{"numbers", "list"}, {"-o", "json", "analytics", "pricing", "--from", "2026-10-01"}, {"phone-numbers", "get", "5"}} {
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
}
