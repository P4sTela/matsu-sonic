# Google Drive Sync — 設計ドキュメント

## 概要

Google Drive → ローカルへの一方向同期ツール。映像・バイナリファイルの大容量データ対応。
Go (backend) + TypeScript/React (frontend) によるシングルバイナリ配布。

---

## 技術スタック

| レイヤー | 技術 | 理由 |
|---------|------|------|
| Backend | Go 1.22+ | シングルバイナリ、goroutineによる並列DL、go:embed |
| HTTP Router | Chi v5 | 軽量、標準net/httpと互換、ミドルウェアチェーン |
| WebSocket | gorilla/websocket | 業界標準、安定 |
| DB | SQLite (mattn/go-sqlite3) | 組み込み、ゼロ設定、cgo依存だがシングルバイナリには問題なし |
| Drive API | google-api-go-client | Google公式 |
| Auth | golang.org/x/oauth2 | Google公式OAuth2ライブラリ |
| 並列制御 | golang.org/x/sync/errgroup | Go公式拡張。SetLimit()で並列数制御、context連携が自然 |
| Frontend | React 18 + TypeScript | 型安全、開発体験 |
| Build | Vite | 高速HMR、go:embedとの相性良し |
| CSS | Tailwind CSS 3 | ユーティリティファースト |
| State管理 | React hooks (useState/useReducer) | 規模的に外部ライブラリ不要 |

---

## ディレクトリ構成

```
gdrive-sync/
├── main.go                      # エントリポイント
├── go.mod
├── go.sum
├── Makefile                     # ビルド自動化
│
├── internal/
│   ├── server/
│   │   ├── server.go            # Chi router定義、ミドルウェア
│   │   ├── middleware.go        # ロギング、リカバリ、CORS
│   │   └── ws.go                # WebSocket hub (接続管理、ブロードキャスト)
│   │
│   ├── handler/
│   │   ├── config.go            # GET/POST /api/config
│   │   ├── auth.go              # POST /api/auth/test
│   │   ├── sync.go              # POST /api/sync/{full,incremental,cancel}, GET /api/sync/status
│   │   ├── files.go             # GET /api/files, GET /api/files/{id}
│   │   ├── revisions.go         # GET /api/files/{id}/revisions, POST .../download
│   │   ├── distribution.go      # 配布関連エンドポイント
│   │   └── browse.go            # GET /api/browse (ディレクトリブラウザ)
│   │
│   ├── drive/
│   │   ├── client.go            # Drive API ラッパー (認証、リスト、DL)
│   │   ├── auth.go              # OAuth2 / ServiceAccount 認証ロジック
│   │   ├── download.go          # チャンクDL、プログレスコールバック
│   │   └── revisions.go         # リビジョン一覧・DL
│   │
│   ├── sync/
│   │   ├── engine.go            # SyncEngine 本体 (Full/Incremental/errgroup並列)
│   │   ├── progress.go          # ProgressEvent 定義、集計
│   │   └── differ.go            # md5比較、Changes API差分検出
│   │
│   ├── distribution/
│   │   ├── target.go            # Target interface定義
│   │   ├── local.go             # LocalTarget実装
│   │   ├── smb.go               # SMBTarget スタブ (将来実装)
│   │   └── manager.go           # DistributionManager
│   │
│   ├── store/
│   │   ├── db.go                # SQLite接続、マイグレーション
│   │   ├── files.go             # synced_files CRUD
│   │   ├── runs.go              # sync_runs CRUD
│   │   ├── revisions.go         # downloaded_revisions CRUD
│   │   └── distribution.go      # distribution_jobs CRUD
│   │
│   └── config/
│       ├── config.go            # 設定構造体、デフォルト値
│       └── loader.go            # JSON読み書き
│
├── frontend/
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   ├── index.html
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── api/
│       │   ├── client.ts        # fetch ラッパー + 型定義
│       │   ├── websocket.ts     # WS接続、自動再接続
│       │   └── types.ts         # 共通型定義 (APIレスポンス等)
│       ├── hooks/
│       │   ├── useSync.ts       # 同期状態管理 hook
│       │   ├── useWebSocket.ts  # WS接続 hook
│       │   └── useConfig.ts     # 設定取得/保存 hook
│       ├── components/
│       │   ├── ProgressBar.tsx
│       │   ├── FileTable.tsx
│       │   ├── DirBrowser.tsx   # ディレクトリ選択モーダル
│       │   └── Badge.tsx
│       └── pages/
│           ├── SyncPage.tsx
│           ├── RevisionsPage.tsx
│           ├── DistributePage.tsx
│           ├── SettingsPage.tsx
│           └── LogPage.tsx
│
└── embed.go                     # //go:embed frontend/dist
```

