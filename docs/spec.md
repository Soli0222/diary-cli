# diary-cli 仕様書

## 概要

Misskeyのノートを元にAI（Claude）と対話しながら日記を生成するGo製CLIツール。
対話を通じてユーザーの言語化力を鍛えつつ、質の高い日記を生成することを目的とする。

## 背景・課題

- 既存フロー: テンプレート生成 → 手書き → misskey-summarizerの結果を貼り付け → git push
- 課題1: 言語化が苦手なため日記の質が低い
- 課題2: misskey-summarizerのサマリーが時系列を無視しており内容が悪い

## 全体フロー

```
1. Misskeyノート取得（当日 or 指定日）
2. ノートの前処理（時系列ソート・時間帯グルーピング・関連ノートまとめ）
3. 前処理済みノートをClaude APIに渡し、CLIで対話セッション開始
   - フェーズ1: 事実確認（ノートの経緯・背景を聞く）
   - フェーズ2: 深掘り（感情・理由・内省を促す）
   - フェーズ3: 締め（一日の総括）
4. 対話履歴を元にClaude APIで日記Markdown生成
5. 同じくClaude APIでMisskeyサマリーも生成
6. テンプレートに組み込んでファイル出力
7. ユーザーが確認（エディタで開く）
8. git commit & push
```

## コマンド体系

```bash
# 今日の日記を作成（対話 → 日記生成 → サマリー生成 → 保存）
diary-cli run

# 昨日の日記を作成
diary-cli run --yesterday

# 特定日の日記を作成
diary-cli run --date 2026-02-14

# サマリーのみ生成（対話スキップ）
diary-cli summary
diary-cli summary --yesterday
diary-cli summary --date 2026-02-14

# 生成済みファイルをgit commit & push
diary-cli push
diary-cli push --yesterday
diary-cli push --date 2026-02-14
```

## 対話セッション設計

### 基本方針

- 所要時間: 5〜10分
- 質問数: 最低3問、最大8問
- 終了方法: `/done` コマンド入力、または最大質問数到達

### フェーズ構成

**フェーズ1: 事実確認（1〜3問）**

ノートの時系列を見て、主要なトピックについて経緯・背景を質問する。

例:
- 「午前中に○○について投稿していましたが、これはどういう経緯でしたか？」
- 「夕方に△△のノートがありましたが、もう少し詳しく教えてください」

**フェーズ2: 深掘り（1〜3問）**

フェーズ1の回答を受けて、感情や理由、内省を促す質問をする。

例:
- 「それに対してどう感じましたか？」
- 「なぜそう思ったのですか？」
- 「その経験から何か気づいたことはありますか？」

**フェーズ3: 締め（1〜2問）**

一日の総括を促す。

例:
- 「今日一日を振り返って、一番印象に残ったことは？」
- 「今日を一言で表すと？」

### 対話中のCLI表示

```
📝 2026-02-15 の日記を作成します
📥 Misskeyから42件のノートを取得しました

--- 対話セッション開始 ---
（/done で終了、最大8問）

🤖 午前中にKubernetesのデプロイについて何度か投稿していましたが、
   何かトラブルがあったのですか？

> [ユーザー入力]

🤖 なるほど。それに対してどう対処しましたか？

> [ユーザー入力]

...

🤖 今日一日を振り返って、一番印象に残ったことは何ですか？

> [ユーザー入力]

--- 対話セッション終了 ---
📄 日記を生成中...
✅ 日記を保存しました: /path/to/2026/0215.md
エディタで開きますか？ (y/N)
```

## 出力形式

```markdown
---
title: 2026-02-15
author: Soli
layout: post
date: 2026-02-15T23:30
category: 日記
---

# <AIが提案するタイトル>

<対話から生成された日記本文（カジュアルな文体）>

# Misskeyサマリー

<Claude APIで生成された時系列サマリー>
```

### 日記本文の文体

- カジュアルな「だ・である」調、もしくは話し言葉寄り
- ユーザー本人が書いたように見える一人称視点
- 対話で引き出された内容を自然な文章にまとめる

### Misskeyサマリー

- ノートの内容を時系列順に整理
- 時間帯ごとのグルーピング（午前/午後/夜 など）
- 重要なトピックを抽出してまとめる

## 設定ファイル

パス: `~/.config/diary-cli/config.yaml`

```yaml
# Misskey設定
misskey:
  instance_url: "https://mi.example.com"
  token: "your-misskey-token"

# Claude API設定
claude:
  api_key: "your-claude-api-key"
  model: "claude-sonnet-4-6"

# 日記設定
diary:
  output_dir: "/path/to/diary"   # 日記リポジトリのパス
  author: "Soli"
  editor: "vim"                  # 確認時に開くエディタ（$EDITORフォールバック）

# 対話設定
chat:
  max_questions: 8
  min_questions: 3
```

## モジュール構成

```
diary-cli/
├── cmd/diary-cli/
│   └── main.go                  # エントリーポイント
├── internal/
│   ├── cli/                     # Cobraコマンド定義
│   │   ├── root.go
│   │   ├── run.go               # run サブコマンド
│   │   ├── summary.go           # summary サブコマンド
│   │   └── push.go              # push サブコマンド
│   ├── config/                  # 設定ファイル読み込み
│   │   └── config.go
│   ├── misskey/                 # Misskey APIクライアント（misskey-summarizerから移植）
│   │   ├── client.go            # API呼び出し
│   │   ├── aid.go               # AIDアルゴリズム（日付→ノートID範囲変換）
│   │   └── aid_test.go
│   ├── models/                  # データモデル（misskey-summarizerから移植）
│   │   └── note.go
│   ├── preprocess/              # ノートの前処理
│   │   └── grouper.go           # 時系列ソート・時間帯グルーピング・関連ノートまとめ
│   ├── chat/                    # 対話セッション管理
│   │   └── session.go           # CLI対話ループ・フェーズ制御
│   ├── claude/                  # Claude APIクライアント
│   │   └── client.go            # Messages API呼び出し
│   ├── generator/               # Markdown生成
│   │   └── diary.go             # 日記本文・サマリー・最終Markdown組み立て
│   └── git/                     # Git操作
│       └── push.go              # add・commit・push
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 技術スタック

| 項目 | 技術 |
|------|------|
| 言語 | Go |
| CLIフレームワーク | Cobra |
| 設定ファイル | Viper（YAML） |
| LLMバックエンド | Claude API（Anthropic Messages API） |
| Misskey API | misskey-summarizerから移植 |
| Git操作 | os/exec経由でgitコマンド実行 |

## misskey-summarizerからの移植範囲

| パッケージ | 移植 | 備考 |
|-----------|------|------|
| `internal/misskey/` | ✅ | ノート取得・AIDアルゴリズム |
| `internal/models/` | ✅ | ノートのデータモデル |
| `internal/openai/` | ❌ | Claude APIに置き換え |
| `internal/discord/` | ❌ | 不要 |

## Claude API利用箇所

1. **対話セッション**: ノート情報＋システムプロンプトを元に質問生成、ユーザー回答を受けて次の質問を生成
2. **日記本文生成**: 対話履歴全体を入力として、カジュアルな文体の日記を生成
3. **サマリー生成**: 前処理済みノートを入力として、時系列サマリーを生成
4. **タイトル生成**: 日記本文を元に、その日を表すタイトルを生成

## 今後の拡張候補（スコープ外）

- 過去の日記を参照して、継続的なトピックの追跡
- 週次・月次の振り返りサマリー生成
- 日記の文体学習（過去の日記からユーザーの文体を学習）
- Discord/Misskey への日記投稿機能