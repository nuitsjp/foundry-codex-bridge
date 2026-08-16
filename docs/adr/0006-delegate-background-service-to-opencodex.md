# opencodex 標準の Windows バックグラウンドサービスを使う

> Windowsで公開`ocx service`コマンドを昇格起動する責務は、ADR 0015で変更した。

FoundryCodex Bridge は初回 Sync で `ocx service install` を実行し、opencodex を Windows ログイン時に起動してクラッシュ時に再起動する Task Scheduler バックグラウンドサービスとして常駐させる。Bridge は Task Scheduler、昇格、安全確認、ロールバックを直接実装せず、すべて opencodex の公開 CLI に委ねる。Codex App は Bridge 終了後もプロキシを必要とするため、Codex CLI 起動時だけ働く Codex shim は使用しない。
