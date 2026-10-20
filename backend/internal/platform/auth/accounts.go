package auth

import (
	"database/sql"
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"unicode/utf8"
)

/*
Function:	NormalizeEmail
Purpose:	the form an email is stored and looked up in: trimmed and lower case, so
"Me@Example.com" and "me@example.com" are one account
*/
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

/*
Function:	ValidateEmail
Purpose:	check that an already normalized email is a plain address (no display name or
angle brackets) of a sensible length
*/
func ValidateEmail(email string) error {
	if len(email) > 254 {
		return fmt.Errorf("email is too long")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || !strings.Contains(email[strings.LastIndex(email, "@"):], ".") {
		return fmt.Errorf("enter a valid email address")
	}
	return nil
}

/*
Function:	ValidateUsername
Purpose:	check a username is 3 to 32 characters with no spaces at either end or control
characters, since it is shown to other users
*/
func ValidateUsername(username string) error {
	n := utf8.RuneCountInString(username)
	if n < 3 || n > 32 {
		return fmt.Errorf("username must be 3 to 32 characters")
	}
	if strings.TrimSpace(username) != username {
		return fmt.Errorf("username can't start or end with a space")
	}
	for _, c := range username {
		if c < 0x20 || c == 0x7f {
			return fmt.Errorf("username can't contain control characters")
		}
	}
	return nil
}

var (
	dummyHashOnce sync.Once
	dummyHash     string
)

/*
Function:	CheckPasswordOrDummy
Purpose:	CheckPassword, but when there is no account (hash is "") it still spends the time of
a real check, so how long a login takes doesn't reveal which emails have accounts
*/
func CheckPasswordOrDummy(password, hash string) bool {
	if hash != "" {
		return CheckPassword(password, hash)
	}
	dummyHashOnce.Do(func() {
		dummyHash, _ = HashPassword("not a real password, only spends time")
	})
	CheckPassword(password, dummyHash)
	return false
}

/*
Function:	RevokeSessions
Purpose:	sign a user out everywhere by deleting their refresh tokens, except keepHash (the
session making the change, "" for none). Access tokens they hold expire within
AccessTokenTTL
*/
func RevokeSessions(db *sql.DB, userID, keepHash string) error {
	_, err := db.Exec(
		`DELETE FROM refresh_tokens WHERE user_id = $1 AND token_hash <> $2`,
		userID, keepHash,
	)
	return err
}
