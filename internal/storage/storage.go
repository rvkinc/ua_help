package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const (
	dialect = "postgres"

	uaLang = "UA"
)

type Config struct {
	DSN string `yaml:"dsn"`
}

type Interface interface {
	UpsertUser(context.Context, *User) (*User, error)
	UserByID(context.Context, uuid.UUID) (*User, error)
	SelectLocalities(context.Context, string) ([]*LocalityRegion, error)
	SelectLocality(context.Context, int) (LocalityRegion, error)

	SelectHelpsByLocalityAndCategoryForCity(context.Context, int, uuid.UUID) ([]*User, error)
	SelectHelpsByLocalityAndCategoryForVillage(context.Context, int, uuid.UUID) ([]*User, error)

	SelectRequestsByUser(context.Context, uuid.UUID) ([]*RequestValue, error)
	InsertRequest(context.Context, *RequestScan) (*RequestValue, error)
	ResolveRequest(context.Context, uuid.UUID) error
	ExpiredRequests(context.Context, time.Time) ([]*RequestValue, error)
	KeepRequest(ctx context.Context, requestID uuid.UUID) error

	SelectHelpsByUser(context.Context, uuid.UUID) ([]*HelpValue, error)
	InsertHelp(context.Context, *HelpScan) error
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
		Language  string    `db:"language"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	RequestScan struct {
		CreatorID   uuid.UUID      `db:"creator_id"`
		CategoryID  uuid.UUID      `db:"category_id"`
		LocalityID  int            `db:"locality_id"`
		Phone       sql.NullString `db:"phone"`
		Description string         `db:"description"`
	}

	RequestValue struct {
		ID                   uuid.UUID      `db:"id"`
		CreatorID            uuid.UUID      `db:"creator_id"`
		CategoryNameEN       string         `db:"name_en"`
		CategoryNameRU       string         `db:"name_ru"`
		CategoryNameUA       string         `db:"name_ua"`
		LocalityPublicNameEN string         `db:"public_name_en"`
		LocalityPublicNameRU string         `db:"public_name_ru"`
		LocalityPublicNameUA string         `db:"public_name_ua"`
		Language             string         `db:"language"`
		Phone                sql.NullString `db:"phone"`
		TgID                 string         `db:"tg_id"`
		Description          string         `db:"description"`
		Resolved             bool           `db:"resolved"`
		CreatedAt            time.Time      `db:"created_at"`
		UpdatedAt            sql.NullTime   `db:"updated_at"`
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

	HelpScan struct {
		CreatorID  uuid.UUID `db:"creator_id"`
		CategoryID uuid.UUID `db:"category_id"`
		LocalityID int       `db:"locality_id"`
	}

	HelpValue struct {
		ID                   uuid.UUID `db:"id"`
		CreatorID            uuid.UUID `db:"creator_id"`
		CategoryNameEN       string    `db:"name_en"`
		CategoryNameRU       string    `db:"name_ru"`
		CategoryNameUA       string    `db:"name_ua"`
		LocalityPublicNameEN string    `db:"public_name_en"`
		LocalityPublicNameRU string    `db:"public_name_ru"`
		LocalityPublicNameUA string    `db:"public_name_ua"`
		Language             string    `db:"language"`
		CreatedAt            time.Time `db:"created_at"`
	}
)

const (
	upsertUserSQL = `
insert into user
	(id, tg_id, chat_id, name, created_at, updated_at)
values ($1, $2, $3, $4, $5, $6, $7) on conflict do update name`

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
	r.id, r.creator_id, c.name_en, c.name_ua, c.name_ru, r.phone, l.public_name_en, l.public_name_ua, l.public_name_ru, r.description, r.resolved, r.created_at, r.updated_at, u.language, u.tg_id
from app_user as u
	join request as r on (u.id = r.creator_id)
	join category as c on c.id = r.category_id
	join locality as l on l.id = r.locality_id
where u.id = $1`

	selectExpiredRequests = `
select 
	r.id, r.creator_id, c.name_en, c.name_ua, c.name_ru, r.phone, l.public_name_en, l.public_name_ua, l.public_name_ru, r.description, r.resolved, r.created_at, r.updated_at, u.language, u.tg_id
from app_user as u
	join request as r on (u.id = r.creator_id)
	join category as c on c.id = r.category_id
	join locality as l on l.id = r.locality_id
where (r.created_at < $1 and r.updated_at is null) OR r.updated_at < $1`

	selectHelpsForVillageByLocalityIDAndCategoryID = `
select u.chat_id, u.language from locality as l
    join locality as l2 on l.parent_id = l2.parent_id
    join help as h on h.locality_id = l2.id
	join app_user as u on h.creator_id = u.id
where l.id = $1 and l.category = $2`

	selectHelpsByLocalityIDAndCategoryID = `
select u.chat_id, u.language from locality as l
	join help as h on h.locality_id = l.id
    join app_user as u on h.creator_id = u.id
where l.id = $1 and l.category = $2`

	insertRequestSQL = `
insert into request as r
    (id, creator_id, category_id, phone, locality_id, description, resolved, created_at) 
values ($1, $2, $3, $4, $5, $6, $7, $8)`

	selectRequestSQL = `
select 
	r.id, c.name_en, c.name_ua, c.name_ru, l.public_name_en, l.public_name_ua, l.public_name_ru, r.created_at, r.updated_at, u.language, u.tg_id
from request as r
	join category as c on c.id = r.category_id
	join locality as l on l.id = r.locality_id
where r.id = $1`

	resolveRequestSQL = `
update request set resolved = false where id = $1`

	selectHelpsByUserSQL = `
select
	h.id, h.creator_id, c.name_en, c.name_ua, c.name_ru, l.public_name_en, l.public_name_ua, l.public_name_ru, h.created_at, h.deleted_at, u.language
from app_user as u
	join help as h on (u.id = h.creator_id)
	join category as c on c.id = r.category_id
	join locality as l on l.id = r.locality_id
where u.id = $1`

	insertHelpSQL = `
insert into help
    (id, creator_id, category_id, locality_id, created_at)
values ($1, $2, $3, $4, $5)`

	deleteHelpSQL = `delete from help where id = $1`

	keepRequestSQL = `
update request set updated_at = now() where id = $1`
)

