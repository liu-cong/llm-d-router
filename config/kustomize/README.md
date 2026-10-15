# Kustomize Deployment Configuration

`config/kustomize/` provides bases, reusable Kustomize `Component` modules, and example overlays for deploying the llm-d Router in standalone or Gateway mode.

## Directory Layout

```text
config/kustomize/
├── base/
│   ├── common/                            # EPP Deployment, Service, RBAC, InferencePool, configMapGenerator, namereference
│   ├── gateway/                           # Gateway-mode base (inherits base/common)
│   └── standalone/                        # Standalone base (adds Envoy sidecar, envoy.yaml ConfigMap, HTTP port 80->8081)
├── components/
│   ├── endpoint-selector/                 # Switches from InferencePool to label-based --endpoint-selector mode
│   ├── proxy/
│   │   ├── agentgateway-sidecar/          # Replaces Envoy sidecar with Agentgateway sidecar
│   │   ├── envoy-service/                 # Runs Envoy proxy as a separate Deployment and Service
│   │   └── proxy-hpa/                     # HorizontalPodAutoscaler for separate proxy Deployment
│   ├── scaling/
│   │   ├── leader-election/               # 3-replica HA with --ha-enable-leader-election and Lease RBAC
│   │   └── epp-hpa/                       # HorizontalPodAutoscaler for active-active EPP Deployment
│   ├── sidecars/
│   │   ├── tokenizer-python/              # vLLM Python render sidecar (vllm launch render)
│   │   ├── tokenizer-rust/                # Rust render sidecar (vllm-rs render)
│   │   └── latency-predictor/             # XGBoost training + prediction sidecars and ConfigMaps
│   ├── provider/
│   │   ├── httproute/                     # Gateway API HTTPRoute bound to InferencePool
│   │   ├── gke/                           # GKE HealthCheckPolicy (HTTP) and GCPBackendPolicy
│   │   ├── gke-grpc/                      # Switches GKE HealthCheckPolicy to gRPC
│   │   └── istio/                         # Istio DestinationRule for EPP TLS
│   └── observability/
│       ├── metrics-auth/                  # ClusterRole/ClusterRoleBinding for tokenreviews and subjectaccessreviews
│       ├── monitoring-prometheus/         # Prometheus Operator ServiceMonitor without bearer auth
│       ├── monitoring-prometheus-auth/    # ServiceMonitor with ServiceAccount bearer token Secret
│       ├── monitoring-gmp/                # GKE Managed Prometheus PodMonitoring
│       └── tracing/                       # OpenTelemetry tracing flag and environment variables
└── overlays/
    ├── standalone-default/                # Reference standalone overlay with Prometheus monitoring
    ├── standalone-agentgateway/           # Standalone overlay with Agentgateway sidecar and endpoint-selector
    ├── standalone-proxy-service/          # Standalone overlay with separate Envoy Service and HPA
    ├── gateway-gke/                       # Gateway overlay with HTTPRoute, GKE policies, and GMP monitoring
    ├── gateway-istio/                     # Gateway overlay with HTTPRoute and Istio DestinationRule
    ├── full-featured/                     # Gateway overlay combining HA leader election, tokenizer, latency predictor, auth metrics, and tracing
    └── autoscaled/                        # Standalone overlay with EPP HPA and Rust tokenizer sidecar
```

## Customizing `EndpointPickerConfig`

`base/common` generates the `epp-config` `ConfigMap` from `epp-config.yaml` using `configMapGenerator`. Kustomize appends a content hash to the generated `ConfigMap` name and updates `Deployment/epp`'s volume reference, triggering a rollout whenever `epp-config.yaml` changes.

To provide a custom `EndpointPickerConfig` in an overlay, use `behavior: replace` on `configMapGenerator`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namePrefix: my-guide-

resources:
  - ../../base/standalone

configMapGenerator:
  - name: epp-config
    behavior: replace
    files:
      - epp-config.yaml
```

## Name Prefix Propagation

`base/common/namereference.yaml` registers custom `nameReference` rules so setting `namePrefix` in an overlay updates:

- `InferencePool.spec.endpointPickerRef.name` (`Service/epp`)
- `Deployment/epp` `--pool-name` argument (`InferencePool/inference-pool`)
- `HTTPRoute.spec.rules[*].backendRefs[*].name` (`InferencePool/inference-pool`)
- `InferenceObjective.spec.poolRef.name` (`InferencePool/inference-pool`)
- `HealthCheckPolicy.spec.targetRef.name` and `GCPBackendPolicy.spec.targetRef.name` (`InferencePool/inference-pool`)
- `DestinationRule.spec.host` (`Service/epp`)
- `Secret.metadata.annotations[kubernetes.io/service-account.name]` (`ServiceAccount/epp`)
- `ServiceMonitor.spec.endpoints[*].authorization.credentials.name` (`Secret`)
- `HorizontalPodAutoscaler.spec.scaleTargetRef.name` (`Deployment`)

## Validation

Run `make verify-kustomize` (or `./hack/verify-kustomize.sh`) to build all bases and overlays with `kustomize build`.
