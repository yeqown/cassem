# Env Copy Design

## Goal

Add a pure-frontend copy flow to the Envs page. Users can copy selected elements from one environment to a new environment without refreshing the page. The copy keeps only the current content for each selected element and does not copy version history, publish history, or operations.

## Scope

- Add a `Copy` button next to `Add environment` on `EnvsPage`.
- Implement a two-step wizard: task creation and task execution.
- Use existing REST APIs only; no backend API changes.
- Load all source elements by paging through the elements list API.
- Copy selected elements concurrently and continue after per-element failures.
- Show progress during execution and final success, skipped, and failed results.

## Interaction Design

### Step 1: Task creation

The user opens the copy dialog from the Envs page.

Controls:

- `Source env`: select an existing env.
- `To env`: type the new env name. Input is normalized to lowercase.
- Element selection list: loads all elements from source env and selects all by default.
- Empty element strategy:
  - `Skip empty elements`
  - `Copy empty elements`

Validation:

- `To env` is required.
- `To env` must not exist in the current env list.
- `To env` must not equal `Source env`.
- `Start copy` stays disabled until validation passes, source elements are loaded, and at least one element is selected.

Summary shown before execution:

- Source env
- Target env
- Selected element count
- Empty element count
- Estimated skipped count based on strategy

### Step 2: Task execution

Clicking `Start copy` switches the dialog into execution mode.

Behavior:

- Dialog cannot be closed while copying.
- Back/cancel controls are hidden or disabled while copying.
- Envs page mutations (`Copy`, `Add environment`, `Delete`) are disabled while copying.
- A `beforeunload` handler warns if the user refreshes or closes the browser tab during copying.
- Completion removes the `beforeunload` handler and enables result actions.

Progress display:

- `Create env`: pending, doing, done, or failed
- `Copy elements`: pending, doing, or done, shown as completed count over total count
- `Complete`: pending or done

Example:

```text
Create env [done] -> Copy elements 2/12 [doing] -> Complete [pending]
```

Final actions:

- `View copied env`: navigate to `/apps/:appId/envs/:toEnv/elements`
- `Close`: close dialog and refresh env list

## Data Flow

### Loading source elements

`requestAllElements(appId, sourceEnv)` calls the existing list endpoint repeatedly:

```text
GET /api/apps/:appId/envs/:sourceEnv/elements?limit=100&seek=:nextSeek
```

It stops when `hasMore` is false. The list result is used for selection, counts, and empty-state hints only.

### Copy execution

1. Create target env:

```text
POST /api/apps/:appId/envs/:toEnv
```

2. Build a queue from selected elements.
3. Run a fixed worker pool with 4 concurrent element copy tasks.
4. For each selected element:
   - Fetch current source detail:

```text
GET /api/apps/:appId/envs/:sourceEnv/elements/:key
```

   - Treat an element as empty if it has no `metadata.usingVersion` or if `raw === ''`.
   - If empty strategy is `Skip empty elements`, record the element as skipped and do not create it in target env.
   - If empty strategy is `Copy empty elements`, create it in target env with `raw` preserved as `''` when empty.
   - Create target element:

```text
POST /api/apps/:appId/envs/:toEnv/elements/:key
```

Body:

```json
{
  "raw": "current source raw content",
  "contentType": 1
}
```

5. Record success, skipped, and failed results per key.
6. Continue copying remaining elements after any per-element failure.

Only the detail response `raw` and `metadata.contentType` are copied. Versions, publish state, operations, and historical content are not copied.

## Error Handling

- Source element list load failure keeps the user in task creation mode and shows an error state.
- Target env creation failure stops the task because no safe target exists.
- Per-element detail or create failure records that element as failed and continues with other elements.
- Final result shows counts and a failed/skipped element list with error messages.
- If target env creation succeeds but all selected elements fail or skip, the target env still exists and the final result makes that clear.

## Code Boundaries

Main work belongs in:

- `internal/cassemadm/web/src/features/envs/EnvsPage.tsx`

If the page becomes hard to read, extract focused code in the same feature folder, such as:

- `CopyEnvDialog.tsx` for dialog state and rendering
- small pure helpers for paging, worker-pool execution, and result summaries

Do not add global abstractions or backend endpoints.

## Testing

Frontend tests should cover:

- `Copy` button appears next to `Add environment`.
- Target env duplicate validation disables `Start copy` while typing.
- Source element list defaults to all selected.
- Empty element strategy affects expected skipped count.
- Execution shows create, copy, and complete progress.
- Per-element failure does not stop remaining copies and appears in final results.

Manual verification should run the Web UI and exercise the happy path and a per-element failure path if backend fixtures make that practical. If a full backend is unavailable, note the limitation and verify with mocked API tests plus UI rendering checks.