func (p *Postgres) UpsertUser(ctx context.Context, user *User) (*User, error) {
	var (
		now = time.Now().UTC()
		uid = uuid.New()
	)

	if user.Language == "" {
		user.Language = uaLang
	}

	_, err := p.driver.ExecContext(ctx, upsertUserSQL,
		uid, user.TgID, user.ChatID, user.Name, now, now, user.Language)

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

func (p *Postgres) SelectRequestsByUser(ctx context.Context, u uuid.UUID) ([]*RequestValue, error) {
	var requests = make([]*RequestValue, 0)
	return requests, p.driver.SelectContext(ctx, &requests, selectRequestsByUserSQL, u)
}

func (p *Postgres) InsertRequest(ctx context.Context, rq *RequestScan) (*RequestValue, error) {
	var (
		now = time.Now().UTC()
		uid = uuid.New()
	)

	_, err := p.driver.ExecContext(ctx, insertRequestSQL,
		uid, rq.CreatorID, rq.CategoryID, rq.Phone, rq.LocalityID, rq.Description, false, now)
	if err != nil {
		return nil, err
	}

	var requestValue RequestValue
	if err = p.driver.GetContext(ctx, &requestValue, selectRequestSQL, uid); err != nil {
		return nil, err
	}

	requestValue.Phone = rq.Phone
	requestValue.Description = rq.Description

	return &requestValue, err
}

func (p *Postgres) ResolveRequest(ctx context.Context, u uuid.UUID) error {
	_, err := p.driver.ExecContext(ctx, resolveRequestSQL, u)
	return err
}

func (p *Postgres) SelectHelpsByUser(ctx context.Context, u uuid.UUID) ([]*HelpValue, error) {
	var helps = make([]*HelpValue, 0)
	return helps, p.driver.SelectContext(ctx, &helps, selectHelpsByUserSQL, u)
}

func (p *Postgres) InsertHelp(ctx context.Context, h *HelpScan) error {
	var (
		now = time.Now().UTC()
		uid = uuid.New()
	)

	_, err := p.driver.ExecContext(ctx, insertHelpSQL,
		uid, h.CreatorID, h.CategoryID, h.LocalityID, now)

	return err
}

func (p *Postgres) DeleteHelp(ctx context.Context, u uuid.UUID) error {
	_, err := p.driver.ExecContext(ctx, deleteHelpSQL, u)
	return err
}

func (p *Postgres) SelectHelpsByLocalityAndCategoryForVillage(ctx context.Context, localityID int, categoryID uuid.UUID) ([]*User, error) {
	var users = make([]*User, 0)
	return users, p.driver.SelectContext(ctx, &users, selectHelpsForVillageByLocalityIDAndCategoryID, localityID, categoryID)
}

func (p *Postgres) SelectHelpsByLocalityAndCategoryForCity(ctx context.Context, localityID int, categoryID uuid.UUID) ([]*User, error) {
	var users = make([]*User, 0)
	return users, p.driver.SelectContext(ctx, &users, selectHelpsByLocalityIDAndCategoryID, localityID, categoryID)
}

func (p *Postgres) ExpiredRequests(ctx context.Context, before time.Time) ([]*RequestValue, error) {
	var requests = make([]*RequestValue, 0)
	return requests, p.driver.SelectContext(ctx, &requests, selectExpiredRequests, before)
}

func (p *Postgres) KeepRequest(ctx context.Context, requestID uuid.UUID) error {
	_, err := p.driver.ExecContext(ctx, keepRequestSQL, requestID)
	return err
}
