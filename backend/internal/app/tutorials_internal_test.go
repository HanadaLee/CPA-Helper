package app

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	backendMigrations "cpa-helper/backend/migrations"
	"github.com/pressly/goose/v3"
)

func TestTutorialsCRUDAndPermissions(t *testing.T) {
	t.Setenv("CPA_HELPER_DATA_DIR", t.TempDir())
	a, err := New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	handler := a.Routes()
	request := func(method, path string, body any, cookies []*http.Cookie, expected int) {
		t.Helper()
		data, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(data))
		for _, cookie := range cookies {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != expected {
			t.Fatalf("%s %s = %d, want %d: %s", method, path, w.Code, expected, w.Body.String())
		}
	}
	admin := requestJSONForPricingTest(t, handler, "POST", "/api/auth/setup", map[string]any{"username": "admin", "password": "test-password", "nickname": "Admin"}, nil, nil)
	requestJSONForPricingTest(t, handler, "POST", "/api/users", map[string]any{"username": "reader", "password": "test-password", "nickname": "Reader"}, admin, nil)
	reader := requestJSONForPricingTest(t, handler, "POST", "/api/auth/login", map[string]any{"username": "reader", "password": "test-password"}, nil, nil)
	var initial []Tutorial
	requestJSONForPricingTest(t, handler, "GET", "/api/tutorials", nil, reader, &initial)
	if len(initial) != 3 {
		t.Fatalf("seed count = %d", len(initial))
	}
	for i, item := range initial {
		if !item.Published || !strings.Contains(item.Markdown, "{{api_key}}") || !strings.Contains(item.MarkdownEN, "{{api_base_url}}") {
			t.Fatalf("invalid seed: %+v", item)
		}
		if want := []string{"Codex CLI Windows", "Codex CLI macOS", "Codex CLI Linux"}[i]; item.Category != want {
			t.Fatalf("category = %q, want %q", item.Category, want)
		}
	}
	payload := Tutorial{Title: "测试", Category: "Codex Windows Desktop", Markdown: "# Draft\n{{api_key}}", SortOrder: 0}
	request("GET", "/api/tutorials", nil, nil, 401)
	request("GET", "/api/settings/tutorials", nil, reader, 403)
	request("POST", "/api/settings/tutorials", payload, reader, 403)
	request("PUT", "/api/settings/tutorials/1", payload, reader, 403)
	request("DELETE", "/api/settings/tutorials/1", nil, reader, 403)
	request("POST", "/api/tutorials", payload, reader, 405)
	var created Tutorial
	requestJSONForPricingTest(t, handler, "POST", "/api/settings/tutorials", payload, admin, &created)
	path := "/api/settings/tutorials/" + strconv.FormatInt(created.ID, 10)
	var published, managed []Tutorial
	requestJSONForPricingTest(t, handler, "GET", "/api/tutorials", nil, reader, &published)
	requestJSONForPricingTest(t, handler, "GET", "/api/settings/tutorials", nil, admin, &managed)
	if len(published) != 3 || len(managed) != 4 {
		t.Fatal("draft was exposed or missing")
	}
	payload.Published = true
	payload.SortOrder = 100
	requestJSONForPricingTest(t, handler, "PUT", path, payload, admin, &created)
	requestJSONForPricingTest(t, handler, "GET", "/api/tutorials", nil, reader, &published)
	if len(published) != 4 || published[3].ID != created.ID {
		t.Fatal("publish/order failed")
	}
	payload.Published = false
	requestJSONForPricingTest(t, handler, "PUT", path, payload, admin, nil)
	requestJSONForPricingTest(t, handler, "GET", "/api/tutorials", nil, reader, &published)
	if len(published) != 3 {
		t.Fatal("unpublish failed")
	}
	request("DELETE", path, nil, admin, 204)
	request("DELETE", path, nil, admin, 404)
	request("PUT", path, payload, admin, 404)
	request("DELETE", "/api/settings/tutorials/bad-id", nil, admin, 404)
	for _, mutate := range []func(*Tutorial){
		func(v *Tutorial) { v.Title = " " },
		func(v *Tutorial) { v.Category = " " },
		func(v *Tutorial) { v.Markdown = " " },
		func(v *Tutorial) { v.Category = strings.Repeat("中", 129) },
		func(v *Tutorial) { v.SortOrder = -1 },
		func(v *Tutorial) { v.SortOrder = 10001 },
		func(v *Tutorial) { v.Markdown = strings.Repeat("x", 128*1024+1) },
		func(v *Tutorial) { v.Title = strings.Repeat("中", 121) },
	} {
		invalid := payload
		mutate(&invalid)
		request("POST", "/api/settings/tutorials", invalid, admin, 422)
	}
	// Seed only during migration: edits/deletions survive restart.
	payload.Title = "Edited seed"
	request("PUT", "/api/settings/tutorials/1", payload, admin, 200)
	request("DELETE", "/api/settings/tutorials/2", nil, admin, 204)
	a.Close()
	a, err = New()
	if err != nil {
		t.Fatal(err)
	}
	managed, err = a.listTutorials(t.Context(), false)
	if err != nil || len(managed) != 2 || managed[1].Title != "Edited seed" {
		t.Fatalf("restart lost changes: %#v, %v", managed, err)
	}
}

