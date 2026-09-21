-- +goose Up
-- The original seeds shared generic titles. Give only their unchanged titles
-- platform-specific names so their independent tabs remain distinguishable.
-- Custom titles, translations, content and publication settings stay intact.
UPDATE tutorials SET title = category
WHERE title = 'Codex CLI 接入教程' AND (
    (id = 1 AND category = 'Codex CLI Windows') OR
    (id = 2 AND category = 'Codex CLI macOS') OR
    (id = 3 AND category = 'Codex CLI Linux')
);
UPDATE tutorials SET title_en = category
WHERE title_en = 'Codex CLI setup' AND (
    (id = 1 AND category = 'Codex CLI Windows') OR
    (id = 2 AND category = 'Codex CLI macOS') OR
    (id = 3 AND category = 'Codex CLI Linux')
);

ALTER TABLE tutorials DROP COLUMN category;

-- +goose Down
-- The removed grouping cannot be recovered; keep one category per title when
-- downgrading instead of guessing or changing any article content.
ALTER TABLE tutorials ADD COLUMN category TEXT NOT NULL DEFAULT '';
UPDATE tutorials SET category = title;
