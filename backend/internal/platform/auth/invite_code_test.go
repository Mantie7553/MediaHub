package auth

import (
	"regexp"
	"testing"
)

var inviteFormat = regexp.MustCompile(`^[` + inviteAlphabet + `]{4}(-[` + inviteAlphabet + `]{4}){3}$`)

func TestGenerateInviteCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		code, err := GenerateInviteCode()
		if err != nil {
			t.Fatal(err)
		}
		if !inviteFormat.MatchString(code) {
			t.Fatalf("code %q does not match XXXX-XXXX-XXXX-XXXX", code)
		}
		if seen[code] {
			t.Fatalf("duplicate code %q", code)
		}
		seen[code] = true
	}
}

func TestInviteCodeMatchesHowUsersTypeIt(t *testing.T) {
	code := "ABCD-EFGH-JKMN-PQRS"
	want := HashInviteCode(code)

	for _, typed := range []string{
		"ABCD-EFGH-JKMN-PQRS",
		"abcd-efgh-jkmn-pqrs",
		"ABCDEFGHJKMNPQRS",
		"  ABCD EFGH JKMN PQRS ",
		"abcd efgh-JKMN pqrs",
	} {
		if HashInviteCode(typed) != want {
			t.Errorf("%q should match %q", typed, code)
		}
	}

	for _, other := range []string{"ABCD-EFGH-JKMN-PQRT", "ABCD-EFGH-JKMN", ""} {
		if HashInviteCode(other) == want {
			t.Errorf("%q should not match %q", other, code)
		}
	}
}

func TestInviteCodeHint(t *testing.T) {
	if got := InviteCodeHint("abcd-efgh-jkmn-pqrs"); got != "PQRS" {
		t.Fatalf("hint = %q, want PQRS", got)
	}
}
