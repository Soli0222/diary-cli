# DEP-0002: Profile UpdatedAt Window and Date Offset Semantics

## Status
In Progress

### Implementation Status (2026-02-22)
- Phase 0: Done
  - `run` 冒頭で `execution_now` を固定
  - `resolveTargetDate(now, flagDate, flagYesterday)` を追加
  - `target_date` をローカル midnight に正規化
- Phase 1: Done
  - `resolveCollectionWindowForRun(...)` を追加
  - `--date` 明示指定時の日単位レンジ分岐を追加
  - `profile.updated_at` の RFC3339 パースとフォールバック理由判定を追加
- Phase 2: Done
  - `fetchNotesForRun(...)` を追加
  - `run` で `GetNotesForTimeRange` / `GetNotesForDay` を分岐
  - `fetchNotes()` は据え置き（`summary` 互換維持）
- Phase 4: Partially Done
  - Done: `resolveTargetDate` の単体テスト追加 (`internal/cli/root_test.go`)
  - Done: `resolveCollectionWindowForRun` の単体テスト追加 (`internal/cli/run_window_test.go`)
  - Pending: 手動確認シナリオの実施・記録
- Phase 3: Done
  - `run` 内の `date` 変数を `targetDate` にリネーム
  - `execution_now` を `run` 冒頭固定へ統一（Phase 0 と整合）
  - `resolveDate()` はラッパー化され、`target_date` 解決は `resolveTargetDate(...)` に分離
- Phase 5: Done
  - Done: `README.md` を新仕様（`--yesterday` / `profile.updated_at` ベース収集）に更新
  - Done: `docs/spec.md` に `target_date` / `collection_window` 分離を反映
- Phase 6: Pending

## Summary
`diary-cli run` のノート収集対象期間（時間レンジ）を、`profile.json` が存在し `updated_at` が有効な場合は `profile.updated_at` から実行開始時刻 (`execution_now`) までとして扱う（ただし `--date` 明示指定時は除く）。

あわせて `--yesterday` は「対象日付を `-1` 日するだけ」の意味に限定し、収集時間レンジの起点を直接決めるフラグとしては扱わない。

## Background
深夜（例: 02:00）に前日分の日記を書く運用では、以下の2つが混ざると意図しない対象範囲になりやすい。

- 日記の対象日（例: 2026-02-22）
- ノート収集の時間レンジ（例: 前回実行後から今まで）

現状認識として、収集レンジが「前日 22:30 から実行時刻まで」のような運用/判定になっている。
これだと `--yesterday` の意図（対象日を1日前にしたい）と、収集レンジの意図（前回実行以降を取りたい）が結合してしまう。

## Goals
- `run` の収集時間レンジを `profile.updated_at` ベースにする
- `--yesterday` の意味を「対象日のオフセット」に限定する
- 深夜に前日分の日記を書く運用を自然に扱えるようにする
- 既存の `--date` / `--yesterday` の利用感を大きく崩さない

## Non-Goals
- `summary` / `push` の収集ロジック変更（`summary` は将来適用の余地はあるが本DEPの主対象は `run`）
- `profile.json` スキーマ変更（`updated_at` は既存項目を利用）
- `22:30` の由来や設定値化の整理

## Terms
- `execution_now`: コマンド実行開始時刻（ローカルタイムゾーン）
- `target_date`: 日記ファイル名・表示・プロンプト上の「対象日」
- `collection_window`: `run` が Misskey ノートを取得する時間レンジ `[start, end)`

## Specification

### 1. 対象日 (`target_date`) の決定
`target_date` は従来どおりルートフラグから決定する。ただし意味を明確化する。

- `--date YYYY-MM-DD` 指定時: その日付を採用
- `--yesterday` 指定時: `execution_now` の日付をローカルタイムゾーンで `-1` 日した日付を採用
- 両方未指定時: `execution_now` の日付を採用
- `--date` と `--yesterday` が同時指定された場合: `--date` 優先（現行互換）

重要: `--yesterday` は **`target_date` をずらすだけ** であり、`collection_window.start` を直接変更しない。
また `target_date` は日付概念として扱い、実装上はローカルタイムゾーンの `00:00:00` に正規化された時刻として保持する。

