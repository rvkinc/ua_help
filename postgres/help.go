package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/rvkinc/uasocial/internal/service"
)

// countries table parameters and query
const (
	helpTable  = `help`
	helpParams = `id, creator_id, category_id, locality_id, created_at, deleted_at`

	selectOnehelpQuery = `SELECT ` + helpParams + ` FROM ` + helpTable + ` WHERE id = $1;`
	createHelpQuery    = `INSERT INTO ` + helpTable +
		`( id, creator_id, category_id, locality_id, created_at ) VALUES ($1, $2, $3, $4, current_timestamp);`

	deleteHelpQuery = `UPDATE ` + helpTable +
		`SET deleted_at = current_timestamp WHERE id = $1;`
)

// help is a help store implementation
type Help struct {
	*sql.DB
}

func NewHelp(db *sql.DB) Help {
	return Help{
		db,
	}
}

func (h Help) Get(ctx context.Context, id string) (service.Help, error) {
	help := service.Help{}

	err := h.QueryRowContext(ctx, selectOnehelpQuery, id).Scan(
		&help.ID,
		&help.CategoryID,
		&help.CategoryID,
		&help.LocalityID,
		&help.CreatedAt,
		&help.DeletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return help, sql.ErrNoRows // TODO create errNotFound
	}
	if err != nil {
		return help, fmt.Errorf("query failed, %w", err)
	}

	return help, nil
}

func (h Help) Create(ctx context.Context, help service.Help) error {
	_, err := h.DB.ExecContext(ctx, createHelpQuery, help.ID, help.CategoryID, help.CategoryID, help.LocalityID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to create help with id: %s", help.ID))
	}

	return nil
}

func (h Help) Delete(ctx context.Context, id string) error {
	_, err := h.DB.ExecContext(ctx, deleteHelpQuery, id)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to delete help with id: %s", id))
	}

	return nil
}
