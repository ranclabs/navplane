import { useEffect, useState } from 'react'

interface ServiceStatus {
  service: string
  version: string
  status: string
}

function App() {
  const [status, setStatus] = useState<ServiceStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const response = await fetch('/api/v1/status')
        if (!response.ok) throw new Error('Failed to fetch status')
        const data = await response.json()
        setStatus(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error')
      } finally {
        setLoading(false)
      }
    }

    fetchStatus()
  }, [])

  return (
    <div className="min-h-screen flex flex-col bg-linear-to-br from-slate-950 via-slate-900 to-slate-950 text-zinc-200 font-sans">
      {/* Header */}
      <header className="p-8 text-center border-b border-white/10">
        <div className="flex items-center justify-center gap-3 text-indigo-400">
          <svg
            width="32"
            height="32"
            viewBox="0 0 32 32"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M16 2L4 8v16l12 6 12-6V8L16 2z"
              stroke="currentColor"
              strokeWidth="2"
              fill="none"
            />
            <path
              d="M16 10l-6 3v6l6 3 6-3v-6l-6-3z"
              fill="currentColor"
            />
          </svg>
          <h1 className="text-3xl font-semibold bg-linear-to-r from-indigo-400 to-purple-400 bg-clip-text text-transparent">
            NavPlane
          </h1>
        </div>
        <p className="mt-2 text-zinc-400">AI Gateway & Control Plane</p>
      </header>

      {/* Main Content */}
      <main className="flex-1 p-8 max-w-6xl mx-auto w-full">
        {/* Status Card */}
        <section className="bg-white/5 border border-white/10 rounded-2xl p-6 mb-8">
          <h2 className="text-xl font-medium mb-4 text-zinc-100">System Status</h2>
          
          {loading && (
            <div className="text-zinc-400 py-4 text-center">Loading...</div>
          )}
          
          {error && (
            <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-4 text-center">
              <p className="text-red-400 font-medium">Unable to connect to backend</p>
              <small className="text-zinc-400 block mt-1">{error}</small>
            </div>
          )}
          
          {status && (
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div className="flex flex-col gap-1">
                <span className="text-xs uppercase tracking-wide text-zinc-500">Service</span>
                <span className="text-lg font-medium">{status.service}</span>
              </div>
              <div className="flex flex-col gap-1">
                <span className="text-xs uppercase tracking-wide text-zinc-500">Version</span>
                <span className="text-lg font-medium">{status.version}</span>
              </div>
              <div className="flex flex-col gap-1">
                <span className="text-xs uppercase tracking-wide text-zinc-500">Status</span>
                <span className={`text-lg font-medium ${status.status === 'operational' ? 'text-green-500' : 'text-yellow-500'}`}>
                  {status.status}
                </span>
              </div>
            </div>
          )}
        </section>

        {/* Features */}
        <section className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="bg-white/3 border border-white/8 rounded-2xl p-6 transition-all duration-200 hover:bg-white/5 hover:border-indigo-500/30 hover:-translate-y-0.5">
            <h3 className="text-lg font-medium mb-2 text-purple-400">Governed LLM Traffic</h3>
            <p className="text-zinc-400 leading-relaxed">
              Route, monitor, and control AI model requests with fine-grained policies.
            </p>
          </div>
          <div className="bg-white/3 border border-white/8 rounded-2xl p-6 transition-all duration-200 hover:bg-white/5 hover:border-indigo-500/30 hover:-translate-y-0.5">
            <h3 className="text-lg font-medium mb-2 text-purple-400">High Performance</h3>
            <p className="text-zinc-400 leading-relaxed">
              Built for speed with Go backend and optimized request handling.
            </p>
          </div>
          <div className="bg-white/3 border border-white/8 rounded-2xl p-6 transition-all duration-200 hover:bg-white/5 hover:border-indigo-500/30 hover:-translate-y-0.5">
            <h3 className="text-lg font-medium mb-2 text-purple-400">Unified Dashboard</h3>
            <p className="text-zinc-400 leading-relaxed">
              Monitor all your AI traffic from a single, intuitive interface.
            </p>
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="p-6 text-center border-t border-white/10 text-zinc-500 text-sm">
        <p>NavPlane &copy; {new Date().getFullYear()}</p>
      </footer>
    </div>
  )
}

export default App
