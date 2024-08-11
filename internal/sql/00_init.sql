CREATE TABLE users (
    id uuid DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE media (
    id SERIAL PRIMARY KEY
);
