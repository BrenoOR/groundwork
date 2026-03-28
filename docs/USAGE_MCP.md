# groundwork-mcp — MCP Server Guide

`groundwork-mcp` exposes groundwork as a
[Model Context Protocol](https://modelcontextprotocol.io) server, letting AI
assistants (Claude Desktop, Claude Code, etc.) scan your project and generate
Terraform/Terragrunt infrastructure files through a safe, guardrailed interface.

## How it works

The AI is the generator — not the server. The server provides two things:

1. **Context** — `groundwork_scan` tells the AI what AWS services your code
   uses and what language it's written in.
2. **Safe writes** — `groundwork_write_files` persists the AI-generated files
   with guardrails (path sandbox, content validation, explicit confirmation).

```
groundwork_scan(project_path)
  → { language: "python", services: ["s3", "dynamodb"] }

AI generates .tf / .hcl content using its own knowledge

groundwork_write_files(base_path, files, confirmed=true)
  → { files_written: ["terragrunt.hcl", "modules/s3/main.tf", ...] }
```

---

## Installation

```bash
go install github.com/groundwork-dev/groundwork/cmd/groundwork-mcp@latest
```

Or build from source:

```bash
make build-mcp        # → ./bin/groundwork-mcp
make build-all        # builds both groundwork and groundwork-mcp
```

---

## Quick start

### Claude Code

Add to your project's `.claude/settings.json` (or the global
`~/.claude/settings.json`):

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
Generate the Terraform infrastructure for the project at ./myapp.
Use prod-tf-state as the state bucket.
```

Claude will call `groundwork_scan`, generate the files, show you a preview,
and ask for confirmation before writing.

### Claude Desktop

Add to `~/.config/claude/claude_desktop_config.json`
(macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "groundwork": {
      "command": "groundwork-mcp",
      "args": [
        "--allowed-root", "/home/user/projects"
      ]
    }
  }
}
```

### SSE mode (CI / web integrations)

```bash
groundwork-mcp \
  --transport sse \
  --port 8080 \
  --allowed-root /workspace
```

---

## Flags reference

| Flag | Default | Description |
|------|---------|-------------|
| `--allowed-root` | *(required)* | Directory the server may read/write. Repeatable. |
| `--readonly` | `false` | Disable `groundwork_write_files` (scan and resources only). |
| `--transport` | `stdio` | Transport: `stdio` or `sse`. |
| `--port` | `8080` | Port for SSE transport. |
| `--max-files` | `10000` | Maximum files to scan per project. |
| `--max-project-bytes` | `104857600` (100 MB) | Maximum total project size. |
| `--max-file-bytes` | `524288` (512 KB) | Maximum size of a single written file. |
| `--rate-limit` | `10` | Maximum tool calls per second. |
| `--allow-exec-provisioners` | `false` | Allow `local-exec`/`remote-exec` blocks in written files. |

---

## Tools

### `groundwork_scan`

Scans a project and returns the AWS services detected in the source code.
Read-only — no files are written.

**Input:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `project_path` | string | yes | Absolute path to the project directory |
| `provider` | string | no | Cloud provider plugin (default: `aws`) |

**Output:**

```json
{
  "language":      "python",
  "files_scanned": 18,
  "services":      ["s3", "dynamodb"],
  "resources": [
    { "provider": "aws", "type": "aws_s3_bucket",      "name": "s3" },
    { "provider": "aws", "type": "aws_dynamodb_table", "name": "dynamodb" }
  ],
  "warnings": []
}
```

The `language` field gives the AI context to generate idiomatic code — for
example, the correct Lambda runtime (`python3.12` vs `nodejs20.x`).

---

### `groundwork_write_files`

Writes AI-generated Terraform/Terragrunt files to disk with full guardrail
validation. Not registered when `--readonly` is active.

**Input:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `base_path` | string | yes | Absolute path of the output directory |
| `files` | array | yes | List of `{path, content}` objects (max 50) |
| `confirmed` | boolean | no | Must be `true` to actually write (default: `false`) |

**Output (confirmed=true):**

```json
{
  "files_written": [
    "terragrunt.hcl",
    "modules/s3/terragrunt.hcl",
    "modules/s3/main.tf"
  ]
}
```

**Output (confirmed=false or omitted):**

```
Error: write operations require confirmed=true. Review the files listed above,
then call groundwork_write_files again with confirmed=true.
```

The two-step flow (show → confirm) ensures the user reviews the generated
content before anything is written to disk.

---

## Resources

Resources are read-only and always available, even in `--readonly` mode.

### `groundwork://services`

Returns the canonical mapping of AWS service names to Terraform resource types.

