# DEP-0003: Chat Session TUI Migration

## Status
In Progress

### Implementation Status (2026-02-22)
- Phase 0: Pending
- Phase 1: Pending
- Phase 2: Pending
- Phase 3: Pending
- Phase 4: Pending

## Summary
`diary-cli run` の対話セッション入力UIを、標準入力の行編集依存（`bufio.Scanner`）からアプリ管理のTUIへ段階的に移行する。

主目的は、対話UXの改善（入力・再描画・状態表示）と、将来の拡張（履歴表示、複数行入力、ショートカット追加）の土台作りである。

## Background
現状の対話セッションは `internal/chat/session.go` で `bufio.Scanner(os.Stdin)` を使って行単位に入力を取得している。

- 入力中の編集（Backspace、カーソル移動、再描画）は端末とIMEに依存する
- 日本語（全角）や絵文字を含む表示で、端末側の文字幅計算と再描画が崩れることがある
- 質問数や状態の表示、入力エラー表示、将来的な複数行入力を自然に追加しにくい

一方で、会話ロジック（質問生成、状態更新、履歴管理）は既に `Session` にまとまっており、UI層を分離して段階的に置き換えやすい。

## Goals
- 対話セッションの入力/表示をアプリ側で制御できるようにする
- 現行CLIの会話ロジックを大きく壊さずにTUIを導入する
- 段階移行（CLIとTUI共存）を可能にする
- 日本語入力環境での実用性を検証しやすい構成にする

## Non-Goals
- 日本語IMEの全環境で完全な挙動保証
- `summary` / `push` / `init` コマンドの同時TUI化
- 対話ロジック（質問方針、プロンプト戦略）の仕様変更
- 最初の段階での高度なUI（複雑なレイアウト、テーマ機構）

## Decision
`run` の対話セッションに対して、TUI実装を追加する。導入は一括置換ではなく、入出力境界の抽象化を先に行う段階移行とする。

制御フロー方針として、初期実装は **チャネルブリッジ方式（Pattern A）** を採用する。

- `Session` は同期的な質問/回答ループの責務を維持する
- TUI（bubbletea）はイベントループを所有し、質問表示/回答入力をUI状態として管理する
- `Session` と TUI の間は channel で質問・状態通知・回答を受け渡す

これにより、既存 `Session` ロジックを大きく分解せずに段階移行できる。制御反転（Pattern B）は将来の改善候補とする。

初期実装は以下を満たす最小TUIとする。

- 質問表示
- 1行入力
- Enter送信
- 終了操作（`/done` と同等）
- 基本的なエラー表示
- 質問数/進捗の簡易表示

## Why TUI (and Why Not “IME Fix Only”)
- 現象の一部は端末/IME/文字幅再描画の相互作用であり、アプリ側で再描画を持つことで改善余地が大きい
- ただしTUI化はIME完全解決の保証ではない（端末・OS・IME・ライブラリ差分が残る）
- それでも、入力体験全体の改善と機能拡張の余地を同時に得られるため、投資対効果が高い

## Architecture Overview
対話ロジックとUI入出力を分離する。

### Current (simplified)
1. `Session.Run()`
2. 質問生成
3. `fmt.Printf(...)`
4. `scanner.Scan()`
5. 回答を状態へ反映

### Target (simplified)
1. `Session.RunWithUI(ctx context.Context, ui ChatUI)`（仮称）
2. 質問生成
3. `ui.ShowQuestion(..., qNum, qMax)`
4. `ui.ReadAnswer(ctx)`
5. 回答を状態へ反映

`ChatUI` は `cli` 実装と `tui` 実装を持てる構成にする。

## Proposed Interface (Concept)
詳細シグネチャは実装時に調整するが、責務は以下に限定する。

- セッション開始表示
- 質問表示
- 待機状態表示（Claude API 呼び出し中）
- 回答入力取得
- 補助メッセージ（警告/エラー/ヒント）表示
- セッション終了表示

重要: `ChatUI` は質問生成・状態更新・履歴構築を持たない。会話ロジックは `Session` 側に残す。

進捗表示（例: `Q 3/10`）は `ShowQuestion` の引数として質問番号/上限を渡す方式を初期方針とする。

