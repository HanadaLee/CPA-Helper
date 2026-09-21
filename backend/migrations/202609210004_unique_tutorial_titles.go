package migrations

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upUniqueTutorialTitles, downUniqueTutorialTitles)
}

func upUniqueTutorialTitles(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT id, title, title_en FROM tutorials ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type article struct {
		id             int64
		title, titleEN string
	}
	var articles []article
	reserved, reservedEN := map[string]bool{}, map[string]bool{}
	for rows.Next() {
		var item article
		if err := rows.Scan(&item.id, &item.title, &item.titleEN); err != nil {
			return err
		}
		articles = append(articles, item)
		reserved[item.title] = true
		english := item.titleEN
		if english == "" {
			english = item.title
		}
		reservedEN[english] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	used, usedEN := map[string]bool{}, map[string]bool{}
	for _, item := range articles {
		title := uniqueTutorialTitle(item.title, used, reserved)
		titleEN := item.titleEN
		english := titleEN
		if english == "" {
			english = title
		}
		if label := uniqueTutorialTitle(english, usedEN, reservedEN); label != english {
			titleEN = label
		}
		if title != item.title || titleEN != item.titleEN {
			if _, err := tx.ExecContext(ctx, `UPDATE tutorials SET title = ?, title_en = ? WHERE id = ?`, title, titleEN, item.id); err != nil {
				return err
			}
		}
	}
	// Enforce unique visible labels in both languages, including English's
	// fallback to the Chinese title. Drafts reserve their titles as well.
	_, err = tx.ExecContext(ctx, `CREATE UNIQUE INDEX ux_tutorials_title ON tutorials(title);
		CREATE UNIQUE INDEX ux_tutorials_english_title ON tutorials(COALESCE(NULLIF(title_en, ''), title));`)
	return err
}

func uniqueTutorialTitle(title string, used, reserved map[string]bool) string {
	candidate := title
	for number := 2; used[candidate] || (candidate != title && reserved[candidate]); number++ {
		suffix := " (" + strconv.Itoa(number) + ")"
		base := []rune(title)
		if limit := 120 - len(suffix); len(base) > limit {
			base = base[:limit]
		}
		candidate = string(base) + suffix
	}
	used[candidate] = true
	return candidate
}

func downUniqueTutorialTitles(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP INDEX ux_tutorials_english_title; DROP INDEX ux_tutorials_title;`)
	return err
}
