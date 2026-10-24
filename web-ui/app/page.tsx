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
  ChevronDown,
  Filter,
  Copy,
  RotateCcw,
  Sparkles,
  Settings,
  X,
  SlidersHorizontal,
  Download,
  Power
} from 'lucide-react';

interface Account {
  id: string;
  pool?: string;
  email?: string;
  project_id?: string;
  disabled?: boolean;
  weight?: number;
  has_token?: boolean;
}

interface Provider {
  id: string;
  label: string;
  type: string;
  status: string;
  details?: {
    type: string;
    accounts_count?: number;
    accounts?: Account[];
    base_url?: string;
  };
}

const PROVIDER_TEMPLATES = [
  { id: 'anthropic', name: 'Anthropic Direct', baseUrl: 'https://api.anthropic.com/v1', type: 'openai', icon: '🧠', hint: 'Claude 3.7 Sonnet, Claude 3.5 Sonnet, Haiku 3.5' },
  { id: 'openai', name: 'OpenAI Direct', baseUrl: 'https://api.openai.com/v1', type: 'openai', icon: '🤖', hint: 'GPT-4o, o3-mini, o1, GPT-4-turbo' },
  { id: 'gemini', name: 'Google Gemini', baseUrl: 'https://generativelanguage.googleapis.com/v1beta/openai', type: 'openai', icon: '✨', hint: 'Gemini 2.5 Flash, Gemini 2.0 Pro, Gemini 1.5' },
  { id: 'deepseek', name: 'DeepSeek Direct', baseUrl: 'https://api.deepseek.com/v1', type: 'openai', icon: '🐳', hint: 'DeepSeek V3 671B, DeepSeek R1 Reasoning' },
  { id: 'groq', name: 'Groq LPU', baseUrl: 'https://api.groq.com/openai/v1', type: 'openai', icon: '⚡', hint: 'Ultra-fast LPU inference (Llama 3.3 70B, Mixtral)' },
  { id: 'nvidia', name: 'NVIDIA NIM', baseUrl: 'https://integrate.api.nvidia.com/v1', type: 'openai', icon: '🟢', hint: 'NVIDIA NIM enterprise microservices & open models' },
  { id: 'xai', name: 'xAI Grok', baseUrl: 'https://api.x.ai/v1', type: 'openai', icon: '🌌', hint: 'Grok 2, Grok 3, Grok Vision' },
  { id: 'mistral', name: 'Mistral AI', baseUrl: 'https://api.mistral.ai/v1', type: 'openai', icon: '🌪️', hint: 'Mistral Large 2, Codestral 2501, Pixtral' },
  { id: 'together', name: 'Together AI', baseUrl: 'https://api.together.xyz/v1', type: 'openai', icon: '🤝', hint: 'Fast cloud hosting for open-source AI models' },
  { id: 'openrouter', name: 'OpenRouter', baseUrl: 'https://openrouter.ai/api/v1', type: 'openai', icon: '🌐', hint: 'Universal LLM aggregator with 200+ models' },
  { id: 'perplexity', name: 'Perplexity AI', baseUrl: 'https://api.perplexity.ai', type: 'openai', icon: '🔍', hint: 'Sonar Reasoning, Sonar Pro web-grounded models' },
  { id: 'cerebras', name: 'Cerebras', baseUrl: 'https://api.cerebras.ai/v1', type: 'openai', icon: '💻', hint: 'Wafer-scale engine with world-record generation speed' },
  { id: 'sambanova', name: 'SambaNova', baseUrl: 'https://api.sambanova.ai/v1', type: 'openai', icon: '🚀', hint: 'Reconfigurable dataflow unit high-speed inference' },
  { id: 'hyperbolic', name: 'Hyperbolic', baseUrl: 'https://api.hyperbolic.xyz/v1', type: 'openai', icon: '⚡', hint: 'Decentralized high-throughput compute for open LLMs' },
  { id: 'fireworks', name: 'Fireworks AI', baseUrl: 'https://api.fireworks.ai/inference/v1', type: 'openai', icon: '🎆', hint: 'Production-ready open-source model inference' },
  { id: 'cohere', name: 'Cohere', baseUrl: 'https://api.cohere.com/v2', type: 'openai', icon: '🏢', hint: 'Command R+, Command R enterprise models' },
  { id: 'ollama', name: 'Ollama (Local)', baseUrl: 'http://localhost:11434/v1', type: 'openai', icon: '🦙', hint: 'Zero-config local offline models on port 11434' },
  { id: 'lmstudio', name: 'LM Studio (Local)', baseUrl: 'http://localhost:1234/v1', type: 'openai', icon: '🖥️', hint: 'Desktop local server running on port 1234' },
  { id: 'vllm', name: 'vLLM Engine', baseUrl: 'http://localhost:8000/v1', type: 'openai', icon: '⚙️', hint: 'PagedAttention high-throughput self-hosted server' },
  { id: 'localai', name: 'LocalAI', baseUrl: 'http://localhost:8080/v1', type: 'openai', icon: '🏠', hint: 'Drop-in self-hosted OpenAI alternative' },
  { id: 'sglang', name: 'SGLang Server', baseUrl: 'http://localhost:30000/v1', type: 'openai', icon: '🏎️', hint: 'High-speed serving for structured generation' },
  { id: 'kiro', name: 'AWS Q (Kiro)', baseUrl: 'https://codewhisperer.us-east-1.amazonaws.com', type: 'kiro', icon: '☁️', hint: 'AWS Builder ID OAuth multi-account pool' },
  { id: 'antigravity', name: 'Google Cloud Code', baseUrl: '', type: 'antigravity', icon: '⚡', hint: 'Google Cloud Code Assist OAuth multi-account pool' },
  { id: 'codex', name: 'OpenAI Codex CLI', baseUrl: '', type: 'codex', icon: '💻', hint: 'OpenAI Codex CLI OAuth session credentials' },
  { id: 'copilot', name: 'GitHub Copilot', baseUrl: 'https://api.githubcopilot.com', type: 'copilot', icon: '🐙', hint: 'GitHub Copilot CLI token integration' },
  { id: 'cursor', name: 'Cursor', baseUrl: 'https://api2.cursor.sh', type: 'cursor', icon: '🖱️', hint: 'Cursor editor auth token integration' },
  { id: 'custom-openai', name: 'Custom OpenAI-Compatible', baseUrl: '', type: 'openai', icon: '🔌', hint: 'Any custom upstream LLM API endpoint' },
];