### 2. 収集時間レンジ (`collection_window`) の決定 (`run`)
`run` のノート収集レンジは以下の優先順位で決定する。

前提:
- `--date` が明示指定された場合は、過去日付指定の期待と整合させるため `profile.updated_at` を使わず、日単位レンジを使う
  - `collection_window = [target_date 00:00, target_date+1day 00:00)`
  - 実装上は従来どおり `GetNotesForDay(target_date)` と同等でよい
- 以下の `profile.updated_at` ベース判定は、`--date` 未指定時（通常実行 / `--yesterday`）にのみ適用する

1. `profile.json` が読み込める
2. `profile.updated_at` が存在する
3. `profile.updated_at` を時刻として正しく解釈できる
4. `profile.updated_at < execution_now`

上記を満たす場合:
- `collection_window.start = profile.updated_at`
- `collection_window.end = execution_now`

満たさない場合（フォールバック）:
- 既存の収集ロジックを維持する
- 本DEPではフォールバックの具体的境界値（例: 前日22:30起点）は既存実装/既存挙動に従う
- `profile_enabled = false` の場合も、`profile` は空として扱われるため常に本フォールバックに入る

### 3. `target_date` と `collection_window` の分離
`target_date` と `collection_window` は独立した概念として扱う。

- 日記タイトル/保存先/表示文言/日付メタデータは `target_date` を使う
- Misskeyノート取得対象は `collection_window` を使う

これにより、深夜 02:00 に `--yesterday` を付けて実行した場合でも、対象日は前日扱いのまま、収集対象は前回実行以降の最新ノートを含められる。

補足:
- `collection_window.end` は `execution_now`（実行開始時刻）で固定する
- 対話セッション中に新たに投稿されたノートは本回 `run` の収集対象に含めない（仕様上の意図的な非対象）
- 実運用上、ユーザーは対話セッション中にノート投稿しない前提を置き、このギャップは許容する

### 4. 異常系・境界条件
- `profile.updated_at` が空文字: フォールバック
- `profile.updated_at` が不正フォーマット: 警告を出してフォールバック
- `profile.updated_at` が未来時刻: 警告を出してフォールバック（`now` に丸めない）
- `profile.updated_at == execution_now` 付近で実質空レンジ: 空取得を許容
- タイムゾーンは `updated_at` のオフセットを尊重し、内部比較は `time.Time` 比較で行う
- `updated_at` は「前回プロファイル保存時刻」であり「前回ノート収集完了時刻」とは厳密一致しない。アンカーとしての近似利用であることを許容する

## Example Scenarios

### Example A: 深夜に前日分を書く
- 前回 `run` 完了時 (`profile.updated_at`): `2026-02-22T22:30:00+09:00`
- 今回実行時刻 (`execution_now`): `2026-02-23T02:00:00+09:00`
- フラグ: `--yesterday`

期待値:
- `target_date = 2026-02-22`
- `collection_window = [2026-02-22 22:30, 2026-02-23 02:00)`

### Example B: 同日中の再実行
- 前回 `run` 完了時: `2026-02-23T08:10:00+09:00`
- 今回実行時刻: `2026-02-23T09:00:00+09:00`
- フラグなし

期待値:
- `target_date = 2026-02-23`
- `collection_window = [08:10, 09:00)`

### Example C: 初回実行（profileなし）
- `profile.json` が存在しない
- フラグ: `--yesterday`

期待値:
- `target_date` は `execution_now` の前日
- `collection_window` は既存フォールバック挙動

### Example D: `--date` 明示指定で過去日を再生成
- 今回実行時刻 (`execution_now`): `2026-02-23T10:00:00+09:00`
- `profile.updated_at`: `2026-02-23T09:30:00+09:00`
- フラグ: `--date 2026-02-15`

期待値:
- `target_date = 2026-02-15`
- `collection_window = [2026-02-15 00:00, 2026-02-16 00:00)`（`updated_at` は使わない）

## Implementation Notes (Planned)
- `resolveDate()` は `target_date` 専用として維持し、返り値は日付（midnight）へ正規化する
- `run` に `collection_window` 解決処理を追加（例: `resolveCollectionWindow(...)`）
- `profile.updated_at` のパースは RFC3339 を第一候補にする（現行保存形式と一致）
- `internal/misskey/client.go` には `GetNotesForTimeRange` が既にあるため、それを利用する分岐を追加する
- `fetchNotes()` は変更せず、`summary` 互換性維持のため `run` 専用取得ヘルパーを追加する

