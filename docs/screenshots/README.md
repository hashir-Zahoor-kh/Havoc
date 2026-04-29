# Havoc — Demo Screenshots

## Local Stack (kind cluster)
- `01-pod-kill-end-to-end.png` — Pod terminating and recovering after a pod-kill experiment
- `02-cpu-pressure-killswitch-abort.png` — CPU pressure experiment aborted via global kill switch
- `03-network-latency-tc-verification.png` — Network latency injected via tc netem, verified inside target pod
- `04-postgres-experiments-and-results.png` — Experiment ledger in Postgres showing all three action types
- `05-kibana-experiment-trace.png` — Single experiment traced across control, agent, and recorder in Kibana
- `06-cluster-overview.png` — Full kind cluster with all Havoc components running

## AWS EKS
- `07-eks-pod-kill-end-to-end.png` — Live end-to-end experiment on EKS showing all four components
- `08-eks-pod-watch.png` — Pod termination and self-healing on EKS
- `09-eks-postgres-experiment-row.png` — Experiment row confirmed in RDS Postgres
- `10-eks-cluster-overview.png` — Full EKS cluster with all Havoc components running
