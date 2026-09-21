## 1. Prepare

Create an API key above and check your balance. Copy a Responses-compatible model ID from **Available Models** for `YOUR_MODEL_ID`. Do not use the management key.

## 2. Install

Install Node.js LTS with npm, then run:

```sh
npm install -g @openai/codex
codex --version
```

## 3. Configure

Edit `{{config_path}}` in your user directory, creating it if needed. Back up and merge existing settings. Put root keys before table headers; avoid duplicate keys and project-local provider settings.

```toml
model = "YOUR_MODEL_ID"
model_provider = "cpa_helper"

[model_providers.cpa_helper]
name = "CPA-Helper"
base_url = "{{api_base_url}}"
wire_api = "responses"
env_key = "CPA_HELPER_API_KEY"
```

Use the default **base URL** above, including `/v1`, without `/responses`. You may substitute an extra endpoint's base URL.

## 4. Launch

Click the key variable to copy your key, or use **Copy code** to substitute it. Run in the same terminal:

```{{shell}}
{{key_command}}
codex
```

Set the variable again in new terminals. Keep keys out of Git and screenshots. This configuration uses a gateway key, not ChatGPT login.

Run from your project directory, send a test request, then check **My Records**. Review permission prompts carefully.

## Troubleshooting

- **401**: check the key, enabled status and terminal environment.
- **404 / unavailable model**: check the base URL and model ID.
- **Quota / rate limit**: check your balance and contact the administrator.

References: [Installation](https://developers.openai.com/codex/quickstart/), [Custom providers](https://developers.openai.com/codex/config-advanced/), [Authentication](https://developers.openai.com/codex/auth/).
