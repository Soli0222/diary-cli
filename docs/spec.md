# diary-cli 仕様書

## 1. 概要

`diary-cli` は Misskey のノートを材料に、Claude との対話で日記を作る Go 製 CLI ツール。
単発の質問生成ではなく、日次利用を前提にユーザープロファイルを学習し、使うほど質問の文脈適合性を高める。

主要目的:
- 日記の言語化支援（事実 -> 感情 -> 総括）
- 継続利用でのインタビュー品質向上
- 実行メトリクス可視化による改善確認

## 2. コマンド

```bash
# 設定ファイルを対話生成
diary-cli init

# 日記作成（対話 -> 学習更新 -> 本文/サマリー/タイトル生成）
diary-cli run
diary-cli run --yesterday
diary-cli run --date 2026-02-14

# サマリーのみ生成（対話スキップ）
diary-cli summary
diary-cli summary --yesterday
diary-cli summary --date 2026-02-14

# 生成済み日記を git add/commit/push
diary-cli push
diary-cli push --yesterday
diary-cli push --date 2026-02-14

# 対話メトリクス集計
# （デフォルト直近7日）
diary-cli stats
diary-cli stats --days 14

# バージョン表示
diary-cli version
```

補足:
- ルートの共通フラグは `--date` / `--yesterday`。
- `stats` は `--days` で参照範囲を指定。
- `--yesterday` は対象日を1日前にするためのフラグ（収集開始時刻の指定ではない）。

## 3. 実行フロー（run）

1. 設定読み込み（`~/.config/diary-cli/config.yaml`）
2. Misskey ノート取得
   - `target_date`（日記の対象日）を決定
   - `collection_window`（ノート収集時間レンジ）を決定
   - `run` は `collection_window` に従って取得
3. ノート前処理（グルーピング・整形、必要に応じ Summaly enrich）
4. プロファイル読み込み（`profile_enabled=true` 時）
5. 対話セッション実行
   - フェーズ制御
   - 構造化ターン生成（JSON）
   - 確認質問の判定・記録
6. プロファイル学習更新
   - 対話から更新候補抽出
   - conflict/pending を考慮してマージ
   - 確認結果を適用して pending -> confirmed 昇格
7. 日記本文・サマリー・タイトル生成
8. Markdown 出力保存
9. 実行メトリクス表示 + 永続化
10. 任意でエディタ起動

ノート件数が 0 件の場合は対話を開始せず終了する。

### 3.1 日付と収集レンジ (`run`)

`run` では、日記の対象日 (`target_date`) と Misskey ノート取得範囲 (`collection_window`) を分離して扱う。

- `target_date`
  - `--date YYYY-MM-DD`: 指定日（ローカルタイムゾーンの 00:00:00）
  - `--yesterday`: 実行開始時刻の日付を `-1` 日した日（00:00:00）
  - 指定なし: 実行開始時刻の当日（00:00:00）
- `collection_window`
  - `--date` 明示指定時: 対象日の1日分 (`[target_date, target_date+1day)`)
  - `--date` 未指定かつ `chat.profile_enabled=true` かつ `profile.updated_at` が有効: `[profile.updated_at, execution_now)`
  - 上記以外: 対象日の1日分取得にフォールバック

補足:
- `profile.updated_at` が不正/未来時刻/未設定の場合はフォールバックする
- `summary` / `push` は引き続き日付ベースで動作し、`collection_window` 拡張は `run` のみ

## 4. 対話設計

### 4.1 フェーズ

- 通常日（ノート件数 >= 10）: 比率 `3:3:2`
  - フェーズ1 事実確認
  - フェーズ2 深掘り
  - フェーズ3 締め
- 少ノート日（ノート件数 < 10）: 比率 `2:4:2`
  - フェーズ1 概要把握
  - フェーズ2 深掘り（未投稿時間帯を積極的に聞く）
  - フェーズ3 締め

`max_questions` に応じて境界を動的計算し、各フェーズ最低1問を保証する。

### 4.2 ターン生成

- Claude へは JSON スキーマでの応答を要求
- 期待フィールド例: `intent`, `summary_check`, `question`, `confirmation_target`
- JSON パース失敗時はテキスト質問へフォールバック

制約:
- 1ターン1質問
- 日本語
- 不要な前置きなし

### 4.3 寄り添い・確認制御

- `summary_every` ターンごとに要約確認を促進
- `max_unknowns_before_confirm` 超過時は確認質問を優先
- `pending_confirmations` がある場合、該当仮説の確認質問を優先

## 5. 学習プロファイル

### 5.1 保存先

- デフォルト: `~/.config/diary-cli/profile.json`
- カスタム: `chat.profile_path`

