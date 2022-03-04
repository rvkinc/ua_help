package storage

import (
	"context"
	"time"

	"github.com/lib/pq"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	dialect = "postgres"
	uaLang  = "UA"
)

type Config struct {
	DSN string `yaml:"dsn"`
}

type Interface interface {
	UpsertUser(context.Context, *User) (*User, error)
	SelectLocalityRegions(context.Context, string) ([]*LocalityRegion, error)

	InsertHelp(context.Context, *HelpInsert) (uuid.UUID, error)
	SelectHelpByID(context.Context, uuid.UUID) (*HelpValue, error)
	SelectHelpsByUser(context.Context, uuid.UUID) ([]*HelpValue, error)
	SelectHelpsByLocalityCategory(context.Context, int, uuid.UUID) ([]*HelpValue, error)
	DeleteHelp(ctx context.Context, uuid2 uuid.UUID) error

	SelectSubscriptionsByLocalityCategory(context.Context, int, uuid.UUID) ([]*HelpValue, error)

	// todo: do we need to select user?
	// SelectUserByID(context.Context, uuid.UUID) (*User, error)
	// todo: do we need to select locality by id?
	// SelectLocality(context.Context, int) (LocalityRegion, error)

	// ExpiredRequests(context.Context, time.Time) ([]*RequestValue, error)
	// KeepRequest(ctx context.Context, requestID uuid.UUID) error
	//
	// SelectHelpsByUser(context.Context, uuid.UUID) ([]*HelpValue, error)
	// InsertHelp(context.Context, *HelpScan) error
	// DeleteHelp(context.Context, uuid.UUID) error
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

	LocalityRegion struct {
		ID         int    `db:"id"`
		Type       string `db:"type"`
		Name       string `db:"public_name_ua"`
		RegionName string `db:"region_public_name_ua"`
	}

	HelpValue struct {
		ID                   uuid.UUID  `db:"id"`
		CreatorID            uuid.UUID  `db:"creator_id"`
		CategoryNameEN       string     `db:"name_en"`
		CategoryNameRU       string     `db:"name_ru"`
		CategoryNameUA       string     `db:"name_ua"`
		LocalityPublicNameEN string     `db:"loc_public_name_en"`
		LocalityPublicNameRU string     `db:"loc_public_name_ru"`
		LocalityPublicNameUA string     `db:"loc_public_name_ua"`
		Language             string     `db:"language"`
		Description          string     `db:"description"`
		CreatedAt            time.Time  `db:"created_at"`
		UpdatedAt            *time.Time `db:"updated_at"`
		DeletedAt            *time.Time `db:"deleted_at"`
	}

	HelpInsert struct {
		CreatorID   uuid.UUID
		CategoryIDs []uuid.UUID
		LocalityID  int
		Description string
	}

	// RequestScan struct {
	// 	CreatorID   uuid.UUID      `db:"creator_id"`
	// 	CategoryID  uuid.UUID      `db:"category_id"`
	// 	LocalityID  int            `db:"locality_id"`
	// 	Phone       sql.NullString `db:"phone"`
	// 	Description string         `db:"description"`
	// }
	//
	// RequestValue struct {
	// 	ID                   uuid.UUID      `db:"id"`
	// 	CreatorID            uuid.UUID      `db:"creator_id"`
	// 	CategoryNameEN       string         `db:"name_en"`
	// 	CategoryNameRU       string         `db:"name_ru"`
	// 	CategoryNameUA       string         `db:"name_ua"`
	// 	LocalityPublicNameEN string         `db:"public_name_en"`
	// 	LocalityPublicNameRU string         `db:"public_name_ru"`
	// 	LocalityPublicNameUA string         `db:"public_name_ua"`
	// 	Language             string         `db:"language"`
	// 	Phone                sql.NullString `db:"phone"`
	// 	TgID                 string         `db:"tg_id"`
	// 	Description          string         `db:"description"`
	// 	Resolved             bool           `db:"resolved"`
	// 	CreatedAt            time.Time      `db:"created_at"`
	// 	UpdatedAt            sql.NullTime   `db:"updated_at"`
	// }

	// Locality struct {
	// 	ID           int    `db:"id"`
	// 	Type         string `db:"type"`
	// 	NameEN       string `db:"name_en"`
	// 	NameRU       string `db:"name_ru"`
	// 	NameUA       string `db:"name_ua"`
	// 	PublicNameEN string `db:"public_name_en"`
	// 	PublicNameRU string `db:"public_name_ru"`
	// 	PublicNameUA string `db:"public_name_ua"`
	// 	Lng          int64  `db:"lng"`
	// 	Lat          int64  `db:"lat"`
	// 	ParentID     int    `db:"parent_id"`
	// }

	// Category struct {
	// 	ID        uuid.UUID `db:"id"`
	// 	NameEN    string    `db:"name_en"`
	// 	NameRU    string    `db:"name_ru"`
	// 	NameUA    string    `db:"name_ua"`
	// 	CreatedAt time.Time `db:"created_at"`
	// }
	//
	// HelpScan struct {
	// 	CreatorID  uuid.UUID `db:"creator_id"`
	// 	CategoryID uuid.UUID `db:"category_id"`
	// 	LocalityID int       `db:"locality_id"`
	// }

	// HelpValue struct {
	// 	ID                   uuid.UUID `db:"id"`
	// 	CreatorID            uuid.UUID `db:"creator_id"`
	// 	CategoryNameEN       string    `db:"name_en"`
	// 	CategoryNameRU       string    `db:"name_ru"`
	// 	CategoryNameUA       string    `db:"name_ua"`
	// 	LocalityPublicNameEN string    `db:"public_name_en"`
	// 	LocalityPublicNameRU string    `db:"public_name_ru"`
	// 	LocalityPublicNameUA string    `db:"public_name_ua"`
	// 	Language             string    `db:"language"`
	// 	CreatedAt            time.Time `db:"created_at"`
	// }
)

