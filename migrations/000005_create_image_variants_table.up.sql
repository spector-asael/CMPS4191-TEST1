BEGIN;

CREATE TABLE image_variants (
    id UUID PRIMARY KEY DEFAULT uuidv7(),

    image_id UUID NOT NULL
        REFERENCES images(id) ON DELETE CASCADE,

    name TEXT NOT NULL
        CHECK (name IN ('thumbnail', 'preview', 'display')),

    stored_filename TEXT NOT NULL
        CHECK (stored_filename <> ''),

    width INTEGER NOT NULL CHECK (width > 0),
    height INTEGER NOT NULL CHECK (height > 0),

    size BIGINT NOT NULL CHECK (size > 0),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (image_id, name),
    UNIQUE (image_id, stored_filename)
);

COMMIT;