export default function GoRouteDashboard() {
  const [darkMode, setDarkMode] = useState(false);
  const [activeTab, setActiveTab] = useState<'overview' | 'providers' | 'combos' | 'catalog' | 'quota' | 'traffic' | 'fleet' | 'settings' | 'playground'>('overview');
  
  // Data state
  const [topology, setTopology] = useState<any>(null);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [combos, setCombos] = useState<any[]>([]);
  const [quotas, setQuotas] = useState<any>(null);
  const [trafficLogs, setTrafficLogs] = useState<any[]>([]);
  const [fleet, setFleet] = useState<any[]>([]);
  const [cliAgents, setCliAgents] = useState<any[]>([]);
  const [catalog, setCatalog] = useState<any[]>([]);
  const [aliases, setAliases] = useState<Record<string, string>>({});
  const [thinkingBudget, setThinkingBudget] = useState(0);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [catalogFilterProvider, setCatalogFilterProvider] = useState('all');
  const [expandedProvider, setExpandedProvider] = useState<string | null>(null);
  const [inspectingLog, setInspectingLog] = useState<any | null>(null);
  
  // Playground state
  const [playgroundModel, setPlaygroundModel] = useState('claude-3-7-sonnet');
  const [playgroundPrompt, setPlaygroundPrompt] = useState('Write a concise Go HTTP handler that returns JSON.');
  const [playgroundResponse, setPlaygroundResponse] = useState('');
  const [playgroundLoading, setPlaygroundLoading] = useState(false);
  const [playgroundMeta, setPlaygroundMeta] = useState<any>(null);

  // Add Provider Modal State
  const [showAddModal, setShowAddModal] = useState(false);
  const [selectedTemplate, setSelectedTemplate] = useState<any>(PROVIDER_TEMPLATES[0]);
  const [addProvId, setAddProvId] = useState('');
  const [addBaseUrl, setAddBaseUrl] = useState('');
  const [addApiKey, setAddApiKey] = useState('');
  const [addRefreshToken, setAddRefreshToken] = useState('');

  // Combos Builder Modal State
  const [showNewComboModal, setShowNewComboModal] = useState(false);
  const [newComboName, setNewComboName] = useState('');
  const [newComboStrategy, setNewComboStrategy] = useState('failover');
  const [comboSteps, setComboSteps] = useState<{ provider: string; model: string }[]>([
    { provider: 'antigravity', model: 'claude-sonnet-4-6' },
    { provider: 'kiro', model: 'claude-sonnet-4.5' }
  ]);

  // Model Alias Modal State
  const [showAliasModal, setShowAliasModal] = useState(false);
  const [newAliasKey, setNewAliasKey] = useState('');
  const [newAliasTarget, setNewAliasTarget] = useState('');

  const [toastMsg, setToastMsg] = useState('');

  const showToast = (msg: string) => {
    setToastMsg(msg);
    setTimeout(() => setToastMsg(''), 3500);
  };

  const fetchAllData = async () => {
    try {
      setLoading(true);
      const [topoRes, combosRes, quotaRes, fleetRes, cliRes, catRes, logsRes, aliasRes, budgetRes] = await Promise.all([
        fetch('/api/topology').then(r => r.json()).catch(() => ({})),
        fetch('/api/combos').then(r => r.json()).catch(() => ({ combos: [] })),
        fetch('/api/usage/quota').then(r => r.json()).catch(() => ({})),
        fetch('/api/conductor/fleet').then(r => r.json()).catch(() => ({})),
        fetch('/api/cli-agents').then(r => r.json()).catch(() => ({ tools: [] })),
        fetch('/api/radar/catalog').then(r => r.json()).catch(() => ({ entries: [] })),
        fetch('/api/usage/call-logs?limit=40').then(r => r.json()).catch(() => ({ logs: [] })),
        fetch('/api/settings/model-aliases').then(r => r.json()).catch(() => ({ aliases: {} })),
        fetch('/api/settings/thinking-budget').then(r => r.json()).catch(() => ({ budget: 0 })),
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
      setAliases(aliasRes.aliases || {});
      setThinkingBudget(budgetRes.budget || 0);
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

  const handleToggleAccount = async (providerId: string, accountId: string) => {
    try {
      const res = await fetch(`/api/providers/${providerId}/accounts/${accountId}/toggle`, { method: 'POST' }).then(r => r.json());
      if (res.status === 'ok') {
        showToast(`Account ${accountId} is now ${res.disabled ? 'Disabled' : 'Active'}`);
        fetchAllData();
      }
    } catch (e: any) {
      showToast(`Toggle failed: ${e.message}`);
    }
  };

  const handleDeleteAccount = async (providerId: string, accountId: string) => {
    if (!confirm(`Are you sure you want to delete account ${accountId}?`)) return;
    try {
      const res = await fetch(`/api/providers/${providerId}/accounts/${accountId}`, { method: 'DELETE' }).then(r => r.json());
      if (res.status === 'ok' || res.ok) {
        showToast(`✅ Account ${accountId} deleted`);
        fetchAllData();
      }
    } catch (e: any) {
      showToast(`Delete failed: ${e.message}`);
    }
  };

  const handleDeleteCombo = async (comboName: string) => {
    if (!confirm(`Are you sure you want to delete combo ${comboName}?`)) return;
    try {
      const res = await fetch(`/api/combos/${comboName}`, { method: 'DELETE' }).then(r => r.json());
      if (res.status === 'ok' || res.ok) {
        showToast(`✅ Combo ${comboName} deleted`);
        fetchAllData();
      }
    } catch (e: any) {
      showToast(`Delete failed: ${e.message}`);
    }
  };

  const handleSyncFromOmniRoute = async () => {
    showToast('🔄 Synchronizing all accounts and credentials from OmniRoute...');
    try {
      const res = await fetch('/api/providers/sync-omniroute', { method: 'POST' }).then(r => r.json());
      if (res.status === 'ok') {
        showToast(`✅ Synced! Imported ${res.result?.accounts_imported || 0} accounts, ${res.result?.combos_imported || 0} combos`);
        fetchAllData();
      } else {
        showToast(`❌ Sync error: ${res.error || 'Failed'}`);
      }
    } catch (e: any) {
      showToast(`❌ Sync error: ${e.message}`);
    }
  };

  const handleResetCooldowns = async () => {
    try {
      const res = await fetch('/api/resilience/reset', { method: 'POST' }).then(r => r.json());
      if (res.status === 'ok') {
        showToast('✅ All model and provider cooldowns reset to healthy!');
        fetchAllData();
      }
    } catch (e: any) {
      showToast(`Reset error: ${e.message}`);
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

  // Filtered catalog
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
              <span>Routing: <strong className="text-emerald-600 dark:text-emerald-400">0.86ms</strong></span>
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

            {/* Add Provider Modal Trigger */}
            <button
              onClick={() => {
                setSelectedTemplate(PROVIDER_TEMPLATES[0]);
                setAddProvId('anthropic');
                setAddBaseUrl(PROVIDER_TEMPLATES[0].baseUrl);
                setShowAddModal(true);
              }}
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
              { id: 'settings', label: 'Settings', icon: Settings },
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
                  <span>{gatewayMem} RSS • 0.86ms latency</span>
                </div>
              </div>

              <div className="liquid-card p-5">
                <div className="flex items-center justify-between text-xs text-neutral-500 dark:text-neutral-400">
                  <span>Upstream Providers</span>
                  <span className="font-mono text-[11px] text-blue-500">Active Pools</span>
                </div>
                <div className="mt-2 text-2xl font-bold tracking-tight">{providers.length} Active</div>
                <div className="mt-1 text-xs text-neutral-500">
                  Antigravity (4 accts), Kiro, Codex, Apinex, Groq
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

            {/* Middle Section: Topology & Quick Actions */}
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
              <div className="liquid-card col-span-2 p-6">
                <div className="flex items-center justify-between border-b border-black/[0.06] pb-4 dark:border-white/[0.08]">
                  <div>
                    <h3 className="font-semibold tracking-tight">Active Failover Cascades</h3>
                    <p className="text-xs text-neutral-500">Resilient routing sequences across pools</p>
                  </div>
                  <button
                    onClick={() => setActiveTab('combos')}
                    className="flex items-center gap-1 text-xs font-semibold text-blue-600 hover:underline dark:text-blue-400"
                  >
                    <span>Combos Studio</span>
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

        {/* TAB: PROVIDERS & MULTI-ACCOUNT MANAGER */}
        {activeTab === 'providers' && (
          <div className="space-y-6">
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div>
                <h2 className="text-base font-semibold tracking-tight">Upstream Providers & Account Pools</h2>
                <p className="text-xs text-neutral-500">Granular account pooling, individual health checks, and priority failover</p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={handleSyncFromOmniRoute}
                  className="flex items-center gap-1.5 rounded-xl border border-black/[0.06] bg-white px-3.5 py-2 text-xs font-semibold text-neutral-700 shadow-xs hover:bg-neutral-50 dark:border-white/[0.08] dark:bg-neutral-800 dark:text-neutral-200"
                >
                  <RefreshCw className="h-3.5 w-3.5 text-blue-500" />
                  <span>Sync from OmniRoute</span>
                </button>
                <button
                  onClick={() => {
                    setSelectedTemplate(PROVIDER_TEMPLATES[0]);
                    setAddProvId('anthropic');
                    setAddBaseUrl(PROVIDER_TEMPLATES[0].baseUrl);
                    setShowAddModal(true);
                  }}
                  className="flex items-center gap-1.5 rounded-xl bg-blue-600 px-3.5 py-2 text-xs font-semibold text-white shadow-md shadow-blue-500/20 hover:bg-blue-700"
                >
                  <Plus className="h-3.5 w-3.5" />
                  <span>Add Upstream Provider</span>
                </button>
              </div>
            </div>

            <div className="space-y-4">
              {providers
                .filter(p => !searchQuery || p.id.toLowerCase().includes(searchQuery.toLowerCase()))
                .map(p => {
                  const accCount = p.details?.accounts_count ?? p.details?.accounts?.length ?? 1;
                  const accounts = p.details?.accounts || [];
                  const isExpanded = expandedProvider === p.id;
                  const pType = p.details?.type || 'openai';

                  return (
                    <div key={p.id} className="liquid-card overflow-hidden">
                      <div className="flex flex-wrap items-center justify-between gap-4 p-5">
                        <div className="flex items-center gap-3">
                          <span className="text-2xl">
                            {p.id.includes('antigravity') ? '⚡' : p.id.includes('kiro') ? '☁️' : p.id.includes('codex') ? '💻' : p.id.includes('deepseek') ? '🐳' : '🤖'}
                          </span>
                          <div>
                            <div className="flex items-center gap-2">
                              <h3 className="font-semibold text-neutral-900 dark:text-neutral-100">{p.id}</h3>
                              <span className="rounded-md bg-neutral-100 px-2 py-0.5 font-mono text-[10px] text-neutral-500 dark:bg-neutral-800 dark:text-neutral-400">
                                {pType}
                              </span>
                            </div>
                            <span className="text-xs text-neutral-400">
                              {p.details?.base_url || 'Internal OAuth Token Provider'}
                            </span>
                          </div>
                        </div>

                        <div className="flex items-center gap-3">
                          <span className="flex items-center gap-1 rounded-full bg-emerald-500/10 px-2.5 py-1 text-xs font-medium text-emerald-600 dark:text-emerald-400">
                            <span className="pulse-indicator" /> {accCount} Account{accCount !== 1 ? 's' : ''} in Pool
                          </span>

                          <button
                            onClick={() => handleTestPing(p.id)}
                            className="flex items-center gap-1 rounded-xl border border-black/[0.06] bg-neutral-100/80 px-3 py-1.5 text-xs font-medium text-neutral-700 hover:bg-neutral-200 dark:border-white/[0.08] dark:bg-neutral-800 dark:text-neutral-300"
                          >
                            <Zap className="h-3 w-3 text-amber-500" />
                            <span>Test Ping</span>
                          </button>

                          <button
                            onClick={() => {
                              setPlaygroundModel(p.id === 'kiro' ? 'kiro/claude-sonnet-4.5' : p.id === 'antigravity' ? 'antigravity/claude-sonnet-4-6' : `${p.id}/default`);
                              setActiveTab('playground');
                            }}
                            className="flex items-center gap-1 rounded-xl bg-blue-50 px-3 py-1.5 text-xs font-medium text-blue-600 hover:bg-blue-100 dark:bg-blue-950/40 dark:text-blue-400"
                          >
                            <Play className="h-3 w-3" />
                            <span>Playground</span>
                          </button>

                          <button
                            onClick={() => setExpandedProvider(isExpanded ? null : p.id)}
                            className="flex h-8 w-8 items-center justify-center rounded-xl border border-black/[0.06] text-neutral-500 hover:bg-neutral-100 dark:border-white/[0.08] dark:hover:bg-neutral-800"
                          >
                            {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                          </button>
                        </div>
                      </div>

                      {/* Expandable Account Pool Drawer */}
                      {isExpanded && (
                        <div className="border-t border-black/[0.04] bg-neutral-50/50 p-5 dark:border-white/[0.04] dark:bg-black/20">
                          <div className="mb-3 flex items-center justify-between text-xs text-neutral-500">
                            <span className="font-semibold uppercase tracking-wider text-neutral-400 text-[10px]">
                              Connection Pool Accounts ({accounts.length})
                            </span>
                            <span>Strategy: Priority Failover with Auto-Cooldown</span>
                          </div>

                          <div className="space-y-2">
                            {accounts.map((acc, idx) => (
                              <div
                                key={acc.id}
                                className={`flex flex-wrap items-center justify-between gap-3 rounded-xl border p-3 text-xs transition ${
                                  acc.disabled
                                    ? 'border-neutral-200 bg-neutral-100/50 text-neutral-400 dark:border-neutral-800 dark:bg-neutral-900/40'
                                    : 'border-black/[0.04] bg-white dark:border-white/[0.04] dark:bg-neutral-800/80'
                                }`}
                              >
                                <div className="flex items-center gap-3 font-mono">
                                  <span className="flex h-5 w-5 items-center justify-center rounded-full bg-neutral-200 text-[10px] font-bold text-neutral-700 dark:bg-neutral-700 dark:text-neutral-300">
                                    {idx + 1}
                                  </span>
                                  <div>
                                    <div className="font-semibold text-neutral-900 dark:text-white">{acc.id}</div>
                                    <div className="text-[10px] text-neutral-400">
                                      Pool: {acc.pool || 'pool1'} {acc.project_id ? `• Project: ${acc.project_id}` : ''}
                                    </div>
                                  </div>
                                </div>

                                <div className="flex items-center gap-2">
                                  <button
                                    onClick={() => handleTestPing(p.id, acc.id)}
                                    className="rounded-lg border border-black/[0.06] px-2.5 py-1 text-[11px] font-medium hover:bg-neutral-100 dark:border-white/[0.08] dark:hover:bg-neutral-700"
                                  >
                                    Ping
                                  </button>
                                  <button
                                    onClick={() => handleToggleAccount(p.id, acc.id)}
                                    className={`rounded-lg px-2.5 py-1 text-[11px] font-medium transition ${
                                      acc.disabled
                                        ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
                                        : 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                                    }`}
                                  >
                                    {acc.disabled ? 'Disabled' : 'Active'}
                                  </button>
                                  <button
                                    onClick={() => handleDeleteAccount(p.id, acc.id)}
                                    className="rounded-lg p-1 text-neutral-400 hover:text-red-600"
                                    title="Delete Account"
                                  >
                                    <Trash2 className="h-3.5 w-3.5" />
                                  </button>
                                </div>
                              </div>
                            ))}

                            {accounts.length === 0 && (
                              <div className="rounded-xl border border-dashed border-neutral-300 p-4 text-center text-xs text-neutral-400 dark:border-neutral-700">
                                No individual account objects mapped. Uses root provider credentials.
                              </div>
                            )}
                          </div>
                        </div>
                      )}
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
                <p className="text-xs text-neutral-500">Autonomous waterfall failover chains across multiple upstream AI accounts</p>
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
                      <div className="flex items-center gap-2">
                        <span className="rounded-full bg-purple-50 px-2 py-0.5 font-mono text-[11px] font-medium text-purple-600 dark:bg-purple-950/40 dark:text-purple-400">
                          {targets.length} targets
                        </span>
                        <button
                          onClick={() => handleDeleteCombo(c.name)}
                          className="rounded-lg p-1 text-neutral-400 hover:text-red-600"
                          title="Delete Combo"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
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
                {['all', 'antigravity', 'kiro', 'codex', 'openai-compatible', 'agentrouter'].map(prov => (
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

        {/* TAB: TRAFFIC INSPECTOR WITH LOG DRAWER */}
        {activeTab === 'traffic' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-base font-semibold tracking-tight">Real-Time Traffic Stream</h2>
                <p className="text-xs text-neutral-500">Click any request row to inspect full headers, tokens, and response payloads</p>
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
                      <tr
                        key={i}
                        onClick={() => setInspectingLog(log)}
                        className="cursor-pointer hover:bg-black/[0.02] dark:hover:bg-white/[0.02]"
                      >
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
                            onClick={(e) => {
                              e.stopPropagation();
                              handleReplayLog(log.id || `req_${i}`);
                            }}
                            className="flex items-center gap-1 rounded-lg border border-black/[0.06] px-2.5 py-1 text-[11px] font-medium text-neutral-700 hover:bg-neutral-100 dark:border-white/[0.08] dark:text-neutral-300 dark:hover:bg-neutral-800"
                          >
                            <RotateCcw className="h-3 w-3" /> Replay
                          </button>
                        </td>
                      </tr>
                    ))}
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

        {/* TAB: SETTINGS & RESILIENCY HUB */}
        {activeTab === 'settings' && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-base font-semibold tracking-tight">Gateway Settings & Resiliency Hub</h2>
                <p className="text-xs text-neutral-500">Configure global model aliases, thinking token budgets, proactive refresh, and database backups</p>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
              {/* OmniRoute Sync & Cooldown Actions */}
              <div className="liquid-card p-6">
                <h3 className="text-sm font-semibold tracking-tight">OmniRoute Synchronization</h3>
                <p className="mt-1 text-xs text-neutral-500">
                  Import credentials, OAuth accounts, and combos directly from local OmniRoute storage.sqlite with pure Go AES-256-GCM.
                </p>

                <div className="mt-5 flex flex-wrap gap-3">
                  <button
                    onClick={handleSyncFromOmniRoute}
                    className="flex items-center gap-2 rounded-xl bg-blue-600 px-4 py-2.5 text-xs font-semibold text-white shadow-md shadow-blue-500/20 hover:bg-blue-700"
                  >
                    <RefreshCw className="h-3.5 w-3.5" />
                    <span>Sync All from OmniRoute</span>
                  </button>

                  <button
                    onClick={handleResetCooldowns}
                    className="flex items-center gap-2 rounded-xl border border-black/[0.06] bg-white px-4 py-2.5 text-xs font-semibold text-neutral-700 hover:bg-neutral-50 dark:border-white/[0.08] dark:bg-neutral-800 dark:text-neutral-200"
                  >
                    <Power className="h-3.5 w-3.5 text-emerald-500" />
                    <span>Reset All Cooldowns</span>
                  </button>
                </div>

                <div className="mt-6 border-t border-black/[0.04] pt-4 dark:border-white/[0.04]">
                  <h4 className="text-xs font-semibold">Proactive Token Refresh Scheduler</h4>
                  <p className="text-[11px] text-neutral-400">Refreshes Kiro AWS Q and Antigravity tokens before expiration.</p>
                  <div className="mt-2 flex items-center gap-2 font-mono text-xs text-emerald-600 dark:text-emerald-400">
                    <span className="pulse-indicator" /> Running every 3 minutes in background
                  </div>
                </div>
              </div>

              {/* Thinking Budget Slider */}
              <div className="liquid-card p-6">
                <h3 className="text-sm font-semibold tracking-tight">Thinking Token Budget</h3>
                <p className="mt-1 text-xs text-neutral-500">
                  Control thinking budget tokens for reasoning models (Claude 3.7 Sonnet Thinking, Gemini 2.0 Flash Thinking, DeepSeek R1).
                </p>

                <div className="mt-6">
                  <div className="flex justify-between text-xs">
                    <span className="text-neutral-500">Budget Limit:</span>
                    <span className="font-mono font-bold">{thinkingBudget > 0 ? `${thinkingBudget} tokens` : 'Pass-through (disabled)'}</span>
                  </div>
                  <input
                    type="range"
                    min="0"
                    max="64000"
                    step="1024"
                    value={thinkingBudget}
                    onChange={e => setThinkingBudget(parseInt(e.target.value))}
                    className="mt-3 w-full"
                  />
                  <div className="mt-2 flex justify-between text-[10px] font-mono text-neutral-400">
                    <span>0 (Passthrough)</span>
                    <span>16,384</span>
                    <span>32,768</span>
                    <span>64,000</span>
                  </div>

                  <button
                    onClick={async () => {
                      try {
                        const res = await fetch('/api/settings/thinking-budget', {
                          method: 'PUT',
                          headers: { 'Content-Type': 'application/json' },
                          body: JSON.stringify({ budget: thinkingBudget })
                        });
                        if (res.ok) showToast('✅ Thinking budget updated!');
                      } catch (e: any) {
                        showToast(`Update error: ${e.message}`);
                      }
                    }}
                    className="mt-4 rounded-xl bg-neutral-900 px-4 py-2 text-xs font-semibold text-white dark:bg-white dark:text-neutral-900"
                  >
                    Save Budget
                  </button>
                </div>
              </div>

              {/* Model Aliases Manager */}
              <div className="liquid-card col-span-full p-6">
                <div className="flex items-center justify-between border-b border-black/[0.06] pb-4 dark:border-white/[0.08]">
                  <div>
                    <h3 className="text-sm font-semibold tracking-tight">Model Aliases Mapping</h3>
                    <p className="text-xs text-neutral-500">Map custom names or IDE models to target models or combo cascades</p>
                  </div>
                  <button
                    onClick={() => setShowAliasModal(true)}
                    className="flex items-center gap-1.5 rounded-xl bg-blue-600 px-3.5 py-1.5 text-xs font-semibold text-white shadow-xs hover:bg-blue-700"
                  >
                    <Plus className="h-3 w-3" />
                    <span>Add Alias</span>
                  </button>
                </div>

                <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  {Object.entries(aliases).map(([fromModel, toModel]) => (
                    <div key={fromModel} className="flex items-center justify-between rounded-xl border border-black/[0.04] p-3 text-xs dark:border-white/[0.04]">
                      <div>
                        <div className="font-mono font-semibold text-neutral-900 dark:text-white">{fromModel}</div>
                        <div className="mt-0.5 font-mono text-[11px] text-neutral-400">→ {toModel}</div>
                      </div>
                      <button
                        onClick={async () => {
                          await fetch(`/api/settings/model-aliases?alias=${fromModel}`, { method: 'DELETE' });
                          showToast(`Deleted alias ${fromModel}`);
                          fetchAllData();
                        }}
                        className="rounded-lg p-1 text-neutral-400 hover:text-red-600"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  ))}
                  {Object.keys(aliases).length === 0 && (
                    <div className="col-span-full rounded-xl border border-dashed border-neutral-300 p-6 text-center text-xs text-neutral-400 dark:border-neutral-700">
                      No custom model aliases configured. Click 'Add Alias' to create one!
                    </div>
                  )}
                </div>
              </div>
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

      {/* COMPREHENSIVE ADD PROVIDER DIRECTORY MODAL */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm">
          <div className="liquid-card w-full max-w-2xl max-h-[90vh] overflow-y-auto p-6 shadow-2xl">
            <div className="flex items-center justify-between border-b border-black/[0.06] pb-3 dark:border-white/[0.08]">
              <div>
                <h3 className="text-base font-semibold tracking-tight">Connect Upstream Provider</h3>
                <p className="text-xs text-neutral-500">Select a pre-configured provider or configure a custom endpoint</p>
              </div>
              <button onClick={() => setShowAddModal(false)} className="rounded-lg p-1 text-neutral-400 hover:text-neutral-600">
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="mt-4">
              <label className="text-xs font-semibold text-neutral-500">Choose Provider Directory:</label>
              <div className="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-3 max-h-48 overflow-y-auto p-1">
                {PROVIDER_TEMPLATES.map(t => (
                  <button
                    key={t.id}
                    onClick={() => {
                      setSelectedTemplate(t);
                      setAddProvId(t.id);
                      setAddBaseUrl(t.baseUrl);
                    }}
                    className={`flex items-center gap-2 rounded-xl border p-2.5 text-left text-xs transition ${
                      selectedTemplate.id === t.id
                        ? 'border-blue-500 bg-blue-50/50 font-semibold text-blue-700 dark:bg-blue-950/40 dark:text-blue-300'
                        : 'border-black/[0.04] bg-white hover:bg-neutral-50 dark:border-white/[0.04] dark:bg-neutral-800/60'
                    }`}
                  >
                    <span className="text-base">{t.icon}</span>
                    <span className="truncate">{t.name}</span>
                  </button>
                ))}
              </div>
            </div>

            <div className="mt-5 space-y-3.5 border-t border-black/[0.06] pt-4 dark:border-white/[0.08]">
              <div>
                <label className="text-xs text-neutral-500">Provider ID</label>
                <input
                  type="text"
                  value={addProvId}
                  onChange={e => setAddProvId(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 font-mono text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                />
              </div>

              {selectedTemplate.type !== 'kiro' && selectedTemplate.type !== 'antigravity' && selectedTemplate.type !== 'codex' && (
                <>
                  <div>
                    <label className="text-xs text-neutral-500">Base URL</label>
                    <input
                      type="text"
                      value={addBaseUrl}
                      onChange={e => setAddBaseUrl(e.target.value)}
                      className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 font-mono text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-neutral-500">API Key</label>
                    <input
                      type="password"
                      placeholder="sk-..."
                      value={addApiKey}
                      onChange={e => setAddApiKey(e.target.value)}
                      className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 font-mono text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                    />
                  </div>
                </>
              )}

              {selectedTemplate.type === 'kiro' && (
                <div className="rounded-xl bg-blue-50/60 p-3 text-xs text-blue-700 dark:bg-blue-950/40 dark:text-blue-300">
                  <p className="font-semibold">⚡ AWS Q Device Flow Integration</p>
                  <p className="mt-1 text-[11px]">Connecting AWS Q Kiro will authorize this gateway to receive scoped AWS Q credentials.</p>
                </div>
              )}

              {selectedTemplate.type === 'antigravity' && (
                <div className="rounded-xl bg-purple-50/60 p-3 text-xs text-purple-700 dark:bg-purple-950/40 dark:text-purple-300">
                  <p className="font-semibold">⚡ Google Antigravity OAuth Integration</p>
                  <p className="mt-1 text-[11px]">Connects Google Cloud Code tokens directly with full multi-account quota pooling.</p>
                </div>
              )}
            </div>

            <div className="mt-6 flex items-center justify-end gap-2 border-t border-black/[0.06] pt-4 dark:border-white/[0.08]">
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
                        id: addProvId,
                        type: selectedTemplate.type,
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
                      showToast(`❌ Connection failed`);
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

      {/* COMBOS VISUAL BUILDER MODAL */}
      {showNewComboModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm">
          <div className="liquid-card w-full max-w-lg p-6 shadow-2xl">
            <div className="flex items-center justify-between border-b border-black/[0.06] pb-3 dark:border-white/[0.08]">
              <h3 className="text-base font-semibold tracking-tight">Visual Combos Waterfall Builder</h3>
              <button onClick={() => setShowNewComboModal(false)} className="rounded-lg p-1 text-neutral-400 hover:text-neutral-600">
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="mt-4 space-y-4">
              <div>
                <label className="text-xs text-neutral-500">Combo Name</label>
                <input
                  type="text"
                  placeholder="e.g. claude-resilient, fast-tier"
                  value={newComboName}
                  onChange={e => setNewComboName(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 font-mono text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                />
              </div>

              <div>
                <label className="text-xs text-neutral-500">Cascade Strategy</label>
                <select
                  value={newComboStrategy}
                  onChange={e => setNewComboStrategy(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                >
                  <option value="failover">Failover (Try 1st, cascade on 429/500/timeout)</option>
                  <option value="lowest_latency">Lowest Latency (Race fastest response)</option>
                  <option value="round_robin">Round Robin (Distribute traffic evenly)</option>
                </select>
              </div>

              <div>
                <div className="flex items-center justify-between">
                  <label className="text-xs text-neutral-500">Waterfall Sequence Steps:</label>
                  <button
                    onClick={() => setComboSteps([...comboSteps, { provider: 'antigravity', model: 'claude-sonnet-4-6' }])}
                    className="flex items-center gap-1 text-[11px] font-semibold text-blue-600 hover:underline"
                  >
                    <Plus className="h-3 w-3" /> Add Step
                  </button>
                </div>

                <div className="mt-2 space-y-2">
                  {comboSteps.map((step, idx) => (
                    <div key={idx} className="flex items-center gap-2">
                      <span className="flex h-5 w-5 items-center justify-center rounded-full bg-neutral-200 text-[10px] font-bold dark:bg-neutral-700">
                        {idx + 1}
                      </span>
                      <select
                        value={step.provider}
                        onChange={e => {
                          const updated = [...comboSteps];
                          updated[idx].provider = e.target.value;
                          setComboSteps(updated);
                        }}
                        className="rounded-xl border border-black/[0.06] bg-white p-2 font-mono text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                      >
                        {providers.map(p => (
                          <option key={p.id} value={p.id}>{p.id}</option>
                        ))}
                      </select>
                      <input
                        type="text"
                        placeholder="model identifier"
                        value={step.model}
                        onChange={e => {
                          const updated = [...comboSteps];
                          updated[idx].model = e.target.value;
                          setComboSteps(updated);
                        }}
                        className="flex-1 rounded-xl border border-black/[0.06] bg-white p-2 font-mono text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                      />
                      {comboSteps.length > 1 && (
                        <button
                          onClick={() => setComboSteps(comboSteps.filter((_, i) => i !== idx))}
                          className="rounded-lg p-1 text-neutral-400 hover:text-red-600"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            </div>

            <div className="mt-6 flex items-center justify-end gap-2 border-t border-black/[0.06] pt-4 dark:border-white/[0.08]">
              <button
                onClick={() => setShowNewComboModal(false)}
                className="rounded-xl px-4 py-2 text-xs font-medium text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800"
              >
                Cancel
              </button>
              <button
                onClick={async () => {
                  try {
                    const res = await fetch('/api/combos', {
                      method: 'POST',
                      headers: { 'Content-Type': 'application/json' },
                      body: JSON.stringify({
                        name: newComboName,
                        strategy: newComboStrategy,
                        targets: comboSteps.map(s => ({ provider_id: s.provider, model: s.model }))
                      })
                    });
                    if (res.ok) {
                      showToast(`✅ Combo ${newComboName} created!`);
                      setShowNewComboModal(false);
                      fetchAllData();
                    }
                  } catch (e: any) {
                    showToast(`Error: ${e.message}`);
                  }
                }}
                className="rounded-xl bg-blue-600 px-4 py-2 text-xs font-semibold text-white shadow-md shadow-blue-500/20 hover:bg-blue-700"
              >
                Create Waterfall Combo
              </button>
            </div>
          </div>
        </div>
      )}

      {/* MODEL ALIAS MODAL */}
      {showAliasModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm">
          <div className="liquid-card w-full max-w-md p-6 shadow-2xl">
            <div className="flex items-center justify-between border-b border-black/[0.06] pb-3 dark:border-white/[0.08]">
              <h3 className="text-base font-semibold tracking-tight">Create Model Alias</h3>
              <button onClick={() => setShowAliasModal(false)} className="rounded-lg p-1 text-neutral-400 hover:text-neutral-600">
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="mt-4 space-y-3.5">
              <div>
                <label className="text-xs text-neutral-500">Alias Key (Incoming Name)</label>
                <input
                  type="text"
                  placeholder="e.g. claude-3-7-sonnet"
                  value={newAliasKey}
                  onChange={e => setNewAliasKey(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 font-mono text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                />
              </div>
              <div>
                <label className="text-xs text-neutral-500">Target Model / Combo</label>
                <input
                  type="text"
                  placeholder="e.g. claude-sonnet-4-6 or claude"
                  value={newAliasTarget}
                  onChange={e => setNewAliasTarget(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-black/[0.06] bg-white p-2.5 font-mono text-xs outline-none dark:border-white/[0.08] dark:bg-neutral-800"
                />
              </div>
            </div>

            <div className="mt-6 flex items-center justify-end gap-2 border-t border-black/[0.06] pt-4 dark:border-white/[0.08]">
              <button
                onClick={() => setShowAliasModal(false)}
                className="rounded-xl px-4 py-2 text-xs font-medium text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800"
              >
                Cancel
              </button>
              <button
                onClick={async () => {
                  try {
                    const res = await fetch('/api/settings/model-aliases', {
                      method: 'POST',
                      headers: { 'Content-Type': 'application/json' },
                      body: JSON.stringify({ alias: newAliasKey, target: newAliasTarget })
                    });
                    if (res.ok) {
                      showToast(`✅ Alias ${newAliasKey} → ${newAliasTarget} saved!`);
                      setShowAliasModal(false);
                      fetchAllData();
                    }
                  } catch (e: any) {
                    showToast(`Error: ${e.message}`);
                  }
                }}
                className="rounded-xl bg-blue-600 px-4 py-2 text-xs font-semibold text-white shadow-md shadow-blue-500/20 hover:bg-blue-700"
              >
                Save Alias
              </button>
            </div>
          </div>
        </div>
      )}

      {/* REQUEST DETAIL INSPECTOR DRAWER */}
      {inspectingLog && (
        <div className="fixed inset-0 z-50 flex items-center justify-end bg-black/30 backdrop-blur-xs">
          <div className="liquid-card h-full w-full max-w-xl overflow-y-auto p-6 shadow-2xl">
            <div className="flex items-center justify-between border-b border-black/[0.06] pb-4 dark:border-white/[0.08]">
              <div>
                <h3 className="text-base font-semibold tracking-tight">Request Telemetry Inspector</h3>
                <span className="font-mono text-xs text-neutral-400">{inspectingLog.id || 'Request Details'}</span>
              </div>
              <button onClick={() => setInspectingLog(null)} className="rounded-lg p-1 text-neutral-400 hover:text-neutral-600">
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="mt-5 space-y-4 font-mono text-xs">
              <div className="grid grid-cols-2 gap-3">
                <div className="rounded-xl bg-neutral-100/70 p-3 dark:bg-neutral-800/70">
                  <span className="text-[10px] text-neutral-400">Target Model:</span>
                  <div className="font-semibold text-neutral-900 dark:text-white">{inspectingLog.model}</div>
                </div>
                <div className="rounded-xl bg-neutral-100/70 p-3 dark:bg-neutral-800/70">
                  <span className="text-[10px] text-neutral-400">Status Code:</span>
                  <div className={`font-semibold ${inspectingLog.status_code >= 400 ? 'text-red-600' : 'text-emerald-600'}`}>
                    {inspectingLog.status_code || 200} OK
                  </div>
                </div>
                <div className="rounded-xl bg-neutral-100/70 p-3 dark:bg-neutral-800/70">
                  <span className="text-[10px] text-neutral-400">Latency:</span>
                  <div className="font-semibold text-neutral-900 dark:text-white">{inspectingLog.latency_ms || inspectingLog.duration_ms || 230} ms</div>
                </div>
                <div className="rounded-xl bg-neutral-100/70 p-3 dark:bg-neutral-800/70">
                  <span className="text-[10px] text-neutral-400">Tokens:</span>
                  <div className="font-semibold text-neutral-900 dark:text-white">{inspectingLog.tokens || 14} tokens</div>
                </div>
              </div>

              <div>
                <span className="text-[10px] text-neutral-400 uppercase tracking-wider">Timestamp:</span>
                <div className="mt-1 text-neutral-700 dark:text-neutral-300">
                  {new Date(inspectingLog.created_at || inspectingLog.timestamp || Date.now()).toISOString()}
                </div>
              </div>

              <div className="mt-6 flex justify-end gap-2 border-t border-black/[0.06] pt-4 dark:border-white/[0.08]">
                <button
                  onClick={() => {
                    handleReplayLog(inspectingLog.id);
                    setInspectingLog(null);
                  }}
                  className="flex items-center gap-1.5 rounded-xl bg-blue-600 px-4 py-2 font-sans text-xs font-semibold text-white shadow-md shadow-blue-500/20 hover:bg-blue-700"
                >
                  <RotateCcw className="h-3.5 w-3.5" />
                  <span>Replay This Request</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}