const (
	upsertUserSQL = `
insert into app_user
	(id, tg_id, chat_id, name, language, created_at, updated_at) 
values (:id, :tg_id, :chat_id, :name, :language, :created_at, :updated_at) 
  	on conflict (tg_id) do update set name = :name`

	// todo: search by different languages
	// todo: sort - city first
	selectLocalityRegionsSQL = `
select l1.id, l1.type, l1.public_name_ua, l3.public_name_ua as region_public_name_ua from locality as l1
    join locality as l2 on (l1.parent_id = l2.id)
    join locality as l3 on (l2.parent_id = l3.id)
where levenshtein(l1.name_ua, $1) <= 1
	and l1.type != 'DISTRICT' and l1.type != 'STATE' and l1.type != 'COUNTRY';`

	selectHelpsByLocalityCategorySQL = `
select
    h.id,
    h.creator_id,
    c.name_ua,
    c.name_ru,
    c.name_en,
    coalesce(reg_l.public_name_ua, l.public_name_ua) as loc_public_name_ua,
    coalesce(reg_l.public_name_ru, l.public_name_ru) as loc_public_name_ru,
    coalesce(reg_l.public_name_en, l.public_name_en) as loc_public_name_en,
    u.language,
    h.description,
    h.created_at,
    h.updated_at,
    h.deleted_at
from locality as l
    left join locality reg_l on (l.parent_id = reg_l.parent_id and
         (l.type = 'VILLAGE' or l.type = 'URBAN' or l.type = 'SETTLEMENT'))
    join help h on coalesce(reg_l.id, l.id) = h.locality_id
    join category c on c.id = any(h.category_ids)
    join app_user u on h.creator_id = u.id
where l.id = $1 and c.id = $2 and h.deleted_at is null;
`

	insertHelpSQL = `
insert into help
    (id, creator_id, category_ids, locality_id, description, created_at, updated_at, deleted_at) 
values ($1, $2, $3, $4, $5, $6, null, null)`

	selectHelpsByUserSQL = `
select
    h.id,
    h.creator_id,
    c.name_ua,
    c.name_ru,
    c.name_en,
    l.public_name_ua as loc_public_name_ua,
    l.public_name_ru as loc_public_name_ru,
    l.public_name_en as loc_public_name_en,
    u.language,
    h.description,
    h.created_at,
    h.updated_at,
    h.deleted_at
from app_user as u
	 join help h on h.creator_id = u.id
	 join locality l on h.locality_id = l.id
	 join category c on c.id = any(h.category_ids)
where u.id = $1 and h.deleted_at is null`

	deleteHelpSQL = `update help set deleted_at = $2 where id = $1`

	// #########################################################################

	// 	selectRequestsByUserSQL = `
	// select
	// 	r.id, r.creator_id, c.name_en, c.name_ua, c.name_ru, r.phone, l.public_name_en, l.public_name_ua, l.public_name_ru, r.description, r.resolved, r.created_at, r.updated_at, u.language, u.tg_id
	// from app_user as u
	// 	join request as r on (u.id = r.creator_id)
	// 	join category as c on c.id = r.category_id
	// 	join locality as l on l.id = r.locality_id
	// where u.id = $1 and not r.resolved`
	//
	// 	selectExpiredRequests = `
	// select
	// 	r.id, r.creator_id, c.name_en, c.name_ua, c.name_ru, r.phone, l.public_name_en, l.public_name_ua, l.public_name_ru, r.description, r.resolved, r.created_at, r.updated_at, u.language, u.tg_id
	// from app_user as u
	// 	join request as r on (u.id = r.creator_id)
	// 	join category as c on c.id = r.category_id
	// 	join locality as l on l.id = r.locality_id
	// where ((r.created_at < $1 and r.updated_at is null) or r.updated_at < $1) and not r.resolved`
	//
	// 	selectHelpsForVillageByLocalityIDAndCategoryID = `
	// select u.chat_id, u.language from locality as l
	//     join locality as l2 on l.parent_id = l2.parent_id
	//     join help as h on h.locality_id = l2.id
	// 	join app_user as u on h.creator_id = u.id
	// where l.id = $1 and l.category = $2`
	//
	// 	selectHelpsByLocalityIDAndCategoryIDForCity = `
	// select u.chat_id, u.language from locality as l
	// 	join help as h on h.locality_id = l.id
	//     join app_user as u on h.creator_id = u.id
	// where l.id = $1 and l.category = $2`
	//
	// 	insertRequestSQL = `
	// insert into request as r
	//     (id, creator_id, category_id, phone, locality_id, description, resolved, created_at)
	// values ($1, $2, $3, $4, $5, $6, $7, $8)`
	//
	// 	selectRequestSQL = `
	// select
	// 	r.id, c.name_en, c.name_ua, c.name_ru, l.public_name_en, l.public_name_ua, l.public_name_ru, r.created_at, r.updated_at, u.language, u.tg_id
	// from request as r
	// 	join category as c on c.id = r.category_id
	// 	join locality as l on l.id = r.locality_id
	// where r.id = $1 and not r.resolved`

	// 	selectHelpsByUserSQL = `
	// select
	// 	h.id, h.creator_id, c.name_en, c.name_ua, c.name_ru, l.public_name_en, l.public_name_ua, l.public_name_ru, h.created_at, h.deleted_at, u.language
	// from app_user as u
	// 	join help as h on (u.id = h.creator_id)
	// 	join category as c on c.id = r.category_id
	// 	join locality as l on l.id = r.locality_id
	// where u.id = $1`

	// 	insertHelpSQL = `
	// insert into help
	//     (id, creator_id, category_id, locality_id, created_at)
	// values ($1, $2, $3, $4, $5)`

	// deleteHelpSQL = `delete from help where id = $1`

	// 	keepRequestSQL = `
	// update request set updated_at = now() where id = $1`
)

