# Converter Plugin 設計案

## 背景・目的

映像・バイナリなど大容量ファイルの一方向同期に加え、取得後のファイル変換ニーズがある。
代表的な例:

- `mp4` → `HAP` / `HAP Q` / `HAP Alpha`（VJ / メディアサーバー向け）
- `mp4` → `ProRes`
- 画像のリサイズ・フォーマット変換
- 将来的には音声変換、ドキュメント変換など

これらをハードコードせず、外部コマンドベースの **Converter Plugin** として汎用的に扱う。

---

## 設計方針

### 1. プラグイン = 外部コマンドテンプレート

Go のネイティブ plugin（`.so`）や WASM などは導入コストが高い。
まずは **外部コマンドを設定テンプレート化** する方式を採用する。

理由:

- `ffmpeg` など既存ツールをそのまま使える
- ユーザーが独自の変換スクリプトを追加しやすい
- 配布バイナリのサイズ・依存が増えない
- セキュリティリスクはサンドボックス化で緩和（後述）

将来的にネイティブプラグインが必要になった場合、`Converter` インターフェースを実装する形で拡張可能。

---

## 型定義

### Config 追加項目

```go
// ConverterConf defines an external command-based converter.
type ConverterConf struct {
 Name            string   `json:"name"`
 Enabled         bool     `json:"enabled"`
 InputPattern    string   `json:"input_pattern"`    // glob, e.g. "*.mp4"
 OutputExtension string   `json:"output_extension"` // e.g. ".mov"
 OutputDir       string   `json:"output_dir"`       // e.g. "converted/hap" (relative to sync root)
 Command         string   `json:"command"`          // e.g. "ffmpeg -i {{input}} -c:v hap -format hap {{output}}"
 Env             []string `json:"env,omitempty"`    // optional KEY=VALUE overrides
 AutoConvert     bool     `json:"auto_convert"`     // run automatically after sync
}

type Config struct {
 // ... existing fields ...
 Converters []ConverterConf `json:"converters"`
}
```

### 内部モデル

```go
package converter

// Converter is the runtime representation of a conversion plugin.
type Converter struct {
 Name            string
 Enabled         bool
 InputPattern    string
 OutputExtension string
 CommandTemplate []string // parsed command tokens
 Env             []string
 AutoConvert     bool
}

// ConversionRecord represents a persisted conversion result.
type ConversionRecord struct {
 ID               string
 FileID           string
 Converter        string
 InputPath        string
 OutputPath       string
 Status           string // pending | running | completed | failed
 Error            string
 StartedAt        string
 FinishedAt       string
 OriginalSize     int64
 OriginalModified string
}

// Job represents a single conversion execution.
type Job struct {
 ID        string
 FileID    string
 Name      string
 InputPath string
 OutputPath string
 Converter string
 Status    string // pending | running | completed | failed
 Error     string
 StartedAt string
 FinishedAt string
}

// Manager runs converters.
type Manager struct {
 converters []Converter
 jobs       chan Job
}
```

### テンプレート変数

| 変数 | 説明 |
|------|------|
| `{{input}}` | 入力ファイルの絶対パス |
| `{{output}}` | 出力ファイルの絶対パス |
| `{{stem}}` | 入力ファイル名（拡張子なし） |
| `{{dir}}` | 入力ファイルの親ディレクトリ |
| `{{output_dir}}` | コンバーター設定の `output_dir`（同期ルートからの相対パス） |

将来的に `{{width}}`, `{{height}}` 等のメタデータ変数も追加可能。

---

## 動作フロー

### 基本方針

**変換は原則手動実行**。同期直後の自動変換は任意（`auto_convert` デフォルト false）。
配布時に元ファイルを除外したい場合は、既存の `select_patterns` で対応する。

### 手動変換

1. Files ページや Sync ページでファイルを選択
2. 変換対象のコンバーターを選択
3. `POST /api/files/{fileID}/convert` で実行
4. 進捗を WebSocket で配信
5. 結果を `conversions` テーブルに記録

### 自動変換（sync 後）

1. ファイルが同期完了し、DB に `local_size` / `local_modified` が記録された後
2. `Converters` を順に評価
3. 入力ファイルパスが `InputPattern` にマッチし、`Enabled && AutoConvert` ならジョブをキューへ
4. ワーカープールで変換を実行
5. 進捗を WebSocket で配信
6. 結果を `conversions` テーブルに記録

