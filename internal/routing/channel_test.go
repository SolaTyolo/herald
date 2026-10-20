package routing

import "testing"

func TestProvidersForChannel(t *testing.T) {
	t.Parallel()
	snap, err := Parse([]byte(`
providers:
  smtp_email:
    type: builtin
    channel: email
  inapp:
    type: builtin
    channel: inapp
  alpha_sms:
    type: remote
    service: sms
    channel: sms
  beta_sms:
    type: remote
    service: sms
    channel: sms
services:
  sms:
    endpoint: http://localhost:9/delivery
`))
	if err != nil {
		t.Fatal(err)
	}
	table := NewTable("mem", snap)
	email := table.ProvidersForChannel("email")
	if len(email) != 1 || email[0].Name != "smtp_email" {
		t.Fatalf("email providers: %+v", email)
	}
	sms := table.ProvidersForChannel("sms")
	if len(sms) != 2 || sms[0].Name != "alpha_sms" || sms[1].Name != "beta_sms" {
		t.Fatalf("sms providers sorted: %+v", sms)
	}
	if got := table.ProvidersForChannel("inapp"); len(got) != 1 || got[0].Name != "inapp" {
		t.Fatalf("inapp providers: %+v", got)
	}
}
