# マルチテナントのブラウザ対話認証を使う

> Status: ADR 0014 により廃止。

FoundryCodex Bridge は、プロジェクト側で管理するマルチテナント Entra アプリ登録の client ID と Azure SDK for Go の `InteractiveBrowserCredential` を使ってユーザーを認証する。Bridge は Public Azure 向けの汎用配布ツールであるため、利用組織ごとのアプリ登録、client secret、Azure CLI への事前ログインを要求せず、各組織の同意ポリシーに従って GUI のサインインから subscription 選択、deployment 管理まで進められるようにする。デスクトップアプリであるためパブリッククライアントとして登録し、client secret は持たない。Azure SDK の名前付き永続キャッシュと非機密の `AuthenticationRecord` で前回のアカウントを再利用し、同時に有効なアカウントは1つとする。再認証が必要な場合だけブラウザを開き、アカウント切り替え時は有効な `AuthenticationRecord` を置き換える。

この方式では、利用者が別Tenantから初めてサインインした際に、Bridgeのサービスプリンシパルと委任同意がそのTenantへ作成され得る。利用組織へBridge独自のEnterprise Applicationや管理者承認を要求しない方針を優先するため、採用を取り消した。
