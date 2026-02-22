import { useEffect, useState } from "react";
import clsx from "clsx";
import {
  Home,
  Layers,
  BarChart3,
  Settings,
  HelpCircle,
  FileText,
  ChevronDown,
  TrendingUp,
  TrendingDown,
  Zap,
  Clock,
  Activity,
  Cpu,
  ChevronRight,
  Menu,
  X,
  Sun,
  Moon,
  Monitor,
} from "lucide-react";
import { useTheme } from "./hooks/useTheme";

interface ServiceStatus {
  service: string;
  version: string;
  status: string;
}

interface StatCardProps {
  title: string;
  value: string;
  change: string;
  changeType: "positive" | "negative" | "neutral";
}

interface RequestRow {
  id: string;
  timestamp: string;
  model: string;
  endpoint: string;
  tokens: number;
  latency: string;
  status: "success" | "error" | "pending";
}

// Navigation items
const mainNavItems = [
  { name: "Home", href: "#", icon: Home, current: true },
  { name: "Models", href: "#", icon: Layers, current: false },
  { name: "Analytics", href: "#", icon: BarChart3, current: false },
  { name: "Settings", href: "#", icon: Settings, current: false },
];

const secondaryNavItems = [
  { name: "Support", href: "#", icon: HelpCircle },
  { name: "Changelog", href: "#", icon: FileText },
];

// Mock data for recent requests
const recentRequests: RequestRow[] = [
  {
    id: "req_001",
    timestamp: "Jan 17, 2026 14:32",
    model: "gpt-4-turbo",
    endpoint: "/v1/chat/completions",
    tokens: 2847,
    latency: "1.2s",
    status: "success",
  },
  {
    id: "req_002",
    timestamp: "Jan 17, 2026 14:28",
    model: "claude-3-opus",
    endpoint: "/v1/messages",
    tokens: 1523,
    latency: "2.1s",
    status: "success",
  },
  {
    id: "req_003",
    timestamp: "Jan 17, 2026 14:25",
    model: "gpt-4-turbo",
    endpoint: "/v1/chat/completions",
    tokens: 892,
    latency: "0.8s",
    status: "success",
  },
  {
    id: "req_004",
    timestamp: "Jan 17, 2026 14:21",
    model: "gemini-pro",
    endpoint: "/v1/generate",
    tokens: 3201,
    latency: "1.5s",
    status: "error",
  },
  {
    id: "req_005",
    timestamp: "Jan 17, 2026 14:18",
    model: "claude-3-sonnet",
    endpoint: "/v1/messages",
    tokens: 1876,
    latency: "1.1s",
    status: "success",
  },
  {
    id: "req_006",
    timestamp: "Jan 17, 2026 14:15",
    model: "gpt-4-turbo",
    endpoint: "/v1/chat/completions",
    tokens: 2156,
    latency: "1.3s",
    status: "success",
  },
  {
    id: "req_007",
    timestamp: "Jan 17, 2026 14:12",
    model: "llama-3-70b",
    endpoint: "/v1/completions",
    tokens: 945,
    latency: "0.6s",
    status: "success",
  },
  {
    id: "req_008",
    timestamp: "Jan 17, 2026 14:08",
    model: "claude-3-opus",
    endpoint: "/v1/messages",
    tokens: 4521,
    latency: "3.2s",
    status: "success",
  },
  {
    id: "req_009",
    timestamp: "Jan 17, 2026 14:05",
    model: "gpt-4-turbo",
    endpoint: "/v1/chat/completions",
    tokens: 1234,
    latency: "0.9s",
    status: "success",
  },
  {
    id: "req_010",
    timestamp: "Jan 17, 2026 14:01",
    model: "gemini-pro",
    endpoint: "/v1/generate",
    tokens: 2089,
    latency: "1.4s",
    status: "success",
  },
];

