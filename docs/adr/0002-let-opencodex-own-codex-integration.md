# Codex 統合は opencodex に委ねる

FoundryCodex Bridge は Codex App の設定ファイルやモデルカタログを直接編集しない。opencodex がすでに冪等で復元可能な Codex 設定注入、モデルカタログ同期、復元を担当しているため、Bridge は公開 `ocx` CLI と連携し、Codex App を opencodex が管理する下流状態として扱う。JSON 対応コマンドは構造化出力を解析し、非対応コマンドは終了コードと標準エラーを扱う。Management API は `ocx` 自身に利用させ、Bridge から直接呼ばない。これにより Management API の admin token を Bridge が扱わず、API key は `ocx account add-key` の標準入力へ渡せる。

現行の `azure-openai` アダプターは API key を必須とし、Azure Entra token refresh を提供しない。このため local authentication が無効な Azure Model Resource は Sync 対象外とし、短寿命の access token を key として保存する代替処理は行わない。

Disconnect では opencodex の default Provider と最終 Provider の制約を回避しない。対象が default の場合は利用者が GUI で代替 Provider を明示選択し、公開 CLI で切り替えてから削除する。代替が存在しない場合は削除しない。
