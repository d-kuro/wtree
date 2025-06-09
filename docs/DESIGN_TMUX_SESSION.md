# tmux セッション管理設計

## 概要

Claude Code実行時のセッション管理とログ保存に特化したtmux連携機能の設計です。複雑なレイアウト管理は行わず、Claude Codeのプロセス管理とログ記録に焦点を当てます。

## 基本コンセプト

### セッション管理の目的

1. **プロセス永続化**: Claude Code実行をターミナル接続に依存させない
2. **ログ保存**: すべてのClaude Code出力を自動的に記録
3. **監視機能**: 実行中のClaude Codeインスタンスの状態確認
4. **デタッチ/アタッチ**: ターミナルを閉じても処理継続

### セッション命名規則

```
gwq-claude-{task-id}-{timestamp}
```

例：
- `gwq-claude-abc123-20240115103045`
- `gwq-claude-def456-20240115110230`

## アーキテクチャ

### tmux Session Manager

```go
type SessionManager struct {
    config    *SessionConfig
    tmuxCmd   *TmuxCommand
    logger    *SessionLogger
}

type ClaudeSession struct {
    ID          string    `json:"id"`
    SessionName string    `json:"session_name"`
    TaskID      string    `json:"task_id"`
    WorktreePath string   `json:"worktree_path"`
    Command     string    `json:"command"`
    PID         int       `json:"pid"`
    StartTime   time.Time `json:"start_time"`
    Status      Status    `json:"status"`
    LogFile     string    `json:"log_file"`
}

type Status string

const (
    StatusRunning   Status = "running"
    StatusCompleted Status = "completed"
    StatusFailed    Status = "failed"
    StatusDetached  Status = "detached"
)
```

## 基本機能

### セッション作成

Claude Code実行時に自動的にtmuxセッションを作成：

```go
func (s *SessionManager) CreateSession(taskID, worktreePath, command string) (*ClaudeSession, error) {
    sessionName := fmt.Sprintf("gwq-claude-%s-%s", taskID, time.Now().Format("20060102150405"))
    logFile := filepath.Join(s.config.LogDir, fmt.Sprintf("%s.log", sessionName))
    
    // tmuxセッション作成
    tmuxCmd := fmt.Sprintf("tmux new-session -d -s %s -c %s", sessionName, worktreePath)
    
    // Claude Code実行（ログ付き）
    claudeCmd := fmt.Sprintf("%s 2>&1 | tee %s", command, logFile)
    
    session := &ClaudeSession{
        ID:           generateID(),
        SessionName:  sessionName,
        TaskID:       taskID,
        WorktreePath: worktreePath,
        Command:      command,
        StartTime:    time.Now(),
        Status:       StatusRunning,
        LogFile:      logFile,
    }
    
    return session, nil
}
```

### ログ管理

```go
type SessionLogger struct {
    baseDir string
}

func (l *SessionLogger) CreateLogFile(sessionName string) (string, error) {
    logFile := filepath.Join(l.baseDir, fmt.Sprintf("%s.log", sessionName))
    
    // ログファイル作成とローテーション設定
    file, err := os.Create(logFile)
    if err != nil {
        return "", err
    }
    defer file.Close()
    
    return logFile, nil
}

func (l *SessionLogger) TailLog(sessionName string, lines int) ([]string, error) {
    logFile := filepath.Join(l.baseDir, fmt.Sprintf("%s.log", sessionName))
    // tail実装
}
```

## コマンド設計

### gwq session サブコマンド

#### `gwq session list`

実行中のClaude Codeセッション一覧（既存のstatusコマンドパターンに準拠）：

```bash
# セッション一覧（シンプルなテーブル形式）
gwq session list

# Output:
# TASK          WORKTREE        STATUS     DURATION
# ● auth-impl   feature/auth    running    1h 25m
#   api-dev     feature/api     running    45m
#   auth-review review/auth     completed  2h 15m

# 詳細情報
gwq session list --verbose

# JSON出力
gwq session list --json

# CSV出力
gwq session list --csv

# リアルタイム監視
gwq session list --watch

# ステータスフィルタ
gwq session list --filter running
gwq session list --filter completed

# ソート
gwq session list --sort duration
gwq session list --sort task
```

#### `gwq session attach`

セッションにアタッチ（既存のget/execパターンに準拠）：

```bash
# パターンマッチでアタッチ
gwq session attach auth

# 完全一致でアタッチ
gwq session attach auth-impl

# 複数マッチ時は自動でfuzzy finder起動
gwq session attach feature  # feature/* がマッチする場合

# 引数なしで全セッションからfuzzy finder選択
gwq session attach

# fuzzy finderを明示的に使用
gwq session attach -i
```

#### `gwq session logs`

ログの表示：

```bash
# パターンマッチでログ表示
gwq session logs auth

# 複数マッチ時は自動でfuzzy finder
gwq session logs feature

# 引数なしでfuzzy finder選択
gwq session logs

# リアルタイムログ（tailに相当）
gwq session logs auth -f
gwq session logs auth --follow

# 最後のN行表示
gwq session logs auth --tail 100

# 行数制限なし（全ログ表示）
gwq session logs auth --all
```