### Exit / Input Result Semantics
`ReadAnswer(ctx)` の終了理由は初期段階で設計を固定する。

- 通常回答
- `/done` による明示終了
- `Ctrl+C` による中断
- EOF
- I/Oエラー

推奨設計（Phase 0 で確定）:
- `ReadAnswer(ctx context.Context) (string, error)`
- sentinel error を用いて終了理由を表現する（例: `ErrAborted`, `io.EOF`）

`/done` の最低質問数バリデーションは `Session` 側に残す（ドメイン/会話ルールであるため）。
したがって `ReadAnswer(ctx)` は `/done` を通常入力文字列として返す。

### Concurrency / Cancellation
チャネルブリッジ方式では `Session` と TUI が並行動作するため、停止協調に `context.Context` を用いる。

- `RunWithUI(ctx context.Context, ui ChatUI)` を初期設計に含める
- `ChatUI` の入力待ちも `ctx` を受け取れる形を優先する（例: `ReadAnswer(ctx)`）
- `Ctrl+C` 時は UI 側で `ErrAborted` を返し、必要に応じて `cancel()` で goroutine 停止を伝播する

## Library Choice
第一候補は `bubbletea` + `bubbles`（`textinput` または `textarea`）とする。

理由:
- Go製で既存コードベースとの統合がしやすい
- 状態遷移ベースで対話UIを組み立てやすい
- 段階的にUI機能を拡張しやすい

注意:
- 日本語IMEの挙動は端末環境差があるため、採用判断はプロトタイプ実機検証を前提とする
- 必要に応じてASCII中心表示（絵文字削減）をデフォルトにする

## UX Requirements (Phase 1 Minimum)
- 入力中の再描画が破綻しにくいこと
- 回答送信後に入力欄が確実にクリアされること
- 質問と回答の履歴が視認できること（直近数件でよい）
- Claude API 呼び出し中に待機状態（例: `Thinking...`）が視認できること
- `Ctrl+C` で安全に中断できること
- `/done` で従来どおり終了できること

## TUI Image (Wireframe)
初期実装では凝ったレイアウトより、読みやすさと再描画の安定性を優先する。

```text
+--------------------------------------------------------------+
| diary-cli run                                                |
| Q 3/10                                      mode: tui (exp)  |
+--------------------------------------------------------------+
| Assistant                                                    |
| 今日はどんな一日でしたか？仕事、生活、気分の順でも大丈夫です。 |
|                                                              |
| History (recent)                                             |
| - Q1: 朝から移動が多くて疲れた                                |
| - Q2: 午後に設計レビューをした                                |
|                                                              |
| Input (/done to finish)                                      |
| > 日本語入力テスト                                            |
|                                                              |
| Enter: send   Ctrl+C: abort                                  |
+--------------------------------------------------------------+
```

ローディング状態（例）:

```text
+--------------------------------------------------------------+
| diary-cli run                                                |
| Q 3/10                                      mode: tui (exp)  |
+--------------------------------------------------------------+
| Assistant                                                    |
| Thinking...                                                  |
|                                                              |
| History (recent)                                             |
| - Q1: 朝から移動が多くて疲れた                                |
| - Q2: 午後に設計レビューをした                                |
|                                                              |
| Input (disabled while waiting)                               |
| >                                                            |
+--------------------------------------------------------------+
```

表示方針（初期）:
- 1カラム構成
- ASCII中心の装飾（絵文字を使わない）
- 履歴は直近数件のみ
- 入力欄は1行（Phase 1）

## Compatibility and Rollout
初期導入では現行CLI経路を残す。

- 既定値は現行CLIのままでもよい
- オプトイン方式（例: `chat.ui = "tui"` または `--tui`）を優先する
- 安定後に既定値切替を検討する

これにより、日本語入力環境や端末差分の実地確認を行いながら移行できる。

## Risks
- TUIライブラリ導入による依存増加・学習コスト
- 日本語IMEが特定端末で依然として不安定な可能性
- テスト戦略の変更（UIイベントを伴うため単体テストしにくい箇所が増える）
- TUI異常終了時に端末状態（raw mode）が壊れるリスク

