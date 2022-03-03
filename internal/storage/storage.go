package storage

import (
	"context"
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

	SelectRequestsByUser(uuid.UUID) ([]*Request, error)
	InsertRequest(context.Context, *Request) (*Request, error)
	ResolveRequest(uuid.UUID) error

	SelectHelpsByUser(context.Context, uuid.UUID) ([]*Help, error)
	SelectHelpsByLocalityAndCategory(context.Context, int, uuid.UUID) ([]*Help, error)
	SelectHelpsByLocalityAndCategoryForVillage(context.Context, int, uuid.UUID) ([]*Help, error)
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
		ID          uuid.UUID `db:"id"`
		CreatorID   uuid.UUID `db:"creator_id"`
		CategoryID  uuid.UUID `db:"category_id"`
		LocalityID  int       `db:"locality_id"`
		Phone       string    `db:"phone"`
		Description string    `db:"description"`
		Resolved    bool      `db:"resolved"`
		CreatedAt   time.Time `db:"createdAt"`
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
		ID         int    `db:"l1.id"`
		Type       string `db:"l1.type"`
		Name       string `db:"l1.public_name_ua"`
		RegionName string `db:"l3.public_name_ua"`
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
		DeletedAt  time.Time `db:"deleted_at"`
	}
)

const (
	upsertUserSQL = `
insert into user
(id, tg_id, chat_id, name, created_at, updated_at)
values ($1, $2, $3, $4, $5, %6)`

	selectHelpsForVillage = `
	SELECT * FROM help WHERE category_id = $1 AND locality_id IN (SELECT * FROM locality WHERE parent_id IN (SELECT l2.id FROM locality AS l1 LEFT JOIN locality AS l2 on (l1.parent_id = l2.id) WHERE l1.id = $2))`

	// todo: search by different languages
	selectLocalitiesSQL = `
SELECT l1.id, l1.type, l1.public_name_ua, l3.public_name_ua from locality as l1
    left join locality as l2 on (l1.parent_id = l2.id)
    left join locality as l3 on (l2.parent_id = l3.id)
where levenshtein(l1.name_ua, $1) <= 1
  AND l1.type != 'DISTRICT' AND l1.type != 'STATE' AND l1.type != 'COUNTRY';`
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

func (p *Postgres) SelectLocalities(s string) ([]*LocalityRegion, error) {
	rows, err := p.driver.Queryx(selectLocalitiesSQL, s)
	if err != nil {
		return nil, err
	}

	var localities = make([]*LocalityRegion, 0)
	for rows.Next() {
		l := new(LocalityRegion)
		err := rows.Scan(l)
		if err != nil {
			return nil, err
		}

		localities = append(localities, l)
	}

	return localities, nil
}

func (p *Postgres) SelectRequestsByUser(u uuid.UUID) ([]*Request, error) {
	// TODO implement me
	panic("implement me")
}

func (p *Postgres) InsertRequest(request *Request) (*Request, error) {
	// TODO implement me
	panic("implement me")
}

func (p *Postgres) ResolveRequest(u uuid.UUID) error {
	// TODO implement me
	panic("implement me")
}

func (p *Postgres) SelectHelpsByUser(u uuid.UUID) ([]*Help, error) {
	// TODO implement me
	panic("implement me")
}

func (p *Postgres) InsertHelp(help *Help) (*Help, error) {
	// TODO implement me
	panic("implement me")
}

func (p *Postgres) DeleteHelp(u uuid.UUID) error {
	// TODO implement me
	panic("implement me")
}

func (p *Postgres) SelectHelpsByLocalityAndCategoryForVillage(context.Context, int, uuid.UUID) ([]*Help, error) {
	rows, err := p.driver.Queryx(selectLocalitiesForVillage, s)
	if err != nil {
		return nil, err
	}

	var localities = make([]*LocalityRegion, 0)
	for rows.Next() {
		l := new(LocalityRegion)
		err := rows.Scan(l)
		if err != nil {
			return nil, err
		}

		localities = append(localities, l)
	}

	return localities, nil
	return nil, nil
}
