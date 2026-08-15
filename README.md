# foundry-codex-bridge

Microsoft Foundry / Azure OpenAI の deployment を opencodex 経由で Codex から利用するための Windows GUI 管理ツール。

## 現在の状態

Issue #2 の第1段階を実装済み。Wails v2 のデスクトップ画面から Azure の既存 Model Resource と Model Deployment を選択し、明示的な Sync 操作で opencodex の Provider、PrimaryKey、モデル、Codex カタログを順番に反映できる。

Azure認証にはAzure CLIのログイン済み資格情報を使用する。Bridge独自のEntraアプリを利用者Tenantへ追加せず、Client IDやclient secretを要求しない。Bridgeの設定ファイルにはsecretを保存しない。Azure CLI、Node.js、npmは事前に導入しておく必要があり、Bridgeは自動導入しない。

実際のAzure接続とResponses接続テストには、Azure CLI、Azureのアクセス権、既存のAzure Model ResourceとModel Deploymentが必要である。Connect画面から`az login`を開始でき、既存のAzure CLIログインも再利用する。

## 開発要件

- Windows
- Go 1.25 以上
- Azure CLI
- Node.js 18 以上と npm
- Wails v2 CLI
- mise

各ツールは`PATH`から実行できる必要がある。miseからNode.jsなどを自動導入しない。

開発モードでアプリを起動する。

```powershell
mise run dev
```

テストとproduction buildをまとめて実行する。

```powershell
mise run build
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
