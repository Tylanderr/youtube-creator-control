CREATE DATABASE creator_control;

\c creator_control

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    email VARCHAR(100) UNIQUE NOT NULL
)

create table media (
    id SERIAL PRIMARY KEY
)
