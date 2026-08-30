package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

var upstreamSectionNames = []string{
	"gemini-api-key",
	"codex-api-key",
	"xai-api-key",
	"claude-api-key",
	"vertex-api-key",
	"openai-compatibility",
}

var upstreamSectionSet = func() map[string]struct{} {
	sections := make(map[string]struct{}, len(upstreamSectionNames))
	for _, name := range upstreamSectionNames {
		sections[name] = struct{}{}
	}
	return sections
}()

func (a *App) handleUpstreams(w http.ResponseWriter, r *http.Request) error {
	user, err := a.readyUser(r.Context(), r)
	if err != nil {
		return err
	}
	if !user.IsAdmin {
		return forbiddenError("需要管理员权限")
	}

	if r.URL.Path == "/api/upstreams" || r.URL.Path == "/api/upstreams/" {
		if err := requireMethod(r, http.MethodGet); err != nil {
			return err
		}
		return a.listUpstreamSections(w, r)
	}

	parts := splitPath(r.URL.Path, "/api/upstreams/")
	if len(parts) != 1 {
		return notFoundError("Not Found")
	}
	section := strings.TrimSpace(parts[0])
	if _, ok := upstreamSectionSet[section]; !ok {
		return notFoundError("不支持的上游类型")
	}

	switch r.Method {
	case http.MethodGet:
		return a.getUpstreamSection(w, r, section)
	case http.MethodPut:
		return a.replaceUpstreamSection(w, r, section)
	default:
		return methodNotAllowed()
	}
}

func (a *App) listUpstreamSections(w http.ResponseWriter, r *http.Request) error {
	cfg, err := a.loadConfig(r.Context())
	if err != nil {
		return err
	}
	_, payload, err := a.keeperRequest(
		r.Context(),
		cfg,
		http.MethodGet,
		"/v0/management/config",
		nil,
		nil,
		time.Duration(cfg.CodexKeeper.CPATimeoutSeconds)*time.Second,
	)
	if err != nil {
		return err
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(payload, &config); err != nil {
		return validationError("CLIProxyAPI 配置响应不是有效 JSON")
	}

	sections := make(map[string]any, len(upstreamSectionNames))
	for _, name := range upstreamSectionNames {
		sections[name] = []any{}
		raw, ok := config[name]
		if !ok || len(raw) == 0 || string(raw) == "null" {
			continue
		}
		var items []any
		if err := json.Unmarshal(raw, &items); err != nil {
			return validationError("CLIProxyAPI 上游配置格式无效：" + name)
		}
		sections[name] = items
	}

	writeJSON(w, http.StatusOK, map[string]any{"sections": sections})
	return nil
}

func (a *App) getUpstreamSection(w http.ResponseWriter, r *http.Request, section string) error {
	cfg, err := a.loadConfig(r.Context())
	if err != nil {
		return err
	}
	_, payload, err := a.keeperRequest(
		r.Context(),
		cfg,
		http.MethodGet,
		"/v0/management/"+section,
		nil,
		nil,
		time.Duration(cfg.CodexKeeper.CPATimeoutSeconds)*time.Second,
	)
	if err != nil {
		return err
	}
	items, err := decodeUpstreamSectionPayload(payload, section)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
	return nil
}

func (a *App) replaceUpstreamSection(w http.ResponseWriter, r *http.Request, section string) error {
	var items []any
	if err := decodeJSON(r, &items); err != nil {
		return err
	}
	if len(items) > 10000 {
		return validationError("单个上游类型最多允许 10000 条配置")
	}
	for _, item := range items {
		if _, ok := item.(map[string]any); !ok {
			return validationError("上游配置必须是 JSON 对象数组")
		}
	}

	cfg, err := a.loadConfig(r.Context())
	if err != nil {
		return err
	}
	_, _, err = a.keeperRequest(
		r.Context(),
		cfg,
		http.MethodPut,
		"/v0/management/"+section,
		nil,
		items,
		time.Duration(cfg.CodexKeeper.CPATimeoutSeconds)*time.Second,
	)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	return nil
}

func decodeUpstreamSectionPayload(payload []byte, section string) ([]any, error) {
	var direct []any
	if err := json.Unmarshal(payload, &direct); err == nil {
		return direct, nil
	}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(payload, &wrapper); err != nil {
		return nil, validationError("CLIProxyAPI 上游配置响应不是有效 JSON")
	}
	for _, key := range []string{section, "items", "data"} {
		raw, ok := wrapper[key]
		if !ok || len(raw) == 0 || string(raw) == "null" {
			continue
		}
		if err := json.Unmarshal(raw, &direct); err == nil {
			return direct, nil
		}
	}
	return nil, validationError("CLIProxyAPI 上游配置响应格式无效")
}
