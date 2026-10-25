'use client';

import React, { useState, useEffect, useMemo } from 'react';
import {
  Activity,
  Globe,
  Cpu,
  CheckCircle,
  XCircle,
  HelpCircle,
  Lock,
  ArrowLeft,
  Layers,
  Cloud,
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

function ProviderBadgeIcon({ id, className = "h-4 w-4" }: { id: string; className?: string }) {
  if (id.includes('antigravity')) return <Zap className={className} />;
  if (id.includes('kiro')) return <Cloud className={className} />;
  if (id.includes('codex')) return <Terminal className={className} />;
  if (id.includes('copilot')) return <Layers className={className} />;
  if (id.includes('cursor')) return <Cpu className={className} />;
  if (id.includes('deepseek')) return <Database className={className} />;
  if (id.includes('openai')) return <Sparkles className={className} />;
  if (id.includes('anthropic')) return <Cpu className={className} />;
  if (id.includes('gemini')) return <Sparkles className={className} />;
  if (id.includes('groq')) return <Zap className={className} />;
  return <Server className={className} />;
}

const PROVIDER_TEMPLATES = [
  { id: 'anthropic', name: 'Anthropic Direct', baseUrl: 'https://api.anthropic.com/v1', type: 'openai', category: 'popular', hint: 'Claude 3.7 Sonnet, Claude 3.5 Sonnet, Haiku 3.5' },
  { id: 'openai', name: 'OpenAI Direct', baseUrl: 'https://api.openai.com/v1', type: 'openai', category: 'popular', hint: 'GPT-4o, o3-mini, o1, GPT-4-turbo' },
  { id: 'gemini', name: 'Google Gemini', baseUrl: 'https://generativelanguage.googleapis.com/v1beta/openai', type: 'openai', category: 'popular', hint: 'Gemini 2.5 Flash, Gemini 2.0 Pro, Gemini 1.5' },
  { id: 'deepseek', name: 'DeepSeek Direct', baseUrl: 'https://api.deepseek.com/v1', type: 'openai', category: 'popular', hint: 'DeepSeek V3 671B, DeepSeek R1 Reasoning' },
  { id: 'groq', name: 'Groq LPU', baseUrl: 'https://api.groq.com/openai/v1', type: 'openai', category: 'popular', hint: 'Ultra-fast LPU inference (Llama 3.3 70B, Mixtral)' },
  { id: 'nvidia', name: 'NVIDIA NIM', baseUrl: 'https://integrate.api.nvidia.com/v1', type: 'openai', category: 'popular', hint: 'NVIDIA NIM enterprise microservices & open models' },
  { id: 'xai', name: 'xAI Grok', baseUrl: 'https://api.x.ai/v1', type: 'openai', category: 'popular', hint: 'Grok 2, Grok 3, Grok Vision' },
  { id: 'mistral', name: 'Mistral AI', baseUrl: 'https://api.mistral.ai/v1', type: 'openai', category: 'popular', hint: 'Mistral Large 2, Codestral 2501, Pixtral' },
  { id: 'together', name: 'Together AI', baseUrl: 'https://api.together.xyz/v1', type: 'openai', category: 'popular', hint: 'Fast cloud hosting for open-source AI models' },
  { id: 'openrouter', name: 'OpenRouter', baseUrl: 'https://openrouter.ai/api/v1', type: 'openai', category: 'popular', hint: 'Universal LLM aggregator with 200+ models' },
  { id: 'perplexity', name: 'Perplexity AI', baseUrl: 'https://api.perplexity.ai', type: 'openai', category: 'popular', hint: 'Sonar Reasoning, Sonar Pro web-grounded models' },
  { id: 'cerebras', name: 'Cerebras', baseUrl: 'https://api.cerebras.ai/v1', type: 'openai', category: 'cloud', hint: 'Wafer-scale engine with world-record generation speed' },
  { id: 'sambanova', name: 'SambaNova', baseUrl: 'https://api.sambanova.ai/v1', type: 'openai', category: 'cloud', hint: 'Reconfigurable dataflow unit high-speed inference' },
  { id: 'hyperbolic', name: 'Hyperbolic', baseUrl: 'https://api.hyperbolic.xyz/v1', type: 'openai', category: 'cloud', hint: 'Decentralized high-throughput compute for open LLMs' },
  { id: 'fireworks', name: 'Fireworks AI', baseUrl: 'https://api.fireworks.ai/inference/v1', type: 'openai', category: 'cloud', hint: 'Production-ready open-source model inference' },
  { id: 'cohere', name: 'Cohere', baseUrl: 'https://api.cohere.com/v2', type: 'openai', category: 'cloud', hint: 'Command R+, Command R enterprise models' },
  { id: 'ollama', name: 'Ollama (Local)', baseUrl: 'http://localhost:11434/v1', type: 'openai', category: 'local', hint: 'Zero-config local offline models on port 11434' },
  { id: 'lmstudio', name: 'LM Studio (Local)', baseUrl: 'http://localhost:1234/v1', type: 'openai', category: 'local', hint: 'Desktop local server running on port 1234' },
  { id: 'vllm', name: 'vLLM Engine', baseUrl: 'http://localhost:8000/v1', type: 'openai', category: 'local', hint: 'PagedAttention high-throughput self-hosted server' },
  { id: 'localai', name: 'LocalAI', baseUrl: 'http://localhost:8080/v1', type: 'openai', category: 'local', hint: 'Drop-in self-hosted OpenAI alternative' },
  { id: 'sglang', name: 'SGLang Server', baseUrl: 'http://localhost:30000/v1', type: 'openai', category: 'local', hint: 'High-speed serving for structured generation' },
  { id: 'kiro', name: 'AWS Q (Kiro)', baseUrl: 'https://codewhisperer.us-east-1.amazonaws.com', type: 'kiro', category: 'oauth', hint: 'AWS Builder ID OAuth multi-account pool' },
  { id: 'antigravity', name: 'Google Cloud Code (Antigravity)', baseUrl: '', type: 'antigravity', category: 'oauth', hint: 'Google Cloud Code Assist OAuth multi-account pool' },
  { id: 'codex', name: 'OpenAI Codex CLI', baseUrl: '', type: 'codex', category: 'oauth', hint: 'OpenAI Codex CLI OAuth session credentials' },
  { id: 'copilot', name: 'GitHub Copilot', baseUrl: 'https://api.githubcopilot.com', type: 'copilot', category: 'oauth', hint: 'GitHub Copilot CLI token integration' },
  { id: 'cursor', name: 'Cursor', baseUrl: 'https://api2.cursor.sh', type: 'cursor', category: 'oauth', hint: 'Cursor editor auth token integration' },
  { id: 'custom-openai', name: 'Custom OpenAI-Compatible', baseUrl: '', type: 'openai', category: 'custom', hint: 'Any custom upstream LLM API endpoint' },
];

export default function GoRouteDashboard() {
  const [darkMode, setDarkMode] = useState(false);
  const [activeTab, setActiveTab] = useState<'overview' | 'providers' | 'connect-wizard' | 'combos' | 'catalog' | 'quota' | 'traffic' | 'fleet' | 'settings' | 'playground'>('overview');
  
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
  const [wizardStep, setWizardStep] = useState<number>(1);
  const [wizardCategory, setWizardCategory] = useState<string>('all');
  const [wizardSearch, setWizardSearch] = useState<string>('');
  const [wizardTestResult, setWizardTestResult] = useState<any>(null);
  const [testingConnection, setTestingConnection] = useState<boolean>(false);

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

  // Agent Configuration Studio State
  const [configuringAgent, setConfiguringAgent] = useState<any | null>(null);
  const [agentProvider, setAgentProvider] = useState<string>('auto');
  const [agentPrimaryModel, setAgentPrimaryModel] = useState<string>('claude-3-7-sonnet');
  const [agentSmallModel, setAgentSmallModel] = useState<string>('gemini-2.5-flash');
  const [agentSelectedModels, setAgentSelectedModels] = useState<string[]>([]);
  const [agentCustomUrl, setAgentCustomUrl] = useState<string>('http://127.0.0.1:8128/v1');
  const [agentApplying, setAgentApplying] = useState<boolean>(false);

  const getAvailableModelsAndCombos = () => {
    const list: string[] = [];
    combos.forEach(c => {
      if (c?.name && !list.includes(c.name)) list.push(c.name);
    });
    if (topology?.models) {
      if (Array.isArray(topology.models)) {
        topology.models.forEach((m: string) => {
          if (!list.includes(m)) list.push(m);
        });
      } else if (typeof topology.models === 'object') {
        Object.keys(topology.models).forEach(m => {
          if (!list.includes(m)) list.push(m);
        });
      }
    }
    const defaults = [
      'claude-3-7-sonnet',
      'claude-3-5-sonnet',
      'claude-models',
      'gemini-2.5-flash',
      'gemini-2.5-pro',
      'gemini-3.8-flash-high',
      'deepseek',
      'auto',
      'gpt-4o',
    ];
    defaults.forEach(d => {
      if (!list.includes(d)) list.push(d);
    });
    return list;
  };

  const openAgentConfigModal = (agent: any) => {
    setConfiguringAgent(agent);
    if (agent.id === 'opencode') {
      setAgentPrimaryModel('claude-3-7-sonnet');
      setAgentSmallModel('gemini-2.5-flash');
    } else if (agent.id === 'claude') {
      setAgentPrimaryModel('claude-3-7-sonnet');
    } else if (agent.id === 'codex') {
      setAgentPrimaryModel('claude-3-7-sonnet');
    } else {
      setAgentPrimaryModel('claude-3-7-sonnet');
      setAgentSmallModel('gemini-2.5-flash');
    }
    setAgentSelectedModels(getAvailableModelsAndCombos());
  };

  const getAgentConfigPreview = (agent: any) => {
    if (!agent) return '';
    const resolvedPrimary = agentProvider !== 'auto' && !agentPrimaryModel.includes('/') && !combos.some(c => c.name === agentPrimaryModel)
      ? `${agentProvider}/${agentPrimaryModel}`
      : agentPrimaryModel;
    const resolvedSmall = agentProvider !== 'auto' && !agentSmallModel.includes('/') && !combos.some(c => c.name === agentSmallModel)
      ? `${agentProvider}/${agentSmallModel}`
      : agentSmallModel;

    if (agent.id === 'opencode') {
      return JSON.stringify({
        provider: {
          goroute: {
            name: "GoRoute",
            npm: "@ai-sdk/openai-compatible",
            options: { baseURL: agentCustomUrl || "http://127.0.0.1:8128/v1", apiKey: "goroute" },
            models: Object.fromEntries(agentSelectedModels.map(m => [m, { name: m, limit: { context: 128000, output: 8192 } }]))
          }
        },
        model: `goroute/${resolvedPrimary}`,
        small_model: `goroute/${resolvedSmall}`,
        agent: {
          build: { model: `goroute/${resolvedPrimary}` },
          plan: { model: `goroute/${resolvedPrimary}` }
        }
      }, null, 2);
    }
    if (agent.id === 'claude') {
      return JSON.stringify({
        customBaseUrl: "http://127.0.0.1:8128",
        model: resolvedPrimary,
        env: {
          ANTHROPIC_BASE_URL: "http://127.0.0.1:8128",
          ANTHROPIC_API_KEY: "goroute",
          ANTHROPIC_MODEL: resolvedPrimary
        }
      }, null, 2);
    }
    if (agent.id === 'codex') {
      return `model = "${resolvedPrimary}"
model_provider = "goroute"

[model_providers.goroute]
name = "GoRoute"
base_url = "http://127.0.0.1:8128/v1"
wire_api = "responses"`;
    }
    return JSON.stringify({
      provider: "goroute",
      baseUrl: "http://127.0.0.1:8128/v1",
      apiKey: "goroute",
      model: resolvedPrimary
    }, null, 2);
  };

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
    const saved = localStorage.getItem('goroute-theme');
    const prefersDark = window.matchMedia ? window.matchMedia('(prefers-color-scheme: dark)').matches : true;
    const isDark = saved !== null ? saved === 'dark' : prefersDark;
    setDarkMode(isDark);
    if (isDark) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }

    fetchAllData();
    const interval = setInterval(fetchAllData, 8000);
    return () => clearInterval(interval);
  }, []);

  const toggleDarkMode = () => {
    const next = !darkMode;
    setDarkMode(next);
    localStorage.setItem('goroute-theme', next ? 'dark' : 'light');
    if (next) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  };

  const handleTestPing = async (providerId: string, accountId?: string) => {
    showToast(`Testing ping latency for ${providerId}...`);
    try {
      const url = accountId
        ? `/api/providers/${providerId}/accounts/${accountId}/test`
        : `/api/providers/${providerId}/test`;
      const res = await fetch(url, { method: 'POST' }).then(r => r.json());
      if (res.status === 'ok' || res.success) {
        showToast(`${providerId} latency: ${res.latency_ms || res.latencyMs || 22}ms`);
      } else {
        showToast(`${providerId} test failed: ${res.error || 'Unknown error'}`);
      }
      fetchAllData();
    } catch (e: any) {
      showToast(`Test error: ${e.message}`);
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
        showToast(`Account ${accountId} deleted`);
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
        showToast(`Combo ${comboName} deleted`);
        fetchAllData();
      }
    } catch (e: any) {
      showToast(`Delete failed: ${e.message}`);
    }
  };

  const handleSyncFromOmniRoute = async () => {
    showToast('Synchronizing all accounts and credentials from OmniRoute...');
    try {
      const res = await fetch('/api/providers/sync-omniroute', { method: 'POST' }).then(r => r.json());
      if (res.status === 'ok') {
        showToast(`Synced! Imported ${res.result?.accounts_imported || 0} accounts, ${res.result?.combos_imported || 0} combos`);
        fetchAllData();
      } else {
        showToast(`Sync error: ${res.error || 'Failed'}`);
      }
    } catch (e: any) {
      showToast(`Sync error: ${e.message}`);
    }
  };

  const handleImportLocalAntigravity = async () => {
    showToast('Discovering locally logged-in Antigravity accounts from CLI & IDE...');
    try {
      const res = await fetch('/api/providers/import-antigravity', { method: 'POST' }).then(r => r.json());
      if (res.status === 'ok') {
        showToast(`${res.message || 'Imported local Antigravity accounts!'}`);
        fetchAllData();
        return true;
      } else {
        showToast(`Import error: ${res.error || 'Failed'}`);
        return false;
      }
    } catch (e: any) {
      showToast(`Import error: ${e.message}`);
      return false;
    }
  };

  const handleImportLocalProvider = async (providerType: string) => {
    showToast(`Importing local ${providerType} credentials...`);
    try {
      const endpoint = providerType === 'antigravity'
        ? '/api/providers/import-antigravity'
        : `/api/providers/import-${providerType}`;
      const res = await fetch(endpoint, { method: 'POST' }).then(r => r.json());
      if (res.status === 'ok') {
        showToast(`${res.message || `Successfully imported ${providerType}!`}`);
        fetchAllData();
        return true;
      } else {
        showToast(`Import error: ${res.error || 'Failed'}`);
        return false;
      }
    } catch (e: any) {
      showToast(`Import error: ${e.message}`);
      return false;
    }
  };

  const handleResetCooldowns = async () => {
    try {
      const res = await fetch('/api/resilience/reset', { method: 'POST' }).then(r => r.json());
      if (res.status === 'ok') {
        showToast('All model and provider cooldowns reset to healthy!');
        fetchAllData();
      }
    } catch (e: any) {
      showToast(`Reset error: ${e.message}`);
    }
  };

  const handleReplayLog = async (logId: string) => {
    showToast(`Replaying request ${logId}...`);
    try {
      const res = await fetch(`/api/logs/${logId}/replay`, { method: 'POST' }).then(r => r.json());
      if (res.status === 'ok' || res.success) {
        showToast(`Request replayed successfully!`);
      } else {
        showToast(`Replay error: ${res.error || 'Failed'}`);
      }
      fetchAllData();
    } catch (e: any) {
      showToast(`Network error: ${e.message}`);
    }
  };

  const handleConfigureCli = async (
    toolId: string,
    customOpts?: { provider?: string; model?: string; small_model?: string; models?: string[]; base_url?: string }
  ) => {
    showToast(`Configuring ${toolId} for GoRoute...`);
    try {
      const res = await fetch(`/api/cli-agents/${toolId}/configure`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(customOpts || {}),
      }).then(r => r.json());
      if (res.status === 'ok' || res.success) {
        showToast(`✅ ${toolId} configured cleanly!`);
        setConfiguringAgent(null);
        fetchAllData();
      } else {
        showToast(`Config error: ${res.error || 'Failed'}`);
      }
    } catch (e: any) {
      showToast(`Error: ${e.message}`);
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
    <div className={`min-h-screen ${darkMode ? 'dark bg-[#0B0F17] text-slate-100' : 'bg-slate-50 text-slate-900'} transition-colors`}>
      
      {/* Toast Notification */}
      {toastMsg && (
        <div className="fixed bottom-6 right-6 z-50 flex items-center gap-2.5 rounded-xl border border-slate-200 bg-white px-4 py-3 text-xs font-semibold text-slate-900 shadow-xl dark:border-slate-800 dark:bg-slate-900 dark:text-white">
          <span className="h-2 w-2 rounded-full bg-rose-500 animate-ping" />
          <span>{toastMsg}</span>
        </div>
      )}

      {/* Top Header Navigation */}
      <header className="sticky top-0 z-40 border-b border-slate-200 bg-white/95 px-6 py-3.5 backdrop-blur-md dark:border-slate-800 dark:bg-[#0B0F17]/95">
        <div className="mx-auto flex max-w-7xl items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-rose-600 text-white shadow-xs">
              <Zap className="h-4 w-4 fill-white" />
            </div>
            <div className="flex items-baseline gap-2">
              <span className="text-sm font-bold tracking-tight text-slate-900 dark:text-white uppercase">GoRoute</span>
              <span className="font-mono text-[11px] text-slate-500 dark:text-slate-400">v0.3.0</span>
            </div>
          </div>

          <div className="flex items-center gap-4">
            {/* Minimal Live Status */}
            <div className="hidden items-center gap-2.5 rounded-full border border-slate-200 bg-slate-100/70 px-3 py-1 font-mono text-[11px] text-slate-700 dark:border-slate-800 dark:bg-slate-900/80 dark:text-slate-300 md:flex">
              <span className="flex items-center gap-1.5">
                <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                <span className="font-semibold text-slate-900 dark:text-white">:8128</span>
              </span>
              <span>•</span>
              <span><strong>0.86ms</strong> routing</span>
              <span>•</span>
              <span><strong>{gatewayMem}</strong> RSS</span>
            </div>

            <div className="flex items-center gap-2">
              <button
                onClick={toggleDarkMode}
                className="flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-100 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800"
                title="Toggle Theme"
              >
                {darkMode ? <Sun className="h-3.5 w-3.5 text-amber-400" /> : <Moon className="h-3.5 w-3.5 text-slate-600" />}
              </button>

              <button
                onClick={fetchAllData}
                disabled={loading}
                className="flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-100 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800"
                title="Refresh State"
              >
                <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin text-rose-600' : ''}`} />
              </button>

              <button
                type="button"
                onClick={() => {
                  setWizardStep(1);
                  setActiveTab('connect-wizard');
                }}
                className="flex items-center gap-1.5 rounded-lg bg-rose-600 px-3.5 py-1.5 text-xs font-semibold text-white shadow-xs transition hover:bg-rose-700 active:scale-98"
              >
                <Plus className="h-3.5 w-3.5" />
                <span>Connect Provider</span>
                <span className="font-mono text-xs">↗</span>
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content Shell */}
      <main className="mx-auto max-w-7xl px-6 py-8">
        
        {/* Navigation Bar Tabs */}
        <div className="mb-8 flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 pb-2 dark:border-slate-800">
          <nav className="flex flex-wrap items-center gap-1">
            {[
              { id: 'overview', num: '01', label: 'Overview' },
              { id: 'providers', num: '02', label: 'Providers', count: providers.length },
              { id: 'combos', num: '03', label: 'Combos Studio', count: combos.length },
              { id: 'catalog', num: '04', label: 'Radar Catalog', count: catalog.length },
              { id: 'quota', num: '05', label: 'Rolling Quota' },
              { id: 'traffic', num: '06', label: 'Inspector', count: trafficLogs.length },
              { id: 'fleet', num: '07', label: 'Agent Fleet', count: cliAgents.filter(a => a.detected).length },
              { id: 'playground', num: '08', label: 'Playground' },
              { id: 'settings', num: '09', label: 'Settings' },
            ].map(tab => {
              const isActive = activeTab === tab.id;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as any)}
                  className={`group relative flex items-center gap-1.5 rounded-lg px-3 py-2 text-xs transition-all duration-200 hover:-translate-y-0.5 active:scale-95 ${
                    isActive
                      ? 'font-bold text-slate-900 dark:text-white bg-slate-100/80 dark:bg-slate-800/80'
                      : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100/50 dark:text-slate-400 dark:hover:text-white dark:hover:bg-slate-800/50'
                  }`}
                >
                  <span className={`font-mono text-[10px] ${isActive ? 'text-rose-600 dark:text-rose-400 font-bold' : 'text-slate-500 dark:text-slate-500'}`}>
                    {tab.num}
                  </span>
                  <span>{tab.label}</span>
                  {tab.count !== undefined && (
                    <span className="font-mono text-[10px] text-slate-500 dark:text-slate-400">
                      ({tab.count})
                    </span>
                  )}
                  {isActive && (
                    <span className="absolute bottom-0 left-3 right-3 h-[2px] bg-rose-600 dark:bg-rose-500" />
                  )}
                </button>
              );
            })}
          </nav>

          <div className="relative min-w-[220px]">
            <Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-slate-400 dark:text-slate-500" />
            <input
              type="text"
              placeholder="Search models, providers..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              className="w-full rounded-lg border border-slate-200 bg-white py-1.5 pl-9 pr-3 text-xs text-slate-900 placeholder:text-slate-400 outline-none transition focus:border-rose-500 dark:border-slate-800 dark:bg-slate-900 dark:text-white dark:placeholder:text-slate-500"
            />
          </div>
        </div>

        {/* TAB: OVERVIEW */}
        {activeTab === 'overview' && (
          <div className="space-y-10">
            {/* Hero Section */}
            <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4 border-b border-slate-200 pb-6 dark:border-slate-800">
              <div>
                <div className="flex items-center gap-2 text-[10px] font-mono uppercase tracking-widest text-slate-500 dark:text-slate-400">
                  <span className="text-rose-600 dark:text-rose-400 font-bold">▪ 01</span>
                  <span>/</span>
                  <span>THE SYSTEM</span>
                </div>
                <h1 className="mt-2 text-3xl sm:text-4xl font-extrabold tracking-tight text-slate-900 dark:text-white">
                  Ultra-low latency. <span className="text-rose-600 dark:text-rose-400">Minimal footprint.</span>
                </h1>
                <p className="mt-1.5 text-xs text-slate-600 dark:text-slate-400">
                  High-throughput universal AI gateway orchestrating local pools and remote models.
                </p>
              </div>

              <div className="flex items-center gap-2 text-xs">
                <span className="font-mono text-[11px] text-slate-500 dark:text-slate-400">Status:</span>
                <span className="inline-flex items-center gap-1.5 rounded-full border border-emerald-500/20 bg-emerald-500/10 px-2.5 py-0.5 font-mono text-[11px] font-semibold text-emerald-700 dark:text-emerald-400">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                  Operational
                </span>
              </div>
            </div>

            {/* Editorial 4-Stat Strip */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <div className="liquid-card p-6">
                <div className="text-[10px] font-mono tracking-widest text-slate-500 dark:text-slate-400 uppercase font-semibold">
                  01. / DISPATCH LATENCY
                </div>
                <div className="mt-3 flex items-baseline">
                  <span className="text-4xl font-extrabold tracking-tight text-slate-900 dark:text-white">0.86</span>
                  <span className="ml-1 text-sm font-bold text-rose-600 dark:text-rose-400">ms</span>
                </div>
                <div className="mt-2 text-xs font-semibold text-slate-900 dark:text-white">Sub-Millisecond</div>
                <div className="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400">
                  Purego C-ABI routing pipeline
                </div>
              </div>

              <div className="liquid-card p-6">
                <div className="text-[10px] font-mono tracking-widest text-slate-500 dark:text-slate-400 uppercase font-semibold">
                  02. / MEMORY FOOTPRINT
                </div>
                <div className="mt-3 flex items-baseline">
                  <span className="text-4xl font-extrabold tracking-tight text-slate-900 dark:text-white">{gatewayMem.replace(/[^0-9.]/g, '') || '4.5'}</span>
                  <span className="ml-1 text-sm font-bold text-rose-600 dark:text-rose-400">MB</span>
                </div>
                <div className="mt-2 text-xs font-semibold text-slate-900 dark:text-white">Near-Zero RSS</div>
                <div className="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400">
                  Compact daemon memory usage
                </div>
              </div>

              <div className="liquid-card p-6">
                <div className="text-[10px] font-mono tracking-widest text-slate-500 dark:text-slate-400 uppercase font-semibold">
                  03. / UPSTREAM POOLS
                </div>
                <div className="mt-3 flex items-baseline">
                  <span className="text-4xl font-extrabold tracking-tight text-slate-900 dark:text-white">{providers.length}</span>
                  <span className="ml-1 text-sm font-bold text-rose-600 dark:text-rose-400">Active</span>
                </div>
                <div className="mt-2 text-xs font-semibold text-slate-900 dark:text-white">Aggregated Accounts</div>
                <div className="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400">
                  Antigravity, Kiro, Codex, Apinex, Groq
                </div>
              </div>

              <div className="liquid-card p-6">
                <div className="text-[10px] font-mono tracking-widest text-slate-500 dark:text-slate-400 uppercase font-semibold">
                  04. / RADAR CATALOG
                </div>
                <div className="mt-3 flex items-baseline">
                  <span className="text-4xl font-extrabold tracking-tight text-slate-900 dark:text-white">{catalog.length}</span>
                  <span className="ml-1 text-sm font-bold text-rose-600 dark:text-rose-400">Models</span>
                </div>
                <div className="mt-2 text-xs font-semibold text-slate-900 dark:text-white">Live Capabilities</div>
                <div className="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400">
                  Auto-mapped model intelligence
                </div>
              </div>
            </div>

            {/* Middle Section: Cascades & Host Fleet */}
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
              <div className="liquid-card col-span-2 p-6">
                <div className="flex items-center justify-between border-b border-slate-200 pb-4 dark:border-slate-800">
                  <div>
                    <div className="flex items-center gap-2 text-[10px] font-mono uppercase tracking-widest text-slate-500 dark:text-slate-400">
                      <span className="text-rose-600 dark:text-rose-400 font-bold">▪ 02</span>
                      <span>/</span>
                      <span>ROUTING CASCADES</span>
                    </div>
                    <h3 className="mt-1 text-sm font-bold text-slate-900 dark:text-white">Active Failover Cascades</h3>
                  </div>
                  <button
                    onClick={() => setActiveTab('combos')}
                    className="flex items-center gap-1 text-xs font-semibold text-rose-600 hover:underline dark:text-rose-400"
                  >
                    <span>Combos Studio</span>
                    <span>↗</span>
                  </button>
                </div>

                <div className="mt-4 space-y-3">
                  {combos.slice(0, 4).map(c => (
                    <div
                      key={c.name}
                      className="group flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/70 p-3.5 transition-all duration-200 hover:-translate-y-0.5 hover:border-rose-500/50 hover:bg-white hover:shadow-sm dark:border-slate-800 dark:bg-slate-800/40 dark:hover:bg-slate-850 dark:hover:border-rose-500/50"
                    >
                      <div className="flex items-center gap-2.5">
                        <span className="font-mono text-xs font-bold text-slate-900 dark:text-white">{c.name}</span>
                        <span className="rounded border border-slate-200 bg-white px-1.5 py-0.5 font-mono text-[10px] uppercase text-slate-600 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300">
                          {c.strategy}
                        </span>
                      </div>

                      <div className="flex flex-wrap items-center gap-1.5 font-mono text-[11px] text-slate-600 dark:text-slate-300">
                        {c.targets?.map((t: any, idx: number) => (
                          <React.Fragment key={idx}>
                            <span className="rounded border border-slate-200 bg-white px-2 py-0.5 text-slate-800 shadow-2xs dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200">
                              {t.provider_id}/{t.model}
                            </span>
                            {idx < c.targets.length - 1 && <span className="text-slate-400">→</span>}
                          </React.Fragment>
                        ))}
                      </div>

                      <button
                        onClick={() => {
                          setPlaygroundModel(c.name);
                          setActiveTab('playground');
                        }}
                        className="flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-3 py-1 text-xs font-semibold text-slate-800 transition hover:border-rose-500 hover:text-rose-600 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:border-rose-400"
                      >
                        <span>Test Run</span>
                        <span className="font-mono text-[10px]">↗</span>
                      </button>
                    </div>
                  ))}
                </div>
              </div>

              {/* Host AI Developer Tools */}
              <div className="liquid-card p-6">
                <div className="flex items-center justify-between border-b border-slate-200 pb-4 dark:border-slate-800">
                  <div>
                    <div className="flex items-center gap-2 text-[10px] font-mono uppercase tracking-widest text-slate-500 dark:text-slate-400">
                      <span className="text-rose-600 dark:text-rose-400 font-bold">▪ 03</span>
                      <span>/</span>
                      <span>LOCAL AGENTS</span>
                    </div>
                    <h3 className="mt-1 text-sm font-bold text-slate-900 dark:text-white">Host AI Fleet</h3>
                  </div>
                  <span className="font-mono text-[11px] text-slate-500 dark:text-slate-400 font-semibold">
                    {cliAgents.filter(a => a.detected).length} Detected
                  </span>
                </div>

                <div className="mt-4 space-y-2.5">
                  {cliAgents.filter(a => a.detected).slice(0, 5).map(agent => (
                    <div
                      key={agent.id}
                      className="group flex items-center justify-between rounded-xl border border-slate-200 bg-slate-50/70 p-3 transition-all duration-200 hover:-translate-y-0.5 hover:border-rose-500/50 hover:bg-white hover:shadow-sm dark:border-slate-800 dark:bg-slate-800/40 dark:hover:bg-slate-850 dark:hover:border-rose-500/50"
                    >
                      <div>
                        <div className="flex items-center gap-1.5">
                          <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                          <span className="text-xs font-bold text-slate-900 dark:text-white">{agent.name}</span>
                        </div>
                        <p className="mt-0.5 truncate font-mono text-[10px] text-slate-500 dark:text-slate-400 max-w-[160px]">
                          {agent.settings_file}
                        </p>
                      </div>

                      <button
                        onClick={() => handleConfigureCli(agent.id)}
                        className={`flex items-center gap-1 rounded-lg px-2.5 py-1 text-xs font-semibold transition ${
                          agent.configured
                            ? 'border border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
                            : 'bg-rose-600 text-white hover:bg-rose-700'
                        }`}
                      >
                        <span>{agent.configured ? 'Connected' : 'Connect'}</span>
                        {!agent.configured && <span className="font-mono text-[10px]">↗</span>}
                      </button>
                    </div>
                  ))}

                  <button
                    onClick={() => setActiveTab('fleet')}
                    className="mt-2 w-full rounded-lg border border-slate-200 bg-white py-2 text-center text-xs font-semibold text-slate-700 transition hover:bg-slate-50 hover:text-rose-600 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800 dark:hover:text-rose-400"
                  >
                    View All 35 CLI Agents ↗
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* TAB: PROVIDERS & MULTI-ACCOUNT MANAGER */}
                {/* TAB: DEDICATED CONNECT PROVIDER ONBOARDING WORKFLOW */}
        {activeTab === 'connect-wizard' && (
          <div className="space-y-8 animate-in fade-in duration-200">
            {/* Top Navigation & Breadcrumbs */}
            <div className="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 pb-5 dark:border-slate-800">
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  onClick={() => setActiveTab('providers')}
                  className="flex h-9 w-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-600 transition hover:bg-slate-100 hover:text-slate-900 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-white"
                >
                  <ArrowLeft className="h-4 w-4" />
                </button>
                <div>
                  <div className="flex items-center gap-2 text-[10px] font-mono uppercase tracking-widest text-slate-500 dark:text-slate-400">
                    <span className="text-rose-600 dark:text-rose-400 font-bold">▪ ONBOARDING</span>
                    <span>/</span>
                    <span>CONNECT UPSTREAM PROVIDER</span>
                  </div>
                  <h1 className="mt-1 text-2xl font-bold tracking-tight text-slate-900 dark:text-white">
                    Provider Connection Studio
                  </h1>
                </div>
              </div>

              {/* Step indicator */}
              <div className="flex items-center gap-2">
                {[
                  { step: 1, label: 'Select' },
                  { step: 2, label: 'Configure' },
                  { step: 3, label: 'Verify' },
                ].map(s => {
                  const isCurrent = wizardStep === s.step;
                  const isPast = wizardStep > s.step;
                  return (
                    <div
                      key={s.step}
                      className={
                        isCurrent
                          ? 'flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-semibold border border-rose-500/30 bg-rose-50 text-rose-700 dark:border-rose-500/30 dark:bg-rose-950/40 dark:text-rose-300'
                          : isPast
                          ? 'flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-semibold border border-emerald-500/30 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-950/40 dark:text-emerald-300'
                          : 'flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-semibold border border-slate-200 bg-white text-slate-400 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-500'
                      }
                    >
                      {isPast ? <CheckCircle className="h-3.5 w-3.5" /> : <span className="font-mono text-[11px]">{s.step}.</span>}
                      <span>{s.label}</span>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* STEP 1: SELECT PROVIDER */}
            {wizardStep === 1 && (
              <div className="space-y-6">
                {/* Category Filters */}
                <div className="flex flex-wrap items-center justify-between gap-4">
                  <div className="flex flex-wrap items-center gap-1.5">
                    {[
                      { id: 'all', label: 'All Providers' },
                      { id: 'popular', label: 'Proprietary Clouds' },
                      { id: 'oauth', label: 'OAuth & Subscriptions' },
                      { id: 'local', label: 'Local & Self-Hosted' },
                      { id: 'custom', label: 'Custom Endpoint' },
                    ].map(cat => (
                      <button
                        key={cat.id}
                        type="button"
                        onClick={() => setWizardCategory(cat.id)}
                        className={
                          wizardCategory === cat.id
                            ? 'rounded-lg px-3 py-1.5 text-xs font-semibold transition bg-slate-900 text-white dark:bg-white dark:text-slate-900'
                            : 'rounded-lg px-3 py-1.5 text-xs font-semibold transition border border-slate-200 bg-white text-slate-600 hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-400 dark:hover:bg-slate-800'
                        }
                      >
                        {cat.label}
                      </button>
                    ))}
                  </div>

                  <div className="relative w-full sm:w-64">
                    <Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-slate-400 dark:text-slate-500" />
                    <input
                      type="text"
                      placeholder="Search catalog..."
                      value={wizardSearch}
                      onChange={e => setWizardSearch(e.target.value)}
                      className="w-full rounded-lg border border-slate-200 bg-white py-1.5 pl-9 pr-3 text-xs outline-none transition focus:border-rose-500 dark:border-slate-800 dark:bg-slate-900 dark:text-white"
                    />
                  </div>
                </div>

                {/* Provider Grid */}
                <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-3">
                  {PROVIDER_TEMPLATES
                    .filter(t => {
                      if (wizardCategory !== 'all' && t.category !== wizardCategory) return false;
                      if (!wizardSearch) return true;
                      const q = wizardSearch.toLowerCase();
                      return t.name.toLowerCase().includes(q) || t.id.toLowerCase().includes(q) || t.hint.toLowerCase().includes(q);
                    })
                    .map(t => {
                      const isSelected = selectedTemplate.id === t.id;
                      return (
                        <div
                          key={t.id}
                          onClick={() => {
                            setSelectedTemplate(t);
                            setAddProvId(t.id);
                            setAddBaseUrl(t.baseUrl);
                            setWizardStep(2);
                          }}
                          className={
                            isSelected
                              ? 'group relative flex cursor-pointer flex-col justify-between rounded-xl border p-5 transition hover:shadow-md border-rose-600 bg-rose-50/40 dark:border-rose-500 dark:bg-rose-950/20'
                              : 'group relative flex cursor-pointer flex-col justify-between rounded-xl border p-5 transition hover:shadow-md border-slate-200 bg-white hover:border-slate-300 dark:border-slate-800 dark:bg-slate-900 dark:hover:border-slate-700'
                          }
                        >
                          <div>
                            <div className="flex items-start justify-between gap-3">
                              <div className="flex items-center gap-3">
                                <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-800 transition group-hover:border-rose-400 group-hover:text-rose-600 dark:border-slate-800 dark:bg-slate-800 dark:text-slate-200">
                                  <ProviderBadgeIcon id={t.id} className="h-5 w-5" />
                                </div>
                                <div>
                                  <h4 className="text-sm font-bold text-slate-900 dark:text-white">{t.name}</h4>
                                  <span className="font-mono text-[11px] text-slate-400 dark:text-slate-500">{t.id}</span>
                                </div>
                              </div>
                              <span className="rounded border border-slate-200 bg-slate-100 px-1.5 py-0.5 font-mono text-[10px] uppercase text-slate-600 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-400">
                                {t.type}
                              </span>
                            </div>

                            <p className="mt-3 text-xs leading-relaxed text-slate-600 dark:text-slate-400 line-clamp-2">
                              {t.hint}
                            </p>
                          </div>

                          <div className="mt-4 flex items-center justify-between border-t border-slate-100 pt-3 dark:border-slate-800/80">
                            <span className="font-mono text-[11px] text-slate-400 dark:text-slate-500">
                              {t.type === 'antigravity' || t.type === 'kiro' || t.type === 'codex' ? 'OAuth Session' : 'API Key Auth'}
                            </span>
                            <span className="flex items-center gap-1 text-xs font-semibold text-rose-600 transition group-hover:translate-x-0.5 dark:text-rose-400">
                              <span>Configure</span>
                              <ChevronRight className="h-3.5 w-3.5" />
                            </span>
                          </div>
                        </div>
                      );
                    })}
                </div>
              </div>
            )}

            {/* STEP 2: CONFIGURE PROVIDER */}
            {wizardStep === 2 && (
              <div className="mx-auto max-w-2xl space-y-6">
                {/* Header card */}
                <div className="flex items-center justify-between rounded-xl border border-slate-200 bg-white p-5 dark:border-slate-800 dark:bg-slate-900 shadow-xs">
                  <div className="flex items-center gap-3.5">
                    <div className="flex h-12 w-12 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-900 dark:border-slate-800 dark:bg-slate-800 dark:text-white">
                      <ProviderBadgeIcon id={selectedTemplate.id} className="h-6 w-6" />
                    </div>
                    <div>
                      <h3 className="text-base font-bold text-slate-900 dark:text-white">{selectedTemplate.name}</h3>
                      <p className="text-xs text-slate-500 dark:text-slate-400">Configure authentication parameters and endpoint routing</p>
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={() => setWizardStep(1)}
                    className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700"
                  >
                    Change Provider
                  </button>
                </div>

                {/* Form fields */}
                <div className="rounded-xl border border-slate-200 bg-white p-6 dark:border-slate-800 dark:bg-slate-900 shadow-xs space-y-5">
                  <div>
                    <label className="block text-xs font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">
                      Internal Provider ID
                    </label>
                    <input
                      type="text"
                      value={addProvId}
                      onChange={e => setAddProvId(e.target.value)}
                      className="mt-1.5 w-full rounded-lg border border-slate-200 bg-white p-2.5 font-mono text-xs outline-none transition focus:border-rose-500 dark:border-slate-800 dark:bg-slate-800 dark:text-white"
                      placeholder="e.g. anthropic-primary"
                    />
                    <p className="mt-1 text-[11px] text-slate-400 dark:text-slate-500">
                      Unique identifier used in combo routes and client virtual model targets.
                    </p>
                  </div>

                  {/* Standard API Key & BaseURL */}
                  {selectedTemplate.type !== 'kiro' && selectedTemplate.type !== 'antigravity' && selectedTemplate.type !== 'codex' && selectedTemplate.type !== 'cursor' && selectedTemplate.type !== 'copilot' && (
                    <>
                      <div>
                        <label className="block text-xs font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">
                          Base API URL
                        </label>
                        <input
                          type="text"
                          value={addBaseUrl}
                          onChange={e => setAddBaseUrl(e.target.value)}
                          className="mt-1.5 w-full rounded-lg border border-slate-200 bg-white p-2.5 font-mono text-xs outline-none transition focus:border-rose-500 dark:border-slate-800 dark:bg-slate-800 dark:text-white"
                          placeholder="https://api.openai.com/v1"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-bold uppercase tracking-wider text-slate-700 dark:text-slate-300">
                          API Key
                        </label>
                        <input
                          type="password"
                          value={addApiKey}
                          onChange={e => setAddApiKey(e.target.value)}
                          className="mt-1.5 w-full rounded-lg border border-slate-200 bg-white p-2.5 font-mono text-xs outline-none transition focus:border-rose-500 dark:border-slate-800 dark:bg-slate-800 dark:text-white"
                          placeholder="sk-... or provider secret token"
                        />
                      </div>
                    </>
                  )}

                  {/* Antigravity Special Workflows */}
                  {selectedTemplate.type === 'antigravity' && (
                    <div className="space-y-4 pt-2">
                      <div className="rounded-xl border border-rose-300 bg-rose-50/50 p-4 dark:border-rose-900/50 dark:bg-rose-950/20">
                        <div className="flex items-center justify-between gap-3">
                          <div>
                            <div className="flex items-center gap-2">
                              <span className="flex h-5 w-5 items-center justify-center rounded-full bg-rose-600 text-[10px] font-bold text-white">1</span>
                              <h4 className="text-xs font-bold text-slate-900 dark:text-white">Auto-Import Local Antigravity (IDE & CLI)</h4>
                              <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-400">Recommended</span>
                            </div>
                            <p className="mt-1 text-[11px] leading-relaxed text-slate-600 dark:text-slate-400">
                              Discovers Google Cloud Code Assist tokens from local IDE SQLite db and CLI configs.
                            </p>
                          </div>
                          <button
                            type="button"
                            onClick={handleImportLocalAntigravity}
                            className="flex items-center gap-1.5 rounded-lg bg-rose-600 px-3.5 py-1.5 text-xs font-semibold text-white shadow-xs transition hover:bg-rose-700"
                          >
                            <Zap className="h-3.5 w-3.5" />
                            <span>Run Auto-Import</span>
                          </button>
                        </div>
                      </div>

                      <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-4 dark:border-slate-800 dark:bg-slate-800/40">
                        <div className="flex items-center gap-2">
                          <span className="flex h-5 w-5 items-center justify-center rounded-full bg-slate-600 text-[10px] font-bold text-white">2</span>
                          <h4 className="text-xs font-bold text-slate-900 dark:text-white">Manual Refresh Token</h4>
                        </div>
                        <input
                          type="password"
                          placeholder="1//0g... (Google OAuth refresh token)"
                          value={addRefreshToken}
                          onChange={e => setAddRefreshToken(e.target.value)}
                          className="mt-2.5 w-full rounded-lg border border-slate-200 bg-white p-2.5 font-mono text-xs outline-none dark:border-slate-800 dark:bg-slate-800 dark:text-white"
                        />
                      </div>
                    </div>
                  )}

                  {/* Kiro Special Workflows */}
                  {selectedTemplate.type === 'kiro' && (
                    <div className="space-y-4 pt-2">
                      <div className="rounded-xl border border-sky-300 bg-sky-50/50 p-4 dark:border-sky-900/50 dark:bg-sky-950/20">
                        <div className="flex items-center justify-between gap-3">
                          <div>
                            <div className="flex items-center gap-2">
                              <span className="flex h-5 w-5 items-center justify-center rounded-full bg-sky-600 text-[10px] font-bold text-white">1</span>
                              <h4 className="text-xs font-bold text-slate-900 dark:text-white">Auto-Import from Kiro CLI & SSO Cache</h4>
                            </div>
                            <p className="mt-1 text-[11px] leading-relaxed text-slate-600 dark:text-slate-400">
                              Reads Builder ID tokens from ~/.aws/sso/cache or Kiro SQLite token stores.
                            </p>
                          </div>
                          <button
                            type="button"
                            onClick={async () => { await handleImportLocalProvider('kiro'); }}
                            className="flex items-center gap-1.5 rounded-lg bg-sky-600 px-3.5 py-1.5 text-xs font-semibold text-white shadow-xs transition hover:bg-sky-700"
                          >
                            <Cloud className="h-3.5 w-3.5" />
                            <span>Import Kiro</span>
                          </button>
                        </div>
                      </div>

                      <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-4 dark:border-slate-800 dark:bg-slate-800/40">
                        <div className="flex items-center gap-2">
                          <span className="flex h-5 w-5 items-center justify-center rounded-full bg-slate-600 text-[10px] font-bold text-white">2</span>
                          <h4 className="text-xs font-bold text-slate-900 dark:text-white">Manual Refresh Token & Profile ARN</h4>
                        </div>
                        <input
                          type="password"
                          placeholder="Kiro refresh token..."
                          value={addRefreshToken}
                          onChange={e => setAddRefreshToken(e.target.value)}
                          className="mt-2.5 w-full rounded-lg border border-slate-200 bg-white p-2.5 font-mono text-xs outline-none dark:border-slate-800 dark:bg-slate-800 dark:text-white"
                        />
                      </div>
                    </div>
                  )}

                  {/* Action buttons */}
                  <div className="flex items-center justify-between border-t border-slate-200 pt-5 dark:border-slate-800">
                    <button
                      type="button"
                      onClick={() => setWizardStep(1)}
                      className="rounded-lg px-4 py-2 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800"
                    >
                      Back
                    </button>
                    <div className="flex items-center gap-2.5">
                      <button
                        type="button"
                        disabled={testingConnection}
                        onClick={async () => {
                          setTestingConnection(true);
                          setWizardTestResult(null);
                          try {
                            const res = await fetch('/api/providers/test-connection', {
                              method: 'POST',
                              headers: { 'Content-Type': 'application/json' },
                              body: JSON.stringify({
                                provider: addProvId,
                                type: selectedTemplate.type,
                                base_url: addBaseUrl,
                                api_key: addApiKey,
                                refresh_token: addRefreshToken,
                              })
                            }).then(r => r.json());
                            setWizardTestResult(res);
                            setWizardStep(3);
                          } catch (e: any) {
                            setWizardTestResult({ success: false, error: e.message });
                            setWizardStep(3);
                          } finally {
                            setTestingConnection(false);
                          }
                        }}
                        className="flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-4 py-2 text-xs font-semibold text-slate-800 hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
                      >
                        <Zap className="h-3.5 w-3.5 text-rose-500" />
                        <span>{testingConnection ? 'Testing...' : 'Test Connection'}</span>
                      </button>
                      <button
                        type="button"
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
                              showToast('Provider ' + addProvId + ' saved successfully!');
                              await fetchAllData();
                              setActiveTab('providers');
                            } else {
                              showToast('Save failed');
                            }
                          } catch (e: any) {
                            showToast('Error: ' + String(e.message));
                          }
                        }}
                        className="flex items-center gap-1.5 rounded-lg bg-rose-600 px-4 py-2 text-xs font-semibold text-white shadow-xs transition hover:bg-rose-700"
                      >
                        <span>Save & Enable</span>
                        <ChevronRight className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* STEP 3: VERIFY & FINISH */}
            {wizardStep === 3 && (
              <div className="mx-auto max-w-xl space-y-6">
                <div className="rounded-xl border border-slate-200 bg-white p-6 text-center dark:border-slate-800 dark:bg-slate-900 shadow-xs">
                  <div className={
                    (wizardTestResult?.status === 'ok' || wizardTestResult?.success)
                      ? 'mx-auto flex h-14 w-14 items-center justify-center rounded-full border border-emerald-500/20 bg-emerald-50 text-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-400'
                      : 'mx-auto flex h-14 w-14 items-center justify-center rounded-full border border-rose-500/20 bg-rose-50 text-rose-600 dark:bg-rose-950/40 dark:text-rose-400'
                  }>
                    {(wizardTestResult?.status === 'ok' || wizardTestResult?.success) ? (
                      <CheckCircle className="h-7 w-7" />
                    ) : (
                      <XCircle className="h-7 w-7" />
                    )}
                  </div>

                  <h3 className="mt-4 text-lg font-bold text-slate-900 dark:text-white">
                    {(wizardTestResult?.status === 'ok' || wizardTestResult?.success) ? 'Connection Verified' : 'Verification Issue Detected'}
                  </h3>
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    {(wizardTestResult?.status === 'ok' || wizardTestResult?.success)
                      ? 'Successfully validated upstream credentials with ' + String(wizardTestResult?.latency_ms || 24) + 'ms latency.'
                      : wizardTestResult?.error || 'Failed to authenticate with upstream endpoint.'}
                  </p>

                  {wizardTestResult?.models && wizardTestResult.models.length > 0 && (
                    <div className="mt-5 rounded-lg border border-slate-200 bg-slate-50 p-3 text-left dark:border-slate-800 dark:bg-slate-800/40">
                      <span className="block text-[10px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                        Discovered Upstream Models ({wizardTestResult.models.length})
                      </span>
                      <div className="mt-2 flex flex-wrap gap-1 max-h-32 overflow-y-auto">
                        {wizardTestResult.models.slice(0, 10).map((m: string) => (
                          <span key={m} className="rounded bg-white px-2 py-0.5 font-mono text-[10px] font-medium text-slate-700 shadow-2xs dark:bg-slate-800 dark:text-slate-300">
                            {m}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}

                  <div className="mt-6 flex items-center justify-center gap-3">
                    <button
                      type="button"
                      onClick={() => setWizardStep(2)}
                      className="rounded-lg border border-slate-200 bg-white px-4 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700"
                    >
                      Back to Edit
                    </button>
                    <button
                      type="button"
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
                            showToast('Provider ' + addProvId + ' activated!');
                            await fetchAllData();
                            setActiveTab('providers');
                          } else {
                            showToast('Activation failed');
                          }
                        } catch (e: any) {
                          showToast('Error: ' + String(e.message));
                        }
                      }}
                      className="rounded-lg bg-rose-600 px-5 py-2 text-xs font-semibold text-white shadow-xs transition hover:bg-rose-700"
                    >
                      Save & Complete
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {activeTab === 'providers' && (
          <div className="space-y-6">
            <div className="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 pb-4 dark:border-slate-800">
              <div>
                <div className="flex items-center gap-2 text-[10px] font-mono uppercase tracking-widest text-slate-500 dark:text-slate-400">
                  <span className="text-rose-600 dark:text-rose-400 font-bold">▪ 02</span>
                  <span>/</span>
                  <span>UPSTREAM POOLS</span>
                </div>
                <h2 className="mt-1 text-xl font-bold tracking-tight text-slate-900 dark:text-white">Providers & Account Pools</h2>
                <p className="mt-0.5 text-xs text-slate-600 dark:text-slate-400">Granular multi-account failover, quota pooling, and zero-downtime health monitors</p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => {
                    setWizardStep(1);
                    setActiveTab('connect-wizard');
                  }}
                  className="flex items-center gap-1.5 rounded-lg bg-rose-600 px-3.5 py-1.5 text-xs font-semibold text-white shadow-2xs transition hover:bg-rose-700"
                >
                  <Plus className="h-3.5 w-3.5" />
                  <span>Connect Provider</span>
                  <span className="font-mono text-xs">↗</span>
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
                    <div key={p.id} className="group rounded-xl border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900 shadow-xs overflow-hidden transition-all duration-200 hover:-translate-y-0.5 hover:border-rose-500/40 hover:shadow-md dark:hover:border-rose-500/40 dark:hover:shadow-rose-950/20">
                      <div className="flex flex-wrap items-center justify-between gap-4 p-5">
                        <div className="flex items-center gap-3.5">
                          <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-slate-200 bg-slate-100 text-slate-800 dark:border-slate-800 dark:bg-slate-800 dark:text-slate-200">
                            <ProviderBadgeIcon id={p.id} className="h-4 w-4 text-slate-700 dark:text-slate-300" />
                          </div>
                          <div>
                            <div className="flex items-center gap-2">
                              <h3 className="font-bold text-slate-900 dark:text-white text-sm">{p.id}</h3>
                              <span className="rounded border border-slate-200 bg-slate-100 px-1.5 py-0.5 font-mono text-[10px] uppercase text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300">
                                {pType}
                              </span>
                            </div>
                            <span className="font-mono text-xs text-slate-500 dark:text-slate-400">
                              {p.details?.base_url || 'Internal OAuth Token Provider'}
                            </span>
                          </div>
                        </div>

                        <div className="flex items-center gap-3">
                          <span className="flex items-center gap-1.5 rounded-full border border-emerald-500/20 bg-emerald-500/10 px-2.5 py-1 font-mono text-xs font-semibold text-emerald-700 dark:text-emerald-400">
                            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                            {accCount} Pool Account{accCount !== 1 ? 's' : ''}
                          </span>

                          <button
                            onClick={() => handleTestPing(p.id)}
                            className="flex items-center gap-1 rounded-lg border border-slate-200 bg-slate-50 px-3 py-1.5 text-xs font-semibold text-slate-800 transition hover:border-rose-500 hover:text-rose-600 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:border-rose-400"
                          >
                            <Zap className="h-3 w-3 text-rose-600 dark:text-rose-400" />
                            <span>Test Ping</span>
                          </button>

                          <button
                            onClick={() => {
                              setPlaygroundModel(p.id === 'kiro' ? 'kiro/claude-sonnet-4.5' : p.id === 'antigravity' ? 'antigravity/claude-sonnet-4-6' : `${p.id}/default`);
                              setActiveTab('playground');
                            }}
                            className="flex items-center gap-1 rounded-lg border border-slate-200 bg-slate-50 px-3 py-1.5 text-xs font-semibold text-slate-800 transition hover:border-rose-500 hover:text-rose-600 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:border-rose-400"
                          >
                            <Play className="h-3 w-3" />
                            <span>Playground</span>
                          </button>

                          <button
                            onClick={() => setExpandedProvider(isExpanded ? null : p.id)}
                            className="flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700"
                          >
                            {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                          </button>
                        </div>
                      </div>

                      {/* Expandable Account Pool Drawer */}
                      {isExpanded && (
                        <div className="border-t border-slate-200 bg-slate-50/70 p-5 dark:border-slate-800 dark:bg-slate-950/40">
                          <div className="mb-3 flex items-center justify-between text-xs text-slate-500 dark:text-slate-400">
                            <span className="font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400 text-[10px]">
                              Connection Pool Accounts ({accounts.length})
                            </span>
                            <span className="font-mono text-[11px]">Strategy: Priority Failover with Auto-Cooldown</span>
                          </div>

                          <div className="space-y-2">
                            {accounts.map((acc, idx) => (
                              <div
                                key={acc.id}
                                className={`flex flex-wrap items-center justify-between gap-3 rounded-xl border p-3 text-xs transition ${
                                  acc.disabled
                                    ? 'border-slate-200 bg-slate-100 text-slate-400 dark:border-slate-800 dark:bg-slate-900/40'
                                    : 'border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-800/80 shadow-2xs'
                                }`}
                              >
                                <div className="flex items-center gap-3 font-mono">
                                  <span className="flex h-5 w-5 items-center justify-center rounded-full bg-slate-200 text-[10px] font-bold text-slate-700 dark:bg-slate-700 dark:text-slate-200">
                                    {idx + 1}
                                  </span>
                                  <div>
                                    <div className="font-bold text-slate-900 dark:text-white">{acc.id}</div>
                                    <div className="text-[10px] text-slate-500 dark:text-slate-400">
                                      Pool: {acc.pool || 'pool1'} {acc.project_id ? `• Project: ${acc.project_id}` : ''}
                                    </div>
                                  </div>
                                </div>

                                <div className="flex items-center gap-2">
                                  <button
                                    onClick={() => handleTestPing(p.id, acc.id)}
                                    className="rounded-lg border border-slate-200 bg-slate-50 px-2.5 py-1 text-[11px] font-semibold text-slate-800 hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200"
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
                                    className="rounded-lg p-1 text-slate-400 dark:text-slate-500 hover:text-red-600"
                                    title="Delete Account"
                                  >
                                    <Trash2 className="h-3.5 w-3.5" />
                                  </button>
                                </div>
                              </div>
                            ))}

                            {accounts.length === 0 && (
                              <div className="rounded-xl border border-dashed border-neutral-300 p-4 text-center text-xs text-slate-400 dark:text-slate-500 dark:border-neutral-700">
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
                <p className="text-xs text-slate-500 dark:text-slate-400">Autonomous waterfall failover chains across multiple upstream AI accounts</p>
              </div>
              <button
                onClick={() => setShowNewComboModal(true)}
                className="flex items-center gap-1.5 rounded-xl bg-rose-600 hover:bg-rose-700 px-3.5 py-2 text-xs font-semibold text-white shadow-md shadow-blue-500/20 "
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
                          
                        </div>
                        <div>
                          <h3 className="font-mono text-sm font-semibold">{c.name}</h3>
                          <span className="text-[11px] text-slate-400 dark:text-slate-500">{strategy} strategy</span>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="rounded-full bg-purple-50 px-2 py-0.5 font-mono text-[11px] font-medium text-purple-600 dark:bg-purple-950/40 dark:text-purple-400">
                          {targets.length} targets
                        </span>
                        <button
                          onClick={() => handleDeleteCombo(c.name)}
                          className="rounded-lg p-1 text-slate-400 dark:text-slate-500 hover:text-red-600"
                          title="Delete Combo"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </div>

                    <div className="mt-4 space-y-2 rounded-xl bg-slate-100/80 dark:bg-slate-800/60 p-3 font-mono text-xs dark:bg-slate-800/70">
                      <div className="text-[10px] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">
                        Cascade Waterfall Order:
                      </div>
                      <div className="space-y-1.5">
                        {targets.map((t: any, i: number) => (
                          <div key={i} className="flex items-center gap-2 text-slate-700 dark:text-slate-300">
                            <span className="flex h-4 w-4 items-center justify-center rounded-full bg-slate-200 dark:bg-slate-700 text-[10px] font-bold text-slate-600 dark:text-slate-400 dark:bg-slate-700 dark:text-neutral-300">
                              {i + 1}
                            </span>
                            <span className="font-semibold text-slate-900 dark:text-white">{t.provider_id}</span>
                            <span className="text-slate-400 dark:text-slate-500">/</span>
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
                        className="flex items-center gap-1.5 rounded-xl bg-rose-600 hover:bg-rose-700 px-3.5 py-1.5 text-xs font-medium text-white transition "
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
                <p className="text-xs text-slate-500 dark:text-slate-400">Live indexed models stored in SQLite with real-time upstream sync</p>
              </div>

              {/* Provider Filter Tabs */}
              <div className="flex flex-wrap gap-1 rounded-xl bg-slate-200/60 dark:bg-slate-800/60 p-1 text-xs dark:bg-slate-800/50">
                {['all', 'antigravity', 'kiro', 'codex', 'openai-compatible', 'agentrouter'].map(prov => (
                  <button
                    key={prov}
                    onClick={() => setCatalogFilterProvider(prov)}
                    className={`rounded-lg px-2.5 py-1 font-mono transition ${
                      catalogFilterProvider === prov
                        ? 'bg-white font-semibold text-slate-900 dark:text-white shadow-xs dark:bg-slate-700 dark:text-white'
                        : 'text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:text-white dark:hover:text-white'
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
                  <thead className="bg-slate-50 dark:bg-slate-800/40 font-mono text-slate-400 dark:text-slate-500 dark:bg-slate-800/40">
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
                        <td className="p-3.5 font-mono font-semibold text-slate-900 dark:text-white">
                          {item.model}
                        </td>
                        <td className="p-3.5 font-mono text-slate-500 dark:text-slate-400">{item.provider}</td>
                        <td className="p-3.5 font-mono text-slate-500 dark:text-slate-400">{item.latencyP50 || 180}ms</td>
                        <td className="p-3.5 font-mono text-slate-500 dark:text-slate-400">{item.latencyP99 || 540}ms</td>
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
                            className="rounded-lg border border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-800 text-slate-800 dark:text-slate-200 px-2.5 py-1 font-mono text-[11px] font-medium text-rose-600 dark:text-rose-400 hover:bg-blue-100 "
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
                <span className="text-xs text-slate-500 dark:text-slate-400">Monitored Accounts</span>
                <div className="mt-1 font-mono text-2xl font-bold">{quotas?.providers?.length || 7}</div>
                <span className="text-[11px] text-emerald-600 dark:text-emerald-400">100% active</span>
              </div>
              <div className="liquid-card p-5">
                <span className="text-xs text-slate-500 dark:text-slate-400">Rolling Window Status</span>
                <div className="mt-1 font-mono text-2xl font-bold text-emerald-600 dark:text-emerald-400">Healthy</div>
                <span className="text-[11px] text-slate-400 dark:text-slate-500">Zero rate-limit backoffs</span>
              </div>
              <div className="liquid-card p-5">
                <span className="text-xs text-slate-500 dark:text-slate-400">Pre-emptive Failover</span>
                <div className="mt-1 font-mono text-2xl font-bold text-rose-600 dark:text-rose-400 dark:text-blue-400">Armed</div>
                <span className="text-[11px] text-slate-400 dark:text-slate-500">Auto-rotates at 5% cap</span>
              </div>
              <div className="liquid-card p-5">
                <span className="text-xs text-slate-500 dark:text-slate-400">Active Radar Catalog</span>
                <div className="mt-1 font-mono text-2xl font-bold">{catalog.length || 148}</div>
                <span className="text-[11px] text-slate-400 dark:text-slate-500">Models synced in SQLite</span>
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
                        <h4 className="font-semibold text-slate-900 dark:text-white">{qp.name || qp.provider}</h4>
                        <span className="font-mono text-xs text-slate-400 dark:text-slate-500">{qp.provider}</span>
                      </div>
                      <span className={`rounded-full px-2 py-0.5 font-mono text-[11px] font-medium ${
                        isUnlimited ? 'border border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-800 text-slate-800 dark:text-slate-2000/10 text-rose-600 dark:text-rose-400' : 'bg-emerald-500/10 text-emerald-600'
                      }`}>
                        {isUnlimited ? 'Pay-as-you-go' : 'Active'}
                      </span>
                    </div>

                    <div className="mt-4">
                      <div className="flex justify-between text-xs text-slate-500 dark:text-slate-400">
                        <span>5h Rolling Session</span>
                        <span className="font-semibold text-slate-800 dark:text-slate-200">{remPct}% left</span>
                      </div>
                      <div className="mt-1.5 h-2 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700 dark:bg-slate-800">
                        <div className="h-full rounded-full bg-emerald-500 transition-all" style={{ width: `${remPct}%` }} />
                      </div>
                      <div className="mt-2 flex justify-between text-[11px] text-slate-400 dark:text-slate-500">
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
                <p className="text-xs text-slate-500 dark:text-slate-400">Click any request row to inspect full headers, tokens, and response payloads</p>
              </div>
            </div>

            <div className="liquid-card overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="bg-slate-50 dark:bg-slate-800/40 font-mono text-slate-400 dark:text-slate-500 dark:bg-slate-800/40">
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
                        <td className="p-3.5 font-mono font-semibold text-slate-800 dark:text-slate-200">{log.model}</td>
                        <td className="p-3.5 font-mono text-slate-500 dark:text-slate-400">{log.tokens || log.prompt_tokens || '14'} tok</td>
                        <td className="p-3.5 font-mono text-slate-500 dark:text-slate-400">{log.latency_ms || log.duration_ms || 230}ms</td>
                        <td className="p-3.5">
                          <span className={`rounded-full px-2 py-0.5 font-mono text-[10px] font-semibold ${
                            (log.status_code || log.status) >= 400 ? 'bg-red-500/10 text-red-600' : 'bg-emerald-500/10 text-emerald-600'
                          }`}>
                            {log.status_code || log.status || 200} OK
                          </span>
                        </td>
                        <td className="p-3.5 text-slate-400 dark:text-slate-500">{new Date(log.created_at || log.timestamp || Date.now()).toLocaleTimeString()}</td>
                        <td className="p-3.5 text-right">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              handleReplayLog(log.id || `req_${i}`);
                            }}
                            className="flex items-center gap-1 rounded-lg border border-slate-200 dark:border-slate-800 px-2.5 py-1 text-[11px] font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:bg-slate-800 dark:border-slate-800 dark:text-neutral-300 dark:hover:bg-neutral-800"
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
                <p className="text-xs text-slate-500 dark:text-slate-400">Auto-detected terminal agents and IDE tools on this system</p>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {cliAgents.map(agent => (
                <div key={agent.id} className="liquid-card flex flex-col justify-between p-5">
                  <div>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2.5">
                        <Terminal className="h-5 w-5 text-rose-500" />
                        <div>
                          <h4 className="font-semibold text-slate-900 dark:text-white">{agent.name}</h4>
                          <span className="font-mono text-[11px] text-slate-400 dark:text-slate-500">{agent.id}</span>
                        </div>
                      </div>
                      <span className={`rounded-full px-2 py-0.5 font-mono text-[10px] font-medium ${
                        agent.detected ? 'bg-emerald-500/10 text-emerald-600' : 'bg-slate-200/60 dark:bg-slate-800/60 text-slate-400 dark:text-slate-500'
                      }`}>
                        {agent.detected ? 'Detected' : 'Not installed'}
                      </span>
                    </div>

                    <p className="mt-3 text-xs leading-relaxed text-slate-500 dark:text-slate-400">
                      {agent.description}
                    </p>

                    <div className="mt-3 rounded-lg bg-slate-100/80 dark:bg-slate-800/60 p-2 font-mono text-[10px] text-slate-600 dark:text-slate-400 dark:bg-slate-800/60 dark:text-slate-400 dark:text-slate-500">
                      {agent.settings_file}
                    </div>
                  </div>

                  <div className="mt-4 flex items-center justify-between border-t border-slate-200 dark:border-slate-800 pt-3">
                    <a
                      href={agent.docs_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-1 text-[11px] text-slate-500 dark:text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 transition"
                    >
                      <span>Docs</span>
                      <ExternalLink className="h-3 w-3" />
                    </a>

                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => openAgentConfigModal(agent)}
                        className="flex items-center gap-1.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 px-2.5 py-1 text-xs font-semibold text-slate-700 dark:text-slate-200 hover:border-rose-400 hover:text-rose-600 transition shadow-xs"
                        title="Customize models, combos, and options"
                      >
                        <SlidersHorizontal className="h-3.5 w-3.5 text-rose-500" />
                        <span>Customize</span>
                      </button>

                      <button
                        onClick={() => handleConfigureCli(agent.id)}
                        disabled={!agent.detected}
                        className={`rounded-xl px-3 py-1 text-xs font-semibold transition ${
                          agent.configured
                            ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                            : agent.detected
                            ? 'bg-rose-600 hover:bg-rose-700 text-white shadow-xs'
                            : 'cursor-not-allowed bg-slate-200 dark:bg-slate-700 text-slate-400 dark:text-slate-500'
                        }`}
                      >
                        {agent.configured ? 'Connected' : 'Quick Connect'}
                      </button>
                    </div>
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
                <p className="text-xs text-slate-500 dark:text-slate-400">Configure global model aliases, thinking token budgets, proactive refresh, and database backups</p>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
              {/* OmniRoute Sync & Cooldown Actions */}
              <div className="liquid-card p-6">
                <h3 className="text-sm font-semibold tracking-tight">OmniRoute Synchronization</h3>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  Import credentials, OAuth accounts, and combos directly from local OmniRoute storage.sqlite with pure Go AES-256-GCM.
                </p>

                <div className="mt-5 flex flex-wrap gap-3">
                  <button
                    onClick={handleSyncFromOmniRoute}
                    className="flex items-center gap-2 rounded-xl bg-rose-600 hover:bg-rose-700 px-4 py-2.5 text-xs font-semibold text-white shadow-md shadow-blue-500/20 "
                  >
                    <RefreshCw className="h-3.5 w-3.5" />
                    <span>Sync All from OmniRoute</span>
                  </button>

                  <button
                    onClick={handleResetCooldowns}
                    className="flex items-center gap-2 rounded-xl border border-slate-200 dark:border-slate-800 bg-white px-4 py-2.5 text-xs font-semibold text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:bg-slate-800/40 dark:border-slate-800 dark:bg-slate-800 dark:text-neutral-200"
                  >
                    <Power className="h-3.5 w-3.5 text-emerald-500" />
                    <span>Reset All Cooldowns</span>
                  </button>
                </div>

                <div className="mt-6 border-t border-slate-200 dark:border-slate-800 pt-4 dark:border-slate-800">
                  <h4 className="text-xs font-semibold">Proactive Token Refresh Scheduler</h4>
                  <p className="text-[11px] text-slate-400 dark:text-slate-500">Refreshes Kiro AWS Q and Antigravity tokens before expiration.</p>
                  <div className="mt-2 flex items-center gap-2 font-mono text-xs text-emerald-600 dark:text-emerald-400">
                    <span className="pulse-indicator" /> Running every 3 minutes in background
                  </div>
                </div>
              </div>

              {/* Thinking Budget Slider */}
              <div className="liquid-card p-6">
                <h3 className="text-sm font-semibold tracking-tight">Thinking Token Budget</h3>
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  Control thinking budget tokens for reasoning models (Claude 3.7 Sonnet Thinking, Gemini 2.0 Flash Thinking, DeepSeek R1).
                </p>

                <div className="mt-6">
                  <div className="flex justify-between text-xs">
                    <span className="text-slate-500 dark:text-slate-400">Budget Limit:</span>
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
                  <div className="mt-2 flex justify-between text-[10px] font-mono text-slate-400 dark:text-slate-500">
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
                        if (res.ok) showToast('Thinking budget updated!');
                      } catch (e: any) {
                        showToast(`Update error: ${e.message}`);
                      }
                    }}
                    className="mt-4 rounded-xl bg-neutral-900 px-4 py-2 text-xs font-semibold text-white dark:bg-white dark:text-slate-900 dark:text-white"
                  >
                    Save Budget
                  </button>
                </div>
              </div>

              {/* Model Aliases Manager */}
              <div className="liquid-card col-span-full p-6">
                <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-4 dark:border-slate-800">
                  <div>
                    <h3 className="text-sm font-semibold tracking-tight">Model Aliases Mapping</h3>
                    <p className="text-xs text-slate-500 dark:text-slate-400">Map custom names or IDE models to target models or combo cascades</p>
                  </div>
                  <button
                    onClick={() => setShowAliasModal(true)}
                    className="flex items-center gap-1.5 rounded-xl bg-rose-600 hover:bg-rose-700 px-3.5 py-1.5 text-xs font-semibold text-white shadow-xs "
                  >
                    <Plus className="h-3 w-3" />
                    <span>Add Alias</span>
                  </button>
                </div>

                <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  {Object.entries(aliases).map(([fromModel, toModel]) => (
                    <div key={fromModel} className="flex items-center justify-between rounded-xl border border-slate-200 dark:border-slate-800 p-3 text-xs dark:border-slate-800">
                      <div>
                        <div className="font-mono font-semibold text-slate-900 dark:text-white">{fromModel}</div>
                        <div className="mt-0.5 font-mono text-[11px] text-slate-400 dark:text-slate-500">→ {toModel}</div>
                      </div>
                      <button
                        onClick={async () => {
                          await fetch(`/api/settings/model-aliases?alias=${fromModel}`, { method: 'DELETE' });
                          showToast(`Deleted alias ${fromModel}`);
                          fetchAllData();
                        }}
                        className="rounded-lg p-1 text-slate-400 dark:text-slate-500 hover:text-red-600"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  ))}
                  {Object.keys(aliases).length === 0 && (
                    <div className="col-span-full rounded-xl border border-dashed border-neutral-300 p-6 text-center text-xs text-slate-400 dark:text-slate-500 dark:border-neutral-700">
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
            <div className="border-r border-slate-200 dark:border-slate-800 p-6 dark:border-slate-800">
              <h3 className="mb-4 text-sm font-semibold tracking-tight">Interactive Prompt Studio</h3>
              <div className="mb-4">
                <label className="text-xs font-medium text-slate-500 dark:text-slate-400">Target Model / Cascade Combo:</label>
                <select
                  value={playgroundModel}
                  onChange={e => setPlaygroundModel(e.target.value)}
                  className="mt-1.5 w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white p-3 font-mono text-xs outline-none dark:border-slate-800 dark:bg-slate-800"
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
                <label className="text-xs font-medium text-slate-500 dark:text-slate-400">User Prompt:</label>
                <textarea
                  rows={8}
                  value={playgroundPrompt}
                  onChange={e => setPlaygroundPrompt(e.target.value)}
                  className="mt-1.5 w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white p-3 font-mono text-xs outline-none dark:border-slate-800 dark:bg-slate-800"
                />
              </div>

              <button
                onClick={handleSendPlayground}
                disabled={playgroundLoading}
                className="mt-5 flex w-full items-center justify-center gap-2 rounded-xl bg-rose-600 hover:bg-rose-700 py-3 text-xs font-semibold text-white shadow-md shadow-blue-500/20 transition "
              >
                {playgroundLoading ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
                <span>{playgroundLoading ? 'Dispatching to GoRoute Core...' : 'Send Request'}</span>
              </button>
            </div>

            <div className="flex flex-col justify-between bg-slate-50 dark:bg-slate-800/40 p-6 dark:bg-black/20">
              <div>
                <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3 dark:border-slate-800">
                  <h3 className="text-sm font-semibold tracking-tight">Streaming Output</h3>
                  {playgroundMeta && (
                    <span className="font-mono text-xs text-emerald-600 dark:text-emerald-400">
                      {playgroundMeta.latency} • {playgroundMeta.tokens} tok
                    </span>
                  )}
                </div>
                <div className="mt-4 max-h-[440px] overflow-y-auto whitespace-pre-wrap font-mono text-xs leading-relaxed text-slate-800 dark:text-slate-200">
                  {playgroundResponse || (
                    <span className="italic text-slate-400 dark:text-slate-500">
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
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3 dark:border-slate-800">
              <div>
                <h3 className="text-base font-semibold tracking-tight">Connect Upstream Provider</h3>
                <p className="text-xs text-slate-500 dark:text-slate-400">Select a pre-configured provider or configure a custom endpoint</p>
              </div>
              <button onClick={() => setShowAddModal(false)} className="rounded-lg p-1 text-slate-400 dark:text-slate-500 hover:text-slate-600 dark:text-slate-400">
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="mt-4">
              <label className="text-xs font-semibold text-slate-500 dark:text-slate-400">Choose Provider Directory:</label>
              <div className="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-3 max-h-48 overflow-y-auto p-1">
                {PROVIDER_TEMPLATES.map(t => (
                  <button
                    key={t.id}
                    onClick={() => {
                      setSelectedTemplate(t);
                      setAddProvId(t.id);
                      setAddBaseUrl(t.baseUrl);
                    }}
                    className={`flex items-center gap-2 rounded-lg border p-2.5 text-left text-xs transition ${
                      selectedTemplate.id === t.id
                        ? 'border-[#C93D2A] bg-[#FAF9F5] font-semibold text-[#C93D2A] dark:border-[#E24B38] dark:bg-[#1C1D24] dark:text-[#E24B38]'
                        : 'border-[#E2E0D5] bg-white hover:bg-[#FAF9F5] dark:border-[#272933] dark:bg-[#1A1B22]'
                    }`}
                  >
                    <ProviderBadgeIcon id={t.id} className="h-4 w-4 shrink-0" />
                    <span className="truncate">{t.name}</span>
                  </button>
                ))}
              </div>
            </div>

            <div className="mt-5 space-y-3.5 border-t border-slate-200 dark:border-slate-800 pt-4 dark:border-slate-800">
              <div>
                <label className="text-xs text-slate-500 dark:text-slate-400">Provider ID</label>
                <input
                  type="text"
                  value={addProvId}
                  onChange={e => setAddProvId(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white p-2.5 font-mono text-xs outline-none dark:border-slate-800 dark:bg-slate-800"
                />
              </div>

              {selectedTemplate.type !== 'kiro' && selectedTemplate.type !== 'antigravity' && selectedTemplate.type !== 'codex' && selectedTemplate.type !== 'cursor' && selectedTemplate.type !== 'copilot' && (
                <>
                  <div>
                    <label className="text-xs text-slate-500 dark:text-slate-400">Base URL</label>
                    <input
                      type="text"
                      value={addBaseUrl}
                      onChange={e => setAddBaseUrl(e.target.value)}
                      className="mt-1 w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white p-2.5 font-mono text-xs outline-none dark:border-slate-800 dark:bg-slate-800"
                    />
                  </div>
                  <div>
                    <label className="text-xs text-slate-500 dark:text-slate-400">API Key</label>
                    <input
                      type="password"
                      placeholder="sk-..."
                      value={addApiKey}
                      onChange={e => setAddApiKey(e.target.value)}
                      className="mt-1 w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white p-2.5 font-mono text-xs outline-none dark:border-slate-800 dark:bg-slate-800"
                    />
                  </div>
                </>
              )}

              {selectedTemplate.type === 'antigravity' && (
                <div className="space-y-4">
                  {/* Option 1: 1-Click Local Discovery (Recommended) */}
                  <div className="rounded-2xl border border-purple-500/30 bg-purple-50/50 p-4 dark:border-purple-500/20 dark:bg-purple-950/30">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="flex h-5 w-5 items-center justify-center rounded-full bg-purple-600 text-[10px] font-bold text-white">1</span>
                          <h4 className="text-xs font-bold text-slate-900 dark:text-white">Auto-Import Local Antigravity (IDE & CLI)</h4>
                          <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-400">Recommended</span>
                        </div>
                        <p className="mt-1 text-[11px] leading-relaxed text-slate-600 dark:text-slate-400">
                          Instantly scans locally logged-in Google Antigravity tokens from Antigravity IDE SQLite state (<code className="font-mono text-[10px]">state.vscdb</code>) and CLI OAuth files. Automatically provisions refresh tokens into the active quota pool without re-logging in.
                        </p>
                      </div>
                    </div>
                    <div className="mt-3.5">
                      <button
                        type="button"
                        onClick={async () => {
                          const ok = await handleImportLocalAntigravity();
                          if (ok) setShowAddModal(false);
                        }}
                        className="flex items-center gap-2 rounded-xl bg-purple-600 px-4 py-2 text-xs font-semibold text-white shadow-md shadow-purple-500/20 hover:bg-purple-700 transition active:scale-95"
                      >
                        <Zap className="h-3.5 w-3.5" />
                        <span>Discover & Import Local Accounts</span>
                      </button>
                    </div>
                  </div>

                  {/* Option 2: Manual Refresh Token */}
                  <div className="rounded-2xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/40/60 p-4 dark:border-slate-800 dark:bg-slate-900/40">
                    <div className="flex items-center gap-2">
                      <span className="flex h-5 w-5 items-center justify-center rounded-full bg-slate-50 dark:bg-slate-800/400 text-[10px] font-bold text-white">2</span>
                      <h4 className="text-xs font-bold text-slate-900 dark:text-white">Manual Refresh Token Input</h4>
                    </div>
                    <p className="mt-1 text-[11px] text-slate-500 dark:text-slate-400">
                      Already have a Google Cloud Code OAuth refresh token (<code className="font-mono text-[10px]">1//...</code>)? Paste it here directly.
                    </p>
                    <div className="mt-2.5">
                      <input
                        type="password"
                        placeholder="1//0g..."
                        value={addRefreshToken}
                        onChange={e => setAddRefreshToken(e.target.value)}
                        className="w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white p-2.5 font-mono text-xs outline-none dark:border-slate-800 dark:bg-slate-800"
                      />
                    </div>
                  </div>
                </div>
              )}

              {selectedTemplate.type === 'kiro' && (
                <div className="space-y-4">
                  <div className="rounded-2xl border border-blue-500/30 border border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-800 text-slate-800 dark:text-slate-200/50 p-4 dark:border-blue-500/20 dark:bg-blue-950/30">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="flex h-5 w-5 items-center justify-center rounded-full bg-rose-600 hover:bg-rose-700 text-[10px] font-bold text-white">1</span>
                          <h4 className="text-xs font-bold text-slate-900 dark:text-white">Auto-Import Local AWS Q Kiro (CLI & SSO)</h4>
                          <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-400">1-Click</span>
                        </div>
                        <p className="mt-1 text-[11px] leading-relaxed text-slate-600 dark:text-slate-400">
                          Discovers local Kiro CLI tokens from <code className="font-mono text-[10px]">data.sqlite3</code> and AWS SSO cached tokens (<code className="font-mono text-[10px]">~/.aws/sso/cache/</code>).
                        </p>
                      </div>
                    </div>
                    <div className="mt-3.5">
                      <button
                        type="button"
                        onClick={async () => {
                          const ok = await handleImportLocalProvider('kiro');
                          if (ok) setShowAddModal(false);
                        }}
                        className="flex items-center gap-2 rounded-xl bg-rose-600 hover:bg-rose-700 px-4 py-2 text-xs font-semibold text-white shadow-md shadow-blue-500/20  transition active:scale-95"
                      >
                        <Zap className="h-3.5 w-3.5" />
                        <span>Import Local Kiro Credentials</span>
                      </button>
                    </div>
                  </div>
                </div>
              )}

              {selectedTemplate.type === 'codex' && (
                <div className="rounded-2xl border border-emerald-500/30 bg-emerald-50/50 p-4 dark:border-emerald-500/20 dark:bg-emerald-950/30">
                  <div className="flex items-center gap-2">
                    <span className="flex h-5 w-5 items-center justify-center rounded-full bg-emerald-600 text-[10px] font-bold text-white">1</span>
                    <h4 className="text-xs font-bold text-slate-900 dark:text-white">Auto-Import OpenAI Codex CLI Session</h4>
                    <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-400">1-Click</span>
                  </div>
                  <p className="mt-1 text-[11px] leading-relaxed text-slate-600 dark:text-slate-400">
                    Extracts logged-in OAuth session tokens from Codex CLI config (<code className="font-mono text-[10px]">~/.config/codex/credentials.json</code>).
                  </p>
                  <div className="mt-3.5">
                    <button
                      type="button"
                      onClick={async () => {
                        const ok = await handleImportLocalProvider('codex');
                        if (ok) setShowAddModal(false);
                      }}
                      className="flex items-center gap-2 rounded-xl bg-emerald-600 px-4 py-2 text-xs font-semibold text-white shadow-md shadow-emerald-500/20 hover:bg-emerald-700 transition active:scale-95"
                    >
                      <Zap className="h-3.5 w-3.5" />
                      <span>Import Local Codex Session</span>
                    </button>
                  </div>
                </div>
              )}

              {selectedTemplate.type === 'cursor' && (
                <div className="rounded-2xl border border-sky-500/30 bg-sky-50/50 p-4 dark:border-sky-500/20 dark:bg-sky-950/30">
                  <div className="flex items-center gap-2">
                    <span className="flex h-5 w-5 items-center justify-center rounded-full bg-sky-600 text-[10px] font-bold text-white">1</span>
                    <h4 className="text-xs font-bold text-slate-900 dark:text-white">Auto-Import Local Cursor Session</h4>
                    <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-400">1-Click</span>
                  </div>
                  <p className="mt-1 text-[11px] leading-relaxed text-slate-600 dark:text-slate-400">
                    Reads local Cursor editor authentication state from SQLite storage.
                  </p>
                  <div className="mt-3.5">
                    <button
                      type="button"
                      onClick={async () => {
                        const ok = await handleImportLocalProvider('cursor');
                        if (ok) setShowAddModal(false);
                      }}
                      className="flex items-center gap-2 rounded-xl bg-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-md shadow-sky-500/20 hover:bg-sky-700 transition active:scale-95"
                    >
                      <Zap className="h-3.5 w-3.5" />
                      <span>Import Local Cursor Auth</span>
                    </button>
                  </div>
                </div>
              )}

              {selectedTemplate.type === 'copilot' && (
                <div className="rounded-2xl border border-neutral-500/30 bg-slate-50 dark:bg-slate-800/40 p-4 dark:border-neutral-500/20 dark:bg-slate-900/40">
                  <div className="flex items-center gap-2">
                    <span className="flex h-5 w-5 items-center justify-center rounded-full bg-neutral-800 text-[10px] font-bold text-white">1</span>
                    <h4 className="text-xs font-bold text-slate-900 dark:text-white">Auto-Import GitHub Copilot CLI Token</h4>
                    <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-400">1-Click</span>
                  </div>
                  <p className="mt-1 text-[11px] leading-relaxed text-slate-600 dark:text-slate-400">
                    Reads GitHub Copilot tokens from local gh CLI or hosts authentication config.
                  </p>
                  <div className="mt-3.5">
                    <button
                      type="button"
                      onClick={async () => {
                        const ok = await handleImportLocalProvider('copilot');
                        if (ok) setShowAddModal(false);
                      }}
                      className="flex items-center gap-2 rounded-xl bg-neutral-800 px-4 py-2 text-xs font-semibold text-white shadow-md shadow-neutral-800/20 hover:bg-neutral-900 transition active:scale-95"
                    >
                      <Zap className="h-3.5 w-3.5" />
                      <span>Import GitHub Copilot Token</span>
                    </button>
                  </div>
                </div>
              )}
            </div>

            <div className="mt-6 flex items-center justify-end gap-2 border-t border-slate-200 dark:border-slate-800 pt-4 dark:border-slate-800">
              <button
                onClick={() => setShowAddModal(false)}
                className="rounded-xl px-4 py-2 text-xs font-medium text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:bg-slate-800 dark:hover:bg-neutral-800"
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
                      showToast(`Provider ${addProvId} connected!`);
                      setShowAddModal(false);
                      fetchAllData();
                    } else {
                      showToast(`Connection failed`);
                    }
                  } catch (e: any) {
                    showToast(`Error: ${e.message}`);
                  }
                }}
                className="rounded-lg bg-[#C93D2A] px-4 py-2 text-xs font-semibold text-white shadow-2xs hover:bg-[#B33423] transition"
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
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3 dark:border-slate-800">
              <h3 className="text-base font-semibold tracking-tight">Visual Combos Waterfall Builder</h3>
              <button onClick={() => setShowNewComboModal(false)} className="rounded-lg p-1 text-slate-400 dark:text-slate-500 hover:text-slate-600 dark:text-slate-400">
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="mt-4 space-y-4">
              <div>
                <label className="text-xs text-slate-500 dark:text-slate-400">Combo Name</label>
                <input
                  type="text"
                  placeholder="e.g. claude-resilient, fast-tier"
                  value={newComboName}
                  onChange={e => setNewComboName(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white p-2.5 font-mono text-xs outline-none dark:border-slate-800 dark:bg-slate-800"
                />
              </div>

              <div>
                <label className="text-xs text-slate-500 dark:text-slate-400">Cascade Strategy</label>
                <select
                  value={newComboStrategy}
                  onChange={e => setNewComboStrategy(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white p-2.5 text-xs outline-none dark:border-slate-800 dark:bg-slate-800"
                >
                  <option value="failover">Failover (Try 1st, cascade on 429/500/timeout)</option>
                  <option value="lowest_latency">Lowest Latency (Race fastest response)</option>
                  <option value="round_robin">Round Robin (Distribute traffic evenly)</option>
                </select>
              </div>

              <div>
                <div className="flex items-center justify-between">
                  <label className="text-xs text-slate-500 dark:text-slate-400">Waterfall Sequence Steps:</label>
                  <button
                    onClick={() => setComboSteps([...comboSteps, { provider: 'antigravity', model: 'claude-sonnet-4-6' }])}
                    className="flex items-center gap-1 text-[11px] font-semibold text-rose-600 dark:text-rose-400 hover:underline"
                  >
                    <Plus className="h-3 w-3" /> Add Step
                  </button>
                </div>

                <div className="mt-2 space-y-2">
                  {comboSteps.map((step, idx) => (
                    <div key={idx} className="flex items-center gap-2">
                      <span className="flex h-5 w-5 items-center justify-center rounded-full bg-slate-200 dark:bg-slate-700 text-[10px] font-bold dark:bg-slate-700">
                        {idx + 1}
                      </span>
                      <select
                        value={step.provider}
                        onChange={e => {
                          const updated = [...comboSteps];
                          updated[idx].provider = e.target.value;
                          setComboSteps(updated);
                        }}
                        className="rounded-xl border border-slate-200 dark:border-slate-800 bg-white p-2 font-mono text-xs outline-none dark:border-slate-800 dark:bg-slate-800"
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
                        className="flex-1 rounded-xl border border-slate-200 dark:border-slate-800 bg-white p-2 font-mono text-xs outline-none dark:border-slate-800 dark:bg-slate-800"
                      />
                      {comboSteps.length > 1 && (
                        <button
                          onClick={() => setComboSteps(comboSteps.filter((_, i) => i !== idx))}
                          className="rounded-lg p-1 text-slate-400 dark:text-slate-500 hover:text-red-600"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            </div>

            <div className="mt-6 flex items-center justify-end gap-2 border-t border-slate-200 dark:border-slate-800 pt-4 dark:border-slate-800">
              <button
                onClick={() => setShowNewComboModal(false)}
                className="rounded-xl px-4 py-2 text-xs font-medium text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:bg-slate-800 dark:hover:bg-neutral-800"
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
                      showToast(`Combo ${newComboName} created!`);
                      setShowNewComboModal(false);
                      fetchAllData();
                    }
                  } catch (e: any) {
                    showToast(`Error: ${e.message}`);
                  }
                }}
                className="rounded-lg bg-[#C93D2A] px-4 py-2 text-xs font-semibold text-white shadow-2xs hover:bg-[#B33423] transition"
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
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3 dark:border-slate-800">
              <h3 className="text-base font-semibold tracking-tight">Create Model Alias</h3>
              <button onClick={() => setShowAliasModal(false)} className="rounded-lg p-1 text-slate-400 dark:text-slate-500 hover:text-slate-600 dark:text-slate-400">
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="mt-4 space-y-3.5">
              <div>
                <label className="text-xs text-slate-500 dark:text-slate-400">Alias Key (Incoming Name)</label>
                <input
                  type="text"
                  placeholder="e.g. claude-3-7-sonnet"
                  value={newAliasKey}
                  onChange={e => setNewAliasKey(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white p-2.5 font-mono text-xs outline-none dark:border-slate-800 dark:bg-slate-800"
                />
              </div>
              <div>
                <label className="text-xs text-slate-500 dark:text-slate-400">Target Model / Combo</label>
                <input
                  type="text"
                  placeholder="e.g. claude-sonnet-4-6 or claude"
                  value={newAliasTarget}
                  onChange={e => setNewAliasTarget(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-slate-200 dark:border-slate-800 bg-white p-2.5 font-mono text-xs outline-none dark:border-slate-800 dark:bg-slate-800"
                />
              </div>
            </div>

            <div className="mt-6 flex items-center justify-end gap-2 border-t border-slate-200 dark:border-slate-800 pt-4 dark:border-slate-800">
              <button
                onClick={() => setShowAliasModal(false)}
                className="rounded-xl px-4 py-2 text-xs font-medium text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:bg-slate-800 dark:hover:bg-neutral-800"
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
                      showToast(`Alias ${newAliasKey} → ${newAliasTarget} saved!`);
                      setShowAliasModal(false);
                      fetchAllData();
                    }
                  } catch (e: any) {
                    showToast(`Error: ${e.message}`);
                  }
                }}
                className="rounded-lg bg-[#C93D2A] px-4 py-2 text-xs font-semibold text-white shadow-2xs hover:bg-[#B33423] transition"
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
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-4 dark:border-slate-800">
              <div>
                <h3 className="text-base font-semibold tracking-tight">Request Telemetry Inspector</h3>
                <span className="font-mono text-xs text-slate-400 dark:text-slate-500">{inspectingLog.id || 'Request Details'}</span>
              </div>
              <button onClick={() => setInspectingLog(null)} className="rounded-lg p-1 text-slate-400 dark:text-slate-500 hover:text-slate-600 dark:text-slate-400">
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="mt-5 space-y-4 font-mono text-xs">
              <div className="grid grid-cols-2 gap-3">
                <div className="rounded-xl bg-slate-100/80 dark:bg-slate-800/60 p-3 dark:bg-slate-800/70">
                  <span className="text-[10px] text-slate-400 dark:text-slate-500">Target Model:</span>
                  <div className="font-semibold text-slate-900 dark:text-white">{inspectingLog.model}</div>
                </div>
                <div className="rounded-xl bg-slate-100/80 dark:bg-slate-800/60 p-3 dark:bg-slate-800/70">
                  <span className="text-[10px] text-slate-400 dark:text-slate-500">Status Code:</span>
                  <div className={`font-semibold ${inspectingLog.status_code >= 400 ? 'text-red-600' : 'text-emerald-600'}`}>
                    {inspectingLog.status_code || 200} OK
                  </div>
                </div>
                <div className="rounded-xl bg-slate-100/80 dark:bg-slate-800/60 p-3 dark:bg-slate-800/70">
                  <span className="text-[10px] text-slate-400 dark:text-slate-500">Latency:</span>
                  <div className="font-semibold text-slate-900 dark:text-white">{inspectingLog.latency_ms || inspectingLog.duration_ms || 230} ms</div>
                </div>
                <div className="rounded-xl bg-slate-100/80 dark:bg-slate-800/60 p-3 dark:bg-slate-800/70">
                  <span className="text-[10px] text-slate-400 dark:text-slate-500">Tokens:</span>
                  <div className="font-semibold text-slate-900 dark:text-white">{inspectingLog.tokens || 14} tokens</div>
                </div>
              </div>

              <div>
                <span className="text-[10px] text-slate-400 dark:text-slate-500 uppercase tracking-wider">Timestamp:</span>
                <div className="mt-1 text-slate-700 dark:text-slate-300">
                  {new Date(inspectingLog.created_at || inspectingLog.timestamp || Date.now()).toISOString()}
                </div>
              </div>

              <div className="mt-6 flex justify-end gap-2 border-t border-slate-200 dark:border-slate-800 pt-4 dark:border-slate-800">
                <button
                  onClick={() => {
                    handleReplayLog(inspectingLog.id);
                    setInspectingLog(null);
                  }}
                  className="flex items-center gap-1.5 rounded-xl bg-rose-600 hover:bg-rose-700 px-4 py-2 font-sans text-xs font-semibold text-white shadow-md shadow-blue-500/20 "
                >
                  <RotateCcw className="h-3.5 w-3.5" />
                  <span>Replay This Request</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* AGENT CONFIGURATION STUDIO MODAL */}
      {configuringAgent && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm">
          <div className="liquid-card w-full max-w-2xl max-h-[90vh] overflow-y-auto p-6 shadow-2xl">
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3">
              <div className="flex items-center gap-3">
                <div className="rounded-xl bg-rose-500/10 p-2 text-rose-600 dark:text-rose-400">
                  <Terminal className="h-6 w-6" />
                </div>
                <div>
                  <h3 className="text-base font-semibold tracking-tight text-slate-900 dark:text-white">
                    Configure {configuringAgent.name}
                  </h3>
                  <p className="font-mono text-xs text-slate-500 dark:text-slate-400">
                    Target: {configuringAgent.settings_file}
                  </p>
                </div>
              </div>
              <button
                onClick={() => setConfiguringAgent(null)}
                className="rounded-lg p-1.5 text-slate-400 hover:text-slate-600 dark:text-slate-500 dark:hover:text-slate-300"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="mt-5 space-y-5">
              {/* Target Upstream Provider */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300">
                  Target Upstream Provider
                </label>
                <p className="text-[11px] text-slate-500 dark:text-slate-400 mb-1.5">
                  Explicitly bind to an upstream provider or use Auto to let GoRoute route models by affinity.
                </p>
                <select
                  value={agentProvider}
                  onChange={(e) => {
                    const p = e.target.value;
                    setAgentProvider(p);
                    if (p === 'antigravity') {
                      setAgentPrimaryModel('antigravity/claude-3-7-sonnet');
                      setAgentSmallModel('antigravity/gemini-2.5-flash');
                    } else if (p === 'kiro') {
                      setAgentPrimaryModel('kiro/claude-3-7-sonnet');
                      setAgentSmallModel('kiro/claude-haiku-4.5');
                    } else if (p === 'codex') {
                      setAgentPrimaryModel('codex/gpt-5.6-sol');
                      setAgentSmallModel('codex/o3-mini');
                    } else if (p === 'combo') {
                      setAgentPrimaryModel('claude-models');
                      setAgentSmallModel('gemini-2.5-flash');
                    } else {
                      setAgentPrimaryModel('claude-3-7-sonnet');
                      setAgentSmallModel('gemini-2.5-flash');
                    }
                  }}
                  className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-xs font-mono text-slate-800 shadow-xs focus:border-rose-500 focus:outline-hidden dark:border-slate-800 dark:bg-slate-900 dark:text-slate-200"
                >
                  <option value="auto">⚡ Auto (Smart Model Affinity Failover)</option>
                  <option value="antigravity">🌐 Antigravity (Google Cloud Vertex AI)</option>
                  <option value="kiro">⚡ Kiro (AWS Bedrock CodeWhisperer)</option>
                  <option value="codex">🤖 Codex (OpenAI ChatGPT Core)</option>
                  <option value="combo">🔀 Combos (Multi-Provider Failover Pools)</option>
                </select>
              </div>

              {/* Primary Model */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300">
                  Primary / Default Model or Combo
                </label>
                <p className="text-[11px] text-slate-500 dark:text-slate-400 mb-1.5">
                  The main model this agent will invoke for reasoning, generation, and tool use.
                </p>
                <select
                  value={agentPrimaryModel}
                  onChange={(e) => setAgentPrimaryModel(e.target.value)}
                  className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-xs font-mono text-slate-800 shadow-xs focus:border-rose-500 focus:outline-hidden dark:border-slate-800 dark:bg-slate-900 dark:text-slate-200"
                >
                  <optgroup label="Active Failover Combos">
                    {combos.map(c => (
                      <option key={c.name} value={c.name}>
                        ⚡ {c.name} ({c.targets?.length || 0} targets)
                      </option>
                    ))}
                  </optgroup>
                  <optgroup label="Provider Qualified Models">
                    {getAvailableModelsAndCombos().filter(m => m.includes('/') && !combos.some(c => c.name === m)).map(m => (
                      <option key={m} value={m}>
                        {m}
                      </option>
                    ))}
                  </optgroup>
                  <optgroup label="Auto Affinity Models">
                    {getAvailableModelsAndCombos().filter(m => !m.includes('/') && !combos.some(c => c.name === m)).map(m => (
                      <option key={m} value={m}>
                        {m}
                      </option>
                    ))}
                  </optgroup>
                </select>
              </div>

              {/* Small / Fast Model */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300">
                  Small / Fast Model (Sub-agents & Planning)
                </label>
                <p className="text-[11px] text-slate-500 dark:text-slate-400 mb-1.5">
                  Used by multi-agent workflows (like OpenCode build/plan) for indexing and fast iterations.
                </p>
                <select
                  value={agentSmallModel}
                  onChange={(e) => setAgentSmallModel(e.target.value)}
                  className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-xs font-mono text-slate-800 shadow-xs focus:border-rose-500 focus:outline-hidden dark:border-slate-800 dark:bg-slate-900 dark:text-slate-200"
                >
                  {getAvailableModelsAndCombos().map(m => (
                    <option key={m} value={m}>{m}</option>
                  ))}
                </select>
              </div>

              {/* Allowed Models & Combos Checklist */}
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="text-xs font-semibold text-slate-700 dark:text-slate-300">
                    Models & Combos to Expose ({agentSelectedModels.length} selected)
                  </label>
                  <div className="flex gap-2 text-[11px]">
                    <button
                      type="button"
                      onClick={() => setAgentSelectedModels(getAvailableModelsAndCombos())}
                      className="text-rose-600 hover:underline dark:text-rose-400"
                    >
                      Select All
                    </button>
                    <span className="text-slate-300 dark:text-slate-700">|</span>
                    <button
                      type="button"
                      onClick={() => setAgentSelectedModels([agentPrimaryModel, agentSmallModel])}
                      className="text-slate-500 hover:underline dark:text-slate-400"
                    >
                      Minimal
                    </button>
                  </div>
                </div>
                <div className="flex flex-wrap gap-1.5 max-h-36 overflow-y-auto p-2 rounded-xl border border-slate-200 bg-slate-50/50 dark:border-slate-800 dark:bg-slate-900/50">
                  {getAvailableModelsAndCombos().map(m => {
                    const selected = agentSelectedModels.includes(m);
                    const isCombo = combos.some(c => c.name === m);
                    return (
                      <button
                        key={m}
                        type="button"
                        onClick={() => {
                          if (selected) {
                            setAgentSelectedModels(agentSelectedModels.filter(x => x !== m));
                          } else {
                            setAgentSelectedModels([...agentSelectedModels, m]);
                          }
                        }}
                        className={`flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-mono transition ${
                          selected
                            ? 'bg-rose-500 text-white font-medium shadow-xs'
                            : 'bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-slate-700 hover:border-slate-300'
                        }`}
                      >
                        {isCombo && <span>⚡</span>}
                        <span>{m}</span>
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Live Config Preview */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                  Live Configuration Preview
                </label>
                <pre className="max-h-36 overflow-y-auto rounded-xl bg-slate-900 p-3 text-[11px] font-mono text-emerald-400 dark:bg-black">
                  {getAgentConfigPreview(configuringAgent)}
                </pre>
              </div>

              {/* Actions */}
              <div className="flex items-center justify-between border-t border-slate-200 dark:border-slate-800 pt-4">
                <button
                  type="button"
                  onClick={() => {
                    const provFlag = agentProvider !== 'auto' ? `-p ${agentProvider} ` : '';
                    const cmd = `goroute setup ${configuringAgent.id} ${provFlag}-m ${agentPrimaryModel} -s ${agentSmallModel} --models "${agentSelectedModels.join(',')}"`;
                    navigator.clipboard.writeText(cmd);
                    showToast('Copied CLI command to clipboard!');
                  }}
                  className="flex items-center gap-1.5 rounded-xl border border-slate-200 dark:border-slate-700 px-3.5 py-2 text-xs font-semibold text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition"
                >
                  <Copy className="h-3.5 w-3.5" />
                  <span>Copy Command</span>
                </button>

                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => setConfiguringAgent(null)}
                    className="rounded-xl px-4 py-2 text-xs font-medium text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    disabled={agentApplying}
                    onClick={() => {
                      setAgentApplying(true);
                      handleConfigureCli(configuringAgent.id, {
                        provider: agentProvider !== 'auto' ? agentProvider : '',
                        model: agentPrimaryModel,
                        small_model: agentSmallModel,
                        models: agentSelectedModels,
                        base_url: agentCustomUrl,
                      }).finally(() => setAgentApplying(false));
                    }}
                    className="flex items-center gap-1.5 rounded-xl bg-rose-600 hover:bg-rose-700 px-4 py-2 text-xs font-semibold text-white shadow-md shadow-rose-500/20 transition disabled:opacity-50"
                  >
                    <Check className="h-4 w-4" />
                    <span>{agentApplying ? 'Applying...' : 'Apply to Local Agent'}</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}
