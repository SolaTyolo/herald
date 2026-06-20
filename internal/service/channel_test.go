package service

import (
	"testing"

	"github.com/SolaTyolo/herald/internal/domain"
)

func TestNormalizeRouteChannel(t *testing.T) {
	t.Parallel()
	cases := map[string]domain.ChannelType{
		"email":  domain.ChannelEmail,
		"SMS":    domain.ChannelSMS,
		"inapp":  domain.ChannelInApp,
		"in_app": domain.ChannelInApp,
		"push":   domain.ChannelPush,
	}
	for in, want := range cases {
		if got := normalizeRouteChannel(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}
