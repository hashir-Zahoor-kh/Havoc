export const experimentHistory = [
  { id: 'hvc-a3f2c1d4', action: 'POD_KILL', target: 'app=checkout', status: 'COMPLETED', timestamp: '2026-05-17 09:14:22', duration: '2.3s' },
  { id: 'hvc-b8e7d2a1', action: 'CPU_PRESSURE', target: 'app=payment', status: 'COMPLETED', timestamp: '2026-05-17 08:55:10', duration: '30.1s' },
  { id: 'hvc-c1d9f4b2', action: 'NETWORK_LATENCY', target: 'app=api-gateway', status: 'ABORTED', timestamp: '2026-05-16 22:30:05', duration: '0.0s' },
  { id: 'hvc-d5a3e8f1', action: 'POD_KILL', target: 'app=inventory', status: 'COMPLETED', timestamp: '2026-05-16 18:42:33', duration: '1.8s' },
  { id: 'hvc-e2b6c9d3', action: 'CPU_PRESSURE', target: 'app=search', status: 'RUNNING', timestamp: '2026-05-17 09:20:01', duration: '—' },
  { id: 'hvc-f7d1a4e9', action: 'NETWORK_LATENCY', target: 'app=notifications', status: 'COMPLETED', timestamp: '2026-05-16 14:11:47', duration: '15.7s' },
]

export const components = [
  { id: 'cli', label: 'CLI', tag: 'GO BINARY', description: 'Command-line interface that assembles ChaosCommand structs and publishes them to the Kafka havoc.commands topic. Validates label selectors, blast radius, and duration before dispatch.' },
  { id: 'control', label: 'CONTROL PLANE', tag: 'GO BINARY', description: 'Central API server that enforces safety guardrails — blast radius limits, Redis-based active-experiment locks, global kill switch, and blackout windows — before forwarding commands.' },
  { id: 'kafka-cmd', label: 'KAFKA\nhavoc.commands', tag: 'KAFKA', description: 'Durable topic that decouples the control plane from agents. Guarantees at-least-once delivery of chaos commands to all subscribed agent consumers.' },
  { id: 'agents', label: 'AGENTS\n(DaemonSet)', tag: 'GO BINARY', description: 'One agent pod per Kubernetes node. Subscribes to havoc.commands, executes chaos actions (pod kill, CPU stress, tc-netem), and publishes results back to havoc.results.' },
  { id: 'postgres-redis', label: 'POSTGRES\n+ REDIS', tag: 'STORAGE', description: 'Postgres stores experiment records and historical results. Redis holds the active-experiment lock and kill-switch flag for sub-millisecond reads on the hot guardrail path.' },
  { id: 'kafka-res', label: 'KAFKA\nhavoc.results', tag: 'KAFKA', description: 'Results topic where agents publish the outcome of each chaos action, including affected pod names, actual duration, and error details on failure.' },
  { id: 'recorder', label: 'RECORDER', tag: 'GO BINARY', description: 'Stateless consumer that drains havoc.results and persists structured experiment records to Postgres. Also ships logs to ELK for dashboarding and post-mortem analysis.' },
  { id: 'storage', label: 'POSTGRES\n+ REDIS + ELK', tag: 'STORAGE', description: 'Persistent storage layer for completed experiment data. ELK (Elasticsearch + Kibana) provides full-text search and visualization across all chaos events.' },
]

export const techStack = {
  LANGUAGE: ['Go 1.22', 'React', 'JavaScript'],
  MESSAGING: ['Apache Kafka', 'Strimzi Operator'],
  STORAGE: ['PostgreSQL', 'Redis', 'Elasticsearch'],
  INFRA: ['Kubernetes', 'Helm', 'Terraform', 'AWS EKS', 'kind'],
  OBSERVABILITY: ['Kibana', 'Kafka UI', 'Health endpoints'],
}

export const chaosActions = [
  {
    name: 'POD_KILL',
    tag: 'AVAILABILITY',
    description: 'Terminates a randomly selected pod matching the label selector, forcing Kubernetes to reschedule it. Tests self-healing, readiness probe configuration, and downstream service resilience.',
  },
  {
    name: 'CPU_PRESSURE',
    tag: 'PERFORMANCE',
    description: 'Injects sustained CPU load on the target pod using a stress-ng worker. Exposes CPU throttling behavior, autoscaler response time, and latency degradation under resource contention.',
  },
  {
    name: 'NETWORK_LATENCY',
    tag: 'RELIABILITY',
    description: 'Adds configurable egress latency via the Linux tc-netem kernel module. Validates timeout configurations, retry logic, and circuit-breaker thresholds across service boundaries.',
  },
]

export const blackoutWindows = [
  { day: 'MON', start: 22, end: 24 },
  { day: 'TUE', start: 22, end: 24 },
  { day: 'WED', start: 22, end: 24 },
  { day: 'THU', start: 22, end: 24 },
  { day: 'FRI', start: 0, end: 2 },
  { day: 'SAT', start: 0, end: 24 },
  { day: 'SUN', start: 0, end: 24 },
]
