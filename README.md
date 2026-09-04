# foundry-codex-bridge

Microsoft Foundry / Azure OpenAI の Model Deployment を、ローカルプロキシ opencodex 経由で Codex から使えるようにする Windows デスクトップアプリ。

## 現状

UX を全面的に見直すため、作り直しの準備段階にある。

- `mock/index.html`: 作り直し後の UI の静的モック。ブラウザで開くだけで動き、左上のドロップダウンで全画面・全状態を切り替えられる。次の実装はこのモックを正とする。
- `poc/`: Wails v2 で作った実証実装（第1〜3段階）。動作するが UX は考慮していない。Azure / opencodex とのやり取りの実装詳細を参照する目的で残している。新規実装には流用しない。
- `docs/`, `CONTEXT.md`: 実証実装の時点で書いた設計ドキュメント、ADR、用語集。システム境界と責務分担（Codex 設定の注入は opencodex に委ねる、Azure 認証は Azure CLI のセッションを借りる、secret を保存しない、など）は作り直し後も引き継ぐ。画面構成に関する記述はモックに置き換わる。

作り直しでは Wails v3 を使う。

## モックが表す UX の要点

- 初回起動は「はじめる準備」のチェックリスト 1 画面。Azure サインイン、Azure リソース選択、Node.js / npm、opencodex、バックグラウンドサービスを自動で検出・実行し、利用者の手が要るもの（ブラウザでの `az login` 承認、UAC 承認）だけを求める。この画面は通常時も「設定」として到達できる。
- Azure リソースの選択は、アクセスできる全テナント × 全サブスクリプションを並列に走査し、見つかった Foundry / Azure OpenAI リソースを 1 つのフラット一覧に届いた順で表示する。テナントやサブスクリプションを利用者に掘り下げさせない。
- ホームは接続中リソース配下の全 Model Deployment の一覧。Codex から使わない Deployment も表示し、行ごとの「Codex で使う」トグルで公開対象を選ぶ。
- Azure への操作（Deployment 作成、Capacity / version 変更）は確認の上で即時実行する。Codex への公開の切替と既定モデルの変更だけをまとめて「Codex へ反映」で適用する。反映待ちは画面下部のバナーに出す。
- Codex の再起動は、どの導線からでも必ず確認ダイアログを経る。

## 開発者向けオンボーディング

### モックを見る

`mock/index.html` をブラウザで開く。ビルド不要。

### 実証実装（poc）を動かす

必要なもの: Windows、Azure CLI、Node.js 18 以上と npm、mise。いずれも `PATH` から実行できること。Go と Wails は mise が導入する。

```powershell
cd poc
mise trust
mise run init
mise run doctor
mise run dev
```

テストと production build は `mise run build`。詳細は `docs/development.md`。

### 読む順番

1. `CONTEXT.md` — 用語。Azure Model Resource、Model Deployment、Bridge-managed Provider、Sync の意味を先に揃える。
2. `mock/index.html` — 作り直し後の画面と状態遷移。
3. `docs/architecture.md` と `docs/adr/` — 変えない境界と、その理由。
4. `poc/internal/` — Azure SDK と `ocx` CLI をどう叩いているかの実例。
