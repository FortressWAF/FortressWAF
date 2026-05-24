'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

const PRESETS = [
  {
    id: 'dev',
    name: 'Development',
    description: 'Single node, SQLite, minimal resources',
    workers: 1,
    maxConns: 100,
    redis: false,
    postgres: false,
    tls: 'none',
    owasp: 'basic',
  },
  {
    id: 'staging',
    name: 'Staging',
    description: 'Single node, PostgreSQL, moderate resources',
    workers: 2,
    maxConns: 1000,
    redis: true,
    postgres: true,
    tls: 'manual',
    owasp: 'extended',
  },
  {
    id: 'prod',
    name: 'Production',
    description: 'Multi-node, PostgreSQL, Redis, HA',
    workers: 4,
    maxConns: 10000,
    redis: true,
    postgres: true,
    tls: 'letsencrypt',
    owasp: 'maximum',
    http3: true,
  },
  {
    id: 'enterprise',
    name: 'Enterprise',
    description: 'Multi-node, full stack with Vault & SIEM',
    workers: 8,
    maxConns: 50000,
    redis: true,
    postgres: true,
    tls: 'letsencrypt',
    owasp: 'maximum',
    http3: true,
    mtls: true,
    vault: true,
    siem: true,
  },
];

const RULE_TEMPLATES = [
  { id: 'owasp', name: 'OWASP Top 10', description: 'Protection against OWASP Top 10 vulnerabilities' },
  { id: 'sqli', name: 'SQL Injection', description: 'Block SQL injection attempts' },
  { id: 'xss', name: 'XSS', description: 'Block cross-site scripting attempts' },
  { id: 'csrf', name: 'CSRF', description: 'Block cross-site request forgery' },
  { id: 'ratelimit', name: 'Rate Limiting', description: 'Limit request rates per IP' },
  { id: 'bot', name: 'Bot Blocking', description: 'Detect and block bots' },
  { id: 'geo', name: 'Geo-Blocking', description: 'Block traffic by country' },
  { id: 'cors', name: 'CORS', description: 'Enforce CORS policies' },
];

