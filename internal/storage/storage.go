package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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
	SelectHelpByID(context.Context, uuid.UUID) (*Help, error)
	SelectHelpsByUser(context.Context, uuid.UUID) ([]*Help, error)
	SelectHelpsByLocalityCategory(context.Context, int, uuid.UUID) ([]*Help, error)
	DeleteHelp(ctx context.Context, uuid2 uuid.UUID) error
	SelectExpiredHelps(context.Context, time.Time) ([]*Help, error)
	KeepHelp(ctx context.Context, requestID uuid.UUID) error

	InsertSubscription(context.Context, *SubscriptionInsert) error
	SelectSubscriptionsByUser(context.Context, uuid.UUID) ([]*SubscriptionValue, error)
	SelectSubscriptionsByLocalityCategories(context.Context, int, []uuid.UUID) ([]*SubscriptionValue, error)
	DeleteSubscription(context.Context, uuid.UUID) error

	SelectCategories(context.Context) ([]*Category, error)
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
		TgID      int       `db:"tg_id"`
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
		Leven      string `db:"leven"`
	}

	Help struct {
		ID                   uuid.UUID  `db:"id"`
		CreatorID            uuid.UUID  `db:"creator_id"`
		Categories           Categories `db:"categories"`
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

	SubscriptionValue struct {
		ID                   uuid.UUID `db:"id"`
		CreatorID            uuid.UUID `db:"creator_id"`
		CategoryID           int       `db:"category_id"`
		ChatID               int64     `db:"chat_id"`
		Language             string    `db:"language"`
		CategoryNameEN       string    `db:"name_en"`
		CategoryNameRU       string    `db:"name_ru"`
		CategoryNameUA       string    `db:"name_ua"`
		LocalityPublicNameEN string    `db:"public_name_en"`
		LocalityPublicNameRU string    `db:"public_name_ru"`
		LocalityPublicNameUA string    `db:"public_name_ua"`
		CreatedAt            time.Time `db:"created_at"`
	}

	SubscriptionInsert struct {
		CreatorID  uuid.UUID
		CategoryID uuid.UUID
		LocalityID int
	}

	CategoryNames struct {
		NameUA string `db:"name_ua"`
		NameRU string `db:"name_ru"`
		NameEN string `db:"name_en"`
	}

	Categories []CategoryNames

	Category struct {
		ID     uuid.UUID `db:"id"`
		NameUA string    `db:"name_ua"`
		NameRU string    `db:"name_ru"`
		NameEN string    `db:"name_en"`
	}
)

func (c *Categories) Scan(src interface{}) error {
	b, ok := src.([]byte)
	if !ok {
		return errors.New("not ok")
	}

	err := json.Unmarshal(b, &c)
	if err != nil {
		return err
	}

	return nil
}