---

## Go パッケージ設計の詳細

### main.go

```go
// 責務: CLI引数パース → Config読み込み → DB初期化 → Server起動
//
// - flag.IntVar(&port, "port", 8765, "server port")
// - flag.StringVar(&configPath, "config", "~/.gdrive-sync/config.json", "config path")
// - signal.NotifyContext で SIGINT/SIGTERM を拾い graceful shutdown
// - 起動時に browser.OpenURL(fmt.Sprintf("http://localhost:%d", port))
```

### embed.go

```go
package main

import "embed"

//go:embed frontend/dist/*
var frontendFS embed.FS
```

### internal/config/config.go

```go
type Config struct {
    AuthMethod      string            `json:"auth_method"`       // "oauth" | "service_account"
    CredentialsPath string            `json:"credentials_path"`
    TokenPath       string            `json:"token_path"`
    SyncFolderID    string            `json:"sync_folder_id"`
    LocalSyncDir    string            `json:"local_sync_dir"`
    Scopes          []string          `json:"scopes"`
    ExportFormats   map[string]Export  `json:"export_formats"`
    ChunkSizeMB     int               `json:"chunk_size_mb"`     // default: 10
    MaxWorkers      int               `json:"max_workers"`       // default: 3
    RevisionNaming  string            `json:"revision_naming"`   // default: "{stem}.rev{rev_id}{suffix}"
    DistTargets     []DistTargetConf  `json:"distribution_targets"`
}

type Export struct {
    MimeType  string `json:"mime"`
    Extension string `json:"ext"`
}

type DistTargetConf struct {
    Name     string `json:"name"`
    Type     string `json:"type"`      // "local" | "smb"
    Path     string `json:"path"`      // local用
    Server   string `json:"server"`    // smb用
    Share    string `json:"share"`
    Username string `json:"username"`
    Password string `json:"password"`
    Domain   string `json:"domain"`
}
```

### internal/server/server.go

```go
// Chi router 構成:
//
// r := chi.NewRouter()
// r.Use(middleware.Logger, middleware.Recoverer, CORSMiddleware)
//
// r.Route("/api", func(r chi.Router) {
//     r.Get("/config",  handler.GetConfig)
//     r.Post("/config", handler.UpdateConfig)
//
//     r.Post("/auth/test", handler.TestAuth)
//
//     r.Post("/sync/full",        handler.StartFullSync)
//     r.Post("/sync/incremental", handler.StartIncrementalSync)
//     r.Post("/sync/cancel",      handler.CancelSync)
//     r.Get("/sync/status",       handler.GetSyncStatus)
//     r.Get("/sync/history",      handler.GetSyncHistory)
//
//     r.Get("/files",             handler.ListFiles)
//     r.Get("/files/{fileID}",    handler.GetFile)
//     r.Get("/files/{fileID}/revisions",                         handler.ListRevisions)
//     r.Post("/files/{fileID}/revisions/{revID}/download",       handler.DownloadRevision)
//     r.Get("/files/{fileID}/revisions/downloaded",              handler.ListDownloadedRevisions)
//
//     r.Get("/distribution/targets",            handler.ListTargets)
//     r.Post("/distribution/targets",           handler.AddTarget)
//     r.Delete("/distribution/targets/{name}",  handler.RemoveTarget)
//     r.Post("/distribution/targets/{name}/test", handler.TestTarget)
//     r.Post("/distribute",                     handler.Distribute)
//     r.Get("/distribution/jobs",               handler.ListDistJobs)
//
//     r.Get("/browse", handler.BrowseDirectory)
// })
//
// r.HandleFunc("/ws", wsHub.HandleWS)
//
// // Embedded frontend (SPAフォールバック)
// r.Handle("/*", SPAHandler(frontendFS))
```

