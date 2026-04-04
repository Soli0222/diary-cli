# diary-cli

Misskey のノートを取得し、AI で要約して日記ベースを生成する CLI ツール。

## 概要

指定した日付の Misskey ノートを収集し、時間帯ごとにグルーピングしたうえで AI に要約・タイトル生成を依頼します。結果は Markdown ファイル、テキスト、JSON のいずれかで出力でき、Discord Webhook への通知にも対応しています。

## インストール

### GitHub Releases

[Releases](https://github.com/soli0222/diary-cli/releases) から各プラットフォーム向けのバイナリをダウンロードできます。

### Go install

```bash
go install github.com/soli0222/diary-cli/cmd/diary-cli@latest
```

### ソースからビルド

```bash
git clone https://github.com/soli0222/diary-cli.git
cd diary-cli
make build    # ./diary-cli が生成される
make install  # $GOPATH/bin にインストール
```

## セットアップ

初回は `init` コマンドで設定ファイルを対話的に生成します。

```bash
diary-cli init
```

`~/.config/diary-cli/config.yaml` が作成されます。

## 使い方

### 基本

```bash
# 今日（05:00以前なら前日）の日記を生成
diary-cli run

# 昨日の日記を生成
diary-cli run --yesterday

# 日付を指定して生成
diary-cli run --date 2026-04-03
```

### AI プロバイダの切り替え

```bash
diary-cli run --provider openai
diary-cli run --provider gemini
```

### 出力形式の変更

```bash
diary-cli run --output summary   # テキストで標準出力
diary-cli run --output json      # JSON で標準出力
diary-cli run --output none      # 出力なし（自動化向け）
```

### Discord 通知

```bash
diary-cli run --discord
diary-cli run --output none --discord   # Discord のみに投稿
```

### Git への push

```bash
diary-cli push   # 生成済み Markdown を git add/commit/push
```

## コマンド一覧

| コマンド | 説明 |
|---------|------|
| `run` | ノート取得 → 前処理 → AI 要約 → タイトル生成 → 出力 |
| `summary` | `run --output summary` 相当（テキスト出力のみ） |
| `push` | 生成済み Markdown を `git add/commit/push` |
| `init` | 設定ファイルを対話的に生成 |
| `version` | バージョンを表示 |

### グローバルフラグ

| フラグ | 短縮 | 説明 |
|-------|------|------|
| `--date` | `-d` | 対象日を `YYYY-MM-DD` で指定（05:00 補正なし） |
| `--yesterday` | `-y` | 昨日の日記を作成 |

### `run` フラグ

| フラグ | 短縮 | デフォルト | 説明 |
|-------|------|----------|------|
| `--output` | `-o` | `markdown` | 出力形式（`markdown` / `summary` / `json` / `none`） |
| `--provider` | `-p` | 設定ファイル準拠 | AI プロバイダ（`claude` / `openai` / `gemini`） |
| `--discord` | — | `false` | Discord Webhook にも投稿 |

### `summary` フラグ

| フラグ | 短縮 | デフォルト | 説明 |
|-------|------|----------|------|
| `--provider` | `-p` | 設定ファイル準拠 | AI プロバイダ |
| `--discord` | — | `false` | Discord Webhook にも投稿 |

## 日付の解釈

日記の 1 日は **05:00 〜 翌 05:00** です。深夜 2 時のノートは前日分として扱われます。

| 条件 | 挙動 |
|------|------|
| フラグなし、05:00 以降に実行 | 当日が対象 |
| フラグなし、05:00 より前に実行 | 前日が対象 |
| `--yesterday` | 上記からさらに 1 日前 |
| `--date YYYY-MM-DD` | 指定日をそのまま使用（時刻補正なし） |

### ノートの時間帯グルーピング

| ラベル | 時間帯 |
|-------|--------|
| 早朝 | 05:00–09:00 |
| 午前 | 09:00–12:00 |
| 午後 | 12:00–17:00 |
| 夕方 | 17:00–21:00 |
| 夜 | 21:00–05:00 |

## 設定

### 設定ファイル

`~/.config/diary-cli/config.yaml`

```yaml
misskey:
  instance_url: "https://misskey.example.com"
  token: "your-token"

ai:
  default_provider: "claude"
  claude:
    api_key: "sk-ant-..."
    model: "claude-sonnet-4-6"
  openai:
    api_key: "sk-..."
    model: "gpt-5.4-mini"
  gemini:
    api_key: "AI..."
    model: "gemini-3.1-flash-preview"

diary:
  output_dir: "./diary"
  author: "your-name"
  timezone: "Asia/Tokyo"

summaly:
  endpoint: ""

discord:
  webhook_url: ""
```

### 環境変数

設定ファイルの値を環境変数で上書きできます。

| 環境変数 | 対応する設定 |
|---------|------------|
| `MISSKEY_INSTANCE_URL` | `misskey.instance_url` |
| `MISSKEY_TOKEN` | `misskey.token` |
| `ANTHROPIC_API_KEY` | `ai.claude.api_key` |
| `OPENAI_API_KEY` | `ai.openai.api_key` |
| `GOOGLE_API_KEY` | `ai.gemini.api_key` |
| `DISCORD_WEBHOOK_URL` | `discord.webhook_url` |

`diary.timezone` で日付解釈と時間帯グルーピングに使うタイムゾーンを指定できます。デフォルトは `Asia/Tokyo` です。

## AI プロバイダ

3 つの AI プロバイダに対応しており、すべて公式 Go SDK を使用しています。

| プロバイダ | デフォルトモデル | SDK |
|-----------|----------------|-----|
| Claude | `claude-sonnet-4-6` | [`anthropic-sdk-go`](https://github.com/anthropics/anthropic-sdk-go) |
| OpenAI | `gpt-5.4-mini` | [`openai-go`](https://github.com/openai/openai-go) |
| Gemini | `gemini-3.1-flash-preview` | [`google.golang.org/genai`](https://pkg.go.dev/google.golang.org/genai) |

`--provider` フラグまたは `ai.default_provider` で切り替えられます。モデルは設定ファイルで変更可能です。

## 出力形式

### Markdown（デフォルト）

`diary.output_dir/YYYY/MMDD.md` に保存されます。

```markdown
---
title: 2026-04-03
author: your-name
layout: post
date: 2026-04-03T05:00
category: 日記
---

# AI が生成したタイトル

# Misskeyサマリー

AI が生成した要約...
```

### Summary

テキスト形式で標準出力に出力されます。

### 標準出力と標準エラー

- `--output summary` は本文のみを標準出力に出力
- `--output json` は JSON のみを標準出力に出力
- 進捗や保存先、Discord 通知結果は標準エラーに出力
- `--output none` は標準出力に何も出力しない

```
2026-04-03 のサマリー
ノート数: 42
タイトル: 春の陽気に誘われて

要約テキスト...
```

### JSON

```json
{
  "date": "2026-04-03",
  "start_time": "2026-04-03T05:00:00+09:00",
  "end_time": "2026-04-04T05:00:00+09:00",
  "note_count": 42,
  "title": "春の陽気に誘われて",
  "summary": "要約テキスト...",
  "notes": [
    {
      "id": "...",
      "created_at": "2026-04-03T10:30:00+09:00",
      "text": "ノート本文"
    }
  ]
}
```

### None

出力なし。CronJob 等で `--discord` と組み合わせて使用します。

## デプロイ

### Docker

```bash
docker build -t diary-cli .
docker run --rm \
  -e MISSKEY_INSTANCE_URL=https://misskey.example.com \
  -e MISSKEY_TOKEN=your-token \
  -e ANTHROPIC_API_KEY=sk-ant-... \
  diary-cli run --yesterday --output summary
```

マルチプラットフォームイメージ（linux/amd64, linux/arm64）が [GitHub Container Registry](https://ghcr.io/soli0222/diary-cli) で公開されています。

### Kubernetes CronJob

毎日自動で日記を生成して Discord に投稿する構成例です。

```bash
# Secret を作成（k8s/secret.example.yaml を参考に）
kubectl apply -f k8s/secret.example.yaml

# CronJob をデプロイ
kubectl apply -f k8s/cronjob.yaml
```

CronJob は毎日 0:00 UTC に `diary-cli run --yesterday --output none --discord` を実行します。

## 開発

```bash
# テスト
make test

# ビルド
make build

# クリーン
make clean
```

### プロジェクト構成

```
cmd/diary-cli/        エントリポイント
internal/
  ai/                 AI プロバイダ（Claude, OpenAI, Gemini）
  cli/                コマンド定義・ワークフロー
  config/             設定ファイル読み込み・環境変数バインド
  discord/            Discord Webhook 連携
  generator/          出力フォーマッタ（Markdown, Summary, JSON）
  git/                git add/commit/push
  misskey/            Misskey API クライアント
  models/             データ構造（Note 等）
  preprocess/         ノートの時間帯グルーピング・Summaly リンク展開
k8s/                  Kubernetes マニフェスト
```

## ライセンス

MIT
