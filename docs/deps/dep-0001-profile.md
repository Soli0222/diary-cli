# DEP-0001: Daily Learning Profile for Interviewer

## Status
In Progress

### Implementation Status (2026-02-20)
- Phase A: Done
  - `internal/profile` の型・保存・読み込み・バージョン管理を実装
  - `run` へのプロファイル読み込み/保存配線を実装
- Phase B: Partial
  - Done: `session` へのプロファイル要約注入、`summary_every` / `max_unknowns_before_confirm` / `empathy_style` を使った質問ヒント制御
  - Not Done: 「自然文出力」と「内部状態更新」を厳密に分離したターンスキーマ運用（現状はヒントベース）
- Phase C: Partial
  - Done: 対話履歴からの更新候補抽出（Claude JSON）とマージ処理
  - Done: 矛盾候補を `conflicts` として永続化し、衝突値を本反映せず `pending_confirmations` に退避
  - Done: `inferred` は本反映せず `pending_confirmations` に保持し、`observed/confirmed` 到来時のみ本反映
  - Not Done: 確認質問の成功を明示的にトラッキングして `confirmed` 昇格させるターン連動制御
- Phase D: Not Started
  - 観測指標（要約確認率・重複質問率・回答文字数推移）の計測基盤
- Cross-cutting: Done
  - Config: `chat.profile_enabled`, `chat.profile_path`, `chat.summary_every`, `chat.max_unknowns_before_confirm`, `chat.empathy_style` を追加
  - Tests: `internal/profile` と `internal/chat` に追加テストを実装

## Summary
`diary-cli run` の対話セッションに、日次で蓄積されるユーザープロファイルを導入する。
これにより、インタビュアー（Claude）が日をまたいでユーザー理解を深め、重複質問を減らし、より引き出しやすい質問を行えるようにする。

## Background
現状の対話は当日のノートと当日会話に依存しており、前日の学びが次回に継承されない。
日記は毎日使う機能であるため、継続利用で対話品質が改善する設計が必要。

## Goals
- 毎日の利用で、質問の文脈適合性を向上させる
- 既知情報の重複質問を減らす
- 継続トピック（数日単位の課題）を追跡する
- 対話失敗時も日記生成を止めない（フォールトトレラント）

## Non-Goals
- 複数ユーザー切替
- クラウド同期
- 長期履歴に対する高度な検索・推薦

## User Experience
- 初回利用時: 従来どおりの対話
- 継続利用時: 前回文脈を踏まえた質問が増える
- 失敗時: プロファイルが使えなくても `run` は完走する

## Architecture Overview
`run` の流れに以下を追加する。

1. プロファイル読み込み
2. 対話セッションへプロファイル要約を注入
3. 対話終了後に学習抽出（当日会話 -> 更新候補）
4. 更新候補を既存プロファイルへマージ
5. 保存

既存フロー（ノート取得 -> 対話 -> 日記生成）は維持する。

## Data Model
新規 `UserProfile` を導入する。

- `version`: スキーマバージョン
- `updated_at`: 最終更新日時
- `stable_facts[]`: 高信頼の継続情報
- `preferences`: 対話スタイル嗜好
- `ongoing_topics[]`: 継続トピック
- `effective_patterns[]`: 有効だった質問パターン
- `sensitive_topics[]`: 慎重に扱う話題

各要素は以下メタを持つ。

- `confidence` (0.0-1.0)
- `last_seen`
- `source_date`
- `status` (`observed` / `inferred` / `confirmed`)

## Storage
- 保存先デフォルト: `~/.config/diary-cli/profile.json`
- 書き込み権限: 0600
- 読み込み失敗: 空プロファイルで継続
- 書き込み失敗: 警告のみ表示し `run` 継続
- 破損時: `.bak` 退避して再初期化

## Prompting and Turn Policy
対話生成は「自然文出力」と「内部状態更新」を分離する。

### Internal Turn Schema (concept)
- `empathy_line`
- `summary_check`
- `next_question`
- `state_updates`

### Policy
- 1ターン1質問
- `summary_every` ターンごとに要約確認を挿入
- `unknowns` が閾値超過時は確認優先
- 感情強度が高い場合は、解決提案より受容・言語化を優先
- 既知情報の再質問を避ける
- 継続トピックは「前回からの変化」を優先して聞く

## Merge Rules
- `observed`: 即時反映可
- `inferred`: 低信頼で保持し、次回確認で昇格
- `confirmed`: `stable_facts` へ昇格
- 矛盾発生時: 上書きせず `conflict` 扱いとして確認質問を生成
- 古い情報は `decay` で優先度低下

## Configuration Changes
`chat` 設定に以下を追加。

- `profile_enabled` (default: `true`)
- `profile_path` (default: empty -> 標準パス)
- `summary_every` (default: `2`)
- `max_unknowns_before_confirm` (default: `3`)
- `empathy_style` (`light` / `balanced` / `deep`, default: `balanced`)

## Implementation Plan

### Phase A (Foundation)
- `internal/profile` 追加（型・読み書き・バージョン）
- `run` でプロファイル読み込み・保存配線

### Phase B (Session Integration)
- `session` にプロファイル要約注入
- 要約確認周期と確認優先ロジック追加

### Phase C (Learning Loop)
- 対話履歴から更新候補抽出
- confidence/矛盾/decay を含むマージ実装

### Phase D (Quality Metrics)
- 観測指標を追加（ログベース）
  - 要約確認率
  - 重複質問率
  - ユーザー回答文字数の推移

## Affected Files (planned)
- New: `internal/profile/types.go`
- New: `internal/profile/store.go`
- New: `internal/profile/merge.go`
- New: `internal/chat/state.go`
- New: `internal/chat/turn.go`
- Update: `internal/chat/session.go`
- Update: `internal/cli/run.go`
- Update: `internal/config/config.go`
- Update: `internal/cli/init.go`
- New tests: `internal/profile/*_test.go`, `internal/chat/*_test.go`

## Failure Handling
- 構造化出力パース失敗: 質問テキストのみのフォールバック
- 学習抽出失敗: 学習スキップで継続
- 保存失敗: 警告のみで継続

## Acceptance Criteria
- 3日連続利用で、前日文脈に触れる質問が確認できる
- 既知情報の重複質問が減少する
- プロファイル読み書きの失敗でも `run` が完走する
- 既存の質問数・フェーズ制御テストが壊れない

## Open Questions
- `effective_patterns` の定義粒度（テンプレート単位か特徴量単位か）
- `decay` の係数と適用タイミング
- センシティブ情報の最小保持方針（完全非保持の境界）
