# diary-cli

Misskeyのノートを元にAI（Claude）と対話しながら日記を生成するCLIツール。

対話を通じてユーザーの言語化力を鍛えつつ、質の高い日記を生成することを目的とする。

## 全体フロー

1. Misskeyノート取得（当日 or 指定日）
2. ノートの前処理（時系列ソート・時間帯グルーピング）
3. Claude APIと対話セッション（事実確認 → 深掘り → 締め）
4. 対話履歴を元に日記Markdown生成
5. Misskeyサマリー生成
6. ファイル出力 → エディタで確認 → git push

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

対話的に設定ファイルを生成できます:

```bash
diary-cli init
```

`~/.config/diary-cli/config.yaml` が作成されます。

### 手動で設定する場合

`~/.config/diary-cli/config.yaml` を作成:

```yaml
misskey:
  instance_url: "https://mi.example.com"
  token: "your-misskey-token"

claude:
  api_key: "your-claude-api-key"
  model: "claude-sonnet-4-20250514"

diary:
  output_dir: "/path/to/diary"
  author: "Soli"
  editor: "vim"

chat:
  max_questions: 8
  min_questions: 3
```

## 使い方

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

## 対話セッション

`run` コマンドでは、Claudeが3つのフェーズで質問します:

1. **事実確認**（1〜3問）: ノートの経緯・背景を聞く
2. **深掘り**（1〜3問）: 感情・理由・内省を促す
3. **締め**（1〜2問）: 一日の総括

対話中に `/done` を入力すると終了します（最低3問は回答が必要）。

## 出力形式

```markdown
---
title: 2026-02-15
author: Soli
layout: post
date: 2026-02-15T23:30
category: 日記
---

# AIが提案するタイトル

対話から生成された日記本文

# Misskeyサマリー

時系列サマリー
```
