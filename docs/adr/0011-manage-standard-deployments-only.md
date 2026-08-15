# 通常の Model Deployment だけを管理する

## Context

Foundry には `Microsoft.CognitiveServices/accounts/deployments` で管理される通常の deployment と、`Microsoft.CognitiveServices/accounts/managedComputeDeployments` で管理される専用 GPU の Managed Compute Deployment がある。後者は API、GPU SKU、accelerator capacity、runtime、起動状態、課金体系が異なり、OpenAI Responses 互換性も選択した runtime に依存する。

## Decision

初期版は Accounts Deployments API で管理される通常の Model Deployment だけを一覧、作成、更新、Sync の対象にする。Managed Compute Deployment は対象外とする。

## Consequences

Standard 系や Provisioned 系など通常の deployment を一貫した画面と Azure SDK client で扱える。専用 GPU 上のモデルは Bridge から管理・Sync できない。将来対応する場合は、既存 Model Deployment の分岐として埋め込まず、別の deployment 種別として設計する必要がある。
