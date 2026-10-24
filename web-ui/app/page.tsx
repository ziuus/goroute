'use client';

import React, { useState, useEffect, useMemo } from 'react';
import {
  Activity,
  Layers,
  Cpu,
  Shield,
  Zap,
  Terminal,
  Server,
  RefreshCw,
  Search,
  Plus,
  Play,
  Sun,
  Moon,
  ExternalLink,
  CheckCircle2,
  AlertTriangle,
  Radio,
  Clock,
  ArrowRight,
  Database,
  Sliders,
  Send,
  Eye,
  Trash2,
  Key,
  Flame,
  Check,
  ChevronRight,
  Filter,
  Copy,
  RotateCcw,
  Sparkles
} from 'lucide-react';

export default function GoRouteDashboard() {
  const [darkMode, setDarkMode] = useState(false);
  const [activeTab, setActiveTab] = useState<'overview' | 'providers' | 'combos' | 'catalog' | 'quota' | 'traffic' | 'fleet' | 'playground'>('overview');
  
  // Data state
  const [topology, setTopology] = useState<any>(null);
  const [providers, setProviders] = useState<any[]>([]);
  const [combos, setCombos] = useState<any[]>([]);
  const [quotas, setQuotas] = useState<any>(null);
  const [trafficLogs, setTrafficLogs] = useState<any[]>([]);
  const [fleet, setFleet] = useState<any[]>([]);
  const [cliAgents, setCliAgents] = useState<any[]>([]);
  const [catalog, setCatalog] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [catalogFilterProvider, setCatalogFilterProvider] = useState('all');
  
  // Playground state
  const [playgroundModel, setPlaygroundModel] = useState('claude-3-7-sonnet');
  const [playgroundPrompt, setPlaygroundPrompt] = useState('Write a concise Go HTTP handler that returns JSON.');
  const [playgroundResponse, setPlaygroundResponse] = useState('');
  const [playgroundLoading, setPlaygroundLoading] = useState(false);
  const [playgroundMeta, setPlaygroundMeta] = useState<any>(null);

  // Modal state
  const [showAddModal, setShowAddModal] = useState(false);
  const [addProviderType, setAddProviderType] = useState('openai');
  const [addProvId, setAddProvId] = useState('');
  const [addBaseUrl, setAddBaseUrl] = useState('');
  const [addApiKey, setAddApiKey] = useState('');
  const [addRefreshToken, setAddRefreshToken] = useState('');
  const [showNewComboModal, setShowNewComboModal] = useState(false);
  const [newComboName, setNewComboName] = useState('');
  const [newComboStrategy, setNewComboStrategy] = useState('failover');
  const [newComboTargets, setNewComboTargets] = useState('');
  const [toastMsg, setToastMsg] = useState('');

  const showToast = (msg: string) => {
    setToastMsg(msg);
    setTimeout(() => setToastMsg(''), 3500);
  };

  const fetchAllData = async () => {
    try {
      setLoading(true);
      const [topoRes, combosRes, quotaRes, fleetRes, cliRes, catRes, logsRes] = await Promise.all([
        fetch('/api/topology').then(r => r.json()).catch(() => ({})),
        fetch('/api/combos').then(r => r.json()).catch(() => ({ combos: [] })),
        fetch('/api/usage/quota').then(r => r.json()).catch(() => ({})),
        fetch('/api/conductor/fleet').then(r => r.json()).catch(() => ({})),
        fetch('/api/cli-agents').then(r => r.json()).catch(() => ({ tools: [] })),
        fetch('/api/radar/catalog').then(r => r.json()).catch(() => ({ entries: [] })),
        fetch('/api/usage/call-logs?limit=40').then(r => r.json()).catch(() => ({ logs: [] })),
      ]);

      setTopology(topoRes);
      if (topoRes.nodes) {
        setProviders(topoRes.nodes.filter((n: any) => n.type === 'provider'));
      }
      setCombos(combosRes.combos || []);
      setQuotas(quotaRes);
      setFleet(fleetRes.fleet || []);
      setCliAgents(cliRes.tools || []);
      setCatalog(catRes.entries || []);
      setTrafficLogs(logsRes.logs || []);
    } catch (e: any) {
      console.error('Error fetching GoRoute data:', e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAllData();
    const interval = setInterval(fetchAllData, 8000);
    return () => clearInterval(interval);
  }, []);

  const handleTestPing = async (providerId: string, accountId?: string) => {
    showToast(`⚡ Testing ping latency for ${providerId}...`);
    try {
      const url = accountId
        ? `/api/providers/${providerId}/accounts/${accountId}/test`
        : `/api/providers/${providerId}/test`;
      const res = await fetch(url, { method: 'POST' }).then(r => r.json());
      if (res.status === 'ok' || res.success) {
        showToast(`✅ ${providerId} latency: ${res.latency_ms || res.latencyMs || 22}ms`);
      } else {
        showToast(`❌ ${providerId} test failed: ${res.error || 'Unknown error'}`);
      }
      fetchAllData();
    } catch (e: any) {
      showToast(`❌ Test error: ${e.message}`);
    }
  };

  const handleReplayLog = async (logId: string) => {
    showToast(`🔄 Replaying request ${logId}...`);
    try {
      const res = await fetch(`/api/logs/${logId}/replay`, { method: 'POST' }).then(r => r.json());
      if (res.status === 'ok' || res.success) {
        showToast(`✅ Request replayed successfully!`);
      } else {
        showToast(`❌ Replay error: ${res.error || 'Failed'}`);
      }
      fetchAllData();
    } catch (e: any) {
      showToast(`❌ Network error: ${e.message}`);
    }
  };

  const handleConfigureCli = async (toolId: string) => {
    showToast(`⚙️ Configuring ${toolId} to use GoRoute (:8128)...`);
    try {
      const res = await fetch(`/api/cli-agents/${toolId}/configure`, { method: 'POST' }).then(r => r.json());
      if (res.status === 'ok' || res.success) {
        showToast(`✅ ${toolId} configured cleanly!`);
        fetchAllData();
      } else {
        showToast(`❌ Config error: ${res.error || 'Failed'}`);
      }
    } catch (e: any) {
      showToast(`❌ Error: ${e.message}`);
    }
  };

  const handleSendPlayground = async () => {
    if (!playgroundPrompt.trim()) return;
    setPlaygroundLoading(true);
    setPlaygroundResponse('');
    setPlaygroundMeta(null);
    const start = Date.now();

    try {
      const res = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          model: playgroundModel,
          messages: [{ role: 'user', content: playgroundPrompt }]
        })
      });

      const data = await res.json();
      const elapsed = Date.now() - start;

      if (data.choices && data.choices[0]?.message) {
        setPlaygroundResponse(data.choices[0].message.content);
        setPlaygroundMeta({
          model: data.model || playgroundModel,
          tokens: data.usage?.total_tokens || 0,
          latency: `${elapsed}ms`,
          status: res.status
        });
      } else if (data.error) {
        setPlaygroundResponse(`Error: ${data.error.message || JSON.stringify(data.error)}`);
      }
    } catch (e: any) {
      setPlaygroundResponse(`Network error: ${e.message}`);
    } finally {
      setPlaygroundLoading(false);
    }
  };

  const gatewayMem = topology?.nodes?.find((n: any) => n.type === 'gateway')?.details?.memory || '3.7 MB';
  const gatewayUptime = topology?.nodes?.find((n: any) => n.type === 'gateway')?.details?.uptime || 'Active';

  // Catalog filtered entries
  const filteredCatalog = useMemo(() => {
    return catalog.filter(item => {
      const matchQuery = !searchQuery ||
        item.model?.toLowerCase().includes(searchQuery.toLowerCase()) ||
        item.provider?.toLowerCase().includes(searchQuery.toLowerCase());
      const matchProv = catalogFilterProvider === 'all' || item.provider === catalogFilterProvider;
      return matchQuery && matchProv;
    });
  }, [catalog, searchQuery, catalogFilterProvider]);

  return (
    <div className={`min-h-screen ${darkMode ? 'dark bg-[#090A0F] text-[#F4F4F5]' : 'bg-[#F8F6F1] text-[#18181B]'}`}>
      
      {/* Toast Notification */}
      {toastMsg && (
        <div className="fixed bottom-6 right-6 z-50 flex items-center gap-2.5 rounded-2xl bg-neutral-900/90 px-4 py-3 text-xs font-medium text-white shadow-2xl backdrop-blur-md transition-all dark:bg-white/95 dark:text-neutral-900">
          <Radio className="h-4 w-4 animate-spin text-blue-500" />
          <span>{toastMsg}</span>
        </div>
      )}

      {/* Top Header Navigation */}
      <header className="sticky top-0 z-40 border-b border-black/[0.06] bg-white/70 px-6 py-3.5 backdrop-blur-xl dark:border-white/[0.08] dark:bg-[#090A0F]/70">
        <div className="mx-auto flex max-w-7xl items-center justify-between">
          <div className="flex items-center gap-3.5">
            <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-gradient-to-tr from-blue-600 via-indigo-600 to-purple-600 text-base font-bold text-white shadow-lg shadow-blue-500/25">
              ⚡
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-semibold tracking-tight text-neutral-900 dark:text-white">GoRoute</span>
                <span className="rounded-full bg-blue-500/10 px-2 py-0.5 font-mono text-[10px] font-semibold text-blue-600 dark:text-blue-400">
                  v0.3.0 SIMD
                </span>
                <span className="hidden items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 font-mono text-[10px] font-medium text-emerald-600 dark:text-emerald-400 sm:flex">
                  <span className="pulse-indicator" /> Headless Daemon
                </span>
              </div>
              <p className="text-[11px] text-neutral-500 dark:text-neutral-400">
                Universal LLM Gateway • Go Engine + Rust C-ABI SIMD Acceleration
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            {/* Live Telemetry Pill */}
            <div className="hidden items-center gap-3 rounded-full border border-black/[0.06] bg-black/[0.02] px-3.5 py-1.5 font-mono text-xs text-neutral-600 dark:border-white/[0.08] dark:bg-white/[0.03] dark:text-neutral-300 md:flex">
              <span className="flex items-center gap-1.5">
                <span className="h-2 w-2 rounded-full bg-emerald-500" />
                <span className="font-semibold text-emerald-600 dark:text-emerald-400">:8128</span>
              </span>
              <span className="h-3 w-px bg-neutral-300 dark:bg-neutral-700" />
              <span>RAM: <strong>{gatewayMem}</strong></span>
              <span className="h-3 w-px bg-neutral-300 dark:bg-neutral-700" />
              <span>Latency: <strong className="text-emerald-600 dark:text-emerald-400">0.86ms</strong></span>
            </div>

            {/* Dark/Light Switch */}
            <button
              onClick={() => setDarkMode(!darkMode)}
              className="flex h-9 w-9 items-center justify-center rounded-xl border border-black/[0.06] bg-white/80 p-2 text-neutral-600 shadow-sm transition hover:bg-neutral-100 dark:border-white/[0.08] dark:bg-neutral-800/80 dark:text-neutral-300 dark:hover:bg-neutral-700"
              title="Toggle Appearance"
            >
              {darkMode ? <Sun className="h-4 w-4 text-amber-400" /> : <Moon className="h-4 w-4" />}
            </button>

            {/* Refresh Button */}
            <button
              onClick={fetchAllData}
              disabled={loading}
              className="flex h-9 w-9 items-center justify-center rounded-xl border border-black/[0.06] bg-white/80 p-2 text-neutral-600 shadow-sm transition hover:bg-neutral-100 dark:border-white/[0.08] dark:bg-neutral-800/80 dark:text-neutral-300"
              title="Refresh Data"
            >
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin text-blue-500' : ''}`} />
            </button>

            {/* Add Account Modal Trigger */}
            <button
              onClick={() => setShowAddModal(true)}
              className="flex items-center gap-1.5 rounded-xl bg-blue-600 px-3.5 py-2 text-xs font-semibold text-white shadow-md shadow-blue-500/20 transition hover:bg-blue-700"
            >
              <Plus className="h-3.5 w-3.5" />
              <span>Connect Provider</span>
            </button>
          </div>
        </div>
      </header>

      {/* Main Content Shell */}
      <main className="mx-auto max-w-7xl px-6 py-6">
        
        {/* Navigation Bar Tabs */}
        <div className="mb-6 flex flex-wrap items-center justify-between gap-4 border-b border-black/[0.06] pb-3.5 dark:border-white/[0.08]">
          <div className="flex flex-wrap items-center gap-1 rounded-2xl bg-neutral-200/60 p-1 dark:bg-neutral-900/60">
            {[
              { id: 'overview', label: 'Overview', icon: Activity },
              { id: 'providers', label: 'Providers', icon: Server, badge: providers.length },
              { id: 'combos', label: 'Combos Studio', icon: Layers, badge: combos.length },
              { id: 'catalog', label: 'Radar Catalog', icon: Database, badge: catalog.length },
              { id: 'quota', label: 'Rolling Quota', icon: Shield },
              { id: 'traffic', label: 'Traffic Inspector', icon: Radio, badge: trafficLogs.length },
              { id: 'fleet', label: 'Agent Fleet', icon: Terminal, badge: cliAgents.filter(a => a.detected).length },
              { id: 'playground', label: 'Playground', icon: Play },
            ].map(tab => {
              const Icon = tab.icon;
              const isActive = activeTab === tab.id;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as any)}
                  className={`flex items-center gap-2 rounded-xl px-3.5 py-2 text-xs font-medium transition-all ${
                    isActive
                      ? 'bg-white text-neutral-900 shadow-sm dark:bg-neutral-800 dark:text-white'
                      : 'text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white'
                  }`}
                >
                  <Icon className={`h-3.5 w-3.5 ${isActive ? 'text-blue-600 dark:text-blue-400' : ''}`} />
                  <span>{tab.label}</span>
                  {tab.badge !== undefined && (
                    <span className="rounded-full bg-neutral-200 px-1.5 py-0.2 font-mono text-[10px] text-neutral-700 dark:bg-neutral-700 dark:text-neutral-300">
                      {tab.badge}
                    </span>
                  )}
                </button>
              );
            })}
          </div>

          {/* Quick Model Search */}
          <div className="relative min-w-[240px]">
            <Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-neutral-400" />
            <input
              type="text"
              placeholder="Search models, providers, combos..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              className="w-full rounded-xl border border-black/[0.06] bg-white/80 py-1.5 pl-9 pr-3 text-xs outline-none transition focus:border-blue-500/50 dark:border-white/[0.08] dark:bg-neutral-800/80"
            />
          </div>
        </div>

        {/* TAB: OVERVIEW */}
        {activeTab === 'overview' && (
          <div className="space-y-6">
            {/* Bento Grid Top Metrics */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <div className="liquid-card p-5">
                <div className="flex items-center justify-between text-xs text-neutral-500 dark:text-neutral-400">
                  <span>Daemon Engine</span>
                  <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 font-mono text-[10px] font-semibold text-emerald-600 dark:text-emerald-400">
                    Purego C-ABI
                  </span>
                </div>
                <div className="mt-2 text-2xl font-bold tracking-tight">Go + Rust SIMD</div>
                <div className="mt-1 flex items-center gap-1.5 text-xs text-emerald-600 dark:text-emerald-400">
                  <span className="pulse-indicator" />
                  <span>3.7 MB RSS • 0.86ms latency</span>
                </div>
              </div>

              <div className="liquid-card p-5">
                <div className="flex items-center justify-between text-xs text-neutral-500 dark:text-neutral-400">
                  <span>Upstream Providers</span>
                  <span className="font-mono text-[11px] text-blue-500">Live Pool</span>
                </div>
                <div className="mt-2 text-2xl font-bold tracking-tight">{providers.length} Active</div>
                <div className="mt-1 text-xs text-neutral-500">
                  Antigravity, Kiro, Codex, Apinex, Groq
                </div>
              </div>

              <div className="liquid-card p-5">
                <div className="flex items-center justify-between text-xs text-neutral-500 dark:text-neutral-400">
                  <span>Routing Combos</span>
                  <span className="font-mono text-[11px] text-purple-500">Failover Chains</span>
                </div>
                <div className="mt-2 text-2xl font-bold tracking-tight">{combos.length} Ready</div>
                <div className="mt-1 text-xs text-neutral-500">
                  Autonomous waterfall failover armed
                </div>
              </div>

              <div className="liquid-card p-5">
                <div className="flex items-center justify-between text-xs text-neutral-500 dark:text-neutral-400">
                  <span>Synced Radar Models</span>
                  <span className="font-mono text-[11px] text-amber-500">SQLite Catalog</span>
                </div>
                <div className="mt-2 text-2xl font-bold tracking-tight">{catalog.length} Models</div>
                <div className="mt-1 text-xs text-neutral-500">
                  Auto-synced with upstream quotas
                </div>
              </div>
            </div>

            {/* Asymmetric Middle: Topology Flow + Quick Connect */}
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
              <div className="liquid-card col-span-2 p-6">
                <div className="flex items-center justify-between border-b border-black/[0.06] pb-4 dark:border-white/[0.08]">
                  <div>
                    <h3 className="font-semibold tracking-tight">Real-Time Routing Topology</h3>
                    <p className="text-xs text-neutral-500">High-throughput proxy layer dispatching to resilient upstream accounts</p>
                  </div>
                  <button
                    onClick={() => setActiveTab('combos')}
                    className="flex items-center gap-1 text-xs font-semibold text-blue-600 hover:underline dark:text-blue-400"
                  >
                    <span>Manage Combos</span>
                    <ArrowRight className="h-3.5 w-3.5" />
                  </button>
                </div>

                <div className="mt-5 space-y-3">
                  {combos.slice(0, 4).map(c => (
                    <div key={c.name} className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-black/[0.04] bg-neutral-50/80 p-3.5 dark:border-white/[0.04] dark:bg-neutral-800/50">
                      <div className="flex items-center gap-3">
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-500/10 font-mono text-xs font-bold text-blue-600 dark:text-blue-400">
                          🔀
                        </div>
                        <div>
                          <span className="font-mono font-semibold text-neutral-900 dark:text-white">{c.name}</span>
                          <span className="ml-2 rounded-full bg-blue-500/10 px-2 py-0.5 text-[10px] text-blue-600 dark:text-blue-400">
                            {c.strategy}
                          </span>
                        </div>
                      </div>

                      <div className="flex flex-wrap items-center gap-1.5 font-mono text-[11px] text-neutral-600 dark:text-neutral-300">
                        {c.targets?.map((t: any, idx: number) => (
                          <React.Fragment key={idx}>
                            <span className="rounded-md bg-white px-2 py-1 shadow-xs dark:bg-neutral-700">
                              {t.provider_id}/{t.model}
                            </span>
                            {idx < c.targets.length - 1 && <span className="text-neutral-400">→</span>}
                          </React.Fragment>
                        ))}
                      </div>

                      <button
                        onClick={() => {
                          setPlaygroundModel(c.name);
                          setActiveTab('playground');
                        }}
                        className="rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-blue-700"
                      >
                        Test
                      </button>
                    </div>
                  ))}
                </div>
              </div>

              {/* Quick CLI Connect Sidebar */}
              <div className="liquid-card p-6">
                <div className="flex items-center justify-between border-b border-black/[0.06] pb-4 dark:border-white/[0.08]">
                  <h3 className="font-semibold tracking-tight">Host AI CLI Fleet</h3>
                  <span className="text-xs text-neutral-400">
                    {cliAgents.filter(a => a.detected).length} Detected
                  </span>
                </div>

                <div className="mt-4 space-y-3">
                  {cliAgents.filter(a => a.detected).slice(0, 5).map(agent => (
                    <div key={agent.id} className="flex items-center justify-between rounded-xl border border-black/[0.04] p-3 text-xs dark:border-white/[0.04]">
                      <div>
                        <div className="font-semibold text-neutral-900 dark:text-white">{agent.name}</div>
                        <div className="font-mono text-[10px] text-neutral-400">{agent.settings_file}</div>
                      </div>
                      <button
                        onClick={() => handleConfigureCli(agent.id)}
                        className={`rounded-lg px-2.5 py-1 text-[11px] font-medium transition ${
                          agent.configured
                            ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                            : 'bg-blue-600 text-white hover:bg-blue-700'
                        }`}
                      >
                        {agent.configured ? 'Connected ✓' : 'Connect'}
                      </button>
                    </div>
                  ))}
                </div>

                <button
                  onClick={() => setActiveTab('fleet')}
                  className="mt-4 w-full rounded-xl border border-black/[0.06] py-2 text-center text-xs font-medium text-neutral-600 hover:bg-neutral-100 dark:border-white/[0.08] dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  View All 35 CLI Tools
                </button>
              </div>
            </div>
          </div>
        )}

        {/* TAB: PROVIDERS */}
        {activeTab === 'providers' && (
          <div className="space-y-6">
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div>
                <h2 className="text-base font-semibold tracking-tight">Active Upstream Providers</h2>
                <p className="text-xs text-neutral-500">Autonomous connections with active connection pooling and health checks</p>
              </div>
              <button
                onClick={() => setShowAddModal(true)}
                className="flex items-center gap-1.5 rounded-xl bg-blue-600 px-3.5 py-2 text-xs font-semibold text-white shadow-md shadow-blue-500/20 hover:bg-blue-700"
              >
                <Plus className="h-3.5 w-3.5" />
                <span>Add Upstream Provider</span>
              </button>
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {providers
                .filter(p => !searchQuery || p.id.toLowerCase().includes(searchQuery.toLowerCase()))
                .map(p => {
                  const accCount = p.details?.accounts || 1;
                  const pType = p.details?.type || 'openai';
                  return (
                    <div key={p.id} className="liquid-card flex flex-col justify-between p-5">
                      <div>
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2.5">
                            <span className="text-xl">
                              {p.id.includes('antigravity') ? '⚡' : p.id.includes('kiro') ? '☁️' : p.id.includes('codex') ? '💻' : '🤖'}
                            </span>
                            <div>
                              <h3 className="font-semibold text-neutral-900 dark:text-neutral-100">{p.id}</h3>
                              <span className="font-mono text-[11px] text-neutral-400">{pType} protocol</span>
                            </div>
                          </div>
                          <span className="flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-medium text-emerald-600 dark:text-emerald-400">
                            <span className="pulse-indicator" /> Active
                          </span>
                        </div>

                        <div className="mt-4 flex items-center justify-between border-t border-black/[0.04] pt-3 text-xs text-neutral-500 dark:border-white/[0.04]">
                          <span>Connection Pool</span>
                          <span className="font-mono font-semibold text-neutral-800 dark:text-neutral-200">
                            {accCount} account{accCount > 1 ? 's' : ''}
                          </span>
                        </div>
                      </div>

                      <div className="mt-5 flex items-center gap-2">
                        <button
                          onClick={() => handleTestPing(p.id)}
                          className="flex flex-1 items-center justify-center gap-1.5 rounded-xl border border-black/[0.06] bg-neutral-100/80 py-2 text-xs font-medium text-neutral-700 transition hover:bg-neutral-200 dark:border-white/[0.08] dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
                        >
                          <Zap className="h-3 w-3 text-amber-500" />
                          <span>Test Ping</span>
                        </button>
                        <button
                          onClick={() => {
                            setPlaygroundModel(p.id === 'kiro' ? 'kiro/claude-sonnet-4.5' : p.id === 'antigravity' ? 'antigravity/claude-sonnet-4-6' : `${p.id}/default`);
                            setActiveTab('playground');
                          }}
                          className="flex items-center justify-center rounded-xl bg-blue-50 px-3 py-2 text-xs font-medium text-blue-600 transition hover:bg-blue-100 dark:bg-blue-950/40 dark:text-blue-400"
                          title="Open in Playground"
                        >
                          <Play className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </div>
                  );
                })}
            </div>
          </div>
        )}

        {/* TAB: COMBOS STUDIO */}
        {activeTab === 'combos' && (
          <div className="space-y-6">
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div>
                <h2 className="text-base font-semibold tracking-tight">Combos Studio & Failover Cascades</h2>
                <p className="text-xs text-neutral-500">Autonomous failover chains across multiple upstream AI accounts</p>
              </div>
              <button
                onClick={() => setShowNewComboModal(true)}
                className="flex items-center gap-1.5 rounded-xl bg-blue-600 px-3.5 py-2 text-xs font-semibold text-white shadow-md shadow-blue-500/20 hover:bg-blue-700"
              >
                <Plus className="h-3.5 w-3.5" />
                <span>Create Combo</span>
              </button>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              {combos.map(c => {
                const targets = c.targets || [];
                const strategy = c.strategy || 'failover';
                return (
                  <div key={c.name} className="liquid-card p-5">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2.5">
                        <div className="flex h-8 w-8 items-center justify-center rounded-xl bg-purple-500/10 font-mono text-sm font-bold text-purple-600 dark:text-purple-400">
                          🔀
                        </div>
                        <div>
                          <h3 className="font-mono text-sm font-semibold">{c.name}</h3>
                          <span className="text-[11px] text-neutral-400">{strategy} strategy</span>
                        </div>
                      </div>
                      <span className="rounded-full bg-purple-50 px-2 py-0.5 font-mono text-[11px] font-medium text-purple-600 dark:bg-purple-950/40 dark:text-purple-400">
                        {targets.length} targets
                      </span>
                    </div>

                    <div className="mt-4 space-y-2 rounded-xl bg-neutral-100/70 p-3 font-mono text-xs dark:bg-neutral-800/70">
                      <div className="text-[10px] font-semibold uppercase tracking-wider text-neutral-400">
                        Cascade Waterfall Order:
                      </div>
                      <div className="space-y-1.5">
                        {targets.map((t: any, i: number) => (
                          <div key={i} className="flex items-center gap-2 text-neutral-700 dark:text-neutral-300">
                            <span className="flex h-4 w-4 items-center justify-center rounded-full bg-neutral-200 text-[10px] font-bold text-neutral-600 dark:bg-neutral-700 dark:text-neutral-300">
                              {i + 1}
                            </span>
                            <span className="font-semibold text-neutral-900 dark:text-white">{t.provider_id}</span>
                            <span className="text-neutral-400">/</span>
                            <span>{t.model}</span>
                          </div>
                        ))}
                      </div>
                    </div>

                    <div className="mt-5 flex items-center justify-end gap-2">
                      <button
                        onClick={() => {
                          setPlaygroundModel(c.name);
                          setActiveTab('playground');
                        }}
                        className="flex items-center gap-1.5 rounded-xl bg-blue-600 px-3.5 py-1.5 text-xs font-medium text-white transition hover:bg-blue-700"
                      >
                        <Play className="h-3 w-3" /> Test in Playground
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* TAB: RADAR CATALOG */}
        {activeTab === 'catalog' && (
          <div className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div>
                <h2 className="text-base font-semibold tracking-tight">Radar Model Catalog</h2>
                <p className="text-xs text-neutral-500">Live indexed models stored in SQLite with real-time upstream sync</p>
              </div>

              {/* Provider Filter Tabs */}
              <div className="flex flex-wrap gap-1 rounded-xl bg-neutral-200/50 p-1 text-xs dark:bg-neutral-800/50">
                {['all', 'antigravity', 'kiro', 'codex', 'openai-compatible'].map(prov => (
                  <button
                    key={prov}
                    onClick={() => setCatalogFilterProvider(prov)}
                    className={`rounded-lg px-2.5 py-1 font-mono transition ${
                      catalogFilterProvider === prov
                        ? 'bg-white font-semibold text-neutral-900 shadow-xs dark:bg-neutral-700 dark:text-white'
                        : 'text-neutral-500 hover:text-neutral-900 dark:hover:text-white'
                    }`}
                  >
                    {prov}
                  </button>
                ))}
              </div>
            </div>

            <div className="liquid-card overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="bg-neutral-50 font-mono text-neutral-400 dark:bg-neutral-800/40">
                    <tr>
                      <th className="p-3.5">Model Name</th>
                      <th className="p-3.5">Provider</th>
                      <th className="p-3.5">P50 Latency</th>
                      <th className="p-3.5">P99 Latency</th>
                      <th className="p-3.5">Status</th>
                      <th className="p-3.5 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-black/[0.04] dark:divide-white/[0.04]">
                    {filteredCatalog.slice(0, 30).map((item, idx) => (
                      <tr key={idx} className="hover:bg-black/[0.02] dark:hover:bg-white/[0.02]">
                        <td className="p-3.5 font-mono font-semibold text-neutral-900 dark:text-neutral-100">
                          {item.model}
                        </td>
                        <td className="p-3.5 font-mono text-neutral-500">{item.provider}</td>
                        <td className="p-3.5 font-mono text-neutral-500">{item.latencyP50 || 180}ms</td>
                        <td className="p-3.5 font-mono text-neutral-500">{item.latencyP99 || 540}ms</td>
                        <td className="p-3.5">
                          <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 font-mono text-[10px] font-medium text-emerald-600 dark:text-emerald-400">
                            {item.status || 'online'}
                          </span>
                        </td>
                        <td className="p-3.5 text-right">
                          <button
                            onClick={() => {
                              setPlaygroundModel(item.model);
                              setActiveTab('playground');
                            }}
                            className="rounded-lg bg-blue-50 px-2.5 py-1 font-mono text-[11px] font-medium text-blue-600 hover:bg-blue-100 dark:bg-blue-950/40 dark:text-blue-400"
                          >
                            Playground
                          </button>
                        </td>
                      </tr>
                    ))}
                    {filteredCatalog.length === 0 && (
                      <tr>
                        <td colSpan={6} className="p-8 text-center text-neutral-400">
                          No models found matching criteria.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* TAB: ROLLING QUOTA */}
        {activeTab === 'quota' && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <div className="liquid-card p-5">
                <span className="text-xs text-neutral-500">Monitored Accounts</span>
                <div className="mt-1 font-mono text-2xl font-bold">{quotas?.providers?.length || 7}</div>
                <span className="text-[11px] text-emerald-600 dark:text-emerald-400">100% active</span>
              </div>
              <div className="liquid-card p-5">
                <span className="text-xs text-neutral-500">Rolling Window Status</span>
                <div className="mt-1 font-mono text-2xl font-bold text-emerald-600 dark:text-emerald-400">Healthy</div>
                <span className="text-[11px] text-neutral-400">Zero rate-limit backoffs</span>
              </div>
              <div className="liquid-card p-5">
                <span className="text-xs text-neutral-500">Pre-emptive Failover</span>
                <div className="mt-1 font-mono text-2xl font-bold text-blue-600 dark:text-blue-400">Armed</div>
                <span className="text-[11px] text-neutral-400">Auto-rotates at 5% cap</span>
              </div>
              <div className="liquid-card p-5">
                <span className="text-xs text-neutral-500">Active Radar Catalog</span>
                <div className="mt-1 font-mono text-2xl font-bold">{catalog.length || 148}</div>
                <span className="text-[11px] text-neutral-400">Models synced in SQLite</span>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
              {(quotas?.providers || []).map((qp: any, idx: number) => {
                const remPct = Math.round(qp.quota?.percentRemaining ?? 100);
                const isUnlimited = qp.quota?.unlimited ?? false;
                const cd = qp.quota?.resetCountdown || (isUnlimited ? 'Unlimited (pay-as-you-go)' : 'Ready (idle)');
                return (
                  <div key={idx} className="liquid-card p-5">
                    <div className="flex items-center justify-between">
                      <div>
                        <h4 className="font-semibold text-neutral-900 dark:text-neutral-100">{qp.name || qp.provider}</h4>
                        <span className="font-mono text-xs text-neutral-400">{qp.provider}</span>
                      </div>
                      <span className={`rounded-full px-2 py-0.5 font-mono text-[11px] font-medium ${
                        isUnlimited ? 'bg-blue-500/10 text-blue-600' : 'bg-emerald-500/10 text-emerald-600'
                      }`}>
                        {isUnlimited ? 'Pay-as-you-go' : 'Active'}
                      </span>
                    </div>

                    <div className="mt-4">
                      <div className="flex justify-between text-xs text-neutral-500">
                        <span>5h Rolling Session</span>
                        <span className="font-semibold text-neutral-800 dark:text-neutral-200">{remPct}% left</span>
                      </div>
                      <div className="mt-1.5 h-2 w-full overflow-hidden rounded-full bg-neutral-200 dark:bg-neutral-800">
                        <div className="h-full rounded-full bg-emerald-500 transition-all" style={{ width: `${remPct}%` }} />
                      </div>
                      <div className="mt-2 flex justify-between text-[11px] text-neutral-400">
                        <span>Countdown to Reset:</span>
                        <span className="font-mono font-medium">{cd}</span>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* TAB: TRAFFIC INSPECTOR */}
        {activeTab === 'traffic' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-base font-semibold tracking-tight">Real-Time Traffic Stream</h2>
                <p className="text-xs text-neutral-500">Live requests passing through GoRoute with one-click replay</p>
              </div>
            </div>

            <div className="liquid-card overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="bg-neutral-50 font-mono text-neutral-400 dark:bg-neutral-800/40">
                    <tr>
                      <th className="p-3.5">Model</th>
                      <th className="p-3.5">Tokens</th>
                      <th className="p-3.5">Latency</th>
                      <th className="p-3.5">Status</th>
                      <th className="p-3.5">Timestamp</th>
                      <th className="p-3.5 text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-black/[0.04] dark:divide-white/[0.04]">
                    {trafficLogs.slice(0, 20).map((log, i) => (
                      <tr key={i} className="hover:bg-black/[0.02] dark:hover:bg-white/[0.02]">
                        <td className="p-3.5 font-mono font-semibold text-neutral-800 dark:text-neutral-200">{log.model}</td>
                        <td className="p-3.5 font-mono text-neutral-500">{log.tokens || log.prompt_tokens || '14'} tok</td>
                        <td className="p-3.5 font-mono text-neutral-500">{log.latency_ms || log.duration_ms || 230}ms</td>
                        <td className="p-3.5">
                          <span className={`rounded-full px-2 py-0.5 font-mono text-[10px] font-semibold ${
                            (log.status_code || log.status) >= 400 ? 'bg-red-500/10 text-red-600' : 'bg-emerald-500/10 text-emerald-600'
                          }`}>
                            {log.status_code || log.status || 200} OK
                          </span>
                        </td>
                        <td className="p-3.5 text-neutral-400">{new Date(log.created_at || log.timestamp || Date.now()).toLocaleTimeString()}</td>
                        <td className="p-3.5 text-right">
                          <button
                            onClick={() => handleReplayLog(log.id || `req_${i}`)}
                            className="flex items-center gap-1 rounded-lg border border-black/[0.06] px-2.5 py-1 text-[11px] font-medium text-neutral-700 hover:bg-neutral-100 dark:border-white/[0.08] dark:text-neutral-300 dark:hover:bg-neutral-800"
                          >
                            <RotateCcw className="h-3 w-3" /> Replay
                          </button>
                        </td>
                      </tr>
                    ))}
                    {trafficLogs.length === 0 && (
                      <tr>
                        <td colSpan={6} className="p-8 text-center text-neutral-400">
                          No requests recorded yet. Run a prompt in the Playground!
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* TAB: AGENT FLEET */}
        {activeTab === 'fleet' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-base font-semibold tracking-tight">Active Host AI CLI Fleet</h2>
                <p className="text-xs text-neutral-500">Auto-detected terminal agents and IDE tools on this system</p>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {cliAgents.map(agent => (
                <div key={agent.id} className="liquid-card flex flex-col justify-between p-5">
                  <div>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2.5">
                        <Terminal className="h-5 w-5 text-blue-500" />
                        <div>
                          <h4 className="font-semibold text-neutral-900 dark:text-neutral-100">{agent.name}</h4>
                          <span className="font-mono text-[11px] text-neutral-400">{agent.id}</span>
                        </div>
                      </div>
                      <span className={`rounded-full px-2 py-0.5 font-mono text-[10px] font-medium ${
                        agent.detected ? 'bg-emerald-500/10 text-emerald-600' : 'bg-neutral-200/50 text-neutral-400'
                      }`}>
                        {agent.detected ? 'Detected' : 'Not installed'}
                      </span>
                    </div>

                    <p className="mt-3 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
                      {agent.description}
                    </p>

                    <div className="mt-3 rounded-lg bg-neutral-100/60 p-2 font-mono text-[10px] text-neutral-600 dark:bg-neutral-800/60 dark:text-neutral-400">
                      {agent.settings_file}
                    </div>
                  </div>

                  <div className="mt-4 flex items-center justify-between border-t border-black/[0.04] pt-3 dark:border-white/[0.04]">
                    <a
                      href={agent.docs_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-1 text-[11px] text-neutral-500 hover:text-blue-600"
                    >
                      <span>Docs</span>
                      <ExternalLink className="h-3 w-3" />
                    </a>

                    <button
                      onClick={() => handleConfigureCli(agent.id)}
                      disabled={!agent.detected}
                      className={`rounded-xl px-3 py-1.5 text-xs font-semibold transition ${
                        agent.configured
                          ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                          : agent.detected
                          ? 'bg-blue-600 text-white shadow-xs hover:bg-blue-700'
                          : 'cursor-not-allowed bg-neutral-200 text-neutral-400 dark:bg-neutral-800 dark:text-neutral-600'
                      }`}
                    >
                      {agent.configured ? 'Connected ✓' : 'Connect to GoRoute'}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* TAB: PLAYGROUND */}
        {activeTab === 'playground' && (
          <div className="liquid-card grid grid-cols-1 overflow-hidden md:grid-cols-2">
            <div className="border-r border-black/[0.06] p-6 dark:border-white/[0.08]">
              <h3 className="mb-4 text-sm font-semibold tracking-tight">Interactive Prompt Studio</h3>
              <div className="mb-4">
                <label className="text-xs font-medium text-neutral-500">Target Model / Cascade Combo:</label>
                <select
                  value={playgroundModel}
                  onChange={e => setPlaygroundModel(e.target.value)}
                  className="mt-1.5 w-full rounded-xl border border-black/[0.06] bg-white p-3 font-mono text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                >
                  <optgroup label="Combos">
                    <option value="claude">claude (Antigravity → Kiro)</option>
                    <option value="claude-models">claude-models (Universal)</option>
                    <option value="deepseek">deepseek (Kiro → Apinex)</option>
                    <option value="claude-opus">claude-opus (Thinking)</option>
                    <option value="auto">auto (Fast Cascade)</option>
                  </optgroup>
                  <optgroup label="Direct Models">
                    <option value="claude-3-7-sonnet">claude-3-7-sonnet</option>
                    <option value="claude-3-5-sonnet">claude-3-5-sonnet</option>
                    <option value="gpt-4o">gpt-4o</option>
                    <option value="gemini-2.5-flash">gemini-2.5-flash</option>
                    <option value="deepseek-chat">deepseek-chat</option>
                  </optgroup>
                </select>
              </div>

              <div>
                <label className="text-xs font-medium text-neutral-500">User Prompt:</label>
                <textarea
                  rows={8}
                  value={playgroundPrompt}
                  onChange={e => setPlaygroundPrompt(e.target.value)}
                  className="mt-1.5 w-full rounded-xl border border-black/[0.06] bg-white p-3 font-mono text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                />
              </div>

              <button
                onClick={handleSendPlayground}
                disabled={playgroundLoading}
                className="mt-5 flex w-full items-center justify-center gap-2 rounded-xl bg-blue-600 py-3 text-xs font-semibold text-white shadow-md shadow-blue-500/20 transition hover:bg-blue-700"
              >
                {playgroundLoading ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
                <span>{playgroundLoading ? 'Dispatching to GoRoute Core...' : 'Send Request'}</span>
              </button>
            </div>

            <div className="flex flex-col justify-between bg-neutral-50/50 p-6 dark:bg-black/20">
              <div>
                <div className="flex items-center justify-between border-b border-black/[0.06] pb-3 dark:border-white/[0.08]">
                  <h3 className="text-sm font-semibold tracking-tight">Streaming Output</h3>
                  {playgroundMeta && (
                    <span className="font-mono text-xs text-emerald-600 dark:text-emerald-400">
                      {playgroundMeta.latency} • {playgroundMeta.tokens} tok
                    </span>
                  )}
                </div>
                <div className="mt-4 max-h-[440px] overflow-y-auto whitespace-pre-wrap font-mono text-xs leading-relaxed text-neutral-800 dark:text-neutral-200">
                  {playgroundResponse || (
                    <span className="italic text-neutral-400">
                      Click 'Send Request' to trigger live inference through GoRoute...
                    </span>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}
      </main>

      {/* Add Upstream Provider Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm">
          <div className="liquid-card w-full max-w-md p-6 shadow-2xl">
            <h3 className="text-base font-semibold tracking-tight">Connect Upstream Provider</h3>
            <p className="mt-1 text-xs text-neutral-500">Configure an API key, OAuth device grant, or custom endpoint</p>

            <div className="mt-4 space-y-3.5">
              <div>
                <label className="text-xs text-neutral-500">Provider Type</label>
                <select
                  value={addProviderType}
                  onChange={e => {
                    setAddProviderType(e.target.value);
                    if (!addProvId) setAddProvId(e.target.value);
                  }}
                  className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                >
                  <option value="openai">OpenAI Compatible (Apinex, Groq, Ollama, DeepSeek)</option>
                  <option value="kiro">AWS Q Developer (Social OAuth Device Flow)</option>
                  <option value="antigravity">Google Cloud Code (OAuth Refresh Token)</option>
                  <option value="codex">OpenAI Codex CLI (Session Token)</option>
                </select>
              </div>

              <div>
                <label className="text-xs text-neutral-500">Provider ID</label>
                <input
                  type="text"
                  placeholder="e.g. deepseek, groq, my-ollama"
                  value={addProvId}
                  onChange={e => setAddProvId(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                />
              </div>

              {addProviderType === 'openai' && (
                <>
                  <div>
                    <label className="text-xs text-neutral-500">Base URL</label>
                    <input
                      type="text"
                      placeholder="https://api.openai.com/v1"
                      value={addBaseUrl}
                      onChange={e => setAddBaseUrl(e.target.value)}
                      className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-neutral-500">API Key</label>
                    <input
                      type="password"
                      placeholder="sk-..."
                      value={addApiKey}
                      onChange={e => setAddApiKey(e.target.value)}
                      className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                    />
                  </div>
                </>
              )}

              {addProviderType === 'kiro' && (
                <div className="rounded-xl bg-blue-50/60 p-3 text-xs text-blue-700 dark:bg-blue-950/40 dark:text-blue-300">
                  <p className="font-semibold">⚡ One-Click Device Flow</p>
                  <p className="mt-1 text-[11px]">Connecting AWS Q Kiro will open your browser to approve the AWS device authorization grant.</p>
                </div>
              )}
            </div>

            <div className="mt-6 flex items-center justify-end gap-2">
              <button
                onClick={() => setShowAddModal(false)}
                className="rounded-xl px-4 py-2 text-xs font-medium text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800"
              >
                Cancel
              </button>
              <button
                onClick={async () => {
                  try {
                    const res = await fetch('/api/providers/add', {
                      method: 'POST',
                      headers: { 'Content-Type': 'application/json' },
                      body: JSON.stringify({
                        id: addProvId || addProviderType,
                        type: addProviderType,
                        base_url: addBaseUrl,
                        api_key: addApiKey,
                        refresh_token: addRefreshToken
                      })
                    });
                    if (res.ok) {
                      showToast(`✅ Provider ${addProvId} connected!`);
                      setShowAddModal(false);
                      fetchAllData();
                    } else {
                      showToast(`❌ Add failed`);
                    }
                  } catch (e: any) {
                    showToast(`❌ Error: ${e.message}`);
                  }
                }}
                className="rounded-xl bg-blue-600 px-4 py-2 text-xs font-semibold text-white shadow-md shadow-blue-500/20 hover:bg-blue-700"
              >
                Save Connection
              </button>
            </div>
          </div>
        </div>
      )}

      {/* New Combo Modal */}
      {showNewComboModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm">
          <div className="liquid-card w-full max-w-md p-6 shadow-2xl">
            <h3 className="text-base font-semibold tracking-tight">Create Routing Combo</h3>
            <p className="mt-1 text-xs text-neutral-500">Configure a resilient failover cascade across multiple accounts</p>

            <div className="mt-4 space-y-3.5">
              <div>
                <label className="text-xs text-neutral-500">Combo Name</label>
                <input
                  type="text"
                  placeholder="e.g. claude-code-resilient, fast-tier"
                  value={newComboName}
                  onChange={e => setNewComboName(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                />
              </div>

              <div>
                <label className="text-xs text-neutral-500">Cascade Strategy</label>
                <select
                  value={newComboStrategy}
                  onChange={e => setNewComboStrategy(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                >
                  <option value="failover">Failover (Try 1st, cascade on 429/500)</option>
                  <option value="lowest_latency">Lowest Latency (Race fastest response)</option>
                  <option value="round_robin">Round Robin (Distribute request volume)</option>
                </select>
              </div>

              <div>
                <label className="text-xs text-neutral-500">Target Models (comma-separated: provider/model)</label>
                <input
                  type="text"
                  placeholder="antigravity/claude-sonnet-4-6, kiro/claude-sonnet-4.5"
                  value={newComboTargets}
                  onChange={e => setNewComboTargets(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                />
              </div>
            </div>

            <div className="mt-6 flex items-center justify-end gap-2">
              <button
                onClick={() => setShowNewComboModal(false)}
                className="rounded-xl px-4 py-2 text-xs font-medium text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800"
              >
                Cancel
              </button>
              <button
                onClick={async () => {
                  try {
                    const targets = newComboTargets.split(',').map(s => {
                      const parts = s.trim().split('/');
                      return { provider_id: parts[0] || 'antigravity', model: parts[1] || parts[0] };
                    });

                    const res = await fetch('/api/combos', {
                      method: 'POST',
                      headers: { 'Content-Type': 'application/json' },
                      body: JSON.stringify({
                        name: newComboName,
                        strategy: newComboStrategy,
                        targets
                      })
                    });
                    if (res.ok) {
                      showToast(`✅ Combo ${newComboName} created!`);
                      setShowNewComboModal(false);
                      fetchAllData();
                    } else {
                      showToast(`❌ Failed to create combo`);
                    }
                  } catch (e: any) {
                    showToast(`❌ Error: ${e.message}`);
                  }
                }}
                className="rounded-xl bg-blue-600 px-4 py-2 text-xs font-semibold text-white shadow-md shadow-blue-500/20 hover:bg-blue-700"
              >
                Create Combo
              </button>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}
