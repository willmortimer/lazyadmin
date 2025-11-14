# User Experience Specification

This document defines the TUI layout, keybindings, and user experience rules.

## Views

### Operations View

**Purpose**: Display and execute individual operations.

**Layout**:
- List of operations (filtered by current principal's roles)
- Status bar showing current filter
- Details area showing last executed operation result

**Display Rules**:
- Only operations allowed by current principal are shown
- Operations are grouped by type in the list
- Filter status is displayed in the status bar
- Last operation result is shown in details area

**Keybindings**:
- `↑` / `↓` or `j` / `k`: Navigate operation list
- `Enter`: Execute selected operation
- `a`: Show all operations (clear filter)
- `h`: Filter to HTTP operations only
- `p`: Filter to Postgres operations only
- `t`: Switch to Tasks view
- `l`: Switch to Logs view
- `?`: Show Help view
- `q` / `Ctrl+C`: Quit application

### Tasks View

**Purpose**: Display and execute multi-step tasks.

**Layout**:
- List of tasks (filtered by current principal's roles)
- Status bar indicating Tasks view
- Details area showing last executed task result and summary

**Display Rules**:
- Only tasks allowed by current principal are shown
- Tasks display risk level in description
- Last task result shows success status and rendered summary
- Summary template output is displayed line by line

**Keybindings**:
- `↑` / `↓` or `j` / `k`: Navigate task list
- `Enter`: Execute selected task
- `t`: Switch to Operations view
- `l`: Switch to Logs view
- `?`: Show Help view
- `q` / `Ctrl+C`: Quit application

### Logs View

**Purpose**: Display recent audit log entries.

**Layout**:
- Table of audit log entries
- Columns: Time, User, Operation, Success
- Most recent entries first

**Display Rules**:
- Shows last 50 entries by default
- Entries ordered by time descending
- Success indicated with ✓ or ✗ symbols
- Table is scrollable

**Keybindings**:
- `↑` / `↓`: Navigate log entries
- `q` / `Esc`: Return to previous view

### Help View

**Purpose**: Display keybinding reference.

**Layout**:
- List of keybindings grouped by view
- Brief description of each binding

**Keybindings**:
- Any key: Return to previous view

## Visual Indicators

### Risk Levels

Tasks display risk level in their description:
- Format: `task:{id} (risk:{level})`
- Levels: `low`, `medium`, `high`

**Future**: High-risk tasks may be visually highlighted (e.g., different color or prefix).

### Operation Results

Operation execution results are displayed as:
- Success: `Output: {result_string}`
- Failure: `Error: {error_message}`

### Task Results

Task execution results include:
- Success status: `Success: true` or `Success: false`
- Summary template output (if provided)
- Step-by-step results available in summary context

## UX Invariants

### Filtering

- Operations view filtering (all/HTTP/Postgres) only applies to operations view
- Tasks view has no filtering
- Filter state persists when switching between operations and tasks views

### Execution Feedback

- Operations execute asynchronously
- TUI remains responsive during execution
- Results appear in details area after completion
- No blocking UI during execution

### Error Display

- Errors are displayed in details area
- Error messages are preserved verbatim
- Failed operations/tasks are clearly marked

### View Transitions

- Switching views preserves list selection where possible
- Help view returns to previous view (not always main)
- Logs view returns to previous view (not always main)

## Accessibility Considerations

### Keyboard Navigation

- All functionality accessible via keyboard
- No mouse requirements
- Standard navigation keys (arrows, Enter, Esc)

### Display

- Works in standard terminal environments
- No special terminal features required
- Color is optional (works in monochrome)

### Error Messages

- Error messages are clear and actionable
- Technical details preserved for debugging
- User-friendly context provided where possible

## Future Enhancements

Potential UX improvements:

- Visual highlighting for high-risk tasks
- Progress indicators for long-running tasks
- Step-by-step progress display during task execution
- Search/filter within operations and tasks lists
- Customizable keybindings
- Color themes
- Operation/task parameter prompts

