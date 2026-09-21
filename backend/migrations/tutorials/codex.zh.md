## 1. 准备密钥与模型

在本页创建并复制 API 密钥，确认可用余额充足。从「可用模型」复制一个支持 Responses 的模型 ID，替换下方的 `YOUR_MODEL_ID`。请勿使用管理密钥。

## 2. 安装 Codex CLI

安装 Node.js LTS（含 npm），然后在终端执行：

```sh
npm install -g @openai/codex
codex --version
```

## 3. 配置网关

打开用户目录中的 `{{config_path}}`（不存在则新建目录和文件）。如已有配置，请备份后合并；根级 `model` 和 `model_provider` 要放在任何 `[表名]` 之前，避免重复字段。不要将这些配置放在项目目录。

```toml
model = "YOUR_MODEL_ID"
model_provider = "cpa_helper"

[model_providers.cpa_helper]
name = "CPA-Helper"
base_url = "{{api_base_url}}"
wire_api = "responses"
env_key = "CPA_HELPER_API_KEY"
```

这里使用上方默认 Endpoint 的**基础 URL**（包含 `/v1`），不要追加 `/responses`。使用备用 Endpoint 时，请复制对应的基础 URL 替换。

## 4. 设置密钥并启动

点击下方密钥变量，选择本页的密钥进行复制；也可点击「复制代码」自动填入所选密钥，在**同一个终端**执行：

```{{shell}}
{{key_command}}
codex
```

环境变量只对当前终端及其启动的进程有效，新终端需要重新设置。不要把真实密钥提交到 Git、写入教程或截图分享。这种接入使用网关密钥，不需要通过 ChatGPT 登录。

进入自己的项目目录再运行 `codex`，发送一条简单请求；在「我的明细」确认模型和用量。涉及文件修改或命令执行时，请仔细核对授权提示。

## 常见问题

- **401**：检查密钥是否完整、是否已启用，以及环境变量是否在当前终端生效。
- **404 / 模型不可用**：核对基础 URL 和模型 ID，不要把 Responses URL 当作基础 URL。
- **额度或限流错误**：检查本页余额，并联系管理员确认上游状态。

参考：[官方安装说明](https://developers.openai.com/codex/quickstart/)、[自定义模型提供商](https://developers.openai.com/codex/config-advanced/)、[认证说明](https://developers.openai.com/codex/auth/)。
