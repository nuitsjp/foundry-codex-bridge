import { useEffect, useMemo, useState } from "react";
import {
  api,
  type ActionResult,
  type Deployment,
  type DeployableModel,
  type DisconnectPreview,
  type ManagedProvider,
  type ModelResource,
  type OpenCodexState,
  type ResourceGroup,
  type Selection,
  type Subscription,
  type SyncPreview,
  type SyncRequest,
  type SyncResult,
  type SyncStage,
  type Tenant,
} from "./api";
import ActivityStatus from "./ActivityStatus";

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

function selectExisting<T>(values: T[], preferred: string | undefined, key: (value: T) => string): string {
  if (preferred && values.some((value) => key(value) === preferred)) return preferred;
  return values.length > 0 ? key(values[0]) : "";
}

function selectInitialTenant(values: Tenant[], savedTenantId: string, authTenantId: string): string {
  return selectExisting(values, savedTenantId || authTenantId, (value) => value.id);
}

function selectDeploymentNames(values: Deployment[], preferred: string[] | undefined): string[] {
  const available = new Set(values.map((value) => value.name));
  const restored = (preferred ?? []).filter((name, index, names) => name && available.has(name) && names.indexOf(name) === index);
  return restored.length > 0 ? restored : values.length > 0 ? [values[0].name] : [];
}

function chooseDefaultDeployment(names: string[], preferred: string | undefined): string {
  return preferred && names.includes(preferred) ? preferred : names[0] ?? "";
}

function syncRequestKey(request: SyncRequest): string {
  return JSON.stringify({
    tenantId: request.tenantId,
    subscriptionId: request.subscriptionId,
    resourceGroup: request.resourceGroup,
    resourceName: request.resourceName,
    deploymentNames: request.deploymentNames,
    defaultDeploymentName: request.defaultDeploymentName,
    providerId: request.providerId,
  });
}

function ActionResultView({ result }: { result: ActionResult }) {
  return <div className={`action-result ${result.ok ? "success" : "failure"}`}><h3>{result.ok ? "操作完了" : "操作は未完了"}</h3>{result.stages.map((stage, index) => <StageRow key={`${stage.name}-${index}`} stage={stage} />)}</div>;
}

function StageRow({ stage }: { stage: SyncStage }) {
  const succeeded = stage.status === "succeeded";
  const pending = stage.status === "pending";
  return <div className="stage"><span className={stage.status}>{succeeded ? "✓" : pending ? "·" : "!"}</span><b>{stage.name}</b><span>{stage.message}</span></div>;
}

function ManagedProviderRow({ provider, onDisconnect }: { provider: ManagedProvider; onDisconnect: (provider: ManagedProvider) => void }) {
  return <div className="managed-row"><div className="managed-details"><div className="managed-heading"><b>{provider.resourceName || provider.resourceId}</b><span className="provider-tag">{provider.providerId}</span></div><div className="managed-meta"><span>Subscription: {provider.subscriptionId || "不明"}</span><span>Region: {provider.location || "不明"}</span><span>Resource group: {provider.resourceGroup || "不明"}</span></div><div className="managed-deployments"><span>Deployments: {provider.deploymentNames.length > 0 ? provider.deploymentNames.join(", ") : "なし"}</span><span>既定: {provider.defaultDeploymentName || "なし"}</span></div></div><button className="secondary compact" onClick={() => onDisconnect(provider)}>Disconnectを確認</button></div>;
}

