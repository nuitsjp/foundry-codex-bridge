# Azure RBAC は診断するが割り当てない

## Context

Azure へのサインインに成功しても、Azure Model Resource の参照、Model Deployment の作成・更新、アカウントキーの取得には別々の ARM 権限が必要である。Bridge が不足ロールを自動割り当てるには、利用者へ `Microsoft.Authorization/roleAssignments/write` を含む強い権限を要求する必要がある。汎用配布ツールがこの権限を持つと、導入審査と誤操作時の影響が大きくなる。

## Decision

Bridge は Azure ロール割り当てを作成・更新しない。Azure SDK の各操作で認可エラーを判定し、不足している操作、対象 scope、管理者へ依頼する内容を GUI に表示する。初期機能では対象リソースの read、`Microsoft.CognitiveServices/accounts/deployments/write`、`Microsoft.CognitiveServices/accounts/listKeys/action` を主要な必要操作として扱う。

## Consequences

必要権限を持たない利用者だけでは初期設定を完結できず、Azure 管理者による事前の RBAC 設定が必要になる。一方、Bridge はロール管理権限を要求せず、組織ごとのカスタムロールにも対応できる。GUI はロール名だけでなく、失敗した操作と scope を説明する必要がある。
