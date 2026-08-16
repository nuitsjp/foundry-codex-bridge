# リポジトリ構成

Wails v2 の標準構成を起点にし、Go 側は外部境界ごとの小さな package に分ける。大きなフレームワーク風の層分けはせず、必要な責務だけを置く。

```text
.
├── build/
│   ├── appicon.png
│   └── windows/
├── docs/
│   ├── adr/
│   ├── architecture.md
│   ├── development.md
│   ├── phase-1-validation.md
│   └── repository-structure.md
├── frontend/
│   ├── src/
│   │   ├── api.ts
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── styles.css
│   │   └── vite-env.d.ts
│   ├── index.html
│   ├── package-lock.json
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── internal/
│   ├── azure/
│   │   ├── cli.go
│   │   ├── client.go
│   │   ├── client_test.go
│   │   ├── candidates.go
│   │   ├── errors.go
│   │   ├── errors_test.go
│   │   └── types.go
│   ├── opencodex/
│   │   ├── cli.go
│   │   ├── installer.go
│   │   ├── lifecycle.go
│   │   ├── provider_test.go
│   │   └── types.go
│   ├── bridge/
│   │   ├── service.go
│   │   ├── settings.go
│   │   ├── sync_test.go
│   │   └── types.go
│   └── platform/
│       ├── paths_other.go
│       ├── paths_windows.go
│       ├── process_other.go
│       └── process_windows.go
├── scripts/
│   ├── doctor.ps1
│   └── init.ps1
├── app.go
├── main.go
├── mise.toml
├── wails.json
├── go.mod
├── go.sum
├── CONTEXT.md
├── README.md
└── .gitignore
```

## ルート

- `main.go`: Wails アプリケーションの起動だけを担当する。
- `app.go`: TypeScript へ公開する Wails bound method を置く。実処理は `internal/bridge` へ委譲する。
- `wails.json`: Wails project configuration。
- `mise.toml`: GoとWailsのバージョン、および開発タスクを定義する。
- `scripts/doctor.ps1`: 開発環境の前提条件を読み取り診断する。
- `scripts/init.ps1`: mise管理ツールとプロジェクト依存を初期化する。
- `CONTEXT.md`: プロジェクト用語集。

## Go packages

- `internal/azure`: Azure CLI実行境界、`AzureCLICredential`を使うAzure SDK client、Azure DTO mapping、認可エラーの分類。
- `internal/opencodex`: opencodex のインストールと、公開 `ocx` CLI を使ったプロセス制御、health、Provider 管理。構造化出力があるコマンドは JSON を解析する。
- `internal/bridge`: Azure と opencodex をまたぐアプリケーション use case。
- `internal/platform`: OS ごとのユーザー設定ディレクトリ解決と、WindowsのCLI非表示起動・UAC昇格起動。

## Frontend

- `frontend/src/App.tsx`: Connect、Deployments、opencodex、Sync の4タブと状態遷移をまとめる。
- `frontend/src/api.ts`: Wails bound method への型付き呼び出しをまとめる。
- `frontend/src/components`: 複数の画面で使う UI component が必要になった場合だけ置く。

## Settings

Bridgeは非secretな端末固有設定だけを`%LOCALAPPDATA%\FoundryCodexBridge\settings.json`に保存する。Azureのログイン状態とトークンキャッシュはAzure CLIに委ね、Bridgeのアプリデータへ認証レコードを保存しない。opencodexのインストール先やProvider対応表をroaming profileへ持ち出さない。

保存対象:

- Azure resource ID と Bridge-managed Provider ID の対応
- last selected tenant
- last selected subscription
- last selected resource group
- last selected Azure Model Resource
- last selected deployment

secretはAzure CLIまたはopencodex credential storageに委ねる。Azure CLIとopencodexのインストール先、healthが返すport、Node.jsのバージョンは状態として検出し、settings.jsonには保存しない。

## Test layout

unit test は対象 package の隣に置く。

```text
internal/
├── azure/
│   ├── client_test.go
│   └── errors_test.go
├── opencodex/
│   └── provider_test.go
└── bridge/
    └── sync_test.go
```

live Azure test は明示的な環境変数がある場合だけ実行する。既定の unit test command では実行しない。
