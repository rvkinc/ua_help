CREATE TABLE "user" (
    id uuid CONSTRAINT user_id PRIMARY KEY,
    tg_id integer,
    name varchar(255) NOT NULL,
    phone varchar(255) NOT NULL,
    created_at timestamp
);

CREATE TABLE category (
    id uuid CONSTRAINT category_id PRIMARY KEY,
    name_ua varchar(255) NOT NULL,
    name_en varchar(255) NOT NULL,
    name_ru varchar(255) NOT NULL,
    created_at timestamp NOT NULL
);

CREATE TABLE request (
    id uuid CONSTRAINT request_id PRIMARY KEY,
    creator_id uuid NOT NULL,
    category_id uuid NOT NULL,
    locality_id uuid NOT NULL,
    description varchar(255),
    resolved bool NOT NULL,
    created_at timestamp
);

CREATE TYPE locality_type AS ENUM (
    'VILLAGE',
    'COMMUNITY',
    'CITY',
    'DISCTRICT',
    'STATE'
);

CREATE TABLE locality (
    id integer CONSTRAINT locality_id PRIMARY KEY,
    type locality_type NOT NULL,
    name_ru varchar(255) NOT NULL,
    name_ua varchar(255) NOT NULL,
    public_name_ua varchar(255) NOT NULL,
    public_name_ru varchar(255) NOT NULL,
    public_name_eu varchar(255) NOT NULL,
    parent_id integer
);