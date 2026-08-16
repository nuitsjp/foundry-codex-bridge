# FoundryCodex Bridge

FoundryCodex Bridge は、Microsoft Foundry または Azure OpenAI の Model Deployment を opencodex 経由で Codex から使えるようにする Windows デスクトップツールである。

## Language

**FoundryCodex Bridge**:
このリポジトリで開発するデスクトップ管理アプリケーション。GUI、Azure リソース管理、opencodex との連携を担当する。
_Avoid_: Bridge proxy, Codex plugin

**Azure CLI session**:
Azure CLIが管理するログイン済みアカウントとトークンキャッシュ。FoundryCodex Bridgeは`AzureCLICredential`経由で再利用し、独自のEntraアプリ登録や認証キャッシュを持たない。Bridgeは共有セッションからサインアウトしない。
_Avoid_: Bridge login cache, embedded Entra client

**Azure Model Resource**:
モデルデプロイを保持する Azure リソース。Microsoft Foundry リソースとスタンドアロン Azure OpenAI リソースの両方を含み、Foundry Project は含まない。
_Avoid_: Foundry Account, Portal, Foundry portal, Azure account

**Model Deployment**:
Azure Model Resource 配下の `Microsoft.CognitiveServices/accounts/deployments` として作成された通常のモデルデプロイ。デプロイ名、モデル識別子、バージョン、SKU、Capacity、プロビジョニング状態を持つ。Managed Compute Deployment は含まない。
_Avoid_: Model, endpoint, Managed Compute Deployment

**Deployable Model**:
Azure Model Resource へデプロイできるモデル。Codex から利用するには Model Deployment が必要である。
_Avoid_: Deployment

**Codex Candidate Model**:
Azure Model Resource のモデル一覧で、OpenAI 形式なら `capabilities.responses`、非 OpenAI 形式なら `capabilities.agentsV2` が `true` の Deployable Model。FoundryCodex Bridge が Codex 用のデプロイ候補として表示するが、この能力値は実リクエストの成功を保証しない。
_Avoid_: opencodex-supported model, supported model, Responses-compatible model

**opencodex**:
Codex のモデルリクエストを上流プロバイダーへ中継するローカルプロキシ。Codex 設定の注入、モデルカタログ同期、復元を担当する。
_Avoid_: Local proxy code, bridge service

**opencodex Provider**:
opencodex が管理する上流プロバイダー設定。エンドポイント、アダプター、認証方式、既定モデル、モデル許可リストを表す。
_Avoid_: Azure provider config, Codex provider

**Bridge-managed Provider**:
FoundryCodex Bridge が1つの Azure Model Resource に対応させて作成・更新する opencodex Provider。初回登録時に Provider ID を決め、登録後は同じ ID を使い続ける。複数の Bridge-managed Provider が同時に存在でき、それぞれ対応する1つ以上の Model Deployment を公開し、そのうち1つを既定モデルとする。
_Avoid_: Shared Azure provider, deployment provider

**Codex Catalog**:
Codex が読むモデルカタログ。FoundryCodex Bridge ではなく opencodex が管理する。
_Avoid_: Model list, picker list

**Sync**:
選択した Azure Model Resource とその Model Deployment を Bridge-managed Provider へ適用し、Codex から利用できる状態へ反映する再実行可能な操作。他の Azure Model Resource に対応する Provider は変更せず、途中失敗時は完了済み操作を巻き戻さず、次回実行で現在状態との差分を収束させる。
_Avoid_: Export, patch, direct Codex edit

**Disconnect**:
Bridge-managed Provider を opencodex から削除し、Azure Model Resource との対応を解除する操作。Azure Model Resource と Model Deployment は削除しない。
_Avoid_: Delete Azure resource, undeploy
