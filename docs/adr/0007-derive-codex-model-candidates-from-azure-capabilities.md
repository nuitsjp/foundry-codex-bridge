# Azure の能力値を Codex 用モデル候補のヒントにする

## Context

Azure Model Resource が返すモデル一覧には、Codex の上流として適さない埋め込み、画像生成、旧 API 専用モデルも含まれる。一方、opencodex の `azure-openai` アダプターはモデル名の固定許可リストを持たず、任意のデプロイ名を Azure OpenAI v1 Responses API へ中継する。モデル名を Bridge に固定すると、新モデル追加のたびに Bridge の更新が必要になる。

## Decision

Bridge は Azure SDK for Go の `AccountsClient.NewListModelsPager` で選択中の Azure Model Resource のモデル一覧を取得する。Codex 用の候補には、OpenAI 形式では `capabilities.responses == true`、非 OpenAI 形式では `capabilities.agentsV2 == true` のモデルだけを表示する。この判定は Microsoft 公式の `Azure-Samples/ai-model-start` と同じ規則とし、モデル名の固定許可リストは持たない。

ただし `AccountModel.Capabilities` は任意キーの map であり、これらのキーは Management API の型契約として Responses API 互換性を保証するものではない。Bridge は条件を満たすモデルを Codex Candidate Model と呼び、実利用可能と断定しない。

選択された Model Deployment は opencodex のカスタムモデルとして登録し、Provider の live model discovery は無効にする。`ocx provider test` はモデル一覧 endpoint の疎通確認であって各デプロイの Responses API 互換性検証ではないため、Azure Provider の利用可否判定には使わない。

Sync 後、Bridge は選択した各 Model Deployment に対して opencodex の公開 Responses endpoint 経由で固定の最小リクエストを送る。このテストの失敗は設定をロールバックせず、Model Deployment 単位の接続状態として表示する。

## Consequences

Azure 側へ新しい Responses 対応モデルが追加され、能力値が設定された場合、Bridge の更新を待たずに候補へ現れる。能力値が未設定または誤っている場合や、実行時の互換性に問題がある場合は、候補から漏れるか、実リクエストを送るまで問題を検出できない。Bridge は Azure の能力値、opencodex の設定同期、実リクエストの成功を別の状態として扱う必要がある。