### internal/server/ws.go — WebSocket Hub

```go
// パターン: Hub + Client
//
// type Hub struct {
//     clients    map[*Client]bool
//     broadcast  chan []byte
//     register   chan *Client
//     unregister chan *Client
// }
//
// type Client struct {
//     hub  *Hub
//     conn *websocket.Conn
//     send chan []byte
// }
//
// Hub.Run() は goroutine で常駐。
// register/unregister で接続管理。
// broadcast チャネルにJSONを送ると全クライアントにファンアウト。
//
// Client.ReadPump()  → ping/pong のみ
// Client.WritePump() → send チャネルから読んでWSに書く
//
// メッセージフォーマット:
// { "type": "sync_progress", "data": { ...ProgressEvent } }
// { "type": "sync_complete", "data": { ...SyncResult } }
// { "type": "log",           "data": { "level": "info", "msg": "..." } }
```

### internal/drive/client.go

```go
// type DriveClient struct {
//     service *drive.Service
//     config  *config.Config
// }
//
// func NewDriveClient(cfg *config.Config) (*DriveClient, error)
//   → auth.go の Authenticate() を呼ぶ
//
// func (d *DriveClient) ListFolder(ctx context.Context, folderID string) ([]*drive.File, error)
//   → pages.NextToken ループ、fields指定
//
// func (d *DriveClient) ListAllRecursive(ctx context.Context, folderID string) ([]*drive.File, error)
//   → BFS で再帰。parent_id を各ファイルに付与。
//
// func (d *DriveClient) DownloadFile(ctx, fileID, destPath, mimeType string, progress func(float64)) (int64, error)
//   → Google Docs は export、それ以外は get_media
//   → io.TeeReader でプログレス計算
//   → チャンクサイズは config.ChunkSizeMB * 1024 * 1024
//
// func (d *DriveClient) GetStartPageToken(ctx) (string, error)
// func (d *DriveClient) GetChanges(ctx, pageToken string) ([]*drive.Change, string, error)
// func (d *DriveClient) GetFileMeta(ctx, fileID string) (*drive.File, error)
```

### internal/drive/auth.go

```go
// func Authenticate(cfg *config.Config) (*drive.Service, error)
//
// OAuth フロー:
//   1. token.json が存在すれば読み込み
//   2. 期限切れなら RefreshToken で更新
//   3. なければ oauth2.Config.AuthCodeURL() → ブラウザで承認
//      → httptest.NewServer() でローカルにコールバック受けてcode取得
//      → Exchange() → token保存
//
// ServiceAccount フロー:
//   1. google.CredentialsFromJSON(ctx, keyJSON, scopes...)
//   2. drive.NewService(ctx, option.WithCredentials(creds))
//
// 注意: OAuth のコールバックサーバーは空きポートを使う (":0")
```

### internal/sync/engine.go