#### `gwq session kill`

セッション終了（既存のremoveパターンに準拠）：

```bash
# パターンマッチで終了
gwq session kill auth

# 複数マッチ時は自動でfuzzy finder
gwq session kill feature

# 引数なしでfuzzy finder選択
gwq session kill

# fuzzy finderを明示的に使用
gwq session kill -i

# 全セッション終了（確認付き）
gwq session kill --all

# 完了済みセッションのみクリーンアップ
gwq session kill --completed
```

## タスクキューとの統合

### タスク実行時の自動セッション作成

```go
func (w *Worker) executeTaskWithSession(task *Task) error {
    // セッション作成
    session, err := w.sessionManager.CreateSession(
        task.ID,
        task.WorktreePath,
        w.buildClaudeCommand(task),
    )
    if err != nil {
        return err
    }
    
    // タスクにセッション情報を記録
    task.SessionID = session.ID
    task.SessionName = session.SessionName
    
    // セッション監視
    go w.monitorSession(session)
    
    return nil
}
```

### タスクステータスとの連携

既存のstatusコマンドを拡張してセッション情報を統合：

```bash
# 既存のstatusコマンドにセッション情報を追加
gwq status --verbose

# Output:
# BRANCH          STATUS       CHANGES           ACTIVITY     SESSION
# ● main          up to date   -                2 hours ago  -
#   feature/auth  changed      5 added, 3 mod   running      auth-impl
#   feature/api   changed      12 added, 8 mod  running      api-dev
#   review/auth   clean        -                completed    auth-review

# セッション情報のみフィルタ
gwq status --filter session
gwq status --filter "no session"

# タスクコマンドでのセッション情報確認
gwq task list --verbose

# Output:
# TASK         BRANCH        STATUS     SESSION      DURATION
# auth-impl    feature/auth  running    attached     1h 25m
# api-dev      feature/api   running    attached     45m
# auth-review  review/auth   completed  detached     2h 15m
```

## ログ管理機能

### ログファイル構造

```
~/.gwq/logs/sessions/
├── gwq-claude-abc123-20240115103045.log
├── gwq-claude-def456-20240115110230.log
└── gwq-claude-ghi789-20240115120015.log
```

### ログローテーション

```toml
[session.logging]
# ログディレクトリ
log_dir = "~/.gwq/logs/sessions"

# ログローテーション
max_log_files = 100
log_retention_days = 30
max_log_size_mb = 100

# ログレベル
log_level = "info"
```

### ログ検索

既存のgrepパターンに準拠したログ検索：

```bash
# 全セッションでキーワード検索
gwq session logs --grep "error"

# 特定セッションでパターン検索
gwq session logs auth --grep "authentication.*failed"

# 複数キーワード検索
gwq session logs --grep "error|failed|exception"

# 時間範囲指定
gwq session logs auth --since "1h"
gwq session logs auth --since "2024-01-15 10:00"

# ログレベルフィルタ
gwq session logs auth --filter error
gwq session logs auth --filter warn
```

## 設定

### tmuxセッション設定

```toml
[session]
# tmux連携を有効化
enabled = true

# セッション自動作成
auto_create_session = true

# セッション作成時の動作
detach_on_create = true
auto_cleanup_completed = true

[session.tmux]
# tmux設定
tmux_command = "tmux"
default_shell = "/bin/bash"

# セッション設定
session_timeout = "24h"
keep_alive = true

[session.logging]
log_dir = "~/.gwq/logs/sessions"
max_log_files = 100
log_retention_days = 30
```

## 使用例

### 基本的な使用フロー

```bash
# タスク実行（自動でセッション作成）
gwq task add -b feature/auth "認証システム実装"

# セッション状態確認（statusコマンドパターン）
gwq session list
gwq status --verbose  # セッション情報含む

# ログ確認（パターンマッチ）
gwq session logs auth --follow

# セッションにアタッチして進捗確認
gwq session attach auth

# セッションからデタッチ（Ctrl+B, D）
# → Claude Codeは継続実行

# 翌朝、結果確認
gwq session list --filter completed
gwq session logs auth --tail 50
```

### ログ分析例

```bash
# エラーが発生したセッションを特定
gwq session logs --grep "error|failed|exception"

# 特定のファイルに関する変更を追跡
gwq session logs --grep "auth.go"

# 長時間実行されているタスクの特定
gwq session list --sort duration

# 実行中のセッションのみ表示
gwq session list --filter running
```

## メリット

1. **プロセス永続化**: ターミナル切断でもClaude Code継続実行
2. **完全なログ記録**: すべての出力が自動保存される
3. **監視機能**: 実行状況の詳細な把握
4. **デバッグ支援**: ログ検索・分析機能
5. **シンプル設計**: 最小限の機能に特化した軽量実装

## 制限事項

1. tmuxがインストールされている環境でのみ動作
2. セッション数が多くなるとリソース消費が増加
3. ログファイルのディスク容量管理が必要

## まとめ

このtmuxセッション管理機能により、Claude Code実行の安定性と監視性が大幅に向上します。複雑なレイアウト管理は行わず、純粋にプロセス管理とログ記録に特化することで、シンプルで信頼性の高いシステムを実現できます。