## Mitigations
- UI境界を薄く保ち、会話ロジックは既存テストを活かす
- TUIは最小機能から開始し、段階的に拡張する
- 実機確認マトリクス（Terminal / iTerm2 / VS Code terminal）を記録する
- 絵文字/装飾を最初は抑え、文字幅リスクを減らす
- TUI起動/終了で端末状態復元を `defer` ベースで徹底し、異常系（`Ctrl+C` / error）での復帰を手動確認する

## Implementation Plan

### Phase 0: UI境界の抽象化（前提）
Status: Pending

- `Session.Run()` の表示/入力部分を小さなインターフェースに分離
- 現行挙動を維持するCLI実装を追加
- `ReadAnswer(ctx)` の終了シグナル（`/done` / abort / EOF / error）設計を確定する
- `/done` 判定と最低質問数バリデーションを `Session` 側責務として明文化する
- 既存テストへの影響を最小化する（UI差分で壊さない）

### Phase 1: 最小TUIプロトタイプ（1行入力）
Status: Pending

- `run` 対話専用のTUI実装を追加
- 質問表示、1行入力、Enter送信、`/done` 終了を実装
- Claude API待機中のローディング表示（最小は `Thinking...`）を実装
- エラー時のメッセージ表示を追加

### Phase 2: 設定/フラグでの切替導入
Status: Pending

- `--tui` フラグまたは `chat.ui` 設定を追加
- `run` でUI実装を切り替える
- TUI終了後の後続プロンプト（例: エディタ起動確認）が正常に動作することを確認する
- 失敗時はCLI UIへフォールバック可能にする（可能なら）

### Phase 3: UX改善（履歴・進捗・中断）
Status: Pending

- 質問数/上限の表示
- 履歴表示改善
- `Ctrl+C` 中断時の扱い整理
- 表示文言をASCII中心に調整（必要に応じて）

### Phase 4: 実機検証と既定値方針の判断
Status: Pending

- 主要端末で日本語入力（削除、変換確定後編集、改行）を確認
- 既知の制約をREADMEへ記載
- 既定値をTUIにするか、オプトイン維持かを判断

## Detailed Implementation Plan
実装順は「ロジック保護 -> 最小TUI導入 -> 切替配線 -> UX改善」の順とする。

### Step 1: UI抽象化 + CLI UI分離（PR1相当, 最優先）
対象:
- `internal/chat/session.go`
- New: `internal/chat/ui.go`
- New: `internal/chat/ui_cli.go`

作業:
- `ChatUI` interface を定義（進捗情報、待機状態、入力取得を含む）
- `cliChatUI` 実装で `bufio.Scanner` ベースの現行挙動を分離
- `fmt.Println` / `fmt.Printf` / `Scanner` 読み取り箇所を `ChatUI` 呼び出しへ置換
- `Session.Run()` は互換維持のため残し、内部で `NewCLIChatUI(...)` を使う形にする
- TUI導入用に `RunWithUI(ctx context.Context, ui ChatUI)`（仮称）を追加する
- `ReadAnswer(ctx)` の戻り値設計（`/done` は文字列、abort/EOF は error）をここで固定する
- `/done` 判定と最低質問数チェックは `Session` 側に残し、UIは文字列入力と表示に専念させる
- `context.Context` による停止協調の流れを `Session` 側に追加する（少なくともシグネチャ導入）

完了条件:
- 現行CLI挙動で `run` がこれまでどおり動く
- `Session` の既存テストが通る（UI差分で壊れない）
- `RunWithUI()` の最低限テスト（mock UI）を追加できる形になっている

### Step 2: 最小TUIプロトタイプを追加する
対象:
- New: `internal/chat/ui_tui.go`
- `go.mod` / `go.sum`（依存追加）

作業:
- `bubbletea` + `bubbles/textinput`（Phase 1 の決定）で1行入力UIを実装
- 状態として最低限以下を持つ
  - 現在の質問文
  - 直近履歴（表示用）
  - 入力文字列
  - ステータスメッセージ（エラー/ヒント）
  - ローディング状態（Claude API 呼び出し中）
  - 質問番号/上限
