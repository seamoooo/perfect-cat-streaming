package chaos

import "testing"

func TestDirective(t *testing.T) {
	cases := []struct {
		name string
		desc string
		want string
	}{
		{"empty", "", ""},
		{"none", "ふつうの猫の動画です", ""},
		{"sre lower", "throughput demo sre", ModeSRE},
		{"sre upper standalone", "これは SRE のデモ", ModeSRE},
		{"player", "player error demo", ModePlayer},
		{"frontend", "browser frontend crash", ModeFrontend},
		{"backend", "api backend 500", ModeBackend},
		{"substring is not a match", "presretired absbackendx", ""},
		{"priority sre over backend", "backend and sre both", ModeSRE},
		{"mixed case", "FrOnTeNd", ModeFrontend},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Directive(tc.desc); got != tc.want {
				t.Fatalf("Directive(%q) = %q, want %q", tc.desc, got, tc.want)
			}
		})
	}
}
