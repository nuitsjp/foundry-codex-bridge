import { useEffect, useMemo, useState } from "react";
import { api, type Deployment, type DeployableModel, type ModelResource, type OpenCodexState, type ResourceGroup, type Subscription, type SyncResult, type Tenant } from "./api";

type Tab = "connect" | "deployments" | "opencodex" | "sync";

const emptyOpenCodex: OpenCodexState = {
  nodeInstalled: false,
  nodeVersion: "",
  npmInstalled: false,
  npmVersion: "",
  requiredNode: ">=18",
  compatible: false,
  installed: false,
  path: "",
  health: { ready: false, pid: 0, port: 0 },
  message: "読み込み中",
};

export default function App() {
  const [tab, setTab] = useState<Tab>("connect");
  const [auth, setAuth] = useState({ signedIn: false, username: "", tenantId: "" });
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [groups, setGroups] = useState<ResourceGroup[]>([]);
  const [resources, setResources] = useState<ModelResource[]>([]);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [models, setModels] = useState<DeployableModel[]>([]);
  const [openCodex, setOpenCodex] = useState<OpenCodexState>(emptyOpenCodex);
  const [tenantId, setTenantId] = useState("");
  const [subscriptionId, setSubscriptionId] = useState("");
  const [resourceGroup, setResourceGroup] = useState("");
  const [resourceName, setResourceName] = useState("");
  const [deploymentName, setDeploymentName] = useState("");
  const [providerId, setProviderId] = useState("");
  const [confirmCosts, setConfirmCosts] = useState(false);
  const [syncResult, setSyncResult] = useState<SyncResult | null>(null);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");

  const selectedResource = useMemo(() => resources.find((item) => item.name === resourceName), [resources, resourceName]);

  useEffect(() => {
    void run("初期状態を読み込み中", async () => {
      const snapshot = await api.snapshot();
      setAuth(snapshot.auth);
      setOpenCodex(snapshot.openCodex);
      setTenantId(snapshot.selection.tenantId);
      setSubscriptionId(snapshot.selection.subscriptionId);
      setResourceGroup(snapshot.selection.resourceGroup);
      setResourceName(snapshot.selection.resourceName);
      setDeploymentName(snapshot.selection.deploymentName);
      setProviderId(snapshot.selection.providerId);
      if (snapshot.auth.signedIn) await loadTenants();
    });
  }, []);

  async function run(label: string, action: () => Promise<void>) {
    setBusy(label);
    setError("");
    try {
      await action();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "処理に失敗しました");
    } finally {
      setBusy("");
    }
  }

  async function loadTenants() {
    const values = await api.tenants();
    setTenants(values);
    if (!tenantId && values.length > 0) setTenantId(values[0].id);
  }

  async function signIn() {
    await run("ブラウザ認証を待機中", async () => {
      const value = await api.signIn();
      setAuth(value);
      await loadTenants();
    });
  }

  async function loadSubscriptions(value: string) {
    setTenantId(value);
    setSubscriptionId("");
    setGroups([]);
    setResources([]);
    await run("Subscriptionを読み込み中", async () => {
      const values = await api.subscriptions(value);
      setSubscriptions(values);
      if (values.length > 0) setSubscriptionId(values[0].id);
    });
  }

  async function loadGroups(value: string) {
    setSubscriptionId(value);
    setResourceGroup("");
    setResources([]);
    await run("Resource groupを読み込み中", async () => {
      const values = await api.resourceGroups(value);
      setGroups(values);
      if (values.length > 0) setResourceGroup(values[0].name);
    });
  }

  async function loadResources(value: string) {
    setResourceGroup(value);
    setResourceName("");
    setDeployments([]);
    setModels([]);
    await run("Azure Model Resourceを読み込み中", async () => {
      const values = await api.resources(subscriptionId, value);
      setResources(values);
      if (values.length > 0) setResourceName(values[0].name);
    });
  }

  async function loadResourceDetails(value: string) {
    setResourceName(value);
    setDeploymentName("");
    await run("Deploymentを読み込み中", async () => {
      const [deploymentValues, modelValues] = await Promise.all([
        api.deployments(subscriptionId, resourceGroup, value),
        api.models(subscriptionId, resourceGroup, value),
      ]);
      setDeployments(deploymentValues);
      setModels(modelValues);
      if (deploymentValues.length > 0) setDeploymentName(deploymentValues[0].name);
    });
  }

  async function prepareOpenCodex() {
    await run("opencodexを準備中", async () => setOpenCodex(await api.prepareOpenCodex()));
  }

  async function refreshOpenCodex() {
    await run("opencodexの状態を確認中", async () => setOpenCodex(await api.openCodexState()));
  }

  async function sync() {
    await run("Syncを実行中", async () => {
      const result = await api.sync({ tenantId, subscriptionId, resourceGroup, resourceName, deploymentName, providerId, confirmCosts });
      setSyncResult(result);
      if (result.providerId) setProviderId(result.providerId);
    });
  }

  const candidateModels = models.filter((model) => model.codexCandidate);

  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">WINDOWS DESKTOP TOOL</p>
          <h1>FoundryCodex Bridge</h1>
          <p className="subtitle">Azure Model Deploymentをopencodex経由でCodexへ接続します。</p>
        </div>
        <div className={`status-pill ${auth.signedIn ? "ready" : "warning"}`}>
          <span className="status-dot" />
          {auth.signedIn ? `サインイン済み: ${auth.username}` : "Azure未接続"}
        </div>
      </header>

      <nav className="tabs" aria-label="主要画面">
        {(["connect", "deployments", "opencodex", "sync"] as Tab[]).map((item) => (
          <button className={tab === item ? "tab active" : "tab"} key={item} onClick={() => setTab(item)}>
            {item === "connect" ? "Connect" : item === "deployments" ? "Deployments" : item === "opencodex" ? "opencodex" : "Sync"}
          </button>
        ))}
      </nav>

      <section className="content">
        {busy && <div className="notice loading">{busy}</div>}
        {error && <div className="notice error" role="alert">{error}</div>}

        {tab === "connect" && <section className="panel">
          <div className="panel-heading"><div><p className="eyebrow">01 / CONNECT</p><h2>Azureへ接続</h2></div><span className="state-label">{auth.signedIn ? "READY" : "REQUIRED"}</span></div>
          <p className="lead">Azure CLIは使わず、プロジェクト所有のマルチテナントEntraアプリでブラウザ認証します。</p>
          {!auth.signedIn ? <button className="primary" onClick={signIn}>ブラウザでサインイン</button> : <button className="secondary" onClick={() => void run("サインアウト中", async () => { await api.signOut(); setAuth({ signedIn: false, username: "", tenantId: "" }); })}>サインアウト</button>}
          <div className="field-grid">
            <label>Tenant<select value={tenantId} onChange={(event) => void loadSubscriptions(event.target.value)} disabled={!auth.signedIn}><option value="">選択してください</option>{tenants.map((item) => <option key={item.id} value={item.id}>{item.displayName || item.id}</option>)}</select></label>
            <label>Subscription<select value={subscriptionId} onChange={(event) => void loadGroups(event.target.value)} disabled={!tenantId}><option value="">選択してください</option>{subscriptions.map((item) => <option key={item.id} value={item.id}>{item.name || item.id}</option>)}</select></label>
            <label>Resource group<select value={resourceGroup} onChange={(event) => void loadResources(event.target.value)} disabled={!subscriptionId}><option value="">選択してください</option>{groups.map((item) => <option key={item.name} value={item.name}>{item.name}</option>)}</select></label>
          </div>
        </section>}

        {tab === "deployments" && <section className="panel">
          <div className="panel-heading"><div><p className="eyebrow">02 / DEPLOYMENTS</p><h2>Model ResourceとDeployment</h2></div><span className="state-label">READ ONLY</span></div>
          <label>Azure Model Resource<select value={resourceName} onChange={(event) => void loadResourceDetails(event.target.value)} disabled={!resourceGroup}><option value="">選択してください</option>{resources.map((item) => <option key={item.id} value={item.name}>{item.name} · {item.kind}</option>)}</select></label>
          {selectedResource?.disableLocalAuth && <div className="notice warning">このリソースはlocal authenticationが無効です。deploymentの閲覧はできますが、API keyを使うopencodex Syncは実行できません。</div>}
          <div className="split-grid">
            <div><h3>既存Deployment</h3>{deployments.length === 0 ? <p className="muted">Deploymentがありません。</p> : <div className="card-list">{deployments.map((item) => <button className={deploymentName === item.name ? "list-card selected" : "list-card"} key={item.id} onClick={() => setDeploymentName(item.name)}><span>{item.name}</span><small>{item.modelName} · {item.modelVersion || "version未指定"} · {item.provisioningState || "状態不明"}</small></button>)}</div>}</div>
            <div><h3>Codex候補</h3>{candidateModels.length === 0 ? <p className="muted">capabilities.responses または capabilities.agentsV2 を満たすモデルがありません。</p> : <div className="card-list">{candidateModels.map((item) => <div className="list-card" key={`${item.name}-${item.version}`}><span>{item.name}</span><small>{item.format} · {item.version || "version未指定"}</small></div>)}</div>}<p className="hint">候補判定はAzureの能力値に基づくヒントであり、利用可否はSync後の接続テストで確認します。</p></div>
          </div>
        </section>}

        {tab === "opencodex" && <section className="panel">
          <div className="panel-heading"><div><p className="eyebrow">03 / OPEN CODEX</p><h2>opencodexの準備</h2></div><span className={`state-label ${openCodex.health.ready ? "good" : ""}`}>{openCodex.health.ready ? "READY" : "NOT READY"}</span></div>
          <div className="prereq-grid"><div className={openCodex.compatible ? "prereq good" : "prereq bad"}><b>Node.js</b><span>{openCodex.nodeInstalled ? openCodex.nodeVersion : "未検出"}</span></div><div className={openCodex.npmInstalled ? "prereq good" : "prereq bad"}><b>npm</b><span>{openCodex.npmInstalled ? openCodex.npmVersion : "未検出"}</span></div><div className={openCodex.installed ? "prereq good" : "prereq bad"}><b>opencodex</b><span>{openCodex.installed ? openCodex.path : "未導入"}</span></div><div className={openCodex.health.ready ? "prereq good" : "prereq bad"}><b>Service</b><span>{openCodex.health.ready ? `port ${openCodex.health.port}` : "停止中"}</span></div></div>
          {!openCodex.compatible ? <div className="notice warning">Node.js {openCodex.requiredNode} とnpmを先に導入してください。BridgeはNode.jsを自動導入しません。</div> : !openCodex.installed ? <button className="primary" onClick={prepareOpenCodex}>利用者の承認でopencodexを導入</button> : <button className="secondary" onClick={refreshOpenCodex}>状態を再確認</button>}
          <p className="hint">Bridgeはopencodexの設定ファイルを直接編集せず、公開ocx CLIだけを使用します。</p>
        </section>}

        {tab === "sync" && <section className="panel">
          <div className="panel-heading"><div><p className="eyebrow">04 / SYNC</p><h2>Codexへ反映</h2></div><span className="state-label">EXPLICIT ACTION</span></div>
          <div className="summary"><span>Resource</span><b>{resourceName || "未選択"}</b><span>Deployment</span><b>{deploymentName || "未選択"}</b></div>
          <label>Provider ID<input value={providerId} onChange={(event) => setProviderId(event.target.value)} placeholder={resourceName ? `az-${resourceName}` : "az-resource-name"} /></label>
          <label className="check"><input type="checkbox" checked={confirmCosts} onChange={(event) => setConfirmCosts(event.target.checked)} /> Sync後に実際のResponsesリクエストを送ることと、Azure料金が発生し得ることを確認しました。</label>
          <button className="primary wide" disabled={!auth.signedIn || !resourceName || !deploymentName || !openCodex.installed || !confirmCosts || Boolean(busy)} onClick={sync}>Syncを明示実行</button>
          {syncResult && <div className={`sync-result ${syncResult.ok ? "success" : "failure"}`}><h3>{syncResult.ok ? "Sync完了" : "Syncは未完了"}</h3>{syncResult.stages.map((stage) => <div className="stage" key={stage.name}><span className={stage.status}>{stage.status === "succeeded" ? "✓" : "!"}</span><b>{stage.name}</b><span>{stage.message}</span></div>)}</div>}
        </section>}
      </section>
      <footer><span>起動時は読み取り専用です。Provider、Catalog、service、接続テストはこの画面でSyncを実行した場合だけ変更します。</span>{busy && <span>{busy}</span>}</footer>
    </main>
  );
}