## Implementation Plan

### Phase 0: 時刻基準の固定と `target_date` 解決の純化（前提）
Status: Done

`execution_now` を `run` 冒頭で固定できる形に先に整える。

- `run` 冒頭で `execution_now := time.Now()` を取得し、以降の処理で使い回す
- `resolveDate()` の内部 `time.Now()` 依存を外すため、小さな純関数を追加する
  - 例: `resolveTargetDate(now time.Time, flagDate string, flagYesterday bool)`
- `resolveDate()` は既存インターフェース維持のラッパーでもよい
- `target_date` として返す値は常にローカルタイムゾーンの midnight に正規化する

### Phase 1: `collection_window` 解決ロジックの導入（`run` 内完結）
Status: Done

`run` コマンドでのみ動作する `collection_window` 解決ロジックを入れる。

- `internal/cli/run.go` に `collection_window` 解決ヘルパーを追加する
- 入力:
  - `execution_now time.Time`
  - `prof *profile.UserProfile`
  - `isExplicitDate bool`
  - `targetDate time.Time`
- 出力:
  - `start time.Time`
  - `end time.Time`
  - `ok bool` (`profile.updated_at` を採用できたか)
  - `reason string`（ログ/警告表示用。例: `explicit_date`, `empty_updated_at`, `invalid_updated_at`, `future_updated_at`）

想定シグネチャ例:

```go
func resolveCollectionWindowForRun(now, targetDate time.Time, isExplicitDate bool, prof *profile.UserProfile) (start, end time.Time, ok bool, reason string)
```

実装方針:
- `isExplicitDate == true` の場合は `targetDate` の日単位レンジを返す（`ok=false, reason=explicit_date` でもよい）
- `prof == nil` なら不採用
- `prof.UpdatedAt == ""` なら不採用
- `time.Parse(time.RFC3339, prof.UpdatedAt)` でパース
- `!parsed.Before(now)` の場合は不採用（未来時刻/同時刻）
- 採用時は `[parsed, now)` を返す

### Phase 2: ノート取得の分岐を `collection_window` 対応に変更
Status: Done

`run` の Misskey ノート取得で、日単位取得と時間レンジ取得を分岐する。

- 既存の `fetchNotes(cfg, date)` は変更しない（`summary` 互換維持のため残す）
- 新規に `run` 専用の取得ヘルパーを追加する
  - 例: `fetchNotesForRun(cfg, targetDate, executionNow, prof)`
- `collection_window` が採用できた場合:
  - `client.GetNotesForTimeRange(me.ID, start, end, false)` を使用
- 採用できない場合:
  - 既存どおり `client.GetNotesForDay(me.ID, targetDate, false)` を使用

ログ表示（任意だが推奨）:
- 採用時: `profile.updated_at` ベースの時間レンジを表示
- 不採用時: フォールバック理由を簡潔に表示（警告レベル）

### Phase 3: `target_date` の意味の明確化（コードコメント/表示文言）
Status: Done

挙動変更時に混乱しやすい箇所を明示する。

- `internal/cli/root.go`
  - `resolveDate()` コメントを `target_date` 決定関数として明確化
- `internal/cli/run.go`
  - `date` 変数名を必要に応じて `targetDate` にリネーム（局所的）
  - `execution_now` を `run` 冒頭で固定した値に統一（Phase 0 で実施済み前提）

注意:
- 日記 frontmatter 用の時刻生成に使う `execution_now` と、収集レンジ終端は同一値を使う方がテストしやすい

### Phase 4: テスト追加（優先度高）
Status: Partially Done

今回の仕様は時間境界バグが起きやすいため、ヘルパー関数を単体テスト可能にする。

#### 4.1 `collection_window` 解決テスト（新規）
新規テストファイル案:
- `internal/cli/run_window_test.go`

ケース:
- `isExplicitDate == true` の場合、`profile.updated_at` を無視して `targetDate` の日単位レンジになる
- `profile nil` -> 不採用
- `updated_at empty` -> 不採用
- `updated_at invalid` -> 不採用
- `updated_at future` -> 不採用
- `updated_at == now` -> 不採用
- `updated_at < now` -> 採用
- タイムゾーン付きRFC3339（`+09:00`）を正しく解釈

