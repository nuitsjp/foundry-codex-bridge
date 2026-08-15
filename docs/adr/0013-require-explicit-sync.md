# Sync は利用者の明示操作で開始する

## Context

Sync は opencodex Provider と Codex Catalog を変更し、完了後に選択した各 Model Deployment へ料金が発生し得る接続テストを送る。Bridge 起動時や常駐監視で自動実行すると、アプリを開いただけでローカル設定変更と Azure 利用が発生する。

## Decision

Bridge 起動時は Azure と opencodex の状態を読み取り、差分があれば `Sync が必要` と表示する。Provider、モデル、Catalog、service の変更は、利用者が GUI で Sync を明示実行した場合だけ行う。自動同期という語は、その1回の Sync 内の一連処理を自動化する意味で使う。

## Consequences

起動時の副作用と予期しない Azure 利用を避けられる。Azure または opencodex を外部から変更した場合、利用者が Bridge を開いて Sync を実行するまで Codex 側へは反映されない。