// Time range options
const timeRanges = [
  "Last hour",
  "Last 24 hours",
  "Last 7 days",
  "Last 30 days",
];

function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  const [dropdownOpen, setDropdownOpen] = useState(false);

  const themeOptions = [
    { value: "light" as const, label: "Light", icon: Sun },
    { value: "dark" as const, label: "Dark", icon: Moon },
    { value: "system" as const, label: "System", icon: Monitor },
  ];

  const currentOption =
    themeOptions.find((opt) => opt.value === theme) || themeOptions[2];
  const CurrentIcon = currentOption.icon;

  return (
    <div className="relative">
      <button
        onClick={() => setDropdownOpen(!dropdownOpen)}
        className="flex h-8 w-8 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-zinc-800 hover:text-white"
        aria-label="Toggle theme"
      >
        <CurrentIcon className="h-4 w-4" />
      </button>

      {dropdownOpen && (
        <>
          <div
            className="fixed inset-0 z-10"
            onClick={() => setDropdownOpen(false)}
          />
          <div className="absolute right-0 z-20 mt-2 w-36 rounded-lg border border-zinc-700 bg-zinc-800 py-1 shadow-lg">
            {themeOptions.map((option) => (
              <button
                key={option.value}
                onClick={() => {
                  setTheme(option.value);
                  setDropdownOpen(false);
                }}
                className={clsx(
                  "flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors",
                  theme === option.value
                    ? "bg-zinc-700 text-white"
                    : "text-zinc-300 hover:bg-zinc-700/50 hover:text-white",
                )}
              >
                <option.icon className="h-4 w-4" />
                {option.label}
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}

function StatCard({ title, value, change, changeType }: StatCardProps) {
  return (
    <div className="rounded-lg border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
      <dt className="text-sm font-medium text-zinc-500 dark:text-zinc-400">
        {title}
      </dt>
      <dd className="mt-2 flex items-baseline gap-2">
        <span className="text-3xl font-semibold tracking-tight text-zinc-900 dark:text-white">
          {value}
        </span>
        <span
          className={clsx(
            "inline-flex items-center gap-0.5 text-sm font-medium",
            changeType === "positive" &&
              "text-emerald-600 dark:text-emerald-400",
            changeType === "negative" && "text-red-600 dark:text-red-400",
            changeType === "neutral" && "text-zinc-500 dark:text-zinc-400",
          )}
        >
          {changeType === "positive" && <TrendingUp className="h-4 w-4" />}
          {changeType === "negative" && <TrendingDown className="h-4 w-4" />}
          {change}
        </span>
      </dd>
    </div>
  );
}

function StatusBadge({ status }: { status: RequestRow["status"] }) {
  return (
    <span
      className={clsx(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
        status === "success" &&
          "bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400",
        status === "error" &&
          "bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-400",
        status === "pending" &&
          "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400",
      )}
    >
      {status}
    </span>
  );
}

function ModelBadge({ model }: { model: string }) {
  const getModelColor = (m: string) => {
    if (m.includes("gpt"))
      return "bg-emerald-100 text-emerald-800 dark:bg-emerald-500/20 dark:text-emerald-300";
    if (m.includes("claude"))
      return "bg-orange-100 text-orange-800 dark:bg-orange-500/20 dark:text-orange-300";
    if (m.includes("gemini"))
      return "bg-blue-100 text-blue-800 dark:bg-blue-500/20 dark:text-blue-300";
    if (m.includes("llama"))
      return "bg-purple-100 text-purple-800 dark:bg-purple-500/20 dark:text-purple-300";
    return "bg-zinc-100 text-zinc-800 dark:bg-zinc-700 dark:text-zinc-300";
  };

  return (
    <span
      className={clsx(
        "inline-flex items-center rounded-md px-2 py-1 text-xs font-medium",
        getModelColor(model),
      )}
    >
      {model}
    </span>
  );
}

function App() {
  const [status, setStatus] = useState<ServiceStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedTimeRange, setSelectedTimeRange] = useState(timeRanges[1]);
  const [timeDropdownOpen, setTimeDropdownOpen] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(false);

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const response = await fetch("/api/v1/status");
        if (!response.ok) throw new Error("Failed to fetch status");
        const data = await response.json();
        setStatus(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unknown error");
      } finally {
        setLoading(false);
      }
    };

    fetchStatus();
  }, []);

  // Get current hour for greeting
  const hour = new Date().getHours();
  const greeting =
    hour < 12 ? "Good morning" : hour < 18 ? "Good afternoon" : "Good evening";

  return (
    <div className="flex h-screen bg-zinc-50 dark:bg-zinc-950">
      {/* Mobile sidebar overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-zinc-900/50 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={clsx(
          "fixed inset-y-0 left-0 z-50 flex w-64 flex-col bg-zinc-900 transition-transform duration-300 lg:static lg:translate-x-0",
          sidebarOpen ? "translate-x-0" : "-translate-x-full",
        )}
      >
        {/* Logo */}
        <div className="flex h-16 items-center justify-between px-6">
          <div className="flex items-center gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-500">
              <Zap className="h-5 w-5 text-white" />
            </div>
            <span className="text-lg font-semibold text-white">Lectr</span>
          </div>
          <div className="flex items-center gap-1">
            <ThemeToggle />
            <button
              className="flex h-8 w-8 items-center justify-center rounded-lg text-zinc-400 hover:bg-zinc-800 hover:text-white lg:hidden"
              onClick={() => setSidebarOpen(false)}
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex flex-1 flex-col px-4 py-4">
          <ul className="space-y-1">
            {mainNavItems.map((item) => (
              <li key={item.name}>
                <a
                  href={item.href}
                  className={clsx(
                    "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                    item.current
                      ? "bg-zinc-800 text-white"
                      : "text-zinc-400 hover:bg-zinc-800 hover:text-white",
                  )}
                >
                  <item.icon className="h-5 w-5" />
                  {item.name}
                </a>
              </li>
            ))}
          </ul>

          {/* Secondary nav */}
          <div className="mt-auto">
            <ul className="space-y-1">
              {secondaryNavItems.map((item) => (
                <li key={item.name}>
                  <a
                    href={item.href}
                    className="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-zinc-400 transition-colors hover:bg-zinc-800 hover:text-white"
                  >
                    <item.icon className="h-5 w-5" />
                    {item.name}
                  </a>
                </li>
              ))}
            </ul>

            {/* User section */}
            <div className="mt-4 border-t border-zinc-800 pt-4">
              <div className="flex items-center gap-3 rounded-lg px-3 py-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-indigo-500 text-sm font-medium text-white">
                  A
                </div>
                <div className="flex-1 min-w-0">
                  <p className="truncate text-sm font-medium text-white">
                    Admin
                  </p>
                  <p className="truncate text-xs text-zinc-500">
                    admin@lectr.ai
                  </p>
                </div>
              </div>
            </div>
          </div>
        </nav>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto">
        {/* Mobile header */}
        <div className="sticky top-0 z-30 flex h-16 items-center gap-4 border-b border-zinc-200 bg-zinc-50 px-4 dark:border-zinc-800 dark:bg-zinc-950 lg:hidden">
          <button
            className="text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white"
            onClick={() => setSidebarOpen(true)}
          >
            <Menu className="h-6 w-6" />
          </button>
          <div className="flex items-center gap-2">
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-indigo-500">
              <Zap className="h-4 w-4 text-white" />
            </div>
            <span className="font-semibold text-zinc-900 dark:text-white">
              Lectr
            </span>
          </div>
        </div>

        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
          {/* Header */}
          <header className="mb-8">
            <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white">
              {greeting}, Admin
            </h1>
            <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
              Monitor and manage your AI gateway traffic
            </p>
          </header>

          {/* Status Banner */}
          {loading && (
            <div className="mb-6 rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
              <div className="flex items-center gap-3">
                <div className="h-5 w-5 animate-spin rounded-full border-2 border-indigo-500 border-t-transparent" />
                <span className="text-sm text-zinc-600 dark:text-zinc-400">
                  Connecting to backend...
                </span>
              </div>
            </div>
          )}

          {error && (
            <div className="mb-6 rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-500/20 dark:bg-red-500/10">
              <div className="flex items-center gap-3">
                <div className="flex h-5 w-5 items-center justify-center rounded-full bg-red-100 dark:bg-red-500/20">
                  <span className="text-xs text-red-600 dark:text-red-400">
                    !
                  </span>
                </div>
                <div>
                  <p className="text-sm font-medium text-red-800 dark:text-red-300">
                    Unable to connect to backend
                  </p>
                  <p className="mt-0.5 text-xs text-red-600 dark:text-red-400">
                    {error}
                  </p>
                </div>
              </div>
            </div>
          )}

          {status && (
            <div className="mb-6 rounded-lg border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-500/20 dark:bg-emerald-500/10">
              <div className="flex items-center gap-3">
                <div className="flex h-5 w-5 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-500/20">
                  <Activity className="h-3 w-3 text-emerald-600 dark:text-emerald-400" />
                </div>
                <p className="text-sm text-emerald-800 dark:text-emerald-300">
                  <span className="font-medium">{status.service}</span> v
                  {status.version} is{" "}
                  <span className="font-medium text-emerald-700 dark:text-emerald-400">
                    {status.status}
                  </span>
                </p>
              </div>
            </div>
          )}

          {/* Overview Section */}
          <section className="mb-8">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-base font-semibold text-zinc-900 dark:text-white">
                Overview
              </h2>

              {/* Time range dropdown */}
              <div className="relative">
                <button
                  onClick={() => setTimeDropdownOpen(!timeDropdownOpen)}
                  className="inline-flex items-center gap-2 rounded-lg border border-zinc-200 bg-white px-3 py-1.5 text-sm font-medium text-zinc-700 shadow-sm transition-colors hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-700"
                >
                  {selectedTimeRange}
                  <ChevronDown className="h-4 w-4 text-zinc-400" />
                </button>

                {timeDropdownOpen && (
                  <>
                    <div
                      className="fixed inset-0 z-10"
                      onClick={() => setTimeDropdownOpen(false)}
                    />
                    <div className="absolute right-0 z-20 mt-1 w-40 rounded-lg border border-zinc-200 bg-white py-1 shadow-lg dark:border-zinc-700 dark:bg-zinc-800">
                      {timeRanges.map((range) => (
                        <button
                          key={range}
                          onClick={() => {
                            setSelectedTimeRange(range);
                            setTimeDropdownOpen(false);
                          }}
                          className={clsx(
                            "block w-full px-4 py-2 text-left text-sm transition-colors",
                            range === selectedTimeRange
                              ? "bg-zinc-100 text-zinc-900 dark:bg-zinc-700 dark:text-white"
                              : "text-zinc-600 hover:bg-zinc-50 dark:text-zinc-300 dark:hover:bg-zinc-700/50",
                          )}
                        >
                          {range}
                        </button>
                      ))}
                    </div>
                  </>
                )}
              </div>
            </div>

            {/* Stats Grid */}
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <StatCard
                title="Total Requests"
                value="128.4K"
                change="+12.3%"
                changeType="positive"
              />
              <StatCard
                title="Avg Latency"
                value="1.24s"
                change="-8.2%"
                changeType="positive"
              />
              <StatCard
                title="Token Usage"
                value="4.2M"
                change="+23.1%"
                changeType="negative"
              />
              <StatCard
                title="Active Models"
                value="12"
                change="+2"
                changeType="neutral"
              />
            </div>
          </section>

          {/* Quick Stats Row */}
          <section className="mb-8 grid gap-4 sm:grid-cols-3">
            <div className="flex items-center gap-4 rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-50 dark:bg-indigo-500/10">
                <Cpu className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
              </div>
              <div>
                <p className="text-sm text-zinc-500 dark:text-zinc-400">
                  Top Model
                </p>
                <p className="font-semibold text-zinc-900 dark:text-white">
                  gpt-4-turbo
                </p>
              </div>
            </div>
            <div className="flex items-center gap-4 rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-50 dark:bg-emerald-500/10">
                <Activity className="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
              </div>
              <div>
                <p className="text-sm text-zinc-500 dark:text-zinc-400">
                  Success Rate
                </p>
                <p className="font-semibold text-zinc-900 dark:text-white">
                  99.2%
                </p>
              </div>
            </div>
            <div className="flex items-center gap-4 rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-50 dark:bg-amber-500/10">
                <Clock className="h-5 w-5 text-amber-600 dark:text-amber-400" />
              </div>
              <div>
                <p className="text-sm text-zinc-500 dark:text-zinc-400">
                  Peak Hour
                </p>
                <p className="font-semibold text-zinc-900 dark:text-white">
                  2:00 PM - 3:00 PM
                </p>
              </div>
            </div>
          </section>

          {/* Recent Requests Table */}
          <section>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-base font-semibold text-zinc-900 dark:text-white">
                Recent Requests
              </h2>
              <a
                href="#"
                className="inline-flex items-center gap-1 text-sm font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
              >
                View all
                <ChevronRight className="h-4 w-4" />
              </a>
            </div>

            <div className="overflow-hidden rounded-lg border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-zinc-200 dark:divide-zinc-800">
                  <thead>
                    <tr className="bg-zinc-50 dark:bg-zinc-800/50">
                      <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
                        Request ID
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
                        Timestamp
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
                        Model
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
                        Endpoint
                      </th>
                      <th className="px-4 py-3 text-right text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
                        Tokens
                      </th>
                      <th className="px-4 py-3 text-right text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
                        Latency
                      </th>
                      <th className="px-4 py-3 text-right text-xs font-medium uppercase tracking-wide text-zinc-500 dark:text-zinc-400">
                        Status
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800">
                    {recentRequests.map((request) => (
                      <tr
                        key={request.id}
                        className="transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                      >
                        <td className="whitespace-nowrap px-4 py-3">
                          <a
                            href="#"
                            className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
                          >
                            {request.id}
                          </a>
                        </td>
                        <td className="whitespace-nowrap px-4 py-3 text-sm text-zinc-500 dark:text-zinc-400">
                          {request.timestamp}
                        </td>
                        <td className="whitespace-nowrap px-4 py-3">
                          <ModelBadge model={request.model} />
                        </td>
                        <td className="whitespace-nowrap px-4 py-3">
                          <code className="rounded bg-zinc-100 px-2 py-0.5 text-xs text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
                            {request.endpoint}
                          </code>
                        </td>
                        <td className="whitespace-nowrap px-4 py-3 text-right text-sm tabular-nums text-zinc-700 dark:text-zinc-300">
                          {request.tokens.toLocaleString()}
                        </td>
                        <td className="whitespace-nowrap px-4 py-3 text-right text-sm tabular-nums text-zinc-700 dark:text-zinc-300">
                          {request.latency}
                        </td>
                        <td className="whitespace-nowrap px-4 py-3 text-right">
                          <StatusBadge status={request.status} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          {/* Footer */}
          <footer className="mt-12 border-t border-zinc-200 pt-6 text-center text-sm text-zinc-500 dark:border-zinc-800 dark:text-zinc-400">
            <p>
              Lectr &copy; {new Date().getFullYear()} &middot; AI Gateway &
              Control Plane
            </p>
          </footer>
        </div>
      </main>
    </div>
  );
}

export default App;
