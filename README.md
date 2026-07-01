# gdrive-sync

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white)](https://react.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Google Drive からローカルへの一方向同期ツール。映像・バイナリなどの大容量ファイルに最適化。
Go バックエンド + React フロントエンドをシングルバイナリで配布。

<p align="center">
  <img src="docs/screenshot.png" alt="gdrive-sync スクリーンショット" width="800">
</p>

## 機能一覧

- **Full / Incremental Sync** — 再帰スキャンまたは Changes API による差分同期
- **並列ダウンロード** — ワーカープール制御（デフォルト 3 並列）
- **リアルタイム進捗** — WebSocket でブラウザ UI にライブ表示
- **Google Docs エクスポート** — PDF, XLSX, PPTX 等への自動変換（フォーマット設定可）
- **リビジョン管理** — 過去リビジョンの一覧・個別ダウンロード（命名パターン設定可）
- **配布 (Distribution)** — 同期済みファイルをローカルパスや SMB 共有へコピー（バッチ配布・配布先ごとの選択パターン設定可）
- **ファイル整合性検証** — ダウンロード後の MD5 チェックサム検証（WebSocket でリアルタイム進捗表示）
- **同期コンフリクト検出** — ローカルファイルの変更・削除を検出し警告（`conflict_strategy` で skip / overwrite を選択）
- **ファイル変換 (Converter Plugin)** — ffmpeg 等の外部コマンドでファイル変換（例: mp4→HAP）。自動/手動実行、stale 検出、再変換対応
- **選択同期** — include パターン（プレフィックス / `*` / `**` ワイルドカード）で同期対象を限定
- **Ignore パターン** — glob ベースのファイル除外（`*.tmp`, `~*` 等）
- **ファイル検索** — 同期済みメタデータに対する全文検索
- **ブラウザベース認証 (PKCE)** — OAuth クレデンシャルをバイナリに埋め込み可能。初回のみブラウザで承認
- **シークレット暗号化** — 配布先パスワード・OAuth トークンをインストールごとの AES-GCM 鍵で暗号化
- **シングルバイナリ** — フロントエンドを `go:embed` で埋め込み、実行時の外部依存なし

## 対応 OS

| OS | ステータス | 備考 |
|----|-----------|------|
| **Windows** | メインターゲット | `make build-windows` で `.exe` を生成 |
| **macOS** | 対応 | 開発環境として使用 |
| **Linux** | 対応 | `make build-linux` でクロスコンパイル |

## クイックスタート

### インストール

**バイナリをダウンロード（推奨）:**

[GitHub Releases](https://github.com/P4sTela/matsu-sonic/releases/latest) から OS に合ったバイナリをダウンロードしてください。

| OS | ファイル |
|----|---------|
| Windows | `gdrive-sync-windows-amd64.exe` |
| macOS | `gdrive-sync-darwin-amd64` |
| Linux | `gdrive-sync-linux-amd64` |

リリースバイナリには OAuth クレデンシャルが埋め込まれているため、追加ファイルなしで認証が可能です。

**ソースからビルド:**

```bash
git clone https://github.com/P4sTela/matsu-sonic.git
cd matsu-sonic

# OAuth クレデンシャルを .env に設定（任意。未設定でも開発は可能）
cp .env.example .env
# .env を編集して GOOGLE_OAUTH_CLIENT_ID / GOOGLE_OAUTH_CLIENT_SECRET を設定

task build
```

必要なもの: Go 1.23+（CGO 有効）、[Bun](https://bun.sh/)。[Nix](https://nixos.org/download/) を使う場合は `nix develop` で環境を構築できます。

### 実行

```bash
./gdrive-sync                              # デフォルト: http://localhost:8765
./gdrive-sync -port 9000                   # ポート指定
./gdrive-sync -config /path/to/config.json # 設定ファイル指定
```

ブラウザで Settings ページを開き、認証と同期先を設定してください。

## セットアップ

### 1. Google Cloud クレデンシャルの準備

1. [Google Cloud Console](https://console.cloud.google.com/) でプロジェクトを作成（または既存を使用）
2. [Drive API を有効化](https://console.cloud.google.com/apis/library/drive.googleapis.com)
3. OAuth クレデンシャルを作成:
   - [認証情報ページ](https://console.cloud.google.com/apis/credentials) > 「認証情報を作成」 > 「OAuth クライアント ID」
   - アプリの種類: **デスクトップ アプリ**
   - 作成後、クライアント ID とクライアントシークレットを控える

### 2. クレデンシャルの設定

gdrive-sync では2通りの方法で OAuth クレデンシャルを指定できます:

| 方式 | 設定 | 用途 |
|------|------|------|
| **埋め込み（推奨）** | `.env` → `ldflags` でビルド時に埋め込み | 配布用バイナリ。ユーザーが追加ファイルを用意不要 |
| **外部ファイル** | Settings ページで `credentials_path` を指定 | 開発時や Service Account 利用時 |

**埋め込み方式（リリースビルド）:**

```bash
cp .env.example .env
# .env を編集
GOOGLE_OAUTH_CLIENT_ID=YOUR_CLIENT_ID
GOOGLE_OAUTH_CLIENT_SECRET=YOUR_CLIENT_SECRET

task build  # バイナリに埋め込まれる
```

**外部ファイル方式（開発時）:**

ダウンロードした OAuth JSON を任意の場所に配置し、Settings ページでパスを指定します。

### 3. 認証と同期設定

1. サーバーを起動し、Web UI の **Settings** ページを開く
2. 認証:
   - **埋め込みクレデンシャル** がある場合: **Start Auth** ボタンを押す → ブラウザで承認 → 自動でトークンが保存
   - **外部クレデンシャル** の場合: `credentials_path` を設定 → **Test Auth** で確認
3. 同期設定:
   - **Sync Folder ID** — 同期対象の Google Drive フォルダ ID（認証後は UI の Drive ブラウザで選択可）
   - **Local Sync Directory** — ローカル同期先ディレクトリ（UI のディレクトリブラウザで作成・選択可）
4. **Save** → Sync ページから同期を開始

### 認証方式

| 方式 | 設定 | 用途 |
|------|------|------|
| **OAuth**（推奨） | `auth_method: "oauth"` | 自分の Drive 全体にアクセス可。共有フォルダ含む。初回のみブラウザ認証、以降自動（PKCE 対応） |
| **Service Account** | `auth_method: "service_account"` | ヘッドレス環境向け。対象フォルダを SA メールアドレスに共有する必要あり |

OAuth トークンは設定ディレクトリ（デフォルト `.gdrive-sync/`）内に暗号化保存されます。`token.json` はインストールごとの AES-GCM 鍵で暗号化されるため、設定ディレクトリを他マシンにコピーしてもトークンは復号できません（セキュリティ上の設計）。

## 設定項目

設定ファイルは `.gdrive-sync/config.json`（初回起動時に作成）。Web UI からも編集可能。

| 項目 | 型 | デフォルト | 説明 |
|------|----|-----------|------|
| `auth_method` | `string` | `"oauth"` | `"oauth"` または `"service_account"` |
| `credentials_path` | `string` | — | OAuth / SA の JSON パス（埋め込みクレデンシャル使用時は省略可） |
| `token_path` | `string` | `"token.json"` | OAuth トークンの保存先（設定ディレクトリからの相対パス） |
| `sync_folder_id` | `string` | — | 同期対象の Google Drive フォルダ ID |
| `local_sync_dir` | `string` | — | ローカル同期先ディレクトリ |
| `scopes` | `[]string` | `[".../drive.readonly"]` | Drive API の OAuth スコープ |
| `export_formats` | `map` | Docs→PDF, Sheets→XLSX, Slides→PPTX | Google Docs のエクスポート形式 |
| `chunk_size_mb` | `int` | `10` | ダウンロードチャンクサイズ (MB) |
| `max_workers` | `int` | `3` | 並列ダウンロード数 |
| `revision_naming` | `string` | `"{stem}.rev{rev_id}{suffix}"` | リビジョンファイルの命名パターン |
| `ignore_patterns` | `[]string` | `[]` | 同期除外の glob パターン |
| `select_patterns` | `[]string` | `[]` | 同期対象を限定する include パターン（空なら全件。プレフィックス / `*` / `**` ワイルドカード対応） |
| `conflict_strategy` | `string` | `"skip"` | ローカル変更検出時の動作: `"skip"`（上書き回避） / `"overwrite"`（警告して上書き） |
| `converters` | `[]object` | `[]` | 外部コマンド変換プラグインの設定（下記参照） |
| `converter_workers` | `int` | `1` | 変換の並列実行数 |
| `distribution_targets` | `[]object` | `[]` | 配布先の設定（下記参照） |

### 変換プラグインの設定例

```json
{
  "converters": [
    {
      "name": "mp4-to-hap",
      "enabled": true,
      "input_pattern": "*.mp4",
      "output_extension": ".mov",
      "output_dir": "converted/hap",
      "command": "ffmpeg -y -i {{input}} -c:v hap -format hap {{output}}",
      "auto_convert": false
    }
  ]
}
```

コマンドテンプレート変数: `{{input}}`（入力パス）, `{{output}}`（出力パス）, `{{stem}}`（拡張子なしファイル名）, `{{dir}}`（親ディレクトリ）

### 配布先の設定例

```json
{
  "distribution_targets": [
    {
      "name": "archive",
      "type": "local",
      "path": "/mnt/archive",
      "select_patterns": ["**/*.mp4", "**/*.mkv"]
    },
    {
      "name": "office-pc",
      "type": "smb",
      "server": "192.168.1.10",
      "share": "shared-folder",
      "username": "user",
      "password": "pass",
      "domain": "WORKGROUP",
      "select_patterns": ["documents/**"]
    }
  ]
}
```

各配布先の `select_patterns` で同期済みファイルのうち配布対象を限定できます。パスワードは設定保存時に AES-GCM で暗号化されます。

## 開発

### 開発モード

```bash
task dev   # フロントエンド (HMR) + Go サーバー (air によるホットリロード) を同時起動
```

<http://localhost:3000> を開く。Go ファイルを編集すると自動でリビルド・再起動されます。

### ビルド

```bash
task build          # フロントエンド + Go バイナリ (./gdrive-sync)
task build-linux    # Linux amd64 向けクロスコンパイル
task build-windows  # Windows amd64 向けクロスコンパイル (.exe)
task types          # Go 構造体から TypeScript 型を自動生成 (tygo)
task test           # Go テスト実行
task clean          # ビルド成果物の削除
```

### 手動ビルド

```bash
cd frontend && bun install && bun run build && cd ..

# 埋め込みクレデンシャルあり
CGO_ENABLED=1 go build -o gdrive-sync -ldflags="-s -w -X github.com/P4sTela/matsu-sonic/internal/drive.DefaultClientID=YOUR_ID -X github.com/P4sTela/matsu-sonic/internal/drive.DefaultClientSecret=YOUR_SECRET" .

# 埋め込みクレデンシャルなし（開発用、credentials_path での認証が必要）
CGO_ENABLED=1 go build -o gdrive-sync -ldflags="-s -w" .
```

### リリース

`v*` タグを push すると GitHub Actions が Windows / macOS / Linux 向けバイナリをビルドし、GitHub Release を自動作成します。

```bash
git tag v1.0.0
git push origin v1.0.0
```

バージョンはビルド時に `ldflags` で埋め込まれます。`--version` フラグで確認可能:

```bash
./gdrive-sync --version
# gdrive-sync v1.0.0
```

## アーキテクチャ

```
gdrive-sync
├── main.go                 # エントリポイント (CLI, ポータブル設定パス解決)
├── embed.go                # go:embed frontend/dist
├── internal/
│   ├── config/             # 設定構造体, JSON ローダー (デフォルト値付き), シークレット暗号化 (secret.go)
│   ├── store/              # SQLite (WAL): files, runs, revisions, distribution_jobs, conversions
│   ├── drive/              # Google Drive API: 認証 (OAuth PKCE / SA), クレデンシャル埋め込み (credentials.go), 一覧, ダウンロード, リビジョン
│   ├── sync/               # 同期エンジン, 差分判定 (MD5), 選択同期 (select.go), コンフリクト検出 (conflict.go), 進捗追跡
│   ├── converter/          # 変換マネージャー, テンプレート展開, 外部コマンド実行
│   ├── server/             # Chi v5 ルーター, ミドルウェア (CORS/Logger/Recovery), WebSocket Hub
│   ├── handler/            # REST API エンドポイント (auth, sync, files, verify, browse, drive_browse, converter, distribution)
│   └── distribution/       # コピー先 Target (ローカル, SMB), 接続テスト, バッチ配布
└── frontend/
    └── src/
        ├── api/            # 型定義 (generated/ — tygo 自動生成), fetch ラッパー, WebSocket クライアント
        ├── hooks/          # useSync, useWebSocket, useConfig, SyncProvider
        ├── pages/          # Sync, Files, Revisions, Distribution (DistributePage), Settings
        └── components/     # ProgressBar, FileTreePicker, DirBrowser, DriveBrowser, shadcn/ui (button, dialog, tabs, ...)
```

### 技術スタック

**バックエンド**

| 技術 | 用途 |
|------|------|
| [Go](https://go.dev/) | サーバー, 同期エンジン, CLI |
| [Chi v5](https://github.com/go-chi/chi) | HTTP ルーティング & ミドルウェア |
| [gorilla/websocket](https://github.com/gorilla/websocket) | リアルタイム進捗通知 |
| [SQLite](https://github.com/mattn/go-sqlite3) (WAL) | メタデータ & 同期履歴 |
| [google-api-go-client](https://github.com/googleapis/google-api-go-client) | Drive API |
| [errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup) | 並列ダウンロード制御 |
| [go-smb2](https://github.com/hirochachacha/go-smb2) | SMB/CIFS ファイル配布 |
| [air](https://github.com/air-verse/air) | Go ホットリロード (開発時) |

**フロントエンド**

| 技術 | 用途 |
|------|------|
| [React 19](https://react.dev/) + TypeScript | UI |
| [Vite](https://vite.dev/) | ビルド & HMR |
| [Tailwind CSS v4](https://tailwindcss.com/) | スタイリング |
| [shadcn/ui](https://ui.shadcn.com/) + [@base-ui/react](https://base-ui.com/) | UI コンポーネント |
| [React Router v7](https://reactrouter.com/) | ルーティング |
| [lucide-react](https://lucide.dev/) | アイコン |
| [sonner](https://sonner.emilkowal.ski/) | トースト通知 |
| [next-themes](https://github.com/pacocoursey/next-themes) | ダークモード |

### API 一覧

| メソッド | エンドポイント | 説明 |
|----------|---------------|------|
| `GET` | `/api/config` | 現在の設定を取得（パスワードはマスク） |
| `POST` | `/api/config` | 設定を更新（部分マージ） |
| `POST` | `/api/auth/test` | Drive API 認証テスト |
| `POST` | `/api/auth/start` | OAuth PKCE 認証フローを開始（認証 URL を返却） |
| `GET` | `/api/auth/status` | 認証状態の取得（authenticated / pending） |
| `POST` | `/api/sync/full` | フル同期を開始 |
| `POST` | `/api/sync/incremental` | 差分同期を開始 |
| `POST` | `/api/sync/cancel` | 実行中の同期をキャンセル |
| `GET` | `/api/sync/status` | 同期進捗スナップショット |
| `GET` | `/api/sync/diff` | ドライラン: 変更プレビュー |
| `POST` | `/api/sync/preview` | 選択同期パターンのプレビュー（ローカル DB 照合） |
| `POST` | `/api/sync/reset` | 同期レコードを全削除 |
| `GET` | `/api/sync/history` | 同期履歴の取得（`?limit=` 指定可） |
| `GET` | `/api/files` | 同期済みファイル一覧（`?search=` 対応） |
| `GET` | `/api/files/{fileID}` | 単一ファイルの詳細取得 |
| `POST` | `/api/files/delete` | 指定ファイルのレコードを削除 |
| `POST` | `/api/files/verify` | ファイル整合性検証（MD5 比較、WebSocket 進捗付き） |
| `POST` | `/api/files/resync` | 指定ファイルのチェックサムをクリア（次回同期で再ダウンロード） |
| `GET` | `/api/files/{fileID}/revisions` | ファイルのリビジョン一覧 |
| `POST` | `/api/files/{fileID}/revisions/{revID}/download` | 特定リビジョンをダウンロード |
| `GET` | `/api/files/{fileID}/revisions/downloaded` | ダウンロード済みリビジョン一覧 |
| `GET` | `/api/distribution/targets` | 配布先一覧（パスワードはマスク） |
| `POST` | `/api/distribution/targets` | 配布先を追加 |
| `PUT` | `/api/distribution/targets/{name}` | 配布先を更新 |
| `DELETE` | `/api/distribution/targets/{name}` | 配布先を削除 |
| `POST` | `/api/distribution/targets/{name}/test` | 配布先への接続テスト |
| `POST` | `/api/distribute` | ファイルを配布先にコピー（バッチ） |
| `GET` | `/api/distribution/jobs` | 配布ジョブ履歴（`?limit=` 指定可） |
| `GET` | `/api/browse` | ローカルディレクトリの閲覧（`?path=` 指定） |
| `POST` | `/api/mkdir` | ディレクトリの作成 |
| `GET` | `/api/drive/browse` | Google Drive フォルダの閲覧（`?folder_id=` / `?source=shared` 指定） |
| `GET` | `/api/conflicts` | ローカル変更・削除によるコンフリクト一覧 |
| `GET` | `/api/converters` | 設定済みコンバーター一覧 |
| `POST` | `/api/files/{fileID}/convert` | 指定ファイルの変換を実行 |
| `POST` | `/api/files/{fileID}/reconvert` | 指定ファイルの再変換（既存レコードを上書き） |
| `GET` | `/api/files/{fileID}/conversions` | ファイルの変換履歴 |
| `GET` | `/api/conversions` | 全体の変換履歴 |
| `GET` | `/api/conversions/stale` | 再変換が必要な変換一覧 |
| `DELETE` | `/api/conversions/{id}` | 変換ジョブ記録の削除 |
| `GET` | `/ws` | WebSocket（リアルタイム進捗・検証進捗・変換進捗・コンフリクト通知） |

## ロードマップ

- [x] **SMB/CIFS 配布先** — ネットワーク共有へのコピー対応（NTLM 認証、バッチ配布）
- [x] **配布先ディレクトリのカスタマイズ** — 配布実行時に配布先サブディレクトリを指定
- [x] **選択同期** — include パターン（プレフィックス / `*` / `**` ワイルドカード）で同期対象を限定
- [x] **配布先ごとの選択配布** — 配布先ターゲットごとに include パターンを設定し、送信するファイルを限定
- [x] **同期コンフリクト検出** — ローカルファイルの変更を警告
- [ ] **通知** — 同期完了/エラー時のデスクトップ・Webhook 通知
- [ ] **マルチフォルダ同期** — 複数の Drive フォルダからの同期
- [x] **ファイル整合性検証** — ダウンロード後のチェックサム検証（WebSocket でリアルタイム進捗表示）

## ライセンス

[MIT](LICENSE)