const (
	upsertUserSQL = `
insert into app_user as u
	(id, tg_id, chat_id, name, language, created_at, updated_at) 
values ($1, $2, $3, $4, $5, $6, $7) 
  	on conflict (tg_id) do update set name = $4 returning u.id`

	selectLocalityRegionsSQL = `
select l1.id, l1.type, l1.public_name_ua, l3.public_name_ua as region_public_name_ua, levenshtein(l1.name_ua, $1) as leven from locality as l1
    join locality as l2 on (l1.parent_id = l2.id)
    join locality as l3 on (l2.parent_id = l3.id)
where levenshtein(l1.name_ua, $1) <= 1 or levenshtein(l1.name_ru, $1) <= 1
  and l1.type != 'DISTRICT' and l1.type != 'STATE' and l1.type != 'COUNTRY'
order by
    case l1.type
        when 'CITY' then 1
        when 'URBAN' then 2
        when 'SETTLEMENT' then 3
        when 'VILLAGE' then 4
        end, leven`

	insertHelpSQL = `
insert into help
    (id, creator_id, category_ids, locality_id, description, created_at, updated_at, deleted_at) 
values ($1, $2, $3, $4, $5, $6, null, null)`

	selectHelpByIDSQL = `
select
    h.id,
    h.creator_id,
    json_agg(json_build_object('name_ua', c.name_ua, 'name_ru', c.name_ru, 'name_en', c.name_en)) as categories,
    l.public_name_ua as loc_public_name_ua,
    l.public_name_ru as loc_public_name_ru,
    l.public_name_en as loc_public_name_en,
    u.language,
    h.description,
    h.created_at,
    h.updated_at,
    h.deleted_at
from help as h
         join app_user u on h.creator_id = u.id
         join locality l on h.locality_id = l.id
         join category c on c.id = any(h.category_ids)
where h.id = $1
group by h.id, u.language, l.public_name_ua, l.public_name_ru, l.public_name_en`

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
where l.id = $1 and c.id = $2 and h.deleted_at is null`

	selectHelpsByUserSQL = `
select
    h.id,
    h.creator_id,
    json_agg(json_build_object('name_ua', c.name_ua, 'name_ru', c.name_ru, 'name_en', c.name_en)) as categories,
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
where u.id = $1 and h.deleted_at is null 
group by h.id, u.language, l.public_name_ua, l.public_name_ru, l.public_name_en`

	deleteHelpSQL = `update help set deleted_at = $2 where id = $1`

	selectExpiredHelps = `
select
    h.id,
    h.creator_id,
    json_agg(json_build_object('name_ua', c.name_ua, 'name_ru', c.name_ru, 'name_en', c.name_en)) as categories,
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
where ((h.created_at < $1 and h.updated_at is null) or h.updated_at < $1) and h.deleted_at is null
group by h.id, u.language, l.public_name_ua, l.public_name_ru, l.public_name_en`

	keepHelpSQL = `update help set updated_at = $2 where id = $1`

	insertSubscriptionSQL = `insert into subscription
	    (id, creator_id, category_id, locality_id, created_at, deleted_at)
	values ($1, $2, $3, $4, $5, null)`

	selectSubscriptionsByUserSQL = `
select s.id,
	s.creator_id,
	s.category_id,
	u.chat_id,
	u.language,
	c.name_ua,
	c.name_ru,
	c.name_en,
	l.public_name_ua,
	l.public_name_ru,
	l.public_name_en,
	s.created_at
from app_user as u
    join subscription s on s.creator_id = u.id
    join category c on c.id = s.category_id
    join locality l on s.locality_id = l.id
where u.id = $1 and s.deleted_at is null`

	selectSubscriptionsByLocalityCategoriesSQL = `
select s.id,
       s.creator_id,
       s.category_id,
       u.chat_id,
       u.language,
       c.name_ua,
       c.name_ru,
       c.name_en,
       l.public_name_ua,
       l.public_name_ru,
       l.public_name_en,
       s.created_at
from app_user as u
         join subscription s on s.creator_id = u.id
         join category c on c.id = s.category_id
         join locality l on s.locality_id = l.id
where l.id = $1 and s.category_id = any($2::uuid[])`

	deleteSubscriptionSQL = `update subscription set deleted_at = $2 where id = $1`

	selectCategoriesSQL = `select id, name_ua, name_en, name_ru from category`
)

func (p *Postgres) UpsertUser(ctx context.Context, user *User) (*User, error) {
	user.ID = uuid.New()
	if user.Language == "" {
		user.Language = uaLang
	}

	var now = time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	var uid uuid.UUID
	err := p.driver.GetContext(ctx, &uid, upsertUserSQL,
		user.ID, user.TgID, user.ChatID, user.Name, user.Language, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	user.ID = uid
	return user, err
}

func (p *Postgres) SelectLocalityRegions(ctx context.Context, s string) ([]*LocalityRegion, error) {
	var localities = make([]*LocalityRegion, 0)
	return localities, p.driver.SelectContext(ctx, &localities, selectLocalityRegionsSQL, s)
}

func (p *Postgres) InsertHelp(ctx context.Context, rq *HelpInsert) (uuid.UUID, error) {
	var (
		now = time.Now().UTC()
		uid = uuid.New()
	)

	_, err := p.driver.ExecContext(ctx, insertHelpSQL,
		uid, rq.CreatorID, pq.Array(rq.CategoryIDs), rq.LocalityID, rq.Description, now)

	return uid, err
}

func (p *Postgres) SelectHelpByID(ctx context.Context, uid uuid.UUID) (*Help, error) {
	var help = new(Help)
	err := p.driver.GetContext(ctx, help, selectHelpByIDSQL, uid)
	if err != nil {
		return nil, err
	}

	return help, nil
}

func (p *Postgres) SelectHelpsByLocalityCategory(ctx context.Context, localityID int, cid uuid.UUID) ([]*Help, error) {
	var helps = make([]*Help, 0)
	return helps, p.driver.SelectContext(ctx, &helps, selectHelpsByLocalityCategorySQL, localityID, cid)
}

func (p *Postgres) SelectHelpsByUser(ctx context.Context, uid uuid.UUID) ([]*Help, error) {
	var helps = make([]*Help, 0)
	return helps, p.driver.SelectContext(ctx, &helps, selectHelpsByUserSQL, uid)
}

func (p *Postgres) DeleteHelp(ctx context.Context, u uuid.UUID) error {
	_, err := p.driver.ExecContext(ctx, deleteHelpSQL, u, time.Now())
	return err
}

func (p *Postgres) SelectExpiredHelps(ctx context.Context, t time.Time) ([]*Help, error) {
	var helps = make([]*Help, 0)
	return helps, p.driver.SelectContext(ctx, &helps, selectExpiredHelps, t)
}

func (p *Postgres) KeepHelp(ctx context.Context, requestID uuid.UUID) error {
	_, err := p.driver.ExecContext(ctx, keepHelpSQL, requestID, time.Now())
	return err
}

func (p *Postgres) InsertSubscription(ctx context.Context, s *SubscriptionInsert) error {
	_, err := p.driver.ExecContext(ctx, insertSubscriptionSQL, uuid.New(), s.CreatorID, s.CategoryID, s.LocalityID, time.Now().UTC())
	return err
}

func (p *Postgres) SelectSubscriptionsByUser(ctx context.Context, uid uuid.UUID) ([]*SubscriptionValue, error) {
	var sub = make([]*SubscriptionValue, 0)
	return sub, p.driver.SelectContext(ctx, sub, selectSubscriptionsByUserSQL, uid)
}

func (p *Postgres) SelectSubscriptionsByLocalityCategories(ctx context.Context, l int, cids []uuid.UUID) ([]*SubscriptionValue, error) {
	var sub = make([]*SubscriptionValue, 0)
	return sub, p.driver.SelectContext(ctx, sub, selectSubscriptionsByLocalityCategoriesSQL, l, pq.Array(cids))
}

func (p *Postgres) DeleteSubscription(ctx context.Context, sid uuid.UUID) error {
	_, err := p.driver.ExecContext(ctx, deleteSubscriptionSQL, sid, time.Now())
	return err
}

func (p *Postgres) SelectCategories(ctx context.Context) ([]*Category, error) {
	var cs = make([]*Category, 0)
	err := p.driver.SelectContext(ctx, &cs, selectCategoriesSQL)
	return cs, err
}
