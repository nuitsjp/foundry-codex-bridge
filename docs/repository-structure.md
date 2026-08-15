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
│   │   ├── client.go
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
│       └── paths_windows.go
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

- `internal/azure`: Azure SDK client、Azure DTO mapping、認証レコード、認可エラーの分類。
- `internal/opencodex`: opencodex のインストールと、公開 `ocx` CLI を使ったプロセス制御、health、Provider 管理。構造化出力があるコマンドは JSON を解析する。
- `internal/bridge`: Azure と opencodex をまたぐアプリケーション use case。
- `internal/platform`: OS ごとのユーザー設定ディレクトリ解決。

## Frontend

- `frontend/src/App.tsx`: Connect、Deployments、opencodex、Sync の4タブと状態遷移をまとめる。
- `frontend/src/api.ts`: Wails bound method への型付き呼び出しをまとめる。
- `frontend/src/components`: 複数の画面で使う UI component が必要になった場合だけ置く。

## Settings

Bridge は非 secret な端末固有設定だけを `%LOCALAPPDATA%\FoundryCodexBridge\settings.json` に保存する。Azure の認証レコードは同じアプリデータディレクトリの `authentication-record.json` に保存し、トークンキャッシュの保存は Azure SDK に委ねる。opencodex のインストール先や Provider 対応表を roaming profile へ持ち出さない。

保存対象:

- Azure resource ID と Bridge-managed Provider ID の対応
- last selected tenant
- last selected subscription
- last selected resource group
- last selected Azure Model Resource
- last selected deployment

secret は Azure SDK token cache または opencodex credential storage に委ねる。opencodex のインストール先、health が返す port、Node.js のバージョンは状態として検出し、settings.json には保存しない。

## Test layout

unit test は対象 package の隣に置く。

```text
internal/
├── azure/
│   └── errors_test.go
├── opencodex/
│   └── provider_test.go
└── bridge/
    └── sync_test.go
```

live Azure test は明示的な環境変数がある場合だけ実行する。既定の unit test command では実行しない。