```go
// type SyncEngine struct {
//     cfg      *config.Config
//     drive    *drive.DriveClient
//     store    *store.DB
//     hub      *server.Hub         // WS broadcast用
//     progress *ProgressTracker
//     cancel   context.CancelFunc
//     mu       sync.Mutex
//     running  bool
// }
//
// ── 並列制御: errgroup.SetLimit() ──────────────────────────────
//
// worker.go は廃止。engine.go 内で errgroup を直接使う。
// 理由:
//   - golang.org/x/sync は Go公式拡張で誰でも読める
//   - 自前のchan+WaitGroupパターンより保守しやすい
//   - context連携、並列数制御が組み込み
//
// func (e *SyncEngine) StartFull(ctx context.Context) error
//   1. e.mu.Lock() で二重起動防止 → running = true → Unlock
//   2. ctx, cancel = context.WithCancel(ctx)
//   3. progressChan := make(chan ProgressEvent, 100)
//   4. go e.progressLoop(ctx, progressChan)  // 100ms throttle で WS broadcast
//   5. drive.ListAllRecursive() でファイル一覧取得
//   6. フォルダ構造をローカルに作成、DBに登録
//   7. errgroup で並列DL:
//
//      g, ctx := errgroup.WithContext(ctx)
//      g.SetLimit(e.cfg.MaxWorkers)  // default 3
//
//      for _, file := range files {
//          f := file  // ループ変数キャプチャ
//          g.Go(func() error {
//              return e.syncOneFile(ctx, f, progressChan)
//          })
//      }
//
//      if err := g.Wait(); err != nil {
//          // 致命的エラー (認証切れ、ディスクフル等) のみここに来る
//      }
//
//   8. GetStartPageToken() でchangeTokenを記録
//   9. store.FinishRun()
//   10. close(progressChan) → progressLoop が終了
//
// func (e *SyncEngine) syncOneFile(ctx, file, progressChan) error
//   1. differ.NeedsSync() でスキップ判定
//   2. スキップ → progressChan <- file_skip → return nil
//   3. DL実行 → progressCallback で file_progress イベント送信
//   4. store.UpsertFile()
//   5. エラー時:
//      - 個別ファイルエラー → progressChan <- file_error → return nil
//        (errgroupには成功として返す → 他のワーカーは止まらない)
//      - 致命的エラー (ctx.Err() != nil, ディスクフル等) → return err
//        (errgroup全体がキャンセルされる)
//
// func (e *SyncEngine) StartIncremental(ctx context.Context) error
//   1. store.GetLastChangeToken()
//   2. token無ければ StartFull() にフォールバック
//   3. drive.GetChanges(token)
//   4. 変更ファイルのみ errgroup で並列処理 (StartFull と同じパターン)
//   5. 削除されたファイルは .gdrive-trash/ に退避
//
// func (e *SyncEngine) Cancel()
//   → cancel() を呼ぶ。errgroup内の各goroutineは ctx.Done() で検知。
//
// func (e *SyncEngine) Status() ProgressSnapshot
//   → progress.Snapshot() を返す
//
// func (e *SyncEngine) progressLoop(ctx, progressChan)
//   → 専用goroutine。progressChanからイベントを読み、
//     tracker.Apply() → 100ms throttle → hub.Broadcast(snapshot)
//   → chan close で自然に終了
```

#### errgroup のエラーハンドリング方針

```
個別ファイルのDLエラー (ネットワーク一時障害、ファイル破損等)
  → progressChan に file_error を送信
  → return nil (他のファイルは継続)

致命的エラー (認証期限切れ、ディスクフル、context cancelled)
  → return err (errgroup が他の全goroutineをキャンセル)

判定基準:
  - ctx.Err() != nil → 致命的 (ユーザーキャンセル or 親context終了)
  - os.IsPermission(err) → 致命的 (ディスクのパーミッション)
  - isDiskFull(err) → 致命的
  - それ以外 → 個別エラー (リトライ後も失敗なら skip)
```

### internal/sync/progress.go

```go
// type ProgressEvent struct {
//     Type             string  `json:"type"`     // "file_start" | "file_progress" | "file_done" | "file_skip" | "file_error" | "scan"
//     FileID           string  `json:"file_id"`
//     FileName         string  `json:"file_name"`
//     FileProgress     float64 `json:"file_progress"`     // 0.0 ~ 1.0
//     BytesDownloaded  int64   `json:"bytes_downloaded"`  // このファイルの現在DL量
//     Error            string  `json:"error,omitempty"`
// }
//
// type ProgressTracker struct {
//     mu              sync.Mutex
//     totalFiles      int
//     completedFiles  int
//     failedFiles     int
//     skippedFiles    int
//     totalBytes      int64
//     currentFile     string
//     currentProgress float64
//     errors          []string  // 直近20件
// }
//
// func (t *ProgressTracker) Apply(event ProgressEvent)
//   → イベント種別に応じてカウンタ更新
//
// func (t *ProgressTracker) Snapshot() ProgressSnapshot
//   → 現在値のコピーを返す (JSON直列化用)
//
// ProgressTracker は Hub への broadcast を直接やらない。
// engine.go 内の goroutine が progressChan から読み、
// tracker.Apply() してから hub.Broadcast(snapshot.JSON()) する。
// これにより broadcast 頻度を制御できる (例: 100ms throttle)。
```

