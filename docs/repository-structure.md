# リポジトリ構成

Wails v2 の標準構成を起点にし、Go 側は外部境界ごとの小さな package に分ける。大きなフレームワーク風の層分けはせず、必要な責務だけを置く。

```text
.
├── build/
│   └── windows/
├── docs/
│   ├── adr/
│   ├── architecture.md
│   └── repository-structure.md
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── features/
│   │   │   ├── azure/
│   │   │   ├── deployments/
│   │   │   ├── opencodex/
│   │   │   └── sync/
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   └── vite.config.ts
├── internal/
│   ├── azure/
│   │   ├── auth.go
│   │   ├── subscriptions.go
│   │   ├── accounts.go
│   │   ├── deployments.go
│   │   ├── models.go
│   │   └── keys.go
│   ├── opencodex/
│   │   ├── cli.go
│   │   ├── installer.go
│   │   ├── provider.go
│   │   └── lifecycle.go
│   ├── bridge/
│   │   ├── service.go
│   │   ├── settings.go
│   │   └── sync.go
│   └── platform/
│       ├── paths_windows.go
│       └── process_windows.go
├── app.go
├── main.go
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
- `CONTEXT.md`: プロジェクト用語集。

## Go packages

- `internal/azure`: Azure SDK client と Azure DTO mapping。
- `internal/opencodex`: opencodex のインストールと、公開 `ocx` CLI を使ったプロセス制御、health、provider 管理。構造化出力があるコマンドは JSON を解析する。
- `internal/bridge`: Azure と opencodex をまたぐアプリケーション use case。
- `internal/platform`: Windows 固有の path と process 処理。

## Frontend

- `frontend/src/features/azure`: サインイン、subscription、resource group、account 選択。
- `frontend/src/features/deployments`: deployment 一覧、作成、更新、状態表示。
- `frontend/src/features/opencodex`: install / running / ready 状態。
- `frontend/src/features/sync`: 選択 deployment を opencodex Provider へ適用する workflow。
- `frontend/src/components`: 複数 feature で使う小さな UI component だけを置く。

## Settings

Bridge は非 secret な設定だけを `%APPDATA%\FoundryCodexBridge\settings.json` に保存する。

保存対象:

- active Azure authentication record
- Azure resource ID to Bridge-managed Provider ID mappings
- last selected tenant
- last selected subscription
- last selected resource group
- last selected Azure Model Resource
- last selected deployment
- opencodex port
- managed opencodex path and version

secret は Azure SDK token cache または opencodex credential storage に委ねる。

## Test layout

unit test は対象 package の隣に置く。

```text
internal/
├── azure/
│   └── deployments_test.go
├── opencodex/
│   └── provider_test.go
└── bridge/
    └── sync_test.go
```

live Azure test は明示的な環境変数がある場合だけ実行する。既定の unit test command では実行しない。