func (p *Postgres) UpsertUser(ctx context.Context, user *User) (*User, error) {
	user.ID = uuid.New()
	if user.Language == "" {
		user.Language = uaLang
	}

	var now = time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := p.driver.NamedExecContext(ctx, upsertUserSQL, user)
	return user, err
}

func (p *Postgres) SelectLocalityRegions(ctx context.Context, s string) ([]*LocalityRegion, error) {
	var localities = make([]*LocalityRegion, 0)
	return localities, p.driver.SelectContext(ctx, &localities, selectLocalityRegionsSQL, s)
}

func (p *Postgres) InsertHelp(ctx context.Context, rq *HelpInsert) error {
	var (
		now = time.Now().UTC()
		uid = uuid.New()
	)

	_, err := p.driver.ExecContext(ctx, insertHelpSQL,
		uid, rq.CreatorID, pq.Array(rq.CategoryIDs), rq.LocalityID, rq.Description, now)

	return err
}

func (p *Postgres) SelectHelpsByLocalityCategory(ctx context.Context, localityID int, cid uuid.UUID) ([]*HelpValue, error) {
	var helps = make([]*HelpValue, 0)
	return helps, p.driver.SelectContext(ctx, &helps, selectHelpsByLocalityCategorySQL, localityID, cid)
}

