CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS profiles(
    id varchar(255) PRIMARY KEY,
    username varchar(255) NOT NULL,
    email citext UNIQUE NOT NULL,
    enriched_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
    );