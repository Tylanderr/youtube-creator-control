CREATE TABLE users (
    id uuid DEFAULT gen_random_uuid(),
    email VARCHAR(100) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE media (
    id SERIAL PRIMARY KEY
    -- Associate media file with users id
    -- Filenames are unique values
);

CREATE TABLE collaborators (
    -- Primary key will be id for user. Foreign key to users table
    -- id list of all users that are collaborators
);
