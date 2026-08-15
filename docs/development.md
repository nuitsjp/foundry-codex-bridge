# 開発手順

## 前提条件

- Windows
- Azure CLI
- Node.js 18 以上と npm
- mise

Azure CLI、Node.js、npm、miseは`PATH`から実行できる必要がある。Go 1.25.13とWails 2.14.0はmiseで導入・固定する。Node.jsとAzure CLIは自動導入しない。`ocx`がPATHにない場合のopencodexだけは、利用者がGUIで導入を承認したときにnpmの`--prefix`を使ってユーザー領域へ導入する。

## 初期化と診断

リポジトリを取得した後、次を実行する。

```powershell
mise trust
mise run init
```

`mise trust`はリポジトリの`mise.toml`を確認した利用者が明示実行する。`init`はmise管理のGoとWailsを導入し、mise shimsをユーザー`PATH`へ登録してPowerShell profileの末尾でmiseを有効化し、Go modulesとfrontend依存を復元してからプロジェクトの`doctor`を実行する。Node.jsまたはAzure CLIがない場合は自動導入せず、導入先を表示して停止する。profile設定は新しいPowerShell sessionから有効になる。

環境だけを再診断する場合は次を実行する。

```powershell
mise run doctor
```

これはFoundryCodex Bridgeの開発環境を診断するタスクである。mise本体とshell連携の診断には`mise doctor`を使用する。

## Azure の設定

Azure CLIをインストールする。BridgeのConnect画面はAzure CLIの導入状態とアクティブなアカウントを診断し、ボタン操作から`az login`を開始する。あらかじめ次のコマンドでログインした状態も再利用できる。

```powershell
az login
```

BridgeはAzure SDK for Goの`AzureCLICredential`を使用する。選択したTenantとSubscriptionを資格情報へ明示し、Azure CLIが管理するログイン状態からARM用トークンを取得する。Bridgeは`az logout`、`az account clear`、Azure CLIの認証キャッシュの直接読み書きを行わない。

対象のSubscriptionとAzure Model Resourceには、既存リソースの参照と、Sync時の`listKeys`実行に必要なAzure権限が必要である。

対象にできる Azure Model Resource は `kind = AIServices` または `kind = OpenAI` の既存リソースである。Managed Compute Deployment、Marketplace 契約、Azure RBAC の変更はこの段階では扱わない。

## ローカル検証

開発モードでアプリを起動する。

```powershell
mise run dev
```

Goのテスト、frontendのclean installとbuild、Wailsのproduction buildをまとめて実行する。

```powershell
mise run build
```

`npm ci` は `frontend/package-lock.json` に固定された依存関係を使う。生成される `frontend/dist`、`frontend/wailsjs`、`build/bin` はリポジトリへコミットしない。

ローカルのAzure CLIを読み取り専用で検出するopt-inテストは、次のように実行する。このテストは`az login`やAzureリソース操作を実行しない。

```powershell
$env:FOUNDRYCODEX_LIVE_AZURE_CLI = "1"
go test ./internal/azure -run TestLiveAzureCLIState -v
Remove-Item Env:FOUNDRYCODEX_LIVE_AZURE_CLI
```

## 実機検証

1. Azure CLIをインストールした状態でアプリを起動する。
2. ConnectタブからAzure CLIのサインインを開始し、tenant、subscription、resource group、Azure Model Resourceを選択する。
3. Deployments タブで既存の Model Deployment と Codex 候補モデルを確認する。
4. opencodex タブで Node.js と npm の状態を確認し、必要なら利用者の承認で opencodex を導入する。
5. Sync タブで Provider ID と対象 Deployment を確認し、Azure 料金が発生し得る接続テストに同意してから Sync を実行する。

Sync は `ocx service install`、Provider 設定、PrimaryKey 登録、custom model、selected model、`ocx sync`、Responses endpoint の接続テストを順番に実行する。途中で失敗しても完了済みの操作は自動で巻き戻さないため、同じ選択で再実行して状態を収束させる。

API key、OAuth token、refresh tokenはBridgeの画面、ログ、Wailsのエラー、settings.jsonに出さない。Azure CLIの認証キャッシュはAzure CLIだけに管理させる。実機検証で使用した資格情報やAzureの値をコミットしない。
