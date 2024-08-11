-- Connect to creator_control database
\c creator_control

-- Table creation

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    email VARCHAR(100) UNIQUE NOT NULL
);

CREATE TABLE media (
    id SERIAL PRIMARY KEY
);