- Enterで送信、`/done` で終了、`Ctrl+C` で中断
- `ChatUI` interface に合わせたブリッジを実装
- bubbletea イベントループと `Session` の同期ループは channel ブリッジで接続する
- `context.Context` / `cancel()` による停止協調を組み込む

完了条件:
- TUI経路で1セッション完走できる
- 日本語入力の基本ケース（確定後編集）が少なくとも1端末で実用レベル
- ローディング中にフリーズ誤認しにくい表示がある

### Step 3: `run` からのUI切替配線を入れる
対象:
- `internal/cli/run.go`
- `internal/config/config.go`（設定方式を採る場合）
- `internal/cli/root.go`（フラグ方式を採る場合）

作業:
- `--tui` もしくは `chat.ui` 設定を追加（DEPのOpen Questionをここで決定）
- `run` 実行時に `ChatUI` 実装を選択
- TUI初期化失敗時の扱いを決める（エラー終了 or CLIフォールバック）
- TUI終了後の `run` 後続プロンプト（エディタ確認など）の端末I/O復帰を確認する

推奨:
- 初期は `--tui` フラグ優先（明示的で試験導入しやすい）
- 後から `chat.ui` 設定を追加してもよい

完了条件:
- 同一バイナリでCLI/TUIを切替可能
- 非TUI利用者への後方互換が保たれる

### Step 4: UX改善と日本語入力検証
対象:
- `internal/chat/ui_tui.go`
- `README.md`

作業:
- 履歴表示の見やすさ改善（直近N件、折返し）
- 質問番号/残数表示
- 表示文字のASCII化（必要に応じて）
- `textarea` への拡張可否を判断（IME安定性を優先して保留/採用を決める）
- 手動検証結果（使える端末/制約）をREADMEに明記

完了条件:
- 最低限の利用ガイドと制約が共有されている
- 再現しやすい既知問題が記録されている

## Concrete Task Breakdown (Initial PR Slices)
レビューしやすい粒度に分割する。

1. PR1: `Session` のUI抽象化 + CLI UI 分離（挙動変更なし）
2. PR2: TUIプロトタイプ追加（未配線でも可、内部実験コマンドでも可）
3. PR3: `run` への `--tui` 配線
4. PR4: UX改善 + README反映 + 手動検証記録

## Testing Strategy
- 既存の `Session` ロジックテストは維持（UI非依存）
- UI境界はモック実装で単体テスト
- TUI固有はスモークテスト中心（手動確認を含む）

`RunWithUI()` に対するモックUIテスト例（Phase 0〜1）:
- 3問応答で `messages` が期待どおり蓄積される
- `/done` 入力で `minQuestions` 未満時に再入力導線が動く
- UIが `ErrAborted` を返したときに中断として終了する
- UIが `io.EOF` を返したときに安全に終了する

手動確認観点（最低限）:
- ASCIIのみ入力
- 日本語（全角）入力
- 日本語確定後のBackspace
- 長文で折り返しが発生するケース
- 連続送信時の入力欄クリア

## Affected Files (planned)
- Update: `internal/chat/session.go`
- New: `internal/chat/ui.go`（または同等のUI境界定義）
- New: `internal/chat/ui_cli.go`（現行CLI相当）
- New: `internal/chat/ui_tui.go`（TUI実装）
- Update: `internal/cli/run.go`（UI選択配線）
- Update: `internal/config/config.go`（設定で切替する場合）
- Update: `README.md`（使い方と既知制約）

## Acceptance Criteria
- 現行CLI経路を維持したまま、TUI経路で `run` 対話を完走できる
- `Session` の会話ロジックテストが大きく壊れない
- 日本語入力環境で少なくとも一部端末（例: macOS Terminal または iTerm2）で実用レベルの入力ができる
- TUI未使用時の既存挙動に後方互換性がある

## Open Questions
- UI切替は `--tui` フラグ優先か、設定ファイル優先か
- `textinput` から `textarea` へ拡張する判断基準（IME安定性・複数行需要・実機検証結果）
- TUI表示で絵文字をデフォルトで無効化するか
- TUI導入後に `summary` / `init` の対話にも横展開するか
