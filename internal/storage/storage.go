package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const dialect = "postgres"

type Config struct {
	DSN string `yaml:"dsn"`
}

type Interface interface {
	UpsertUser(context.Context, *User) (*User, error)
	UserByID(context.Context, uuid.UUID) (*User, error)
	SelectLocalities(context.Context, string) ([]*LocalityRegion, error)
	SelectLocality(context.Context, int) (LocalityRegion, error)

	SelectHelpsByLocalityAndCategory(context.Context, int, uuid.UUID) ([]*Help, error)
	SelectHelpsByLocalityAndCategoryForVillage(context.Context, int, uuid.UUID) ([]*Help, error)

	SelectRequestsByUser(context.Context, uuid.UUID) ([]*Request, error)
	InsertRequest(context.Context, *Request) (*Request, error)
	ResolveRequest(context.Context, uuid.UUID) error

	SelectHelpsByUser(context.Context, uuid.UUID) ([]*Help, error)
	InsertHelp(context.Context, *Help) (*Help, error)
	DeleteHelp(context.Context, uuid.UUID) error
}

type Postgres struct {
	config *Config
	driver *sqlx.DB
}

func NewPostgres(c *Config) (*Postgres, error) {
	db, err := sqlx.Open(dialect, c.DSN)
	if err != nil {
		return nil, err
	}

	err = db.PingContext(context.Background())
	if err != nil {
		return nil, err
	}

	return &Postgres{
		config: c,
		driver: db,
	}, nil
}

