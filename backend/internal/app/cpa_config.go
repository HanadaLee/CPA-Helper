package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	cpaConfigMaxBytes        = 8 << 20
	cpaConfigRequestMaxBytes = (cpaConfigMaxBytes * 2) + 1024
)

type cpaConfigPayload struct {
	Content string `json:"content"`
}

func (a *App) handleCPAConfig(w http.ResponseWriter, r *http.Request) error {
	if _, err := a.adminUser(r.Context(), r); err != nil {
		return err
	}

	cfg, err := a.loadConfig(r.Context())
	if err != nil {
		return err
	}
	timeout := time.Duration(cfg.CodexKeeper.CPATimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	switch r.Method {
	case http.MethodGet:
		content, err := a.loadCPAConfigYAML(r, cfg, timeout)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, cpaConfigPayload{Content: content})
		return nil
	case http.MethodPut:
		payload, err := decodeCPAConfigPayload(r)
		if err != nil {
			return err
		}
		if strings.TrimSpace(payload.Content) == "" {
			return validationError("CPA 配置内容不能为空")
		}
		if len(payload.Content) > cpaConfigMaxBytes {
			return validationError("CPA 配置内容超过大小限制")
		}
		if _, _, err := a.keeperRawRequest(
			r.Context(),
			cfg,
			http.MethodPut,
			"/v0/management/config.yaml",
			nil,
			[]byte(payload.Content),
			"application/yaml",
			timeout,
			cpaConfigMaxBytes,
		); err != nil {
			return err
		}
		content, err := a.loadCPAConfigYAML(r, cfg, timeout)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, cpaConfigPayload{Content: content})
		return nil
	default:
		return methodNotAllowed()
	}
}

func decodeCPAConfigPayload(r *http.Request) (cpaConfigPayload, error) {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, cpaConfigRequestMaxBytes+1))
	if err != nil || len(body) > cpaConfigRequestMaxBytes {
		return cpaConfigPayload{}, validationError("CPA 配置请求超过大小限制")
	}
	var payload cpaConfigPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return cpaConfigPayload{}, validationError("请求体不是有效 JSON")
	}
	return payload, nil
}

func (a *App) loadCPAConfigYAML(r *http.Request, cfg AppConfig, timeout time.Duration) (string, error) {
	_, payload, err := a.keeperRawRequest(
		r.Context(),
		cfg,
		http.MethodGet,
		"/v0/management/config.yaml",
		nil,
		nil,
		"",
		timeout,
		cpaConfigMaxBytes,
	)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}