### 古くなった変換の検出

元ファイルが同期等で上書きされると、変換済みファイルは古くなる。
そのため `conversions` テーブルには変換実行時の元ファイル状態を記録し、現在の元ファイルと比較して **stale（再変換が必要）** か判定する。

| 状態 | 判定条件 |
|------|---------|
| `fresh` | 元ファイルのサイズ・更新時刻が変換時と一致 |
| `stale` | 元ファイルのサイズ・更新時刻が変換時と不一致 |
| `missing` | 元ファイルまたは変換後ファイルが存在しない |

### 出力ファイルの配置

- デフォルト: `{{sync_root}}/{{output_dir}}/<relative_dir>/<stem><output_extension>`
  - 例: `videos/intro.mp4` → `converted/hap/videos/intro.mov`
- `output_dir` が空の場合: 元ファイルと同じディレクトリに配置
- 元ファイルは上書きしない
- 出力ディレクトリは変換実行時に自動作成

---

## DB スキーマ追加

```sql
CREATE TABLE conversions (
    id          TEXT PRIMARY KEY, -- uuid or ulid
    file_id     TEXT NOT NULL,
    converter   TEXT NOT NULL,
    input_path  TEXT NOT NULL,
    output_path TEXT,
    status      TEXT DEFAULT 'pending',
    error_message TEXT,
    started_at  TEXT,
    finished_at TEXT,
    original_size     INTEGER,      -- input file size at conversion time
    original_modified TEXT,         -- input file mtime at conversion time
    UNIQUE(file_id, converter)
);
CREATE INDEX idx_conversions_file ON conversions(file_id);
CREATE INDEX idx_conversions_status ON conversions(status);
```

---

## Distribution との連携

配布先へ変換済みファイルを送信できるようにする。

### ユースケース

- アーカイブ用配布先には元の `mp4` を送信
- 放映用サーバーには `HAP` 変換済み `.mov` を送信
- 同一ファイルを異なる配布先に異なる形式で配布

### 設計

配布先に **デフォルトコンバーター** を設定可能にする。
配布実行時には **リクエストで上書き** も可能。

```go
// DistTargetConf 追加項目
type DistTargetConf struct {
    // ... existing fields ...
    Converter string `json:"converter,omitempty"` // default converter for this target
}
```

```go
// DistributeRequest 追加項目
type DistributeRequest struct {
    FileIDs    []string `json:"file_ids"`
    TargetName string   `json:"target_name"`
    DestDir    string   `json:"dest_dir"`
    Converter  string   `json:"converter,omitempty"` // optional override
}
```

### 配布時のファイル解決フロー

1. 対象ファイルの `local_path`（元ファイル）を取得
2. 使用するコンバーター名を決定:
   - `DistributeRequest.Converter` があれば使用
   - なければ `DistTargetConf.Converter`
   - どちらもなければ元ファイルを配布
3. コンバーターが指定されている場合:
   - `conversions` テーブルから `file_id + converter` で変換を検索
   - 見つからなければエラー（`conversion_not_found`）
   - 見つかっても **stale** ならエラー（`conversion_stale`）
   - `fresh` または `completed` かつ出力ファイルが存在すれば `output_path` を配布
   - 将来的には「未変換 / stale の場合は変換をキューし、完了後に配布」を検討

### 選択配布パターンとの兼ね合い

既存の `select_patterns` は元ファイルの相対パスで判定する。
変換後の拡張子は異なる場合があるため、配布時のパス解決は **元ファイルの相対パス** を基準に行い、出力ファイル名のみを置き換える。

例:

- 元: `videos/intro.mp4`
- コンバーター `output_dir`: `converted/hap`
- 変換後: `converted/hap/videos/intro.mov`
- `select_patterns: ["videos/**"]` → 元ファイルの相対パス `videos/intro.mp4` で判定し配布対象
- 配布先ファイル名: `videos/intro.mov`（または `dest_dir/videos/intro.mov`）
  - 元ファイルの相対ディレクトリ構造を維持し、拡張子のみ置き換える

### API 変更

