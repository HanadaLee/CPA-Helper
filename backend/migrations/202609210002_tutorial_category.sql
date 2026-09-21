-- +goose Up
ALTER TABLE tutorials RENAME COLUMN client TO category;

-- Preserve every article and its custom client name while flattening the two
-- dimensions. General articles keep their name without an "all" suffix.
UPDATE tutorials SET category = category || CASE platform
    WHEN 'windows' THEN ' Windows'
    WHEN 'macos' THEN ' macOS'
    WHEN 'linux' THEN ' Linux'
    WHEN 'ios' THEN ' iOS'
    WHEN 'android' THEN ' Android'
    ELSE ''
END;

ALTER TABLE tutorials DROP COLUMN platform;

-- +goose Down
-- Keep the full category name when downgrading; do not guess how a user-defined
-- category should be split back into client/platform.
ALTER TABLE tutorials ADD COLUMN platform TEXT NOT NULL DEFAULT 'all'
    CHECK(platform IN ('all', 'windows', 'macos', 'linux', 'ios', 'android'));
ALTER TABLE tutorials RENAME COLUMN category TO client;
