package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	er "github.com/pkg/errors"
	"github.com/rvkinc/uasocial/internal/service"
)

// countries table parameters and query
const (
	userTable  = `user`
	userParams = `id, tg_id, name, created_at, updated_at`

	selectOneuserQuery = `SELECT ` + userParams + ` FROM ` + userTable + ` WHERE id = $1;`
	upsertUserQuery    = `INSERT INTO ` + userTable +
		`( id, tg_id, name, created_at, updated_at ) VALUES ($1, $2, $3, current_timestamp, current_timestamp)` +
		`ON CONFLICT (id) DO UPDATE name = $3 AND updated_at = current_timestamp;`
)

// user is a user store implementation
type user struct {
	*sql.DB
}

func Newuser(db *sql.DB) user {
	return user{
		db,
	}
}

func (u user) Create(ctx context.Context, user service.User) error {
	_, err := u.DB.ExecContext(ctx, upsertUserQuery,
		user.ID,
		user.TelegramID,
		user.Name,
	)
	if err != nil {
		return er.Wrap(err, fmt.Sprintf("failed to create user with id: %s", user.ID))
	}

	return nil
}

func (u user) Get(ctx context.Context, id string) (service.User, error) {
	user := service.User{}

	err := u.QueryRowContext(ctx, selectOneuserQuery, id).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Name,
		&user.CreatedAt,
		&user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return user, sql.ErrNoRows // TODO create errNotFound
	}
	if err != nil {
		return user, fmt.Errorf("query failed, %w", err)
	}

	return user, nil
}