`POST /api/distribute` のリクエストボディに `converter` フィールドを追加（省略可）。

### フロントエンド

- Distribute ページで「配布形式」セレクタを追加
  - デフォルト（元ファイル）
  - 各コンバーター
- 配布先設定でデフォルトコンバーターを選択可能に

---

## API 追加

| Method | Endpoint | 説明 |
|--------|----------|------|
| `GET` | `/api/converters` | 設定済みコンバーター一覧 |
| `POST` | `/api/files/{fileID}/convert` | 指定ファイルを手動変換 |
| `POST` | `/api/files/{fileID}/reconvert` | 指定ファイルを再変換（stale/completed を上書き） |
| `GET` | `/api/files/{fileID}/conversions` | ファイルの変換履歴（stale 状態含む） |
| `GET` | `/api/conversions` | 全体の変換履歴 |
| `GET` | `/api/conversions/stale` | stale（再変換が必要）な変換一覧 |
| `DELETE` | `/api/conversions/{id}` | 変換ジョブ記録を削除 |

WebSocket メッセージ:

- `convert_progress` — 変換進捗（stdout 解析 or ファイルサイズ増加）
- `convert_complete` — 完了/失敗

---

## セキュリティ

外部コマンド実行はリスクを伴う。以下を行う:

1. **テンプレート変数のみ許可**: コマンド文字列は予めトークン化し、変数以外はそのまま渡さない
2. **コマンドインジェクション防止**: `{{...}}` 以外のユーザー入力をコマンドに含めない
3. **実行前検証**: 設定保存時に `{{input}}` / `{{output}}` の存在を確認
4. **実行ユーザーの権限**: 同期先ディレクトリ内のみ書き込み
5. **タイムアウト**: 長時間変換を防止（デフォルト 30 分、設定可能に）
6. **将来**: 可能であれば `firejail` や `bubblewrap` 等でのサンドボックス実行を検討

---

## フロントエンド

- Settings ページに「Converters」セクションを追加
  - プリセット選択（ffmpeg mp4→hap 等）
  - カスタムコマンド編集
  - 有効/無効、自動変換 ON/OFF
- Files ページでファイルを右クリック / 選択して「Convert」 / "Reconvert"
- 変換済みファイルに stale バッジを表示
- Sync ページに変換進捗表示（オプション）

---

## 実装計画

1. `internal/converter` パッケージ新規作成
   - `config.go` — `ConverterConf` 型（既存 `config` パッケージに追加）
   - `converter.go` — `Converter` / `Manager`
   - `template.go` — テンプレート変数展開
   - `runner.go` — コマンド実行 + 進捗追跡
2. DB マイグレーション v3: `conversions` テーブル
3. `store/conversions.go` — CRUD
4. `handler/converter.go` — API エンドポイント
5. `sync/engine.go` — ダウンロード後の自動変換フック
6. フロントエンド: Settings/Files ページ拡張
7. README 更新

---

## 検討事項

### Q1. 出力ファイルの追跡方法

- A: `conversions` テーブルで追跡。元の `synced_files` には追加しない（元ファイルとの 1:N 関係）。

### Q2. 変換済みファイルの再変換

- A:
  - `POST /api/files/{fileID}/reconvert` で既存レコードを上書き再変換
  - 再変換時は既存の出力ファイルを削除してから実行
  - 元ファイルが stale 状態の場合、UI 上で警告アイコンを表示しワンクリックで再変換可能
  - `GET /api/conversions/stale` で一括確認

### Q3. HAP 変換のプリセット

```json
{
  "name": "mp4-to-hap",
  "enabled": true,
  "input_pattern": "*.mp4",
  "output_extension": ".mov",
  "output_dir": "converted/hap",
  "command": "ffmpeg -y -i {{input}} -c:v hap -format hap {{output}}",
  "auto_convert": false
}
```

### Q4. 並列実行数

- A: 別途 `converter_workers` 設定を追加（デフォルト 1）。動画変換は CPU/GPU リソースを圧迫するため保守的に。

---

## 結論

**外部コマンドテンプレート方式の Converter Plugin** を導入し、まずは ffmpeg ベースの動画変換（mp4→HAP）をサポートする。
将来的にインターフェースを拡張して、Go plugin や WASM 等の実装も差し替え可能にする。