### internal/sync/differ.go

```go
// ファイル同期が必要かどうかの判定ロジック:
//
// func NeedsSync(remote *drive.File, local *store.SyncedFile) bool
//   1. local が nil → 新規ファイル → true
//   2. remote.Md5Checksum != "" && remote.Md5Checksum == local.MD5 → false (スキップ)
//   3. remote.ModifiedTime > local.DriveModified → true
//   4. ローカルファイルが存在しない → true
//   5. それ以外 → false
//
// Google Docs は md5 が無い → ModifiedTime で比較
```

### internal/distribution/target.go

```go
// type Target interface {
//     Type() string
//     Distribute(ctx context.Context, src string, destRelative string) (string, error)
//     TestConnection(ctx context.Context) error
//     ListContents(ctx context.Context, path string) ([]DirEntry, error)
// }
//
// type DirEntry struct {
//     Name  string `json:"name"`
//     IsDir bool   `json:"is_dir"`
//     Size  int64  `json:"size"`
//     Path  string `json:"path"`
// }
```

### internal/distribution/local.go

```go
// type LocalTarget struct {
//     BasePath string
// }
//
// Distribute: os.MkdirAll + io.Copy (preserveTimestamps)
// TestConnection: os.MkdirAll + tempfileの書き込み/削除テスト
// ListContents: os.ReadDir
```

### internal/distribution/smb.go

```go
// type SMBTarget struct {
//     Server   string
//     Share    string
//     Username string
//     Password string
//     Domain   string
// }
//
// 将来実装: github.com/hirochachacha/go-smb2 を使用
// 現在は全メソッドが ErrNotImplemented を返す
// ただし構造体とインターフェース実装は完成させておく
```

### internal/store/db.go

```go
// func New(dbPath string) (*DB, error)
//   → sqlite3.Open
//   → migrate() で全テーブル作成 (IF NOT EXISTS)
//   → WALモード有効化 (PRAGMA journal_mode=WAL)
//   → busy_timeout=5000
//
// マイグレーション:
//   バージョン管理テーブル (schema_version) + 連番SQLで管理
//   将来の列追加にも対応できるようにする
```

---

## SQLite スキーマ

```sql
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY
);

CREATE TABLE synced_files (
    file_id         TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    mime_type       TEXT,
    md5_checksum    TEXT,
    size            INTEGER,
    drive_modified  TEXT,   -- RFC3339
    local_path      TEXT,
    last_synced     TEXT,   -- RFC3339
    parent_id       TEXT,
    is_folder       INTEGER DEFAULT 0
);
CREATE INDEX idx_synced_parent ON synced_files(parent_id);

CREATE TABLE sync_runs (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at       TEXT NOT NULL,
    finished_at      TEXT,
    status           TEXT DEFAULT 'running',  -- running|completed|failed|cancelled
    files_synced     INTEGER DEFAULT 0,
    files_failed     INTEGER DEFAULT 0,
    bytes_downloaded INTEGER DEFAULT 0,
    change_token     TEXT
);

CREATE TABLE downloaded_revisions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id       TEXT NOT NULL,
    revision_id   TEXT NOT NULL,
    local_path    TEXT NOT NULL,
    downloaded_at TEXT NOT NULL,
    size          INTEGER,
    UNIQUE(file_id, revision_id)
);
CREATE INDEX idx_revisions_file ON downloaded_revisions(file_id);

CREATE TABLE distribution_jobs (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id       TEXT NOT NULL,
    source_path   TEXT NOT NULL,
    target_type   TEXT NOT NULL,
    target_path   TEXT NOT NULL,
    status        TEXT DEFAULT 'pending',  -- pending|completed|failed
    created_at    TEXT NOT NULL,
    completed_at  TEXT,
    error_message TEXT
);
CREATE INDEX idx_dist_status ON distribution_jobs(status);
```

