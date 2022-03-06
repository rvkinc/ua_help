package uasocial

import (
	"database/sql"
	"embed"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations
var migrations embed.FS // nolint:gochecknoglobals

func Migrate(db *sql.DB) (*migrate.Migrate, error) {
	d, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, err
	}

	i, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, err
	}

	return migrate.NewWithInstance("iofs", d, "uasocial", i)
}
