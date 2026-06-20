package routing

import (
	"os"
	"testing"
	"time"
)

func TestParseBuiltinAndRemote(t *testing.T) {
	t.Parallel()
	yaml := `
services:
  user-sms-worker:
    endpoint: "http://user-sms-worker:8080/delivery"
    healthCheck:
      path: "/health"
      interval: "5s"
      timeout: "2s"
providers:
  smtp_email:
    type: builtin
    channel: email
    timeout: 10s
  inapp:
    type: builtin
    channel: inapp
    timeout: 5s
  tencent_sms:
    type: remote
    service: user-sms-worker
    channel: sms
    timeout: 10s
`
	snap, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Providers) != 3 {
		t.Fatalf("providers=%d", len(snap.Providers))
	}
	p := snap.Providers["tencent_sms"]
	if p.Type != TypeRemote || p.Endpoint != "http://user-sms-worker:8080/delivery" {
		t.Fatalf("remote provider: %+v", p)
	}
	if snap.Providers["smtp_email"].Timeout != 10*time.Second {
		t.Fatalf("smtp timeout: %v", snap.Providers["smtp_email"].Timeout)
	}
}

func TestValidateRejects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		yaml string
	}{
		{
			name: "builtin_with_service",
			yaml: `
providers:
  smtp_email:
    type: builtin
    service: x
    channel: email
`,
		},
		{
			name: "remote_missing_service",
			yaml: `
providers:
  foo:
    type: remote
    channel: sms
`,
		},
		{
			name: "remote_unknown_service",
			yaml: `
services:
  a:
    endpoint: "http://a:8080/delivery"
providers:
  foo:
    type: remote
    service: b
    channel: sms
`,
		},
		{
			name: "invalid_builtin_name",
			yaml: `
providers:
  aliyun_sms:
    type: builtin
    channel: sms
`,
		},
		{
			name: "bad_endpoint",
			yaml: `
services:
  a:
    endpoint: "not-a-url"
providers:
  foo:
    type: remote
    service: a
    channel: sms
`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Parse([]byte(tc.yaml)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestReloadRejectsBadKeepsOld(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/providers.yaml"
	good := []byte(`
providers:
  smtp_email:
    type: builtin
    channel: email
`)
	if err := writeFile(path, good); err != nil {
		t.Fatal(err)
	}
	table, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !table.HasProvider("smtp_email") {
		t.Fatal("missing smtp")
	}
	if err := writeFile(path, []byte(`providers: {}`)); err != nil {
		t.Fatal(err)
	}
	if err := table.Reload(); err == nil {
		t.Fatal("expected reload error")
	}
	if !table.HasProvider("smtp_email") {
		t.Fatal("old snapshot should remain")
	}
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