補足:
- `updated_at` はプロファイル保存時刻（RFC3339）で、`run` のノート収集レンジ開始の近似アンカーとして利用される（`--date` 明示指定時を除く）

### 5.2 データ構造（主要）

- `stable_facts`
- `ongoing_topics`
- `effective_patterns`
- `sensitive_topics`
- `preferences`
- `conflicts`
- `pending_confirmations`
- `confirmation_history`

### 5.3 更新ルール

- `observed` / `confirmed`: 本反映対象
- `inferred`: 本反映せず `pending_confirmations` に保持
- `conflicts`: 本反映せず pending 化
- decay: `last_seen` が古い項目は confidence を減衰

### 5.4 確認結果の適用

セッションで確認質問が発生した場合、回答を判定してプロファイルへ反映する。

判定:
- ルールベース（肯定/否定トークン）
- 不確実時は LLM 判定（`confirmed|denied|uncertain` + reason）

反映:
- `confirmed`: pending の確証カウント到達で確定昇格
- `denied`: pending から除外
- 全結果を `confirmation_history` に記録（`method`, `reason`, `question`, `answer`）

## 6. メトリクス

### 6.1 run 完了時表示

- 質問数
- 要約確認ターン数
- 構造化ターン数 / フォールバック数
- 確認結果（試行/確定/否定/不確実）
- 平均回答文字数
- 重複質問率
- プロファイル変化（stable / pending / conflicts）

### 6.2 永続化

- 保存先: `~/.config/diary-cli/metrics.jsonl`
- 形式: 1行1JSON（RunMetrics）

### 6.3 stats 集計

`diary-cli stats --days N` で以下を表示:
- 期間全体の実行回数・平均質問数
- 要約確認率
- 確認成功率
- 平均回答文字数
- 平均重複質問率
- 日別サマリ

## 7. 設定ファイル

パス: `~/.config/diary-cli/config.yaml`

```yaml
misskey:
  instance_url: "https://mi.example.com"
  token: "your-misskey-token"

claude:
  api_key: "your-claude-api-key"
  model: "claude-sonnet-4-6"

diary:
  output_dir: "/path/to/diary"
  author: "Soli"
  editor: "vim"

chat:
  max_questions: 8
  min_questions: 3
  summary_every: 2
  max_unknowns_before_confirm: 3
  empathy_style: "balanced" # light|balanced|deep
  profile_enabled: true
  profile_path: "" # 空なら ~/.config/diary-cli/profile.json

summaly:
  endpoint: ""
```

デフォルト:
- `claude.model`: `claude-sonnet-4-6`
- `chat.max_questions`: `8`
- `chat.min_questions`: `3`
- `chat.summary_every`: `2`
- `chat.max_unknowns_before_confirm`: `3`
- `chat.empathy_style`: `balanced`
- `chat.profile_enabled`: `true`

## 8. 出力形式

```markdown
---
title: YYYY-MM-DD
author: <author>
layout: post
date: YYYY-MM-DDTHH:mm
category: 日記
---

# <生成タイトル>

<日記本文>

# Misskeyサマリー

<時系列サマリー>
```

## 9. 失敗時挙動

- プロファイル読み込み失敗: 空プロファイルで継続
- プロファイル JSON 破損: バックアップ退避後に空プロファイルへフォールバック
- 学習抽出失敗: 日記生成は継続
- 構造化ターン失敗: テキスト質問へフォールバック
- メトリクス保存失敗: 警告表示のみで継続

## 10. モジュール構成

```text
diary-cli/
├── cmd/diary-cli/main.go
├── internal/
│   ├── chat/
│   │   ├── session.go
│   │   ├── state.go
│   │   └── turn.go
│   ├── profile/
│   │   ├── types.go
│   │   ├── store.go
│   │   ├── learn.go
│   │   ├── merge.go
│   │   └── confirm.go
│   ├── metrics/
│   │   └── store.go
│   ├── cli/
│   │   ├── root.go
│   │   ├── init.go
│   │   ├── run.go
│   │   ├── summary.go
│   │   ├── push.go
│   │   ├── stats.go
│   │   └── version.go
│   ├── claude/client.go
│   ├── generator/diary.go
│   ├── preprocess/
│   ├── misskey/
│   ├── models/
│   └── git/push.go
└── docs/spec.md
```

## 11. 現在の到達点

- DEP-0001（プロファイル学習 + 対話構造化 + メトリクス可視化）: 実装済み
- 追加改善余地:
  - メトリクスの外部出力（CSV/Markdown）
  - より厳密な類似質問判定
  - 複数ユーザー切替対応
