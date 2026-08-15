# 開発手順

## 前提条件

- Windows
- Go 1.25 以上
- Node.js 18 以上と npm
- Wails v2 CLI
- ブラウザ認証に使用する、プロジェクト所有の Entra パブリッククライアント ID

Bridge は Node.js、npm、Wails CLI を自動導入しない。`ocx` が PATH にない場合の opencodex だけは、利用者が GUI で導入を承認したときに npm の `--prefix` を使ってユーザー領域へ導入する。

## Azure の設定

プロジェクト所有のマルチテナント Entra アプリケーションを用意する。リリースビルドではパブリッククライアント ID を `main.embeddedAzureClientID` へ埋め込み、開発時だけ環境変数で上書きする。client secret は作成せず、Bridge や利用者に入力させない。

```powershell
$env:FOUNDRYCODEX_AZURE_CLIENT_ID = "<public-client-id>"
```

Bridge は `InteractiveBrowserCredential` と Azure SDK の永続キャッシュを使用する。初回サインイン後は認証レコードを再利用する。対象の subscription と Azure Model Resource には、既存リソースの参照と、Sync 時の `listKeys` 実行に必要な Azure 権限が必要である。

対象にできる Azure Model Resource は `kind = AIServices` または `kind = OpenAI` の既存リソースである。Managed Compute Deployment、Marketplace 契約、Azure RBAC の変更はこの段階では扱わない。

## ローカル検証

リポジトリのルートで次を実行する。

```powershell
go test ./...
Push-Location frontend
npm ci
npm run build
Pop-Location
wails build
```

`npm ci` は `frontend/package-lock.json` に固定された依存関係を使う。生成される `frontend/dist`、`frontend/wailsjs`、`build/bin` はリポジトリへコミットしない。

リリース用に client ID を埋め込む場合は、次のように Wails の linker flag を指定する。client ID は public client の識別子であり、client secret は埋め込まない。

```powershell
wails build -ldflags "-X main.embeddedAzureClientID=<public-client-id>"
```

## 実機検証

1. `FOUNDRYCODEX_AZURE_CLIENT_ID` を設定してアプリを起動する。
2. Connect タブでブラウザ認証し、tenant、subscription、resource group、Azure Model Resource を選択する。
3. Deployments タブで既存の Model Deployment と Codex 候補モデルを確認する。
4. opencodex タブで Node.js と npm の状態を確認し、必要なら利用者の承認で opencodex を導入する。
5. Sync タブで Provider ID と対象 Deployment を確認し、Azure 料金が発生し得る接続テストに同意してから Sync を実行する。

Sync は `ocx service install`、Provider 設定、PrimaryKey 登録、custom model、selected model、`ocx sync`、Responses endpoint の接続テストを順番に実行する。途中で失敗しても完了済みの操作は自動で巻き戻さないため、同じ選択で再実行して状態を収束させる。

API key、OAuth token、refresh token は Bridge の画面、ログ、Wails のエラー、settings.json に出さない。実機検証で使用した資格情報や Azure の値をコミットしない。