func (p *Postgres) SelectHelpsByUser(ctx context.Context, uid uuid.UUID) ([]*HelpValue, error) {
	var helps = make([]*HelpValue, 0)
	return helps, p.driver.SelectContext(ctx, &helps, selectHelpsByUserSQL, uid)
}

func (p *Postgres) DeleteHelp(ctx context.Context, u uuid.UUID) error {
	_, err := p.driver.ExecContext(ctx, deleteHelpSQL, u, time.Now())
	return err
}

// ###############################################################################

// func (p *Postgres) SelectRequestsByUser(ctx context.Context, u uuid.UUID) ([]*RequestValue, error) {
// 	var requests = make([]*RequestValue, 0)
// 	return requests, p.driver.SelectContext(ctx, &requests, selectRequestsByUserSQL, u)
// }

// func (p *Postgres) ResolveRequest(ctx context.Context, u uuid.UUID) error {
// 	_, err := p.driver.ExecContext(ctx, resolveRequestSQL, u)
// 	return err
// }
//
// func (p *Postgres) SelectHelpsByUser(ctx context.Context, u uuid.UUID) ([]*HelpValue, error) {
// 	var helps = make([]*HelpValue, 0)
// 	return helps, p.driver.SelectContext(ctx, &helps, selectHelpsByUserSQL, u)
// }

// func (p *Postgres) InsertHelp(ctx context.Context, h *HelpScan) error {
// 	var (
// 		now = time.Now().UTC()
// 		uid = uuid.New()
// 	)
//
// 	_, err := p.driver.ExecContext(ctx, insertHelpSQL,
// 		uid, h.CreatorID, h.CategoryID, h.LocalityID, now)
//
// 	return err
// }

// func (p *Postgres) DeleteHelp(ctx context.Context, u uuid.UUID) error {
// 	_, err := p.driver.ExecContext(ctx, deleteHelpSQL, u)
// 	return err
// }
//
// func (p *Postgres) SelectHelpsByLocalityAndCategoryForVillage(ctx context.Context, localityID int, categoryID uuid.UUID) ([]*User, error) {
// 	var users = make([]*User, 0)
// 	return users, p.driver.SelectContext(ctx, &users, selectHelpsForVillageByLocalityIDAndCategoryID, localityID, categoryID)
// }
//
// func (p *Postgres) SelectHelpsByLocalityAndCategoryForCity(ctx context.Context, localityID int, categoryID uuid.UUID) ([]*User, error) {
// 	var users = make([]*User, 0)
// 	return users, p.driver.SelectContext(ctx, &users, selectHelpsByLocalityIDAndCategoryIDForCity, localityID, categoryID)
// }

// func (p *Postgres) ExpiredRequests(ctx context.Context, before time.Time) ([]*RequestValue, error) {
// 	var requests = make([]*RequestValue, 0)
// 	return requests, p.driver.SelectContext(ctx, &requests, selectExpiredRequests, before)
// }

// func (p *Postgres) KeepRequest(ctx context.Context, requestID uuid.UUID) error {
// 	_, err := p.driver.ExecContext(ctx, keepRequestSQL, requestID)
// 	return err
// }