---

## REST API 仕様

### Config
| Method | Path | Request | Response |
|--------|------|---------|----------|
| GET | /api/config | — | `Config` (password系フィールドはマスク) |
| POST | /api/config | `Partial<Config>` | `{ status: "saved" }` |

### Auth
| Method | Path | Request | Response |
|--------|------|---------|----------|
| POST | /api/auth/test | — | `{ status: "ok", user: { displayName, emailAddress } }` |

### Sync
| Method | Path | Request | Response |
|--------|------|---------|----------|
| POST | /api/sync/full | — | `{ status: "started", mode: "full" }` |
| POST | /api/sync/incremental | — | `{ status: "started", mode: "incremental" }` |
| POST | /api/sync/cancel | — | `{ status: "cancel_requested" }` |
| GET | /api/sync/status | — | `{ is_running: bool, progress: ProgressSnapshot }` |
| GET | /api/sync/history | `?limit=20` | `SyncRun[]` |

409 Conflict を返す条件: 同期が既に実行中

### Files
| Method | Path | Request | Response |
|--------|------|---------|----------|
| GET | /api/files | `?search=keyword` | `SyncedFile[]` |
| GET | /api/files/:id | — | `SyncedFile` |

### Revisions
| Method | Path | Request | Response |
|--------|------|---------|----------|
| GET | /api/files/:id/revisions | — | `DriveRevision[]` (Drive APIから直接) |
| POST | /api/files/:id/revisions/:revId/download | `{ dest_dir?: string }` | `{ path, size }` |
| GET | /api/files/:id/revisions/downloaded | — | `DownloadedRevision[]` (DBから) |

### Distribution
| Method | Path | Request | Response |
|--------|------|---------|----------|
| GET | /api/distribution/targets | — | `DistTargetConf[]` |
| POST | /api/distribution/targets | `DistTargetConf` | `{ status: "added" }` |
| DELETE | /api/distribution/targets/:name | — | `{ status: "removed" }` |
| POST | /api/distribution/targets/:name/test | — | `{ status: "ok" }` or error |
| POST | /api/distribute | `{ file_ids, target_name, dest_dir? }` | `DistResult[]` |
| GET | /api/distribution/jobs | `?limit=50` | `DistJob[]` |

### Utility
| Method | Path | Request | Response |
|--------|------|---------|----------|
| GET | /api/browse | `?path=/home/user` | `{ current, parent, items: [{name,path,is_dir}] }` |

---

## WebSocket メッセージ仕様

接続: `ws://localhost:8765/ws`

### サーバー → クライアント

```typescript
// 同期進捗 (100ms throttle)
{
  type: "sync_progress",
  data: {
    total_files: number,
    completed_files: number,
    failed_files: number,
    skipped_files: number,
    bytes_downloaded: number,
    current_file: string,
    current_file_progress: number,  // 0.0-1.0
    is_running: boolean,
    errors: string[]                // 直近20件
  }
}

// 同期完了
{
  type: "sync_complete",
  data: {
    status: "completed" | "failed" | "cancelled",
    files_synced: number,
    files_failed: number,
    bytes_downloaded: number,
    duration_ms: number
  }
}

// ログ
{
  type: "log",
  data: { level: "info" | "warn" | "error", msg: string, ts: string }
}

// pong (keep-alive応答)
{ type: "pong" }
```

### クライアント → サーバー

```typescript
// keep-alive
{ action: "ping" }
```

---

## フロントエンド TypeScript 型定義