export default function App() {
  const [tab, setTab] = useState<Tab>("connect");
  const [initialized, setInitialized] = useState(false);
  const [initializationFailed, setInitializationFailed] = useState(false);
  const [auth, setAuth] = useState({ cliInstalled: false, cliVersion: "", signedIn: false, username: "", tenantId: "", message: "" });
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [groups, setGroups] = useState<ResourceGroup[]>([]);
  const [resources, setResources] = useState<ModelResource[]>([]);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [models, setModels] = useState<DeployableModel[]>([]);
  const [managedProviders, setManagedProviders] = useState<ManagedProvider[]>([]);
  const [openCodex, setOpenCodex] = useState<OpenCodexState>(emptyOpenCodex);
  const [tenantId, setTenantId] = useState("");
  const [subscriptionId, setSubscriptionId] = useState("");
  const [resourceGroup, setResourceGroup] = useState("");
  const [resourceName, setResourceName] = useState("");
  const [deploymentNames, setDeploymentNames] = useState<string[]>([]);
  const [defaultDeploymentName, setDefaultDeploymentName] = useState("");
  const [providerId, setProviderId] = useState("");
  const [confirmCosts, setConfirmCosts] = useState(false);
  const [syncPreview, setSyncPreview] = useState<SyncPreview | null>(null);
  const [syncPreviewKey, setSyncPreviewKey] = useState("");
  const [syncResult, setSyncResult] = useState<SyncResult | null>(null);
  const [opencodexAction, setOpencodexAction] = useState<ActionResult | null>(null);
  const [catalogAction, setCatalogAction] = useState<ActionResult | null>(null);
  const [disconnectAction, setDisconnectAction] = useState<ActionResult | null>(null);
  const [disconnectTarget, setDisconnectTarget] = useState<ManagedProvider | null>(null);
  const [disconnectPreview, setDisconnectPreview] = useState<DisconnectPreview | null>(null);
  const [replacementProviderId, setReplacementProviderId] = useState("");
  const [confirmDisconnect, setConfirmDisconnect] = useState(false);
  const [confirmUpdate, setConfirmUpdate] = useState(false);
  const [confirmRestart, setConfirmRestart] = useState(false);
  const [port, setPort] = useState("");
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");

  const selectedResource = useMemo(() => resources.find((item) => item.name === resourceName), [resources, resourceName]);
  const selectedManagedProvider = useMemo(() => managedProviders.find((item) => item.resourceId === selectedResource?.id), [managedProviders, selectedResource]);
  const providerIdInput = providerId || selectedManagedProvider?.providerId || "";
  const currentSyncRequest = useMemo<SyncRequest>(() => ({ tenantId, subscriptionId, resourceGroup, resourceName, deploymentNames, defaultDeploymentName, providerId: providerIdInput, confirmCosts }), [tenantId, subscriptionId, resourceGroup, resourceName, deploymentNames, defaultDeploymentName, providerIdInput, confirmCosts]);
  const currentSyncKey = syncRequestKey(currentSyncRequest);
  const candidateModels = models.filter((model) => model.codexCandidate);
  const portNumber = Number(port);
  const validPort = Number.isInteger(portNumber) && portNumber >= 1 && portNumber <= 65535;
  const canPreview = auth.signedIn && Boolean(resourceName) && deploymentNames.length > 0 && Boolean(defaultDeploymentName) && openCodex.installed && !busy;
  const canSync = canPreview && Boolean(confirmCosts) && Boolean(syncPreview?.ok) && syncPreviewKey === currentSyncKey;
  const disconnectManaged = disconnectPreview?.managed ?? disconnectTarget;
  const canDisconnect = Boolean(disconnectPreview?.ok) && (!disconnectPreview?.isDefault || Boolean(replacementProviderId)) && confirmDisconnect && !busy;

  useEffect(() => {
    void run("初期状態を読み込み中", async () => {
      const snapshot = await api.snapshot();
      setAuth(snapshot.auth);
      applyOpenCodexState(snapshot.openCodex);
      if (snapshot.auth.signedIn) await loadHierarchy(snapshot.selection, snapshot.auth.tenantId);
      await loadManagedProviders();
    }).then((succeeded) => {
      setInitializationFailed(!succeeded);
      setInitialized(true);
    });
  }, []);

  async function run(label: string, action: () => Promise<void>) {
    setBusy(label);
    setError("");
    try { await action(); return true; } catch (cause) { setError(cause instanceof Error ? cause.message : "処理に失敗しました"); return false; } finally { setBusy(""); }
  }

  function applyOpenCodexState(value: OpenCodexState) {
    setOpenCodex(value);
    if (value.health.port > 0) setPort(String(value.health.port));
  }

  async function loadManagedProviders() { setManagedProviders(await api.managedProviders()); }

  async function loadResourceDetailsFor(tenant: string, subscription: string, group: string, resource: string, preferredDeployments?: string[], preferredDefaultDeployment?: string) {
    if (!resource) { setDeployments([]); setModels([]); setDeploymentNames([]); setDefaultDeploymentName(""); return { deploymentNames: [], defaultDeploymentName: "" }; }
    const [deploymentValues, modelValues] = await Promise.all([api.deployments(tenant, subscription, group, resource), api.models(tenant, subscription, group, resource)]);
    setDeployments(deploymentValues);
    setModels(modelValues);
    const selected = selectDeploymentNames(deploymentValues, preferredDeployments);
    const defaultDeployment = chooseDefaultDeployment(selected, preferredDefaultDeployment);
    setDeploymentNames(selected);
    setDefaultDeploymentName(defaultDeployment);
    return { deploymentNames: selected, defaultDeploymentName: defaultDeployment };
  }

  async function loadResourcesFor(tenant: string, subscription: string, group: string, preferredResource?: string, preferredDeployments?: string[], preferredDefaultDeployment?: string) {
    if (!group) { setResources([]); setResourceName(""); await loadResourceDetailsFor(tenant, subscription, group, ""); return { resource: "", deploymentNames: [], defaultDeploymentName: "" }; }
    const values = await api.resources(tenant, subscription, group);
    setResources(values);
    const resource = selectExisting(values, preferredResource, (value) => value.name);
    setResourceName(resource);
    const details = await loadResourceDetailsFor(tenant, subscription, group, resource, preferredDeployments, preferredDefaultDeployment);
    return { resource, ...details };
  }

  async function loadGroupsFor(tenant: string, subscription: string, preferredGroup?: string, preferredResource?: string, preferredDeployments?: string[], preferredDefaultDeployment?: string) {
    if (!subscription) { setGroups([]); setResourceGroup(""); return loadResourcesFor(tenant, subscription, ""); }
    const values = await api.resourceGroups(tenant, subscription);
    setGroups(values);
    const group = selectExisting(values, preferredGroup, (value) => value.name);
    setResourceGroup(group);
    return loadResourcesFor(tenant, subscription, group, preferredResource, preferredDeployments, preferredDefaultDeployment);
  }

  async function loadSubscriptionsFor(tenant: string, preferredSubscription?: string, preferredGroup?: string, preferredResource?: string, preferredDeployments?: string[], preferredDefaultDeployment?: string) {
    if (!tenant) { setSubscriptions([]); setSubscriptionId(""); return loadGroupsFor(tenant, ""); }
    const values = await api.subscriptions(tenant);
    setSubscriptions(values);
    const subscription = selectExisting(values, preferredSubscription, (value) => value.id);
    setSubscriptionId(subscription);
    return loadGroupsFor(tenant, subscription, preferredGroup, preferredResource, preferredDeployments, preferredDefaultDeployment);
  }

  async function loadHierarchy(preferred?: Partial<Selection>, authTenantId = auth.tenantId) {
    const values = await api.tenants();
    setTenants(values);
    const tenant = selectInitialTenant(values, preferred?.tenantId ?? "", authTenantId);
    setTenantId(tenant);
    const selected = await loadSubscriptionsFor(tenant, preferred?.subscriptionId, preferred?.resourceGroup, preferred?.resourceName, preferred?.deploymentNames, preferred?.defaultDeploymentName);
    setProviderId(selected.resource === preferred?.resourceName ? preferred?.providerId ?? "" : "");
    invalidateSync(false);
  }

  function invalidateSync(clearProvider: boolean) {
    if (clearProvider) setProviderId("");
    setConfirmCosts(false);
    setSyncPreview(null);
    setSyncPreviewKey("");
    setSyncResult(null);
  }

  async function signIn() {
    await run("Azure CLIのサインインを待機中", async () => { const value = await api.signIn(); setAuth(value); await loadHierarchy(undefined, value.tenantId); await loadManagedProviders(); });
  }

  async function selectTenant(value: string) { await run("Subscriptionを読み込み中", async () => { setTenantId(value); invalidateSync(true); await loadSubscriptionsFor(value); }); }
  async function selectSubscription(value: string) { await run("Resource groupを読み込み中", async () => { setSubscriptionId(value); invalidateSync(true); await loadGroupsFor(tenantId, value); }); }
  async function selectGroup(value: string) { await run("Azure Model Resourceを読み込み中", async () => { setResourceGroup(value); invalidateSync(true); await loadResourcesFor(tenantId, subscriptionId, value); }); }
  async function selectResource(value: string) { await run("Deploymentを読み込み中", async () => { setResourceName(value); invalidateSync(true); await loadResourceDetailsFor(tenantId, subscriptionId, resourceGroup, value); }); }

  function updateDeploymentSelection(next: string[]) {
    const available = new Set(deployments.map((item) => item.name));
    const selected = next.filter((name, index, names) => available.has(name) && names.indexOf(name) === index);
    setDeploymentNames(selected);
    setDefaultDeploymentName((current) => selected.includes(current) ? current : selected[0] ?? "");
    invalidateSync(false);
  }

  function toggleDeployment(name: string, checked: boolean) { updateDeploymentSelection(checked ? [...deploymentNames, name] : deploymentNames.filter((item) => item !== name)); }
  function selectDefaultDeployment(name: string) { if (deploymentNames.includes(name)) { setDefaultDeploymentName(name); invalidateSync(false); } }
  function changeProviderId(value: string) { setProviderId(value); invalidateSync(false); }

  async function prepareOpenCodex() { await run("opencodexを準備中", async () => applyOpenCodexState(await api.prepareOpenCodex())); }
  async function refreshOpenCodex() { await run("opencodexの状態を確認中", async () => applyOpenCodexState(await api.openCodexState())); }
  async function refreshAfterAction() { applyOpenCodexState(await api.openCodexState()); }

  async function openCodexAction(label: string, action: () => Promise<ActionResult>) {
    await run(label, async () => { const result = await action(); setOpencodexAction(result); await refreshAfterAction(); });
  }

  async function previewSync() {
    const request = currentSyncRequest;
    setSyncPreview(null);
    setSyncPreviewKey("");
    await run("Sync差分を確認中", async () => { const preview = await api.previewSync(request); setSyncPreview(preview); setSyncPreviewKey(syncRequestKey(request)); });
  }

  async function sync() {
    if (!canSync) return;
    await run("Syncを実行中", async () => { const result = await api.sync(currentSyncRequest); setSyncResult(result); if (result.providerId) setProviderId(result.providerId); setSyncPreview(null); setSyncPreviewKey(""); setConfirmCosts(false); await loadManagedProviders(); });
  }

  async function restartCodexCatalog() {
    if (!confirmRestart) return;
    await run("Codexカタログを再同期中", async () => { const result = await api.restartCodexCatalog(); setCatalogAction(result); setConfirmRestart(false); await refreshAfterAction(); });
  }

  async function openDisconnect(provider: ManagedProvider) {
    setDisconnectTarget(provider);
    setDisconnectPreview(null);
    setReplacementProviderId("");
    setConfirmDisconnect(false);
    setDisconnectAction(null);
    await run("Disconnectの確認を準備中", async () => setDisconnectPreview(await api.previewDisconnect(provider.resourceId)));
  }

  function closeDisconnect() { setDisconnectTarget(null); setDisconnectPreview(null); setReplacementProviderId(""); setConfirmDisconnect(false); }

  async function disconnect() {
    if (!disconnectTarget || !disconnectPreview || !canDisconnect) return;
    await run("ProviderをDisconnect中", async () => { const result = await api.disconnect({ resourceId: disconnectTarget.resourceId, replacementProviderId }); setDisconnectAction(result); await refreshAfterAction(); await loadManagedProviders(); });
  }

  return <main className="shell">
    <header className="topbar"><div><p className="eyebrow">WINDOWS DESKTOP TOOL</p><h1>FoundryCodex Bridge</h1><p className="subtitle">Azure Model Deploymentをopencodex経由でCodexへ接続します。</p></div><div className={`status-pill auth-status ${!initialized ? "pending" : auth.signedIn ? "ready" : "warning"}`} aria-hidden={!initialized}><span className="status-dot" />{initialized ? initializationFailed ? "Azure CLI状態取得失敗" : auth.signedIn ? `Azure CLI: ${auth.username}` : auth.cliInstalled ? "Azure CLI未接続" : "Azure CLI未検出" : "Azure CLI"}</div></header>
    <nav className="tabs" aria-label="主要画面">{(["connect", "deployments", "opencodex", "sync"] as Tab[]).map((item) => <button className={tab === item ? "tab active" : "tab"} key={item} onClick={() => setTab(item)}>{item === "connect" ? "Connect" : item === "deployments" ? "Deployments" : item === "opencodex" ? "opencodex" : "Sync"}</button>)}</nav>
    <section className="content">
      {error && <div className="notice error" role="alert">{error}</div>}
      {tab === "connect" && <section className="panel"><div className="panel-heading"><div><p className="eyebrow">01 / CONNECT</p><h2>Azureへ接続</h2></div><span className={`state-label ${!initialized || initializationFailed ? "pending" : ""}`} aria-hidden={!initialized || initializationFailed}>{auth.signedIn ? "READY" : "REQUIRED"}</span></div><p className="lead">Azure CLIのログイン済み資格情報を使用します。Bridge独自のEntraアプリを利用者のTenantへ追加しません。</p><div className="auth-actions">{initialized && !initializationFailed && <>{!auth.cliInstalled && <div className="notice warning">Azure CLIをインストールしてからBridgeを再起動してください。</div>}<button className={auth.signedIn ? "secondary" : "primary"} onClick={signIn} disabled={!auth.cliInstalled}>{auth.signedIn ? "Azure CLIでアカウントを変更" : "Azure CLIでサインイン"}</button>{auth.cliInstalled && <p className="hint">Azure CLI {auth.cliVersion || "version不明"}。Bridgeは共有セッションからサインアウトしません。</p>}</>}</div><div className="field-grid"><label>Tenant<select value={tenantId} onChange={(event) => void selectTenant(event.target.value)} disabled={!auth.signedIn}><option value="">選択してください</option>{tenants.map((item) => <option key={item.id} value={item.id}>{item.displayName || item.id}</option>)}</select></label><label>Subscription<select value={subscriptionId} onChange={(event) => void selectSubscription(event.target.value)} disabled={!tenantId}><option value="">選択してください</option>{subscriptions.map((item) => <option key={item.id} value={item.id}>{item.name || item.id}</option>)}</select></label><label>Resource group<select value={resourceGroup} onChange={(event) => void selectGroup(event.target.value)} disabled={!subscriptionId}><option value="">選択してください</option>{groups.map((item) => <option key={item.name} value={item.name}>{item.name}</option>)}</select></label></div>{initialized && !initializationFailed && subscriptionId && !busy && !error && groups.length === 0 && <p className="hint">このSubscriptionにはAzure Model Resourceがありません。</p>}</section>}
      {tab === "deployments" && <section className="panel"><div className="panel-heading"><div><p className="eyebrow">02 / DEPLOYMENTS</p><h2>Model ResourceとDeployment</h2></div><span className="state-label">READ ONLY</span></div><label>Azure Model Resource<select value={resourceName} onChange={(event) => void selectResource(event.target.value)} disabled={!resourceGroup}><option value="">選択してください</option>{resources.map((item) => <option key={item.id} value={item.name}>{item.name} · {item.kind}</option>)}</select></label>{selectedResource?.disableLocalAuth && <div className="notice warning">このリソースはlocal authenticationが無効です。deploymentの閲覧はできますが、API keyを使うopencodex Syncは実行できません。</div>}<div className="split-grid"><div><h3>既存Deployment</h3>{deployments.length === 0 ? <p className="muted">Deploymentがありません。</p> : <div className="deployment-list">{deployments.map((item) => { const selected = deploymentNames.includes(item.name); return <div className={selected ? "deployment-row selected" : "deployment-row"} key={item.id}><label className="deployment-target"><input type="checkbox" aria-label={`${item.name}を公開対象にする`} checked={selected} onChange={(event) => toggleDeployment(item.name, event.target.checked)} /><span><b>{item.name}</b><small>{item.modelName} · {item.modelVersion || "version未指定"} · {item.provisioningState || "状態不明"}</small></span></label><label className="deployment-default"><input type="radio" name="default-deployment" aria-label={`${item.name}を既定にする`} checked={defaultDeploymentName === item.name} disabled={!selected} onChange={() => selectDefaultDeployment(item.name)} /><span>既定</span></label></div>; })}</div>}<p className="hint">チェックしたDeploymentだけをopencodexへ公開します。既定は選択対象の中から1件指定します。</p></div><div><h3>Codex候補</h3>{candidateModels.length === 0 ? <p className="muted">capabilities.responses または capabilities.agentsV2 を満たすモデルがありません。</p> : <div className="card-list">{candidateModels.map((item) => <div className="list-card" key={`${item.name}-${item.version}`}><span>{item.name}</span><small>{item.format} · {item.version || "version未指定"}</small></div>)}</div>}<p className="hint">候補判定はAzureの能力値に基づくヒントであり、利用可否はSync後の接続テストで確認します。</p></div></div></section>}
      {tab === "opencodex" && <section className="panel"><div className="panel-heading"><div><p className="eyebrow">03 / OPEN CODEX</p><h2>opencodexの管理</h2></div><span className={`state-label ${openCodex.health.ready ? "good" : ""}`}>{openCodex.health.ready ? "READY" : "NOT READY"}</span></div><div className="prereq-grid"><div className={openCodex.compatible ? "prereq good" : "prereq bad"}><b>Node.js</b><span>{openCodex.nodeInstalled ? openCodex.nodeVersion : "未検出"}</span></div><div className={openCodex.npmInstalled ? "prereq good" : "prereq bad"}><b>npm</b><span>{openCodex.npmInstalled ? openCodex.npmVersion : "未検出"}</span></div><div className={openCodex.installed ? "prereq good" : "prereq bad"}><b>opencodex</b><span>{openCodex.installed ? openCodex.path : "未導入"}</span></div><div className={openCodex.health.ready ? "prereq good" : "prereq bad"}><b>Service</b><span>{openCodex.health.ready ? `port ${openCodex.health.port}` : "停止中"}</span></div></div>{!openCodex.compatible ? <div className="notice warning">Node.js {openCodex.requiredNode} とnpmを先に導入してください。BridgeはNode.jsを自動導入しません。</div> : !openCodex.installed ? <button className="primary" onClick={prepareOpenCodex}>利用者の承認でopencodexを導入</button> : <div className="action-grid"><button className="secondary" disabled={Boolean(busy)} onClick={() => void openCodexAction("opencodexを起動中", api.startOpenCodex)}>起動</button><button className="secondary" disabled={Boolean(busy)} onClick={() => void openCodexAction("opencodexを停止中", api.stopOpenCodex)}>停止</button><button className="secondary" disabled={Boolean(busy)} onClick={refreshOpenCodex}>状態を再確認</button><button className="secondary" disabled={Boolean(busy)} onClick={() => void openCodexAction("opencodexを修復中", api.repairOpenCodex)}>修復</button></div>}{openCodex.installed && <><div className="management-row"><label>Service port<input type="number" min="1" max="65535" value={port} onChange={(event) => setPort(event.target.value)} /></label><button className="secondary" disabled={!validPort || Boolean(busy)} onClick={() => void openCodexAction("opencodexのポートを変更中", () => api.changeOpenCodexPort(portNumber))}>ポートを変更</button></div><label className="check"><input type="checkbox" checked={confirmUpdate} onChange={(event) => setConfirmUpdate(event.target.checked)} /> opencodexを既定のlatest版へ更新することを確認しました。</label><button className="secondary" disabled={!confirmUpdate || Boolean(busy)} onClick={() => void openCodexAction("opencodexをlatestへ更新中", async () => { const result = await api.updateOpenCodex(); setConfirmUpdate(false); return result; })}>latestへ更新</button></>}{<p className="hint">Bridgeはopencodexの設定ファイルを直接編集せず、公開ocx CLIだけを使用します。Node.jsがない場合は、利用者が先に導入してください。</p>}{opencodexAction && <ActionResultView result={opencodexAction} />}</section>}
      {tab === "sync" && <section className="panel"><div className="panel-heading"><div><p className="eyebrow">04 / SYNC</p><h2>Codexへ反映</h2></div><span className="state-label">EXPLICIT ACTION</span></div><div className="summary"><span>Resource</span><b>{resourceName || "未選択"}</b><span>Provider</span><b>{providerIdInput || "未登録"}</b><span>Deployments</span><b>{deploymentNames.length > 0 ? deploymentNames.join(", ") : "未選択"}</b><span>既定Deployment</span><b>{defaultDeploymentName || "未選択"}</b></div><label>Provider ID<input value={providerIdInput} onChange={(event) => changeProviderId(event.target.value)} disabled={Boolean(selectedManagedProvider)} placeholder={resourceName ? `az-${resourceName}` : "az-resource-name"} /></label>{selectedManagedProvider ? <p className="hint locked-hint">このResourceはすでにBridge管理Providerへ登録されています。Provider IDはDisconnectまで変更できません。</p> : <><p className="hint">未登録ResourceのProvider IDだけ編集できます。</p><p className="hint">opencodex serviceが未登録の場合、初回のSyncでWindowsの管理者承認が表示されます。</p></>}<div className="preview-actions"><button className="secondary" disabled={!canPreview} onClick={() => void previewSync()}>差分を確認</button><span className="preview-state">{syncPreview ? syncPreview.ok ? "差分確認済み" : "差分確認で問題があります" : "Sync前に差分確認が必要です"}</span></div>{syncPreview && <div className={`preview-result ${syncPreview.ok ? "success" : "failure"}`}><h3>{syncPreview.ok ? "予定操作" : "Syncできません"}</h3>{syncPreview.message && <p className="preview-message">{syncPreview.message}</p>}{syncPreview.changes.length > 0 ? syncPreview.changes.map((change, index) => <div className="stage" key={`${change.area}-${index}`}><span className="planned">•</span><b>{change.area}</b><span>{change.action}: {change.details}</span></div>) : <p className="muted">変更予定はありません。</p>}</div>}<label className="check"><input type="checkbox" checked={confirmCosts} onChange={(event) => setConfirmCosts(event.target.checked)} /> Syncで実際のResponsesリクエストを送り、Azure料金が発生し得ることを確認しました。</label><button className="primary wide" disabled={!canSync} onClick={() => void sync()}>Syncを明示実行</button>{syncResult && <div className={`sync-result ${syncResult.ok ? "success" : "failure"}`}><h3>{syncResult.ok ? "Sync完了" : "Syncは未完了"}</h3>{syncResult.stages.map((stage, index) => <StageRow key={`${stage.name}-${index}`} stage={stage} />)}{syncResult.connections.length > 0 && <div className="connections"><h3>Deployment別接続テスト</h3>{syncResult.connections.map((connection) => <div className="stage" key={connection.deployment}><span className={connection.status === "succeeded" ? "succeeded" : "failed"}>{connection.status === "succeeded" ? "✓" : "!"}</span><b>{connection.deployment}</b><span>{connection.message}</span></div>)}</div>}</div>}<div className="catalog-action"><h3>Codexカタログ</h3><label className="check"><input type="checkbox" checked={confirmRestart} onChange={(event) => setConfirmRestart(event.target.checked)} /> Codexを再起動し、opencodexのカタログを再同期することを確認しました。</label><button className="secondary" disabled={!confirmRestart || Boolean(busy)} onClick={() => void restartCodexCatalog()}>Codex再起動を伴う再同期</button>{catalogAction && <ActionResultView result={catalogAction} />}</div><div className="managed-section"><div className="section-heading"><div><h3>Bridge管理Provider</h3><p className="hint">Syncで登録したProviderです。DisconnectしてもAzure Model ResourceやDeploymentは削除されません。</p></div></div>{managedProviders.length === 0 ? <p className="muted">Bridge管理Providerはありません。</p> : <div className="managed-list">{managedProviders.map((provider) => <ManagedProviderRow key={provider.resourceId} provider={provider} onDisconnect={(value) => void openDisconnect(value)} />)}</div>}</div>{disconnectTarget && <div className="disconnect-confirm"><div className="section-heading"><div><h3>Disconnectの確認</h3><p className="hint">opencodexからProviderとBridgeの管理情報だけを削除します。Azure資源は削除しません。</p></div><button className="secondary compact" onClick={closeDisconnect}>閉じる</button></div>{disconnectPreview ? <><div className="summary"><span>Resource</span><b>{disconnectManaged?.resourceName || disconnectManaged?.resourceId}</b><span>Provider</span><b>{disconnectManaged?.providerId}</b><span>Subscription</span><b>{disconnectManaged?.subscriptionId || "不明"}</b><span>Region</span><b>{disconnectManaged?.location || "不明"}</b></div>{disconnectPreview.message && <div className={`notice ${disconnectPreview.ok ? "warning" : "error"}`}>{disconnectPreview.message}</div>}{disconnectPreview.dependentCombos.length > 0 && <div className="notice error">ComboがこのProviderを参照しているためDisconnectできません。対象: {disconnectPreview.dependentCombos.join(", ")}</div>}{disconnectPreview.isDefault && <label>置き換える既定Provider<select value={replacementProviderId} onChange={(event) => setReplacementProviderId(event.target.value)} disabled={!disconnectPreview.ok || disconnectPreview.dependentCombos.length > 0}><option value="">選択してください</option>{disconnectPreview.replacementProviders.map((value) => <option key={value} value={value}>{value}</option>)}</select></label>}<label className="check"><input type="checkbox" checked={confirmDisconnect} onChange={(event) => setConfirmDisconnect(event.target.checked)} disabled={!disconnectPreview.ok || disconnectPreview.dependentCombos.length > 0} /> 表示されたResource/Providerを確認し、Azure資源を削除せずにDisconnectすることを確認しました。</label><button className="primary" disabled={!canDisconnect} onClick={() => void disconnect()}>Disconnectを実行</button></> : <p className="muted">確認情報を読み込み中です。</p>}{disconnectAction && <ActionResultView result={disconnectAction} />}</div>}</section>}
    </section>
    <footer><span>起動時は読み取り専用です。Provider、Catalog、service、接続テストは明示操作を実行した場合だけ変更します。</span><ActivityStatus message={busy} /></footer>
  </main>;
}
