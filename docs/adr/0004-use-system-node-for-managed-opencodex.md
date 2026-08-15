# システムの Node.js で opencodex をユーザー単位に導入する

FoundryCodex Bridge は Node.js を同梱、ダウンロード、インストールせず、opencodex が要求するバージョンの Node.js と npm がシステムに存在することを前提とする。要件を満たさない場合は GUI でユーザーに導入を求める。既存 `ocx` があれば利用し、なければシステムの npm を使って `%LOCALAPPDATA%\FoundryCodexBridge\opencodex` 配下へユーザー単位で導入する。これにより Node.js 導入時の環境依存トラブルとグローバル npm 環境への変更を Bridge の責務から除外する。
