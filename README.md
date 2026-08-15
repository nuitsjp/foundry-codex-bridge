# foundry-codex-bridge

Microsoft Foundry / Azure OpenAI の deployment を opencodex 経由で Codex から利用するための Windows GUI 管理ツール。

## 現在の状態

Issue #2 の第1段階を実装済み。Wails v2 のデスクトップ画面から Azure の既存 Model Resource と Model Deployment を選択し、明示的な Sync 操作で opencodex の Provider、PrimaryKey、モデル、Codex カタログを順番に反映できる。

Azure には `InteractiveBrowserCredential` でサインインし、認証レコードとトークンキャッシュはユーザー領域に保存する。Bridge の設定ファイルには secret を保存しない。Node.js と npm は事前に導入しておく必要があり、Bridge は自動導入しない。

実際の Azure 接続と Responses 接続テストには、プロジェクト所有の Entra パブリッククライアント ID、Azure のアクセス権、既存の Azure Model Resource と Model Deployment が必要である。リリースビルドには client ID を埋め込み、開発時だけ環境変数で上書きする。

## 開発要件

- Windows
- Go 1.25 以上
- Node.js 18 以上と npm
- Wails v2 CLI

開発時はプロジェクト所有のパブリッククライアント ID を設定する。リリースビルドでは `main.embeddedAzureClientID` へ値を埋め込む。

```powershell
$env:FOUNDRYCODEX_AZURE_CLIENT_ID = "<public-client-id>"
```

Node.js が利用できる状態で、次を実行する。

```powershell
go test ./...
Push-Location frontend
npm ci
npm run build
Pop-Location
wails build
```

リリース用に client ID を埋め込む場合は、Wails の linker flag を指定する。

```powershell
wails build -ldflags "-X main.embeddedAzureClientID=<public-client-id>"
```

`ocx` が PATH にない場合、GUI の「利用者の承認でopencodexを導入」から npm を使って `%LOCALAPPDATA%\FoundryCodexBridge\opencodex` に導入する。グローバル npm 環境と PATH は変更しない。

## 第1段階の実装境界

- 対象は Public Azure の `kind = AIServices` または `kind = OpenAI` の既存リソースと、通常の既存 Model Deployment だけである。
- Azure Model Resource の作成、Deployment の作成・更新、Marketplace 契約、Azure RBAC の変更は行わない。
- local authentication が無効なリソースは、API key を使う現行 opencodex アダプターの制約により Sync 対象外とする。
- 起動時は読み取り専用で、Provider、Catalog、service、接続テストを自動変更しない。
- Sync の接続テストは実際の Azure リクエストになるため、画面上の確認が必要である。

## 設計ドキュメント

- [アーキテクチャ](./docs/architecture.md)
- [リポジトリ構成](./docs/repository-structure.md)
- [開発手順](./docs/development.md)
- [用語集](./CONTEXT.md)
- [ADR](./docs/adr/)
