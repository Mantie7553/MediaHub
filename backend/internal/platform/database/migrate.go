package database

import (
	"database/sql"
	"os"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/migrations"
	"github.com/pressly/goose/v3"
)

// seededAdminHash is the password hash migration 00002 gives the default admin account
const seededAdminHash = "$2a$10$hP.c96FjCVog/ktlMbje.OLhbNDt1hmwbor41Ou6fLLSGchF5SRpi"

/*
Function:	Migrate
Purpose:	Apply any pending goose migrations from the embedded migrations folder
Params:
  - db: the database connection
*/
func Migrate(db *sql.DB) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, ".")
}

/*
Function:	SetInitialAdminPassword
Purpose:	Replace the seeded default admin password with ADMIN_PASSWORD. Only touches the account
            while it still has the seeded hash, so a password the admin changed is never overwritten.
            Returns whether the password was changed
Params:
  - db: the database connection
*/
func SetInitialAdminPassword(db *sql.DB) (bool, error) {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		return false, nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return false, err
	}
	res, err := db.Exec(
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE username = 'admin' AND password_hash = $2`,
		hash, seededAdminHash,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
