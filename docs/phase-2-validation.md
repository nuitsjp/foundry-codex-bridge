# 第2段階 検証記録

## 自動検証

2026-08-16に次を実行し、成功した。

```powershell
go test ./...
go vet ./...
mise run build
```

`mise run build`はGo unit test、`npm ci`、17件のfrontend test、TypeScript検査、Vite production build、Wails Windows/amd64 production buildを含む。

fake `ocx`を使うunit testでは、次を確認した。

- 複数selected modelとcustom modelが差分だけで収束する。
- 対象外Providerのcustom modelを変更しない。
- Provider設定が一致する場合は変更コマンドを実行しない。
- Combo参照中のProviderをDisconnectしない。
- 既定Providerの付け替え成功後にだけ対象Providerを削除する。
- Provider削除失敗時はBridgeの対応表を保持する。
- port変更は設定、service停止、service起動、catalog syncの順に実行する。
- PrimaryKeyをprocess argument、設定、結果へ含めない。

## 実環境の読み取り確認

ユーザー環境に導入済みのopencodex `2.21.0`で、状態を変更しない次の公開CLI契約を確認した。

- `ocx provider list --json`
- `ocx provider show <provider-id> --json`
- `ocx models list-custom --json`
- `ocx models selected <provider-id> --json`
- `ocx combo list --json`

production buildを起動し、Azure CLIの共有ログイン、保存済みtenant、subscription、Azure Model Resourceを読み込んだConnect画面が空白化せず表示されることを確認した。確認後にプロセスを終了した。

## 実環境の受け入れ確認

2026-08-16に、利用者がGUIから次の操作を実行し、すべて正常に完了することを確認した。

- 2件目のBridge-managed Provider追加
- 複数Deploymentの実Responses接続テスト
- ProviderのDisconnect
- opencodex port変更
- opencodex latest更新
- Codex再起動を伴うcatalog sync

確認後のopencodexは`2.21.0`で、serviceはport `10100`でready状態だった。