#### 4.2 `--yesterday` 意味の維持テスト（既存/追加）
`resolveDate()` は既に単純だが、仕様明文化に合わせてテスト追加を検討。

新規テストファイル案:
- `internal/cli/root_test.go`

ケース:
- フラグなし -> 当日
- `--yesterday` -> `-1 day`
- `--date` 指定 -> 指定日
- `--date` + `--yesterday` -> `--date` 優先
- 返り値が常に midnight（時分秒ゼロ）であること

備考:
- Phase 0 で `resolveTargetDate(now, flagDate, flagYesterday)` を抽出し、その純関数を主にテスト対象とする

### Phase 5: ドキュメント更新
Status: Done

実装と合わせて利用者向け説明を更新する。

- `README.md`
  - `--yesterday` の説明を「対象日を1日前にする」に修正
  - `run` が `profile.updated_at` を用いて前回実行以降を収集する旨を追記（`profile_enabled` 時）
- `docs/spec.md`
  - `target_date` と `collection_window` の概念分離を反映
  - `run` のみ `collection_window` を使うことを明記

### Phase 6: 動作確認（手動）
Status: Pending

自動テストに加え、ローカルで最低限の手動確認を行う。

1. `profile.json` なしで `run --yesterday`
2. `profile.json.updated_at` を過去時刻にした状態で `run --yesterday`
3. `profile.json.updated_at` を不正文字列にした状態で `run`
4. `profile.json.updated_at` を未来時刻にした状態で `run`

確認ポイント:
- 対象日の表示は `--yesterday` で前日になる
- ノート取得は `updated_at` ベースに切り替わる（採用時）
- 不正 `updated_at` でもクラッシュしない
- `--date YYYY-MM-DD` 指定時は `updated_at` を無視して対象日の日単位取得になる

## Rollout Strategy
- まず `run` のみに適用する（`summary` / `push` は据え置き）
- フォールバック挙動は維持して、既存利用者への影響を限定する
- 実装後に実利用で違和感があれば `collection_window` 表示/警告の粒度を調整する
- `fetchNotes()` は変更しないことで `summary` の既存挙動を保つ

## Risks and Mitigations
- リスク: `updated_at` が「プロファイル保存時刻」であり、ユーザーの期待する「前回ノート収集完了時刻」と完全一致しない
  - 対応: 本DEPでは既存項目流用を優先し、必要なら将来 `last_run_at` 導入を別DEPで検討
- リスク: `updated_at` をアンカーにしたことで、将来 `run` 以外の処理で `updated_at` が更新されると収集レンジが意図せず狭まる
  - 対応: 本実装では `run` における `profile.Save()` 起点の更新のみを前提として明文化し、`updated_at` 更新責務の見直しは別タスクで扱う
- リスク: `target_date` と収集レンジの不一致で、日記内容に翌日未明の投稿が混ざる
  - 対応: これは仕様上許容（深夜運用対応のため）。必要なら将来オプション化
- リスク: `run.go` の責務肥大化
  - 対応: `resolveCollectionWindow...` / `fetchNotesForRun...` の小関数化で分離

## Affected Areas (Planned)
- `internal/cli/root.go` (`target_date` の意味の明確化)
- `internal/cli/run.go` (`collection_window` 解決と取得分岐)
- `internal/misskey/client.go` (`GetNotesForTimeRange` 利用)
- `README.md` / `docs/spec.md` (`--yesterday` 説明の明確化)

## Acceptance Criteria
- `profile.updated_at` がある `run` 実行で、ノート収集が `updated_at` 以降に限定される
- `--yesterday` 指定時でも収集レンジ開始は `updated_at` が優先される
- 深夜実行（例: 02:00）で `--yesterday` を付けると、対象日は前日になりつつ、当日深夜分ノートも収集できる
- `profile.updated_at` 不正時にクラッシュせずフォールバックする

## Open Questions
- フォールバック挙動（現状の「前日22:30起点」想定）をこの機会に明文化・設定化するか
- `summary` コマンドにも同じ `collection_window` 概念を導入するか
- `target_date` と `collection_window` の不一致が大きい場合のUI表示（警告/確認）を入れるか