type (
	User struct {
		ID        uuid.UUID `db:"id"`
		TgID      int64     `db:"tg_id"`
		ChatID    int64     `db:"chat_id"`
		Name      string    `db:"name"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	Request struct {
		ID          uuid.UUID      `db:"id"`
		CreatorID   uuid.UUID      `db:"creator_id"`
		CategoryID  uuid.UUID      `db:"category_id"`
		LocalityID  int            `db:"locality_id"`
		Phone       sql.NullString `db:"phone"`
		Description string         `db:"description"`
		Resolved    bool           `db:"resolved"`
		CreatedAt   time.Time      `db:"created_at"`
		DeletedAt   *time.Time     `db:"deleted_at"`
	}

	Locality struct {
		ID           int    `db:"id"`
		Type         string `db:"type"`
		NameEN       string `db:"name_en"`
		NameRU       string `db:"name_ru"`
		NameUA       string `db:"name_ua"`
		PublicNameEN string `db:"public_name_en"`
		PublicNameRU string `db:"public_name_ru"`
		PublicNameUA string `db:"public_name_ua"`
		Lng          int64  `db:"lng"`
		Lat          int64  `db:"lat"`
		ParentID     int    `db:"parent_id"`
	}

	LocalityRegion struct {
		ID         int    `db:"id"`
		Type       string `db:"type"`
		Name       string `db:"public_name_ua"`
		RegionName string `db:"region_public_name_ua"`
	}

	Category struct {
		ID        uuid.UUID `db:"id"`
		NameEN    string    `db:"name_en"`
		NameRU    string    `db:"name_ru"`
		NameUA    string    `db:"name_ua"`
		CreatedAt time.Time `db:"created_at"`
	}

	Help struct {
		ID         uuid.UUID `db:"id"`
		CreatorID  uuid.UUID `db:"creator_id"`
		CategoryID uuid.UUID `db:"category_id"`
		LocalityID int       `db:"locality_id"`
		CreatedAt  time.Time `db:"created_at"`
	}
)

const (
	upsertUserSQL = `
insert into user
	(id, tg_id, chat_id, name, created_at, updated_at)
values ($1, $2, $3, $4, $5, %6) on conflict do update name`

	// todo: search by different languages
	// todo: sort - city first
	selectLocalitiesSQL = `
select l1.id, l1.type, l1.public_name_ua, l3.public_name_ua as region_public_name_ua from locality as l1
    join locality as l2 on (l1.parent_id = l2.id)
    join locality as l3 on (l2.parent_id = l3.id)
where levenshtein(l1.name_ua, $1) <= 1
	and l1.type != 'DISTRICT' and l1.type != 'STATE' and l1.type != 'COUNTRY';`

	selectRequestsByUserSQL = `
select 
	r.id, r.creator_id, r.category_id, r.phone, r.locality_id, r.description, r.resolved, r.created_at, r.deleted_at
from app_user as u
	join request as r on (u.id = r.creator_id)
where u.id = $1`

	insertRequestSQL = `
insert into request
    (id, creator_id, category_id, phone, locality_id, description, resolved, created_at, deleted_at) 
values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	resolveRequestSQL = `
update request set resolved = false where id = $1`

	selectHelpsByUserSQL = `
select
	h.id, h.creator_id, h.category_id, h.locality_id, h.created_at, h.deleted_at
from app_user as u
	left join help as h on (u.id = h.creator_id)
where u.id = $1`

	insertHelpSQL = `
insert into help
    (id, creator_id, category_id, locality_id, created_at)
values ($1, $2, $3, $4, $5)`

	deleteHelpSQL = `delete from help where id = $1`
)

func (p *Postgres) UpsertUser(ctx context.Context, user *User) (*User, error) {
	var (
		now = time.Now().UTC()
		uid = uuid.New()
	)

	_, err := p.driver.ExecContext(ctx, upsertUserSQL,
		uid, user.TgID, user.ChatID, user.Name, now, now)

	if err != nil {
		return nil, err
	}

	return &User{
		ID:        uid,
		TgID:      user.TgID,
		ChatID:    user.ChatID,
		Name:      user.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (p *Postgres) SelectLocalities(ctx context.Context, s string) ([]*LocalityRegion, error) {
	var localities = make([]*LocalityRegion, 0)
	return localities, p.driver.SelectContext(ctx, &localities, selectLocalitiesSQL, s)
}

func (p *Postgres) SelectRequestsByUser(ctx context.Context, u uuid.UUID) ([]*Request, error) {
	var requests = make([]*Request, 0)
	return requests, p.driver.SelectContext(ctx, &requests, selectRequestsByUserSQL, u)
}

func (p *Postgres) InsertRequest(ctx context.Context, rq *Request) (*Request, error) {
	var (
		now = time.Now().UTC()
		uid = uuid.New()
	)

	_, err := p.driver.ExecContext(ctx, insertRequestSQL,
		uid, rq.CreatorID, rq.CategoryID, rq.Phone, rq.LocalityID, rq.Description, rq.Resolved, now, nil)

	if err != nil {
		return nil, err
	}

	return &Request{
		ID:          uid,
		CreatorID:   rq.CreatorID,
		CategoryID:  rq.CategoryID,
		LocalityID:  rq.LocalityID,
		Phone:       rq.Phone,
		Description: rq.Description,
		Resolved:    rq.Resolved,
		CreatedAt:   now,
		DeletedAt:   nil,
	}, nil
}

func (p *Postgres) ResolveRequest(ctx context.Context, u uuid.UUID) error {
	_, err := p.driver.ExecContext(ctx, resolveRequestSQL, u)
	return err
}

func (p *Postgres) SelectHelpsByUser(ctx context.Context, u uuid.UUID) ([]*Help, error) {
	var helps = make([]*Help, 0)
	return helps, p.driver.SelectContext(ctx, &helps, selectHelpsByUserSQL, u)
}

func (p *Postgres) InsertHelp(ctx context.Context, h *Help) (*Help, error) {
	var (
		now = time.Now().UTC()
		uid = uuid.New()
	)

	_, err := p.driver.ExecContext(ctx, insertHelpSQL,
		uid, h.CreatorID, h.CategoryID, h.LocalityID, now)

	if err != nil {
		return nil, err
	}

	return &Help{
		ID:         uid,
		CreatorID:  h.CreatorID,
		CategoryID: h.CategoryID,
		LocalityID: h.LocalityID,
		CreatedAt:  now,
	}, nil
}

func (p *Postgres) DeleteHelp(ctx context.Context, u uuid.UUID) error {
	_, err := p.driver.ExecContext(ctx, deleteHelpSQL, u)
	return err
}