```typescript
// api/types.ts

export interface Config {
  auth_method: "oauth" | "service_account";
  credentials_path: string;
  token_path: string;
  sync_folder_id: string;
  local_sync_dir: string;
  chunk_size_mb: number;
  max_workers: number;
  distribution_targets: DistTarget[];
}

export interface DistTarget {
  name: string;
  type: "local" | "smb";
  path: string;
  server?: string;
  share?: string;
  username?: string;
}

export interface SyncedFile {
  file_id: string;
  name: string;
  mime_type: string;
  md5_checksum: string;
  size: number;
  drive_modified: string;
  local_path: string;
  last_synced: string;
  parent_id: string;
  is_folder: boolean;
}

export interface SyncProgress {
  total_files: number;
  completed_files: number;
  failed_files: number;
  skipped_files: number;
  bytes_downloaded: number;
  current_file: string;
  current_file_progress: number;
  is_running: boolean;
  errors: string[];
}

export interface SyncRun {
  id: number;
  started_at: string;
  finished_at: string;
  status: "running" | "completed" | "failed" | "cancelled";
  files_synced: number;
  files_failed: number;
  bytes_downloaded: number;
}

export interface DriveRevision {
  id: string;
  modifiedTime: string;
  size: string;
  lastModifyingUser?: { displayName: string };
  mimeType: string;
  keepForever: boolean;
  originalFilename: string;
}

export interface DistJob {
  id: number;
  file_id: string;
  source_path: string;
  target_type: string;
  target_path: string;
  status: "pending" | "completed" | "failed";
  created_at: string;
  error_message: string;
}

export interface BrowseResult {
  current: string;
  parent: string;
  items: { name: string; path: string; is_dir: boolean }[];
}

// WebSocket messages
export type WSMessage =
  | { type: "sync_progress"; data: SyncProgress }
  | { type: "sync_complete"; data: SyncComplete }
  | { type: "log"; data: LogEntry }
  | { type: "pong" };

export interface SyncComplete {
  status: string;
  files_synced: number;
  files_failed: number;
  bytes_downloaded: number;
  duration_ms: number;
}

export interface LogEntry {
  level: "info" | "warn" | "error";
  msg: string;
  ts: string;
}
```

---

## ビルド & 配布

### Makefile

```makefile
.PHONY: dev build clean

# 開発: フロントエンドHMR + Goサーバー
dev:
	cd frontend && npm run dev &
	go run main.go -port 8765

# 本番ビルド: フロント → embed → Goバイナリ
build:
	cd frontend && npm ci && npm run build
	CGO_ENABLED=1 go build -o gdrive-sync -ldflags="-s -w" .

# クロスコンパイル (Linux向け、macOSから)
build-linux:
	cd frontend && npm ci && npm run build
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o gdrive-sync-linux -ldflags="-s -w" .

clean:
	rm -f gdrive-sync gdrive-sync-linux
	rm -rf frontend/dist
```

### 開発ワークフロー

1. `cd frontend && npm run dev` → Vite dev server (:3000)
2. `go run main.go` → Go server (:8765)
3. Vite の proxy設定で `/api/*` と `/ws` を :8765 に転送
4. ブラウザは :3000 を開く

### 本番

```bash
make build
./gdrive-sync                    # → http://localhost:8765 が自動で開く
./gdrive-sync -port 9000         # ポート指定
./gdrive-sync -config /etc/gdrive-sync.json  # 設定ファイル指定
```

---

## 並列DL制御の詳細

```
                ┌─────────────┐
                │ SyncEngine  │
                │ StartFull() │
                └──────┬──────┘
                       │ ListAllRecursive()
                       ▼
              ┌─────────────────┐
              │   errgroup      │  g.SetLimit(MaxWorkers)
              │   .WithContext  │  default: 3
              └────┬───┬───┬───┘
                   │   │   │     g.Go(func() error { ... })
            ┌──────┘   │   └──────┐
            ▼          ▼          ▼
      goroutine 1  goroutine 2  goroutine 3
      syncOneFile  syncOneFile  syncOneFile
            │          │          │
            │   個別エラー → return nil (継続)
            │   致命的エラー → return err (全停止)
            ▼          ▼          ▼
      ┌──────────────────────────────┐
      │    progressChan              │  buffered: 100
      │    (ProgressEvent)           │
      └──────────────┬───────────────┘
                     │
                     ▼
            ┌─────────────────┐
            │ progressLoop    │  100ms throttle
            │ (1 goroutine)   │  tracker.Apply() → snapshot
            └────────┬────────┘
                     │ hub.Broadcast(snapshot.JSON())
                     ▼
            ┌─────────────────┐
            │ WebSocket Hub   │
            │ → all clients   │
            └─────────────────┘
```