```json
{
  "s3":             "aws_s3_bucket",
  "dynamodb":       "aws_dynamodb_table",
  "sqs":            "aws_sqs_queue",
  "sns":            "aws_sns_topic",
  "lambda":         "aws_lambda_function",
  "rds":            "aws_db_instance",
  "secretsmanager": "aws_secretsmanager_secret"
}
```

### `groundwork://template/{service}`

Returns the reference `.tf` template for a given service. The AI can use this
as a starting point or ignore it — it is documentation, not a generation engine.

```
groundwork://template/s3
groundwork://template/dynamodb
groundwork://template/lambda
```

---

## Guardrails

### Path sandbox

Every path is validated against `--allowed-root` before any read or write.
Symlinks are resolved before checking. Path traversal (`../`) is rejected.

```
Error: sandbox: path "/etc/passwd" is outside allowed roots ["/home/user/projects"]
Error: sandbox: path "../../.bashrc" escapes base directory
```

### Explicit confirmation

`groundwork_write_files` requires `confirmed: true` in the request. Without
it, the tool returns an error message — nothing is written. This ensures the
AI always presents a preview before persisting files.

### Content validation

Before writing, each file is checked for:

- **Allowed extensions** — only `.tf`, `.hcl`, `.tfvars`, `.json`, `.yaml`, `.yml`
- **Binary content** — files containing null bytes are rejected
- **Exec provisioners** — `local-exec` and `remote-exec` blocks are blocked by
  default; opt-in with `--allow-exec-provisioners`

```
Error: file "main.tf" contains a local-exec or remote-exec provisioner block,
       which is disabled by default; start groundwork-mcp with
       --allow-exec-provisioners to permit.
```

### Size limits

| Limit | Default | Flag |
|-------|---------|------|
| Files per scan | 10 000 | `--max-files` |
| Total project size | 100 MB | `--max-project-bytes` |
| Single file size | 512 KB | `--max-file-bytes` |
| Files per write call | 50 | hardcoded in schema |

### Rate limiting

Token-bucket limiter: `--rate-limit` calls/second with 2× burst. Prevents
runaway AI loops from hammering the filesystem.

### Read-only mode

```bash
groundwork-mcp --allowed-root /workspace --readonly
```

`groundwork_write_files` is not registered. Only `groundwork_scan` and the
resources are available. Useful for code-review pipelines, audits, and demos.

### No credentials, no shell

The server never reads AWS credentials (`AWS_ACCESS_KEY_ID`, `~/.aws/`).
No shell commands are executed (`os/exec` is forbidden in the MCP layer).
The only I/O operations are: reading source files and writing `.tf`/`.hcl` text.

### Audit log

Every tool call is logged as a JSON line to stderr:

```json
{"ts":"2026-03-27T14:32:01Z","tool":"groundwork_write_files","result":"ok","files":3,"duration_ms":12}
{"ts":"2026-03-27T14:32:05Z","tool":"groundwork_write_files","result":"blocked","reason":"sandbox: path \"/etc\" is outside allowed roots"}
```

Redirect to a file without affecting the stdio transport:

```bash
groundwork-mcp --allowed-root /home/user/projects 2>> groundwork-mcp.log
```

---

## Example conversation

```
User: Generate the Terraform infra for the project at ./myapp.
      Use prod-tf-state as the state bucket, region us-east-1.

AI calls: groundwork_scan({ project_path: "/abs/path/myapp" })
→ { language: "python", services: ["s3", "dynamodb"] }

AI generates (using its own Terraform knowledge + detected context):
  - terragrunt.hcl            (remote state with prod-tf-state bucket)
  - modules/s3/terragrunt.hcl
  - modules/s3/main.tf        (bucket + versioning + SSE)
  - modules/dynamodb/terragrunt.hcl
  - modules/dynamodb/main.tf  (PAY_PER_REQUEST table)

AI: "I'll create these 5 files in ./myapp/infra. Shall I proceed?"

User: Yes

AI calls: groundwork_write_files({
  base_path: "/abs/path/myapp/infra",
  files: [ ... ],
  confirmed: true
})
→ { files_written: ["terragrunt.hcl", "modules/s3/...", "modules/dynamodb/..."] }

AI: "Done. Files written to ./myapp/infra.
     Next steps:
     1. Fill in the variable values in each module's terragrunt.hcl inputs block.
     2. Run: cd ./myapp/infra && terragrunt run-all init"
```

---

## Comparison: CLI vs MCP server

| | `groundwork` (CLI) | `groundwork-mcp` (MCP) |
|---|---|---|
| Generator | Go templates embedded in binary | The AI model |
| Interaction | One-shot command | Conversational, multi-turn |
| Review step | `--dry-run` flag | Built-in (confirmed=true required) |
| Customisation | Flags only | AI adapts to project context |
| Template updates | New binary release required | None — AI knowledge is up to date |
| Credentials needed | No | No |
| Shell execution | No | No |
