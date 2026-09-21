package migrations

import (
	"context"
	"database/sql"
	_ "embed"
	"strings"

	"github.com/pressly/goose/v3"
)

//go:embed tutorials/codex.zh.md
var codexTutorialZH string

//go:embed tutorials/codex.en.md
var codexTutorialEN string

func init() {
	goose.AddMigrationContext(upTutorials, downTutorials)
}

func upTutorials(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `CREATE TABLE tutorials (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		title_en TEXT NOT NULL DEFAULT '',
		client TEXT NOT NULL,
		platform TEXT NOT NULL DEFAULT 'all' CHECK(platform IN ('all', 'windows', 'macos', 'linux', 'ios', 'android')),
		markdown TEXT NOT NULL,
		markdown_en TEXT NOT NULL DEFAULT '',
		sort_order INTEGER NOT NULL DEFAULT 0,
		published INTEGER NOT NULL DEFAULT 0 CHECK(published IN (0, 1)),
		updated_at TEXT NOT NULL
	);
	CREATE INDEX ix_tutorials_published_order ON tutorials(published, sort_order, id);`); err != nil {
		return err
	}
	for i, platform := range []string{"windows", "macos", "linux"} {
		path := "~/.codex/config.toml"
		shell := "bash"
		command := "export CPA_HELPER_API_KEY='{{api_key}}'"
		if platform == "windows" {
			path = `%USERPROFILE%\.codex\config.toml`
			shell = "powershell"
			command = `$env:CPA_HELPER_API_KEY = '{{api_key}}'`
		}
		replacer := strings.NewReplacer("{{config_path}}", path, "{{shell}}", shell, "{{key_command}}", command)
		if _, err := tx.ExecContext(ctx, `INSERT INTO tutorials
			(title, title_en, client, platform, markdown, markdown_en, sort_order, published, updated_at)
			VALUES (?, ?, 'Codex CLI', ?, ?, ?, ?, 1, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))`,
			"Codex CLI 接入教程", "Codex CLI setup", platform, replacer.Replace(codexTutorialZH), replacer.Replace(codexTutorialEN), i); err != nil {
			return err
		}
	}
	return nil
}

func downTutorials(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP TABLE tutorials`)
	return err
}