- `errgroup.SetLimit(MaxWorkers)` で並列数制御。デフォルト3
- `context.WithCancel` で errgroup 全体にキャンセル伝播
- 個別ファイルエラーは `return nil` で他ワーカーに影響しない
- 致命的エラー (認証切れ、ディスクフル) のみ `return err` で全停止
- プログレスの WS broadcast は 100ms にスロットルして帯域節約
- `g.Wait()` 完了後に `close(progressChan)` → progressLoop が自然終了

---

## エラーハンドリング方針

| 場面 | 対応 |
|------|------|
| 認証失敗 | API 401 → フロントで「認証テストしてください」表示 |
| Drive APIレート制限 | exponential backoff (初回1s, 最大60s, 5回リトライ) |
| 個別ファイルDL失敗 | エラーログ + スキップ、他ファイルの同期は継続 |
| WebSocket切断 | フロント側で3秒後に自動再接続 |
| SQLiteロック | WALモード + busy_timeout=5000ms |
| ディスク容量不足 | DL前に残容量チェック、閾値以下で同期停止 |
| 同期中の二重起動 | sync.Mutex で排他、409 Conflict |

---

## 将来拡張ポイント

1. **SMB配布**: `internal/distribution/smb.go` に go-smb2 で実装追加
2. **NFS配布**: `Target` interfaceの新実装を追加するだけ
3. **S3配布**: 同上
4. **スケジュール同期**: `internal/scheduler/` に cron-like なスケジューラ追加、Config に cron式を追加
5. **通知**: 同期完了時のWebhook/メール通知 (`internal/notify/`)
6. **マルチフォルダ**: Config.SyncFolderID を配列化
7. **選択的同期**: 除外パターン (glob) を Config に追加

---

## 実装順序 (Claude Code向け推奨)

依存関係が少ない順に:

1. `internal/config` → 設定構造体と JSON 読み書き
2. `internal/store` → SQLite スキーマ、マイグレーション、全 CRUD
3. `internal/drive/auth.go` → OAuth / ServiceAccount 認証
4. `internal/drive/client.go` + `download.go` + `revisions.go` → Drive API操作
5. `internal/sync/differ.go` → NeedsSync 判定ロジック
6. `internal/sync/progress.go` → ProgressEvent / ProgressTracker
7. `internal/sync/engine.go` → errgroup ベースの同期エンジン本体
8. `internal/server/ws.go` → WebSocket Hub + Client
9. `internal/server/server.go` + `middleware.go` → Chi router
10. `internal/handler/*` → 全 REST API エンドポイント
11. `internal/distribution/*` → Target interface + Local + SMB stub
12. `embed.go` + `main.go` + `Makefile` → 結合・ビルド
13. `frontend/` → TypeScript 化 (型定義 → hooks → pages)

各ステップ完了時にユニットテスト作成を推奨。
特に `store`, `differ`, `engine` は テスト必須。

---

## 設計判断の記録

| 判断 | 選択 | 理由 |
|------|------|------|
| HTTP router | Chi v5 | 標準net/http互換、ルートグルーピングで15+エンドポイントの一覧性確保。Ginは独自Contextで標準から乖離 |
| 並列DL制御 | errgroup.SetLimit() | worker.go 廃止。Go公式x/syncで誰でも読める。自前chan+WaitGroupより保守しやすい |
| エラー継続方針 | 個別nil/致命的err | errgroupデフォルトの「最初のエラーで全停止」を回避。個別エラーはprogressChan経由でUIに伝達 |
| DB | SQLite (WAL) | 組み込み、ゼロ設定。WALモードで同期中の読み取りがブロックされない |
| フロントembed | go:embed | シングルバイナリ配布。利用者にNode.js不要 |
| 配布抽象化 | Target interface | Local/SMB/NFS/S3 を同一インターフェースで追加可能 |
