package auth

import (
	"crypto/rand"
	"strings"
)

// no 0/O, 1/I/L so codes can be read out or typed without mistakes
const inviteAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

const (
	inviteGroups    = 4
	inviteGroupSize = 4
)

/*
Function:	GenerateInviteCode
Purpose:	Create a random single use sign up code formatted as XXXX-XXXX-XXXX-XXXX
*/
func GenerateInviteCode() (string, error) {
	n := inviteGroups * inviteGroupSize
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	var b strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 && i%inviteGroupSize == 0 {
			b.WriteByte('-')
		}
		// byte % 31 would favour some symbols, so redraw bytes past the last full multiple of 31
		c := raw[i]
		for int(c) >= 256-(256%len(inviteAlphabet)) {
			var one [1]byte
			if _, err := rand.Read(one[:]); err != nil {
				return "", err
			}
			c = one[0]
		}
		b.WriteByte(inviteAlphabet[int(c)%len(inviteAlphabet)])
	}
	return b.String(), nil
}

/*
Function:	NormalizeInviteCode
Purpose:	Make codes match regardless of case, dashes, or spaces the user typed
Params:
  - code: the code as entered
*/
func NormalizeInviteCode(code string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '-', ' ', '\t':
			return -1
		}
		if r >= 'a' && r <= 'z' {
			return r - 'a' + 'A'
		}
		return r
	}, strings.TrimSpace(code))
}

/*
Function:	HashInviteCode
Purpose:	Hash a code for storage/lookup, only the hash is kept in the database
Params:
  - code: the code as generated or entered
*/
func HashInviteCode(code string) string {
	return hashToken(NormalizeInviteCode(code))
}

/*
Function:	InviteCodeHint
Purpose:	Last few characters of a code so admins can tell invites apart
Params:
  - code: the code as generated
*/
func InviteCodeHint(code string) string {
	norm := NormalizeInviteCode(code)
	if len(norm) <= inviteGroupSize {
		return norm
	}
	return norm[len(norm)-inviteGroupSize:]
}
