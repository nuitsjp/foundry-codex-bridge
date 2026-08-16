# Windowsではopencodex serviceコマンドをBridgeから昇格起動する

## Context

WailsのGUIプロセスからWindows版の`ocx.cmd`を実行すると、`cmd.exe`またはPowerShellのコンソールが表示される。さらに、非昇格の`ocx service install`にTask Scheduler登録を委ねた実機検証では、opencodex内部の昇格処理が終了コード199で失敗し、UAC確認画面も表示されなかった。

BridgeはTask Schedulerの登録方法を再実装せず、opencodexの公開CLI契約を維持する必要がある。一方、GUIから開始した操作では、管理者承認を利用者へ確実に提示し、補助コンソールを表示しない必要がある。

## Decision

Windowsでは、Bridgeが公開`ocx service install`または`ocx service repair`を`ShellExecuteExW`の`runas` verbで昇格起動し、終了を待つ。起動対象はPATHまたはBridge管理ディレクトリから解決したNode.jsと、npm packageが`bin.opencodex`および`bin.ocx`として公開する`bin/ocx.mjs`とし、Windowsのcommand shimである`ocx.cmd`を介さない。起動後のプロセスは非表示にするが、WindowsのUAC確認画面は表示する。

Bridgeが担当するのは公開CLIプロセスの昇格起動と終了コードの取得だけである。Task Schedulerのタスク定義、登録、起動、安全確認、競合判定、ロールバックは`ocx service`に委ねる。Bridgeは`Schtasks.exe`、Task Scheduler API、opencodex設定ファイル、Codex設定ファイルを直接操作しない。

その他のAzure CLI、Node.js、npm、ocxコマンドにはWindowsの`CREATE_NO_WINDOW`を指定し、GUI操作からコンソールを生成しない。UACがキャンセルされた場合は、秘密情報を含まない操作可能なエラーをGUIへ表示する。

## Consequences

初回Syncとservice修復ではWindowsの管理者承認が表示される。BridgeはWindows固有の昇格プロセス境界を持つが、opencodexのサービス管理方式には依存せず、公開`ocx service`の作法を維持する。昇格プロセスの標準出力と標準エラーは取得できないため、Bridgeは起動失敗、UACキャンセル、終了コードを区別して扱う。