export default function SetupPage() {
  const router = useRouter();
  const [step, setStep] = useState(1);
  const [selectedPreset, setSelectedPreset] = useState<string>('');
  const [config, setConfig] = useState({
    listenAddr: '0.0.0.0:80',
    adminAddr: '0.0.0.0:8443',
    http3: false,
    http3Port: 443,
    mtls: false,
    backendUrl: 'http://localhost:3000',
    healthCheck: true,
    healthPath: '/health',
    domains: '',
    tlsMode: 'none',
    certFile: '',
    keyFile: '',
    rules: [] as string[],
    owaspLevel: 'basic',
    redis: false,
    redisUrl: 'redis://localhost:6379',
    postgres: false,
    postgresUrl: 'postgres://localhost:5432/fortresswaf',
    vault: false,
    siem: false,
  });

  const applyPreset = (presetId: string) => {
    const preset = PRESETS.find(p => p.id === presetId);
    if (!preset) return;

    setSelectedPreset(presetId);
    setConfig(prev => ({
      ...prev,
      workers: preset.workers,
      maxConns: preset.maxConns,
      redis: preset.redis,
      postgres: preset.postgres,
      tlsMode: preset.tls,
      owaspLevel: preset.owasp,
      http3: preset.http3 || false,
      mtls: preset.mtls || false,
      vault: preset.vault || false,
      siem: preset.siem || false,
    }));
  };

  const handleRuleToggle = (ruleId: string) => {
    setConfig(prev => ({
      ...prev,
      rules: prev.rules.includes(ruleId)
        ? prev.rules.filter(r => r !== ruleId)
        : [...prev.rules, ruleId],
    }));
  };

  const generateConfig = () => {
    const configYaml = `
sites:
  - name: default
    domains:
      - ${config.domains.split(',').map(d => `"${d.trim()}"`).join('\n      - ')}
    upstream: ${config.backendUrl}
    port: 80
    waf_enabled: true
    ${config.tlsMode !== 'none' ? `
    tls: true
    cert_file: ${config.certFile}
    key_file: ${config.keyFile}` : ''}

redis:
  enabled: ${config.redis}
  addr: ${config.redisUrl}

db:
  driver: ${config.postgres ? 'postgresql' : 'sqlite3'}
  dsn: ${config.postgres ? config.postgresUrl : 'fortresswaf.db'}

ml:
  enabled: true
  endpoint: http://127.0.0.1:9090

rules:
${config.rules.map(rule => `  - id: ${rule.toUpperCase()}-001
    name: ${RULE_TEMPLATES.find(r => r.id === rule)?.name || rule}
    enabled: true
    severity: high
    action: block`).join('\n')}
`;

    const blob = new Blob([configYaml], { type: 'text/yaml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'config.yaml';
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-900 to-slate-800 text-white">
      <div className="container mx-auto px-4 py-8">
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold mb-2">FortressWAF Setup</h1>
          <p className="text-slate-400">Configure your Web Application Firewall</p>
        </div>

        <div className="max-w-4xl mx-auto">
          <div className="flex items-center justify-center mb-8">
            {[1, 2, 3, 4, 5, 6].map(s => (
              <div key={s} className="flex items-center">
                <div
                  className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold
                    ${step >= s ? 'bg-blue-600 text-white' : 'bg-slate-700 text-slate-400'}`}
                >
                  {s}
                </div>
                {s < 6 && (
                  <div className={`w-12 h-0.5 ${step > s ? 'bg-blue-600' : 'bg-slate-700'}`} />
                )}
              </div>
            ))}
          </div>

          <Card className="bg-slate-800 border-slate-700">
            {step === 1 && (
              <>
                <CardHeader>
                  <CardTitle>Select Preset</CardTitle>
                  <CardDescription>Choose a configuration preset or customize manually</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-4">
                    {PRESETS.map(preset => (
                      <div
                        key={preset.id}
                        onClick={() => applyPreset(preset.id)}
                        className={`p-4 rounded-lg border cursor-pointer transition-all
                          ${selectedPreset === preset.id
                            ? 'border-blue-500 bg-blue-900/30'
                            : 'border-slate-600 bg-slate-700/50 hover:border-slate-500'}`}
                      >
                        <div className="flex items-center justify-between mb-2">
                          <span className="font-bold">{preset.name}</span>
                          {selectedPreset === preset.id && (
                            <Badge className="bg-blue-600">Selected</Badge>
                          )}
                        </div>
                        <p className="text-sm text-slate-400">{preset.description}</p>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </>
            )}

            {step === 2 && (
              <>
                <CardHeader>
                  <CardTitle>Network Configuration</CardTitle>
                  <CardDescription>Configure network settings</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <label className="block text-sm mb-1">Proxy Listen Address</label>
                    <Input
                      value={config.listenAddr}
                      onChange={e => setConfig(prev => ({ ...prev, listenAddr: e.target.value }))}
                      className="bg-slate-700 border-slate-600"
                    />
                  </div>
                  <div>
                    <label className="block text-sm mb-1">Admin API Listen Address</label>
                    <Input
                      value={config.adminAddr}
                      onChange={e => setConfig(prev => ({ ...prev, adminAddr: e.target.value }))}
                      className="bg-slate-700 border-slate-600"
                    />
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="http3"
                      checked={config.http3}
                      onChange={e => setConfig(prev => ({ ...prev, http3: e.target.checked }))}
                      className="rounded"
                    />
                    <label htmlFor="http3">Enable HTTP/3 (QUIC)</label>
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="mtls"
                      checked={config.mtls}
                      onChange={e => setConfig(prev => ({ ...prev, mtls: e.target.checked }))}
                      className="rounded"
                    />
                    <label htmlFor="mtls">Enable mTLS Client Authentication</label>
                  </div>
                </CardContent>
              </>
            )}

            {step === 3 && (
              <>
                <CardHeader>
                  <CardTitle>Backend Configuration</CardTitle>
                  <CardDescription>Configure your upstream backend</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <label className="block text-sm mb-1">Backend URL</label>
                    <Input
                      value={config.backendUrl}
                      onChange={e => setConfig(prev => ({ ...prev, backendUrl: e.target.value }))}
                      placeholder="http://localhost:3000"
                      className="bg-slate-700 border-slate-600"
                    />
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="healthCheck"
                      checked={config.healthCheck}
                      onChange={e => setConfig(prev => ({ ...prev, healthCheck: e.target.checked }))}
                      className="rounded"
                    />
                    <label htmlFor="healthCheck">Enable Health Checks</label>
                  </div>
                  {config.healthCheck && (
                    <div>
                      <label className="block text-sm mb-1">Health Check Path</label>
                      <Input
                        value={config.healthPath}
                        onChange={e => setConfig(prev => ({ ...prev, healthPath: e.target.value }))}
                        className="bg-slate-700 border-slate-600"
                      />
                    </div>
                  )}
                </CardContent>
              </>
            )}

            {step === 4 && (
              <>
                <CardHeader>
                  <CardTitle>Domain & TLS</CardTitle>
                  <CardDescription>Configure domains and TLS settings</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <label className="block text-sm mb-1">Domains (comma-separated)</label>
                    <Input
                      value={config.domains}
                      onChange={e => setConfig(prev => ({ ...prev, domains: e.target.value }))}
                      placeholder="example.com, *.example.com"
                      className="bg-slate-700 border-slate-600"
                    />
                  </div>
                  <div>
                    <label className="block text-sm mb-1">TLS Mode</label>
                    <Select
                      value={config.tlsMode}
                      onValueChange={v => setConfig(prev => ({ ...prev, tlsMode: v }))}
                    >
                      <SelectTrigger className="bg-slate-700 border-slate-600">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">None (HTTP only)</SelectItem>
                        <SelectItem value="manual">Manual (provide cert/key)</SelectItem>
                        <SelectItem value="letsencrypt">Let's Encrypt (auto-cert)</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  {config.tlsMode === 'manual' && (
                    <>
                      <div>
                        <label className="block text-sm mb-1">Certificate File</label>
                        <Input
                          value={config.certFile}
                          onChange={e => setConfig(prev => ({ ...prev, certFile: e.target.value }))}
                          placeholder="/etc/fortresswaf/cert.pem"
                          className="bg-slate-700 border-slate-600"
                        />
                      </div>
                      <div>
                        <label className="block text-sm mb-1">Key File</label>
                        <Input
                          value={config.keyFile}
                          onChange={e => setConfig(prev => ({ ...prev, keyFile: e.target.value }))}
                          placeholder="/etc/fortresswaf/key.pem"
                          className="bg-slate-700 border-slate-600"
                        />
                      </div>
                    </>
                  )}
                </CardContent>
              </>
            )}

            {step === 5 && (
              <>
                <CardHeader>
                  <CardTitle>Rule Templates</CardTitle>
                  <CardDescription>Select security rules to enable</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-3">
                    {RULE_TEMPLATES.map(rule => (
                      <div
                        key={rule.id}
                        onClick={() => handleRuleToggle(rule.id)}
                        className={`p-3 rounded-lg border cursor-pointer transition-all
                          ${config.rules.includes(rule.id)
                            ? 'border-blue-500 bg-blue-900/30'
                            : 'border-slate-600 bg-slate-700/50 hover:border-slate-500'}`}
                      >
                        <div className="flex items-center gap-2">
                          <input
                            type="checkbox"
                            checked={config.rules.includes(rule.id)}
                            onChange={() => handleRuleToggle(rule.id)}
                            className="rounded"
                          />
                          <span className="font-medium">{rule.name}</span>
                        </div>
                        <p className="text-xs text-slate-400 mt-1">{rule.description}</p>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </>
            )}

            {step === 6 && (
              <>
                <CardHeader>
                  <CardTitle>Integrations & Review</CardTitle>
                  <CardDescription>Configure external integrations</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="redis"
                      checked={config.redis}
                      onChange={e => setConfig(prev => ({ ...prev, redis: e.target.checked }))}
                      className="rounded"
                    />
                    <label htmlFor="redis">Enable Redis</label>
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="postgres"
                      checked={config.postgres}
                      onChange={e => setConfig(prev => ({ ...prev, postgres: e.target.checked }))}
                      className="rounded"
                    />
                    <label htmlFor="postgres">Enable PostgreSQL</label>
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="vault"
                      checked={config.vault}
                      onChange={e => setConfig(prev => ({ ...prev, vault: e.target.checked }))}
                      className="rounded"
                    />
                    <label htmlFor="vault">Enable HashiCorp Vault</label>
                  </div>
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="siem"
                      checked={config.siem}
                      onChange={e => setConfig(prev => ({ ...prev, siem: e.target.checked }))}
                      className="rounded"
                    />
                    <label htmlFor="siem">Enable SIEM Integration</label>
                  </div>

                  <div className="mt-6 p-4 bg-slate-700/50 rounded-lg">
                    <h3 className="font-bold mb-2">Configuration Summary</h3>
                    <div className="text-sm space-y-1">
                      <p>Preset: {selectedPreset || 'Custom'}</p>
                      <p>Listen: {config.listenAddr}</p>
                      <p>Backend: {config.backendUrl}</p>
                      <p>Domains: {config.domains || 'Not set'}</p>
                      <p>TLS: {config.tlsMode}</p>
                      <p>Rules: {config.rules.length > 0 ? config.rules.join(', ') : 'None'}</p>
                    </div>
                  </div>

                  <Button onClick={generateConfig} className="w-full mt-4">
                    Download config.yaml
                  </Button>
                </CardContent>
              </>
            )}

            <div className="flex justify-between p-6 border-t border-slate-700">
              <Button
                variant="outline"
                onClick={() => setStep(s => Math.max(1, s - 1))}
                disabled={step === 1}
              >
                Previous
              </Button>
              <Button
                onClick={() => {
                  if (step === 6) {
                    router.push('/dashboard');
                  } else {
                    setStep(s => s + 1);
                  }
                }}
              >
                {step === 6 ? 'Finish' : 'Next'}
              </Button>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}
