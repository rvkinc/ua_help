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
	requestTable  = `request`
	requestParams = `id, category_id, locality_id, phone, description, resolved, created_at`

	selectOnerequestQuery = `SELECT ` + requestParams + ` FROM ` + requestTable + ` WHERE id = $1;`
	createrequestQuery    = `INSERT INTO ` + requestTable +
		`( id, category_id, locality_id, phone, description, created_at ) VALUES ($1, $2, $3, $4, $5, current_timestamp);`

	deleterequestQuery = `UPDATE ` + requestTable +
		`SET resolved = true WHERE id = $1;`
)

// request is a request store implementation
type request struct {
	*sql.DB
}

func Newrequest(db *sql.DB) request {
	return request{
		db,
	}
}

func (r request) Get(ctx context.Context, id string) (service.Request, error) {
	request := service.Request{}

	err := r.QueryRowContext(ctx, selectOnerequestQuery, id).Scan(
		&request.ID,
		&request.CategoryID,
		&request.LocalityID,
		&request.Phone,
		&request.Description,
		&request.Resolved,
		&request.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return request, sql.ErrNoRows // TODO create errNotFound
	}
	if err != nil {
		return request, fmt.Errorf("query failed, %w", err)
	}

	return request, nil
}

func (r request) Create(ctx context.Context, request service.Request) error {
	_, err := r.DB.ExecContext(ctx, createrequestQuery,
		request.ID,
		request.CategoryID,
		request.LocalityID,
		request.Phone,
		request.Description)
	if err != nil {
		return er.Wrap(err, fmt.Sprintf("failed to create request with id: %s", request.ID))
	}

	return nil
}

func (r request) Delete(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx, deleterequestQuery, id)
	if err != nil {
		return er.Wrap(err, fmt.Sprintf("failed to delete request with id: %s", id))
	}

	return nil
}
