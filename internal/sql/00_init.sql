CREATE TABLE users (
    id UUID DEFAULT gen_random_uuid(),
    email VARCHAR(100) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE media (
    file_id UUID PRIMARY KEY,
    user_id UUID,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE collaborators (
    -- Primary key will be id for user. Foreign key to users table
    -- id list of all users that are collaborators
);
