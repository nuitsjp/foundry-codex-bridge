# Azure CLI のログイン済み資格情報を使う

## Context

プロジェクト所有のマルチテナント Entra パブリッククライアントを使うと、利用者TenantでBridgeのサービスプリンシパルと委任同意が作成される。Tenantの同意ポリシーによっては、初回利用時に管理者承認も必要になる。汎用ローカルツールの導入によって、利用組織へBridge独自のEnterprise Applicationを追加することは避けたい。

独自Client IDを使う限り、Interactive Browser、Device Code、WAMのどの認証フローでもこのTenant側の同意境界はなくならない。一方、Azure CLIのログイン済み資格情報はAzure SDK for Goの`AzureCLICredential`から公式に利用できる。

## Decision

BridgeはAzure CLIを認証ブローカーとして使用する。Connect画面でAzure CLIの導入状態とアクティブなアカウントを診断し、利用者の明示操作で`az login`を開始する。GUIに端末入力を要求しないよう、そのコマンドプロセスだけ`AZURE_CORE_LOGIN_EXPERIENCE_V2=off`を設定してSubscription selectorを無効にする。Azure CLIの永続設定は変更しない。ARMクライアントには、選択中のTenant IDとSubscription IDを指定した`AzureCLICredential`を渡す。

BridgeはAzure CLIを同梱、自動インストール、更新しない。未導入の場合はGUIでインストールを求める。Azure CLIの認証キャッシュは共有状態であるため、Bridgeは`az logout`や`az account clear`を実行しない。アカウント変更は再度`az login`を開始して行う。

Bridge独自のEntraアプリ登録、Client ID、client secret、`InteractiveBrowserCredential`、`AuthenticationRecord`、Azure SDK永続トークンキャッシュは使用しない。

## Consequences

利用者TenantにBridge独自のEnterprise Applicationや委任同意を作成せずに済み、リリースビルドへClient IDを埋め込む必要もなくなる。一方、Azure CLIの事前導入が必須になり、Bridgeと他のAzure CLI利用ツールは同じログイン状態を共有する。Azure CLIからサインアウトまたはアカウント変更された場合、Bridgeも次の状態取得またはARM操作からその影響を受ける。
