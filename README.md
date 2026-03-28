# groundwork

Analyses a project in any language, detects AWS SDK usage, and generates
[Terragrunt](https://terragrunt.gruntwork.io/) + Terraform scaffolding for
the corresponding cloud resources.

Two usage modes:

| Mode | How it works | Guide |
|------|-------------|-------|
| **CLI** | One-shot command — scan project, render templates, write files | [docs/USAGE_CLI.md](docs/USAGE_CLI.md) |
| **MCP server** | AI scans the project, generates code from its own knowledge, writes through a guardrailed bridge | [docs/USAGE_MCP.md](docs/USAGE_MCP.md) |

---

## Installation

**CLI:**
```bash
go install github.com/groundwork-dev/groundwork/cmd/groundwork@latest
```

**MCP server:**
```bash
go install github.com/groundwork-dev/groundwork/cmd/groundwork-mcp@latest
```

**From source:**
```bash
make build-all   # → ./bin/groundwork + ./bin/groundwork-mcp
```

---

## CLI — quick start

```bash
groundwork \
  --input  ./my-project \
  --output ./infra \
  --state-bucket my-tf-state
```

```
groundwork: scanned 12 file(s) from "./my-project"
groundwork: 3 resource spec(s) collected
groundwork: output written to "./infra"
```

→ Full CLI guide: [docs/USAGE_CLI.md](docs/USAGE_CLI.md)

---

## MCP server — quick start

Add to `.claude/settings.json` (Claude Code) or `claude_desktop_config.json`
(Claude Desktop):

```json
{
  "mcpServers": {
    "groundwork": {
      "command": "groundwork-mcp",
      "args": ["--allowed-root", "/home/user/projects"]
    }
  }
}
```

Then in a conversation:

```
Generate the Terraform infra for the project at ./myapp.
Use prod-tf-state as the state bucket.
```

→ Full MCP guide: [docs/USAGE_MCP.md](docs/USAGE_MCP.md)

---

## Supported languages and services

| Language | Detection strategy |
|----------|--------------------|
| Go | `github.com/aws/aws-sdk-go-v2/service/<service>` imports |
| Python | `boto3.client('<service>')` / `boto3.resource('<service>')` calls |
| Node.js | `@aws-sdk/client-<service>` in `package.json` or `require`/`import` |

| AWS service | Terraform resource |
|-------------|-------------------|
| `s3` | `aws_s3_bucket` |
| `dynamodb` | `aws_dynamodb_table` |
| `sqs` | `aws_sqs_queue` |
| `sns` | `aws_sns_topic` |
| `lambda` | `aws_lambda_function` |
| `rds` | `aws_db_instance` |
| `secretsmanager` | `aws_secretsmanager_secret` |

---

## Development

```bash
make build      # compile ./bin/groundwork
make build-mcp  # compile ./bin/groundwork-mcp
make test       # go test ./... -race
make lint       # golangci-lint run
```

---

## License

MIT — see [LICENSE](LICENSE) for details.

## Acknowledgements

This project was designed and built with the assistance of [Claude](https://claude.ai) (Anthropic).
