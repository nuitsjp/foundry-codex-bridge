# Model version 更新ポリシーは Azure の既定に従う

## Context

Model Deployment は `OnceNewDefaultVersionAvailable`、`OnceCurrentVersionExpired`、`NoAutoUpgrade` の version upgrade policy を持てる。Bridge が特定値を独自の既定にすると、Azure portal や API で作成した deployment と挙動が異なり、Azure 側の既定変更にも追従できない。現行 Azure では未設定の `null` は `OnceCurrentVersionExpired` 相当である。

## Decision

Bridge は独自の既定値を設けない。新規作成時は、利用者が明示的に選択しない限り `versionUpgradeOption` を送信せず、Azure の既定に委ねる。既存 deployment の更新では、利用者が変更しない限り現在値を維持する。Azure が選択肢を提供する場合は3つの policy を GUI から明示選択できる。

## Consequences

Bridge と Azure portal の既定動作が揃い、Azure 側の仕様変更に追従できる。利用者が新しい既定版への即時追従を望む場合は、GUI で `OnceNewDefaultVersionAvailable` を選択する必要がある。GUI は未設定と明示設定を区別して表示する。
