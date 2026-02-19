#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${MO_BIN:-}" ]]; then
  if command -v mo >/dev/null 2>&1; then
    MO_BIN="$(command -v mo)"
  else
    MO_BIN="/tmp/mo"
  fi
fi
SMOKE_TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-60}"
SMOKE_TO_EMAIL="${SMOKE_TO_EMAIL:-}"
SMOKE_TASK_LIST_ID="${SMOKE_TASK_LIST_ID:-}"

log() {
  printf '[smoke] %s\n' "$*"
}

fail() {
  printf '[smoke][FAIL] %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

run_mo_json() {
  local out
  if ! out=$("$MO_BIN" --json --force "$@" 2> >(sed 's/^/[mo stderr] /' >&2)); then
    fail "command failed: $MO_BIN --json --force $*"
  fi
  if ! jq -e . >/dev/null <<<"$out"; then
    fail "non-JSON output for command: $*"
  fi
  printf '%s' "$out"
}

assert_jq() {
  local json="$1"
  local expr="$2"
  local description="$3"
  shift 3
  if ! jq -e "$expr" "$@" >/dev/null <<<"$json"; then
    printf '[smoke][FAIL] %s\n' "$description" >&2
    printf '[smoke][FAIL] jq expr: %s\n' "$expr" >&2
    printf '[smoke][FAIL] json: %s\n' "$json" >&2
    exit 1
  fi
  log "PASS: $description"
}

wait_for_mail_in_sent() {
  local subject="$1"
  local deadline=$((SECONDS + SMOKE_TIMEOUT_SECONDS))
  while ((SECONDS < deadline)); do
    local sent_list
    sent_list=$(run_mo_json mail list --folder sentitems --max 25)
    local msg_id
    msg_id=$(jq -r --arg subject "$subject" '.items[]? | select(.subject == $subject) | .id' <<<"$sent_list" | head -n1)
    if [[ -n "$msg_id" ]]; then
      printf '%s' "$msg_id"
      return 0
    fi
    sleep 2
  done
  return 1
}

wait_for_calendar_subject() {
  local event_id="$1"
  local expected_subject="$2"
  local from_ts="$3"
  local to_ts="$4"
  local deadline=$((SECONDS + SMOKE_TIMEOUT_SECONDS))
  while ((SECONDS < deadline)); do
    local list_out
    list_out=$(run_mo_json calendar list --from "$from_ts" --to "$to_ts" --max 100)
    if jq -e --arg id "$event_id" --arg subject "$expected_subject" '.items | any(.id == $id and .subject == $subject)' >/dev/null <<<"$list_out"; then
      return 0
    fi
    sleep 2
  done
  return 1
}

wait_for_calendar_absent() {
  local event_id="$1"
  local from_ts="$2"
  local to_ts="$3"
  local deadline=$((SECONDS + SMOKE_TIMEOUT_SECONDS))
  while ((SECONDS < deadline)); do
    local list_out
    list_out=$(run_mo_json calendar list --from "$from_ts" --to "$to_ts" --max 100)
    if ! jq -e --arg id "$event_id" '.items | any(.id == $id)' >/dev/null <<<"$list_out"; then
      return 0
    fi
    sleep 2
  done
  return 1
}

require_cmd jq
require_cmd date
[[ -x "$MO_BIN" ]] || fail "MO binary is not executable: $MO_BIN"

task_id=""
task_list_id=""
event_id=""

cleanup() {
  set +e
  if [[ -n "$task_id" ]]; then
    local args=()
    if [[ -n "$task_list_id" ]]; then
      args+=(--list-id "$task_list_id")
    fi
    "$MO_BIN" --json --force tasks delete "$task_id" "${args[@]}" >/dev/null 2>&1 || true
  fi
  if [[ -n "$event_id" ]]; then
    "$MO_BIN" --json --force calendar delete "$event_id" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

log "Checking auth status"
auth_status=$(run_mo_json auth status)
assert_jq "$auth_status" '.has_credentials == true' 'credentials are available'

selected_client=$(jq -r '.client // empty' <<<"$auth_status")
if [[ -z "$selected_client" ]]; then
  selected_client="default"
fi

account=$(jq -r '.account // .default_account // empty' <<<"$auth_status")
if [[ -z "$account" ]]; then
  auth_list=$(run_mo_json auth list)
  account=$(jq -r --arg client "$selected_client" '([.items[]? | select(.client == $client)] | .[0].email) // .items[0].email // empty' <<<"$auth_list")
fi
[[ -n "$account" ]] || fail 'no account selected; run mo auth add <email> first'
export MO_CLIENT="$selected_client"
export MO_ACCOUNT="$account"

auth_status_selected=$(run_mo_json auth status)
if ! jq -e '.token_available == true' >/dev/null <<<"$auth_status_selected"; then
  fail "auth token is not available for $account (run: mo auth add $account --device)"
fi
log 'PASS: auth token is available for selected account'

send_to="$SMOKE_TO_EMAIL"
if [[ -z "$send_to" ]]; then
  send_to="$account"
fi
log "Using account: $account"
log "Send target: $send_to"

log "Testing mail list/get"
mail_list=$(run_mo_json mail list --max 5)
assert_jq "$mail_list" '.items | type == "array"' 'mail list returns an items array'

first_mail_id=$(jq -r '.items[0].id // empty' <<<"$mail_list")
if [[ -n "$first_mail_id" ]]; then
  mail_get=$(run_mo_json mail get "$first_mail_id")
  assert_jq "$mail_get" '.id == $id' 'mail get returns requested id' --arg id "$first_mail_id"
else
  log 'Mailbox has no messages; skipping existing-message read assertion'
fi

log "Testing mail send/read-back"
mail_subject="mo smoke mail $(date -u +%Y%m%dT%H%M%SZ)"
mail_body="mo smoke body $(date -u +%s)"
mail_send=$(run_mo_json mail send --to "$send_to" --subject "$mail_subject" --body "$mail_body")
assert_jq "$mail_send" '.status == "sent"' 'mail send returns sent status'

sent_mail_id=$(wait_for_mail_in_sent "$mail_subject") || fail 'sent message not found in sent items before timeout'
sent_mail=$(run_mo_json mail get "$sent_mail_id")
assert_jq "$sent_mail" '.subject == $subject' 'sent message subject matches' --arg subject "$mail_subject"

log "Testing tasks list/create/read/update/complete/delete"
default_tasks_list=$(run_mo_json tasks list --max 5)
assert_jq "$default_tasks_list" '.items | type == "array"' 'tasks list (default list) returns an items array'

create_task_args=(tasks create --title "mo smoke task $(date -u +%Y%m%dT%H%M%SZ)" --body "mo smoke task body")
if [[ -n "$SMOKE_TASK_LIST_ID" ]]; then
  create_task_args+=(--list-id "$SMOKE_TASK_LIST_ID")
fi
created_task=$(run_mo_json "${create_task_args[@]}")
assert_jq "$created_task" '.id | type == "string" and (. | length > 0)' 'tasks create returns task id'

task_id=$(jq -r '.id' <<<"$created_task")
task_list_id=$(jq -r '.parentListId // empty' <<<"$created_task")
if [[ -z "$task_list_id" && -n "$SMOKE_TASK_LIST_ID" ]]; then
  task_list_id="$SMOKE_TASK_LIST_ID"
fi

task_list_args=()
if [[ -n "$task_list_id" ]]; then
  task_list_args+=(--list-id "$task_list_id")
fi

list_after_create=$(run_mo_json tasks list --max 100 "${task_list_args[@]}")
assert_jq "$list_after_create" '.items | any(.id == $id)' 'created task appears in task list' --arg id "$task_id"

run_mo_json tasks update "$task_id" --status inProgress --importance high "${task_list_args[@]}" >/dev/null
list_after_update=$(run_mo_json tasks list --max 100 "${task_list_args[@]}")
assert_jq "$list_after_update" '.items | any(.id == $id and .status == "inProgress")' 'task update persisted in list output' --arg id "$task_id"

run_mo_json tasks complete "$task_id" "${task_list_args[@]}" >/dev/null
list_after_complete=$(run_mo_json tasks list --max 100 "${task_list_args[@]}")
assert_jq "$list_after_complete" '.items | any(.id == $id and .status == "completed")' 'task complete persisted in list output' --arg id "$task_id"

delete_task=$(run_mo_json tasks delete "$task_id" "${task_list_args[@]}")
assert_jq "$delete_task" '.deleted == true' 'tasks delete confirms deletion'
deleted_task_id="$task_id"
task_id=""

list_after_delete=$(run_mo_json tasks list --max 100 "${task_list_args[@]}")
assert_jq "$list_after_delete" '(.items | any(.id == $id)) | not' 'deleted task no longer appears in list' --arg id "$deleted_task_id"

log "Testing calendar create/read/update/delete"
cal_from=$(date -u -d '+20 minutes' +%Y-%m-%dT%H:%M:%SZ)
cal_to=$(date -u -d '+50 minutes' +%Y-%m-%dT%H:%M:%SZ)
cal_window_from=$(date -u -d '-10 minutes' +%Y-%m-%dT%H:%M:%SZ)
cal_window_to=$(date -u -d '+2 hours' +%Y-%m-%dT%H:%M:%SZ)
cal_subject="mo smoke calendar $(date -u +%Y%m%dT%H%M%SZ)"
cal_subject_updated="$cal_subject updated"

created_event=$(run_mo_json calendar create --summary "$cal_subject" --from "$cal_from" --to "$cal_to" --description 'mo smoke event')
assert_jq "$created_event" '.id | type == "string" and (. | length > 0)' 'calendar create returns event id'

event_id=$(jq -r '.id' <<<"$created_event")
if ! wait_for_calendar_subject "$event_id" "$cal_subject" "$cal_window_from" "$cal_window_to"; then
  fail 'created calendar event did not appear in list before timeout'
fi
log 'PASS: created calendar event appears in list output'

run_mo_json calendar update "$event_id" --summary "$cal_subject_updated" >/dev/null
if ! wait_for_calendar_subject "$event_id" "$cal_subject_updated" "$cal_window_from" "$cal_window_to"; then
  fail 'updated calendar summary did not appear in list before timeout'
fi
log 'PASS: calendar update persisted in list output'

delete_event=$(run_mo_json calendar delete "$event_id")
assert_jq "$delete_event" '.deleted == true' 'calendar delete confirms deletion'
deleted_event_id="$event_id"
event_id=""

if ! wait_for_calendar_absent "$deleted_event_id" "$cal_window_from" "$cal_window_to"; then
  fail 'deleted calendar event still appears in list after timeout'
fi
log 'PASS: deleted calendar event no longer appears in list output'

log 'All live smoke checks passed.'
