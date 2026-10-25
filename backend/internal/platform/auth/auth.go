package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// how long an access token works. Short, because it can't be revoked: a signed out, demoted
// or deleted user keeps access only this long. The frontend renews it with the session cookie
const AccessTokenTTL = 15 * time.Minute

/*
UserID: The id for the current user
Role: the role for the user
*/
type Claims struct {
	Type   string `json:"typ"`
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

/*
Function:	HashPassword
Purpose:	Hash function for hashing a users password
Params:
  - password: the password we are hashing
*/
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

/*
Function:	CheckPassword
Purpose:	Check the hashed password against the raw password
Params:
  - password: the password we are checking against
  - hash: the hash we are checking
*/
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePassword checks that a password meets the minimum security requirements
func ValidatePassword(password string) error {
	if len(password) < 12 {
		return fmt.Errorf("password must be at least 12 characters")
	}
	// bcrypt only uses the first 72 bytes and refuses longer passwords
	if len(password) > 72 {
		return fmt.Errorf("password must be at most 72 bytes")
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsNumber(c):
			hasNumber = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

/*
Function:	GenerateToken
Purpose:	Create a JWT token for a user
Params:
  - userID: the id of the current user
  - role: the role of the current user
*/
func GenerateToken(userID, role string) (string, error) {
	claims := Claims{
		Type:   TokenTypeAccess,
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

/*
Function:	ValidateToken
Purpose:	Check that a token is valid
Params:
  - tokenStr: the token we are checking
*/
func ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	// tokens without typ (minted before media tokens existed) are rejected and get refreshed
	if !token.Valid || claims.Type != TokenTypeAccess {
		return nil, errors.New("invalid access token")
	}
	return claims, nil
}

/*
Function:	hashToken
Purpose:	Hash function for hashing a JWT token
Params:
  - token: the token we are hashing
*/
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

/*
Function:	touchLastSeen
Purpose:	note that a user just signed in or refreshed their session, and that they were around
today, for the Insights page. A failure only leaves these stale, so it doesn't fail the sign in
*/
func touchLastSeen(db *sql.DB, userID string) {
	db.Exec(`UPDATE users SET last_seen_at = NOW() WHERE id = $1`, userID)
	MarkActiveToday(db, userID)
}

/*
Function:	MarkActiveToday
Purpose:	note that a user was around today, for the Insights activity chart. Called on sign in
and whenever something is watched, read or played. A failure only leaves the chart short, so
callers don't fail over it
Params:
  - db: a connection to the database
  - userID: the user
*/
func MarkActiveToday(db *sql.DB, userID string) {
	db.Exec(
		`INSERT INTO user_active_days (user_id, day)
		SELECT id, CURRENT_DATE FROM users WHERE id = $1
		ON CONFLICT DO NOTHING`,
		userID,
	)
}

/*
Function:	GenerateRefreshToken
Purpose:	Create a new token for a user
Params:
  - db: a connection to the database
  - userID: the id of the current user
*/
func GenerateRefreshToken(db *sql.DB, userID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	token := hex.EncodeToString(raw)
	tokenHash := hashToken(token)

	_, err := db.Exec(
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		userID, tokenHash, time.Now().Add(RefreshTokenTTL),
	)

	if err != nil {
		return "", err
	}

	return token, nil
}
