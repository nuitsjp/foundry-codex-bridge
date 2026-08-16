# 第1段階 実機検証記録

## 実施情報

- 実施日: 2026-08-16
- OS: Microsoft Windows 11 Pro 10.0.26200
- Go: 1.25.13 windows/amd64
- Wails: 2.14.0
- Node.js: 24.18.0
- npm: 11.6.2
- Azure CLI: 2.88.0
- opencodex: 2.21.0

## 自動検証

`mise build`を実行し、次を確認した。

- `go test ./...`: 成功
- frontend Vitest: 3 test files、9 tests成功
- `tsc --noEmit`: 成功
- Vite production build: 成功
- Wails Windows/amd64 production build: 成功
- `go vet ./...`: 成功

## Windows実機E2E

GUIから次の縦断操作を確認した。

1. Azure CLIの既存ログイン状態を再利用し、tenantとsubscriptionを選択した。
2. Azure Model Resourceを含むresource groupだけが表示された。
3. 既存Azure Model ResourceとModel Deploymentを選択した。
4. 初回SyncでWindowsのUAC確認画面を承認した。PowerShell、Windows Terminal、cmdの補助コンソールは表示されなかった。
5. opencodexのTask Scheduler service登録、Provider登録、PrimaryKey登録、selected model、custom model、Codex Catalog同期、Responses endpoint接続テストが成功した。
6. Bridge開発プロセス終了後もopencodex serviceが稼働し、`ocx status --json`でservice installed、viable、health OKを確認した。
7. `ocx models list-custom --json`で`<provider-id>/gpt-5.6-sol`が存在することを確認した。
8. Codex Appを再起動し、`<provider-id>/gpt-5.6-sol`がモデル選択に表示されることを確認した。

## Secret検査

opencodexが保存したPrimaryKeyと同じ値が次の対象に含まれないことを、値自体を出力せずに検査した。

- `%LOCALAPPDATA%\FoundryCodexBridge\settings.json`: 含まれない
- `~/.opencodex/service.log`: 含まれない
- Git追跡ファイル: 含まれない

PrimaryKeyは`ocx account add-key`の標準入力だけへ渡し、Bridgeの引数、設定、画面、エラーへ保存しない。

## 未検証

- Windows 10での実機E2E
- Azure CLI未ログイン状態からGUIで開始する`az login`
- UACキャンセル時の実機表示
- local authenticationが無効なAzure Model Resourceでの実機エラー表示