func TestTutorialCategoryMigrationPreservesArticles(t *testing.T) {
	ctx := t.Context()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "tutorial-upgrade.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	goose.SetBaseFS(backendMigrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(ctx, db, ".", 202609210001); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE tutorials SET title = 'Custom title', title_en = 'Custom EN', markdown = 'custom {{api_key}}', markdown_en = 'custom {{api_base_url}}', published = 0, sort_order = 88 WHERE id = 1;
		DELETE FROM tutorials WHERE id = 2;`); err != nil {
		t.Fatal(err)
	}
	platformLabels := map[string]string{"all": "", "windows": " Windows", "macos": " macOS", "linux": " Linux", "ios": " iOS", "android": " Android"}
	for _, platform := range []string{"all", "windows", "macos", "linux", "ios", "android"} {
		if _, err := db.Exec(`INSERT INTO tutorials (title, title_en, client, platform, markdown, markdown_en, sort_order, published, updated_at)
			VALUES ('Custom article', 'EN article', 'My Client', ?, 'body {{api_key}}', 'EN {{responses_url}}', 5, 1, '2026-09-21T01:00:00Z')`, platform); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := db.Query(`SELECT id, title, title_en, client, platform, markdown, markdown_en, sort_order, published, updated_at FROM tutorials ORDER BY sort_order, id`)
	if err != nil {
		t.Fatal(err)
	}
	var expected []Tutorial
	for rows.Next() {
		var item Tutorial
		var client, platform string
		if err := rows.Scan(&item.ID, &item.Title, &item.TitleEN, &client, &platform, &item.Markdown, &item.MarkdownEN, &item.SortOrder, &item.Published, &item.UpdatedAt); err != nil {
			t.Fatal(err)
		}
		item.Category = client + platformLabels[platform]
		expected = append(expected, item)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	a := &App{db: db}
	verify := func() {
		t.Helper()
		if !testColumnExists(t, db, "tutorials", "category") || testColumnExists(t, db, "tutorials", "platform") || testColumnExists(t, db, "tutorials", "client") {
			t.Fatal("tutorial schema still has multiple dimensions")
		}
		actual, err := a.listTutorials(ctx, false)
		if err != nil || !reflect.DeepEqual(actual, expected) {
			t.Fatalf("migration changed article data: got %+v, want %+v; err=%v", actual, expected, err)
		}
		if !testIndexExists(t, db, "ix_tutorials_published_order") {
			t.Fatal("migration removed the tutorial listing index")
		}
	}
	for range 2 {
		if err := a.runMigrations(ctx); err != nil {
			t.Fatal(err)
		}
		verify()
	}
	// A downgrade preserves the merged name as a general client. Re-upgrading
	// must not append a second OS suffix or alter any article content.
	if err := goose.DownToContext(ctx, db, ".", 202609210001); err != nil {
		t.Fatal(err)
	}
	if err := a.runMigrations(ctx); err != nil {
		t.Fatal(err)
	}
	verify()
}
