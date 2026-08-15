# Azure Marketplace は既存契約だけを利用する

## Context

Foundry のパートナーおよびコミュニティモデルには、Azure Marketplace の購入プランと利用条件への同意が必要なものがある。Bridge で新規契約まで行うには、subscription-level の Marketplace agreement 署名権限、SaaS resource provider 登録、SaaS リソース作成、価格と法的条件の表示および明示同意を扱う必要がある。

## Decision

初期版は、対象 subscription ですでに Marketplace 契約済みのモデルだけを通常の deployment 作成フローで利用する。Bridge は Marketplace offer の購入、利用条件への同意、agreement 署名、SaaS リソース作成を行わない。未契約が原因の Azure エラーは専用状態として GUI に表示する。

## Consequences

Marketplace 契約が必要なモデルを初めて利用する作業は Bridge だけでは完結しない。既存契約済みモデルと、Azure が直接販売するモデルの管理は継続できる。Bridge が subscription-level の Marketplace 契約権限を通常利用者へ要求せずに済む。
