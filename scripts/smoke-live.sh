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
SMOKE_DRIVE_ENABLED="${SMOKE_DRIVE_ENABLED:-0}"
SMOKE_DRIVE_PARENT="${SMOKE_DRIVE_PARENT:-}"
SMOKE_DRIVE_SHARE_EMAIL="${SMOKE_DRIVE_SHARE_EMAIL:-}"
SMOKE_DRIVE_SHARED_CHECK="${SMOKE_DRIVE_SHARED_CHECK:-0}"

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

run_mo_error_json() {
  local stdout_file stderr_file out err
  stdout_file="$(mktemp /tmp/mo-smoke-stdout-XXXXXX.log)"
  stderr_file="$(mktemp /tmp/mo-smoke-stderr-XXXXXX.log)"

  if "$MO_BIN" --json --force "$@" >"$stdout_file" 2>"$stderr_file"; then
    rm -f "$stdout_file" "$stderr_file"
    fail "expected command to fail but it succeeded: $MO_BIN --json --force $*"
  fi

  out="$(cat "$stdout_file")"
  err="$(cat "$stderr_file")"
  rm -f "$stdout_file" "$stderr_file"

  if [[ -n "$err" ]]; then
    sed 's/^/[mo stderr] /' <<<"$err" >&2
  fi

  if jq -e '.error.code | type == "string"' >/dev/null <<<"$out"; then
    printf '%s' "$out"
    return 0
  fi
  if jq -e '.error.code | type == "string"' >/dev/null <<<"$err"; then
    printf '%s' "$err"
    return 0
  fi

  fail "expected JSON error output for command: $*"
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
drive_item_id=""
drive_folder_id=""
drive_tmp_file=""
drive_download_file=""

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
  if [[ -n "$drive_item_id" ]]; then
    "$MO_BIN" --json --force drive delete "$drive_item_id" >/dev/null 2>&1 || true
  fi
  if [[ -n "$drive_folder_id" ]]; then
    "$MO_BIN" --json --force drive delete "$drive_folder_id" >/dev/null 2>&1 || true
  fi
  if [[ -n "$drive_tmp_file" ]]; then
    rm -f "$drive_tmp_file" >/dev/null 2>&1 || true
  fi
  if [[ -n "$drive_download_file" ]]; then
    rm -f "$drive_download_file" >/dev/null 2>&1 || true
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

if [[ "$SMOKE_DRIVE_ENABLED" == "1" || "$SMOKE_DRIVE_ENABLED" == "true" ]]; then
  require_cmd cmp
  log "Testing drive commands"

  drive_list_out=$(run_mo_json drive drives --max 20)
  assert_jq "$drive_list_out" '.items | type == "array"' 'drive drives returns an items array'

  drive_tmp_file=$(mktemp /tmp/mo-smoke-drive-src-XXXXXX.txt)
  drive_download_file=$(mktemp /tmp/mo-smoke-drive-dst-XXXXXX.txt)
  printf 'mo smoke drive %s\n' "$(date -u +%Y%m%dT%H%M%SZ)" >"$drive_tmp_file"

  folder_name="mo smoke folder $(date -u +%Y%m%dT%H%M%SZ)"
  mkdir_args=(drive mkdir "$folder_name")
  if [[ -n "$SMOKE_DRIVE_PARENT" ]]; then
    mkdir_args+=(--parent "$SMOKE_DRIVE_PARENT")
  fi
  created_folder=$(run_mo_json "${mkdir_args[@]}")
  assert_jq "$created_folder" '.id | type == "string" and (. | length > 0)' 'drive mkdir returns folder id'
  drive_folder_id=$(jq -r '.id' <<<"$created_folder")

  upload_name="mo-smoke-upload-$(date -u +%Y%m%dT%H%M%SZ).txt"
  upload_args=(drive upload "$drive_tmp_file" --name "$upload_name" --conflict fail)
  if [[ -n "$SMOKE_DRIVE_PARENT" ]]; then
    upload_args+=(--parent "$SMOKE_DRIVE_PARENT")
  fi
  uploaded_item=$(run_mo_json "${upload_args[@]}")
  assert_jq "$uploaded_item" '.id | type == "string" and (. | length > 0)' 'drive upload returns item id'
  drive_item_id=$(jq -r '.id' <<<"$uploaded_item")

  ls_args=(drive ls --max 200)
  if [[ -n "$SMOKE_DRIVE_PARENT" ]]; then
    ls_args+=(--parent "$SMOKE_DRIVE_PARENT")
  fi
  listed_items=$(run_mo_json "${ls_args[@]}")
  assert_jq "$listed_items" '.items | any(.id == $id)' 'uploaded item appears in drive ls results' --arg id "$drive_item_id"

  get_item=$(run_mo_json drive get "$drive_item_id")
  assert_jq "$get_item" '.id == $id' 'drive get returns requested item id' --arg id "$drive_item_id"

  downloaded_item=$(run_mo_json drive download "$drive_item_id" --out "$drive_download_file")
  assert_jq "$downloaded_item" '.downloaded == true' 'drive download reports success'
  if ! cmp -s "$drive_tmp_file" "$drive_download_file"; then
    fail 'downloaded drive file content does not match uploaded content'
  fi
  log 'PASS: drive download content matches uploaded content'

  run_mo_json drive move "$drive_item_id" --parent "$drive_folder_id" >/dev/null
  moved_list=$(run_mo_json drive ls --parent "$drive_folder_id" --max 200)
  assert_jq "$moved_list" '.items | any(.id == $id)' 'drive move places item under destination folder' --arg id "$drive_item_id"

  renamed_name="${upload_name%.txt}-renamed.txt"
  run_mo_json drive rename "$drive_item_id" "$renamed_name" >/dev/null
  renamed_item=$(run_mo_json drive get "$drive_item_id")
  assert_jq "$renamed_item" '.name == $name' 'drive rename updates item name' --arg name "$renamed_name"

  perms_list=$(run_mo_json drive permissions "$drive_item_id" --max 50)
  assert_jq "$perms_list" '.items | type == "array"' 'drive permissions returns an items array'

  comments_err=$(run_mo_error_json drive comments "$drive_item_id")
  assert_jq "$comments_err" '.error.code == "not_implemented"' 'drive comments returns not_implemented'

  if [[ -n "$SMOKE_DRIVE_SHARE_EMAIL" ]]; then
    share_out=$(run_mo_json drive share "$drive_item_id" --to user --email "$SMOKE_DRIVE_SHARE_EMAIL" --role read)
    assert_jq "$share_out" '.shared == true' 'drive share reports success'
    shared_perm_id=$(jq -r '.value[0].id // .permission.id // .id // empty' <<<"$share_out")
    if [[ -n "$shared_perm_id" ]]; then
      unshare_out=$(run_mo_json drive unshare "$drive_item_id" "$shared_perm_id")
      assert_jq "$unshare_out" '.unshared == true' 'drive unshare reports success'
    else
      log 'Share response did not include a permission id; skipping unshare assertion'
    fi
  fi

  if [[ "$SMOKE_DRIVE_SHARED_CHECK" == "1" || "$SMOKE_DRIVE_SHARED_CHECK" == "true" ]]; then
    shared_out=$(run_mo_json drive shared --max 20)
    assert_jq "$shared_out" '.items | type == "array"' 'drive shared returns an items array'
  fi

  deleted_drive_item=$(run_mo_json drive delete "$drive_item_id")
  assert_jq "$deleted_drive_item" '.deleted == true' 'drive delete removes uploaded item'
  drive_item_id=""

  deleted_drive_folder=$(run_mo_json drive delete "$drive_folder_id")
  assert_jq "$deleted_drive_folder" '.deleted == true' 'drive delete removes created folder'
  drive_folder_id=""
fi

log 'All live smoke checks passed.'
