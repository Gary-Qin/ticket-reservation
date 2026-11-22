CREATE TABLE events (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    remaining_tickets INT NOT NULL CHECK (remaining_tickets >= 0)
);

CREATE TABLE reservations (
    id UUID PRIMARY KEY,
    event_id TEXT NOT NULL REFERENCES events(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON reservations (event_id);

INSERT INTO events(id, name, remaining_tickets) VALUES 
    ('ariana-grande', 'Ariana Grande', 100),
    ('bruno-mars', 'Bruno Mars', 100),
    ('calvin-harris', 'Calvin Harris', 100);
