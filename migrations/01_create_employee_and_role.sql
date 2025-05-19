-- Create role table
CREATE TABLE IF NOT EXISTS "role" (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name TEXT NOT NULL,
    description TEXT,
    status BOOLEAN DEFAULT TRUE,
    parent_id BIGINT REFERENCES "role"(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create employee table
CREATE TABLE IF NOT EXISTS employee (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    position TEXT,
    department TEXT,
    role_id BIGINT REFERENCES "role"(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);