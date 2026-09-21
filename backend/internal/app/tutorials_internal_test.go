package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
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
	for _, item := range initial {
		if !item.Published || !strings.Contains(item.Markdown, "{{api_key}}") || !strings.Contains(item.MarkdownEN, "{{api_base_url}}") {
			t.Fatalf("invalid seed: %+v", item)
		}
	}
	payload := Tutorial{Title: "测试", Client: "Other CLI", Platform: "all", Markdown: "# Draft\n{{api_key}}", SortOrder: 0}
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
		func(v *Tutorial) { v.Client = " " },
		func(v *Tutorial) { v.Markdown = " " },
		func(v *Tutorial) { v.Platform = "invalid" },
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
