# ab – Azure DevOps Boards CLI wrapper

ab is a lightweight, ergonomic wrapper around the Azure CLI (az) for
day‑to‑day Azure DevOps Boards workflows. It focuses on work items
(User Stories and Tasks), providing interactive pickers and forms,
pretty Markdown rendering, and convenient commands for creating,
editing, listing, and moving items through your Kanban flow.

> 🤖 *The development of `ab` was assisted by a large-language model
> (GPT-5 with Medium reasoning), specifically using
> [Codex CLI](https://developers.openai.com/codex/cli).*

## ab --help

```console
ab is a thin wrapper around Azure CLI (az) boards commands for common workflows.

Usage:
  ab [command]

Available Commands:
  backward    Move a work-item backward one Kanban column
  close       Set work-item state to Closed
  completion  Generate the autocompletion script for the specified shell
  create      Create work-items
  delete      Delete a work-item
  edit        Edit a work-item (title, description, assignee, state, column)
  forward     Push a work-item forward
  help        Help about any command
  list        List work-items
  renew       Set work-item state to New
  resolve     Set work-item state to Resolved
  show        Show a work-item and its details
  workon      Assign to me and move to Active

Flags:
      --confirm string    Confirmation mode: always|mutations|never (overrides AB_CONFIRM)
  -d, --default-columns   Use default Agile columns: New,Active,Resolved,Closed (overrides AB_COLUMNS)
  -h, --help              help for ab
  -P, --po-order          Order items by PO priority where possible (StackRank for Stories/Bugs)
  -s, --silent            Silent mode: do not print az commands, only outputs
  -v, --version           version for ab
  -y, --yes               Do not prompt; equivalent to --confirm never

Use "ab [command] --help" for more information about a command.

Author: Michel Blomgren <michel.blomgren@nionit.com>
```

## Features

- Interactive create/edit forms
  ([charmbracelet/huh](https://github.com/charmbracelet/huh)) with
  required Title validation.
- Pretty output with Markdown rendering
  ([charmbracelet/glamour](https://github.com/charmbracelet/glamour))
  and terminal-width wrapping.
- Cross-platform terminal width detection (Linux, macOS/Darwin, FreeBSD,
  Windows) without platform-specific syscalls.
- Parent-aware listing and pickers for bulk actions.
- Kanban column transitions using dynamic `WEF_*_Kanban.Column`
  fields.
- Bulk resolve/renew/close/delete with multi-select.
- Safe execution with confirm prompts; `--yes` and `--silent` to
  streamline.

## Install

- With Go: `go install github.com/sa6mwa/ab@latest`
- Or build from source in this repo:

```console
go build -trimpath -ldflags "-s -w" && sudo install ab /usr/local/bin/
```

### Cross-compiling

`ab` cross-compiles cleanly using the standard Go toolchain. Examples:

```bash
# Windows (x86_64)
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o ab.exe

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o ab

# FreeBSD (x86_64)
GOOS=freebsd GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o ab
```

Note: While the codebase compiles for Linux, macOS/Darwin, FreeBSD, and
Windows, the tool is currently tested regularly on Linux only. If you run
into a platform-specific issue, please open an issue with details.

## Prerequisites

- Azure CLI installed and signed in (`az login`).
- Azure DevOps defaults configured (organization and project):
  - `az devops configure --defaults organization=https://dev.azure.com/<org> project=<project>`

## `ab` is opinionated

User Stories progress through a predefined set of columns
(`board.ColumnOrder`). These columns are tailored to our team's
workflow at [Nion (We Make IT Easy)](https://nionit.com/) and are not
part of the default Azure DevOps *Agile process*. Extracting the
column setup via the `az` CLI or REST endpoints proved too complex,
which is why it is hardcoded. We utilize the default *States*, but
have customized and renamed columns for User Stories. For Tasks that
lack a `_Kanban.Column` field, the behavior follows the standard
`Agile process` with its default states. If you wish to configure your
Kanban board differently, you can adjust the columns and their order
using the `AB_COLUMNS` environment variable, separating each column
with a comma (`,`). Alternatively, use the global flag `-d` / `--default-columns`
to switch to the Azure DevOps Agile default flow: `New,Active,Resolved,Closed`.

## Quick Start

- List open items: `ab list`
- Create a story interactively: `ab create story -a @me`
- Create a task under a story: `ab create task -p 1234`
- Edit an item: `ab edit 1234`
- Start work on an item: `ab workon 1234` or `ab workon` (picker)
- Move forward/backward on the board: `ab forward 1234`, `ab backward 1234`

## Usage Examples

- List everything that’s not Closed
  - `ab list`
  - Save listing to a file: `ab list -o backlog.md` or `ab list -O` (prompts; default `ab-list.md`).
- List only Tasks or only User Stories
  - `ab list tasks`
  - `ab list stories`
- Parent-aware listing
  - `ab list 1234` prints parent summary and its children table.
  - Save children listing: `ab list 1234 -o ab1234.md` or `ab list 1234 -O` (prompts; default `ab1234.md`).

- Create a User Story
  - Interactive (no title): `ab create story -a @me`
    - Opens a form with Title, Kanban Column, Assignee, Description, and Acceptance Criteria.
    - If `-a @me` is used, Assignee is prefilled with your UPN.
  - Non-interactive: `ab create story "As a user, I want..." [-a @me]`

- Create a Task
  - Requires a parent User Story.
  - With picker: `ab create task` then select a parent and fill the form.
  - With flags: `ab create task -p 1234 -a @me "Do the thing"`

- Create a Bug
  - Requires a parent User Story (bugs are linked as children).
  - With picker: `ab create bug` then select a parent and fill the form.
  - With flags: `ab create bug -p 1234 --severity 2 -a @me "Something is broken"`
  - Severity accepts `1|2|3|4` and maps to `1 - Critical`, `2 - High`, `3 - Medium` (default), `4 - Low`.

- Show a Work Item
  - `ab show` opens a picker; or `ab show 1234` directly.
  - Outputs a Markdown document with compact headings:
    - Title; for Bugs, Severity appears immediately under Title.
    - Created By, Assignee.
    - For User Stories: Column, Acceptance Criteria.
    - State, Description.
  - Appends a `# Children` section listing child work-items (same table as `list <id>`).
  - Save output to file:
    - `ab show 1234 -o ab1234.md` writes the generated Markdown after printing it.
    - `ab show 1234 -O` prompts for a path (default `ab1234.md`). Paths starting with `~/` or `~user/` are expanded to home directories.

- Edit a Work Item
  - `ab edit` opens a picker; or `ab edit 1234` directly.
  - User Story form: Title, Kanban Column, Assignee, Description (MD), Acceptance Criteria (MD).
  - Bug form: Title, Severity, State (New/Active/Resolved/Closed), Assignee, Description (MD).
  - Task form: Title, State (New/Active/Closed), Assignee, Description (MD).
  - Title is required; Description/Acceptance Criteria convert Markdown ↔ HTML automatically.

- Flow and State
  - `ab workon [id]` assigns the item to you and moves it to Active.
  - `ab forward [id]` / `ab backward [id]` move by Kanban column using board order.
  - Bulk state changes (multi-select when no IDs):
    - `ab resolve`, `ab renew`, `ab close`, `ab delete`

## Flags and Behavior

- `--yes, -y`: Skips confirmations (same as `--confirm never`).
- `--confirm <always|mutations|never>`: Confirmation policy (default is to confirm).
- `--silent, -s`: Suppress printing az commands; only show outputs.
 - `--default-columns, -d`: Use Azure DevOps Agile default columns (`New,Active,Resolved,Closed`).
 - `--po-order, -P`: Global flag. Order items by PO priority where possible (Stories/Bugs by StackRank, others by date). Affects list output, pickers, and commands. Can be set via `AB_PO_ORDER=true` (also accepts `AB_STACKRANK=true`).

## Rendering and TUI

- Output uses glamour for Markdown rendering, wrapped to your terminal width.
- Interactive forms and pickers are powered by huh.
- Picker rows use “ID | T | Title” where T is the type’s initial.

## How It Works

- ab shells out to `az` and uses Azure Boards JSON responses for behavior.
- Transitions set the relevant WEF_*_Kanban.Column field; Azure maps states.
- Assignee `@me` resolves to your signed-in userPrincipalName via `az ad`.

## Troubleshooting

- Ensure `az devops configure` defaults are set; many commands depend on them.
- Use `--confirm always` to see and approve every `az` command.
- Use `--silent` to hide `az` command lines if your terminal is noisy.

## License

MIT, see `LICENSE` file.
