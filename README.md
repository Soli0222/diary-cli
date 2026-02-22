# diary-cli

Misskeyノートをもとに Claude と対話しながら日記を作成する CLI ツールです。

単発生成ではなく、毎日の利用でユーザープロファイルを学習し、質問の質を改善します。

## 主な機能

- Misskey ノート取得（当日/昨日/指定日）
- 3フェーズ対話（事実確認 -> 深掘り -> 締め）
- ノートが少ない日の専用対話モード
- 学習プロファイル（継続情報、未確認仮説、矛盾管理）
- 対話結果を使った日記本文・サマリー・タイトル生成
- run 完了時のメトリクス表示
- `stats` コマンドによる時系列メトリクス集計

## インストール

```bash
go install github.com/soli0222/diary-cli/cmd/diary-cli@latest
```

または:

```bash
git clone https://github.com/soli0222/diary-cli.git
cd diary-cli
make build
```

## セットアップ

対話式セットアップ:

```bash
diary-cli init
```

`~/.config/diary-cli/config.yaml` が作成されます。

## 設定ファイル

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

### 主要デフォルト

- `claude.model`: `claude-sonnet-4-6`
- `chat.max_questions`: `8`
- `chat.min_questions`: `3`
- `chat.summary_every`: `2`
- `chat.max_unknowns_before_confirm`: `3`
- `chat.empathy_style`: `balanced`
- `chat.profile_enabled`: `true`

## 使い方

```bash
# 日記作成（対話 -> 学習更新 -> 本文/サマリー/タイトル生成）
diary-cli run
diary-cli run --yesterday
diary-cli run --date 2026-02-14

# サマリーのみ生成（対話スキップ）
diary-cli summary
diary-cli summary --yesterday
diary-cli summary --date 2026-02-14

# 生成済みファイルをgit commit & push
diary-cli push
diary-cli push --yesterday
diary-cli push --date 2026-02-14

# 対話メトリクス集計
diary-cli stats
diary-cli stats --days 14
```

## run の挙動

`run` は以下を実行します。

1. Misskey ノート取得・前処理
2. 学習済みプロファイル読み込み
3. 対話セッション
4. プロファイル更新（学習抽出・矛盾管理・確認昇格）
5. 日記本文/サマリー/タイトル生成
6. Markdown 保存
7. 実行メトリクス表示・保存

対話中に `/done` で終了できます（`min_questions` 未満では終了不可）。

日付/収集範囲の扱い:
- `--yesterday` は「対象日を1日前にする」だけです（収集開始時刻を直接変えません）
- `--date YYYY-MM-DD` 指定時は、指定日のノートを1日分取得します
- `chat.profile_enabled=true` かつ `profile.json.updated_at` が有効な場合、`run` は `profile.updated_at` から実行開始時刻までのノートを取得します（`--date` 明示指定時を除く）
- `profile.updated_at` が不正/未来時刻/未設定の場合は、従来どおり対象日の1日分取得にフォールバックします

## 学習プロファイル

保存先:
- デフォルト: `~/.config/diary-cli/profile.json`

扱う情報（抜粋）:
- `stable_facts`
- `ongoing_topics`
- `effective_patterns`
- `sensitive_topics`
- `conflicts`
- `pending_confirmations`
- `confirmation_history`

概要:
- `inferred` は即反映せず pending に保持
- 確認質問で `confirmed` へ昇格
- 矛盾候補は `conflicts` に保持し、本反映は保留

## メトリクス

### run 直後に表示される項目

- 質問数
- 要約確認ターン数
- 構造化ターン数 / フォールバック数
- 確認結果（試行/確定/否定/不確実）
- 平均回答文字数
- 重複質問率
- プロファイル変化（stable/pending/conflicts）

### 保存先

- `~/.config/diary-cli/metrics.jsonl`（1行1JSON）

### stats で見られるもの

`diary-cli stats --days N` で、期間集計と日別サマリを表示します。

## 出力形式

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

## 開発

```bash
go test ./...
```

仕様詳細は `docs/spec.md` を参照してください。
