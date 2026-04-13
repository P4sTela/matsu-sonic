# gdrive-sync

Google Drive からローカルへの一方向同期ツール。映像・バイナリファイルの大容量データに対応。
Go バックエンド + React フロントエンドによるシングルバイナリ配布。

## 技術スタック

### Backend

| 技術 | 用途 |
|------|------|
| Go 1.22+ | バックエンド全般 |
| Chi v5 | HTTP ルーティング |
| gorilla/websocket | リアルタイム進捗通知 |
| SQLite (WAL) | メタデータ・同期履歴管理 |
| google-api-go-client | Drive API |
| errgroup | 並列ダウンロード制御 |

### Frontend

| 技術 | 用途 |
|------|------|
| React 19 + TypeScript | UI |
| Vite | ビルド・HMR |
| Tailwind CSS v4 | スタイリング |
| shadcn/ui | UIコンポーネント |
| React Router v7 | ルーティング |
| lucide-react | アイコン |

## セットアップ

### 前提条件

- [Nix](https://nixos.org/download/) (推奨) または Go 1.22+, Node.js 20+, GCC
- Google Cloud プロジェクトで Drive API を有効化済みの OAuth クレデンシャル

### インストール

```bash
git clone https://github.com/P4sTela/matsu-sonic.git
cd matsu-sonic
```

Nix を使う場合:

```bash
nix develop
```

## 実行

### 開発モード

ターミナルを2つ使用:

```bash
# ターミナル1: フロントエンド (HMR)
cd frontend && npm install && npm run dev

# ターミナル2: Go サーバー
go run . -port 8765
```

ブラウザで http://localhost:3000 を開く。
Vite が `/api` と `/ws` を `:8765` にプロキシします。

### 本番ビルド

```bash
cd frontend && npm ci && npx vite build && cd ..
CGO_ENABLED=1 go build -o gdrive-sync -ldflags="-s -w" .
```

```bash
./gdrive-sync                                # http://localhost:8765
./gdrive-sync -port 9000                     # ポート指定
./gdrive-sync -config /path/to/config.json   # 設定ファイル指定
```

または Makefile:

```bash
make build    # フロント + Go ビルド
make test     # テスト実行
make clean    # 成果物削除
```

## 初期設定

1. 初回起動時にカレントディレクトリの `.gdrive-sync/config.json` が自動生成される
2. Settings ページ、または直接 JSON を編集して以下を設定:
   - `credentials_path`: OAuth クレデンシャルファイルのパス
   - `sync_folder_id`: 同期対象の Google Drive フォルダ ID
   - `local_sync_dir`: ローカル同期先ディレクトリ
3. 「Test Auth」で認証確認

## 機能

- **Full Sync** — Drive フォルダ配下の全ファイルを再帰的にスキャンし、差分があるもののみダウンロード
- **Incremental Sync** — Changes API で前回以降の変更のみを取得・同期
- **並列ダウンロード** — errgroup による並列数制御 (デフォルト 3 ワーカー)
- **リアルタイム進捗** — WebSocket で同期進捗をブラウザにプッシュ
- **リビジョン管理** — ファイルの過去リビジョンを一覧・個別ダウンロード
- **配布** — 同期済みファイルをローカルパスにコピー (SMB は将来対応)
- **ファイル検索** — 同期済みファイルの検索・一覧
- **ログビューア** — リアルタイムログ表示

## プロジェクト構成

```
├── main.go                 # エントリポイント
├── embed.go                # go:embed frontend/dist
├── Makefile
├── internal/
│   ├── config/             # 設定構造体、JSON 読み書き
│   ├── store/              # SQLite スキーマ、CRUD
│   ├── drive/              # Drive API (認証、一覧、DL、リビジョン)
│   ├── sync/               # 同期エンジン、差分判定、進捗追跡
│   ├── server/             # Chi router、WebSocket Hub
│   ├── handler/            # REST API エンドポイント
│   └── distribution/       # 配布先 Target interface + 実装
└── frontend/
    └── src/
        ├── api/            # 型定義、fetch ラッパー、WebSocket
        ├── hooks/          # useSync, useWebSocket, useConfig
        ├── pages/          # Sync, Files, Revisions, Distribute, Settings, Logs
        └── components/     # ProgressBar, FileTable + shadcn/ui
```

## テスト

```bash
go test ./...
```

テスト対象: config, store, sync (differ, progress, engine), distribution

## License

MIT
