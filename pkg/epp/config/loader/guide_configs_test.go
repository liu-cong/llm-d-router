package loader

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"

	configapi "github.com/llm-d/llm-d-inference-scheduler/apix/config/v1alpha1"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins"
	fwkplugin "github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/interface/plugin"
	igwtestutils "github.com/llm-d/llm-d-inference-scheduler/test/utils/igw"

	// Import GIE built-in plugins for registration
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/scorer/prefix"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/picker/maxscore"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/picker/random"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/picker/weightedrandom"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/profilehandler/single"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/scorer/kvcacheutilization"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/scorer/queuedepth"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/scorer/runningrequests"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/scorer/loraaffinity"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/scorer/tokenload"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/flowcontrol/fairness/globalstrict"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/flowcontrol/fairness/roundrobin"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/flowcontrol/ordering/fcfs"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/flowcontrol/ordering/edf"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/flowcontrol/ordering/slodeadline"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/flowcontrol/usagelimits"
	reqdataprodprefix "github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/requestcontrol/dataproducer/approximateprefix"
	attrprefix "github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/datalayer/attribute/prefix"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/requestcontrol/dataproducer/inflightload"
	attrconcurrency "github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/datalayer/attribute/concurrency"
	latencyproducer "github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/requestcontrol/dataproducer/predictedlatency"
	attrlatency "github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/datalayer/attribute/latency"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/requestcontrol/admitter/latencyslo"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/requestcontrol/admitter/probabilisticadmitter"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/filter/prefixcacheaffinity"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/filter/sloheadroomtier"
	latencyscorer "github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/scorer/latency"
	testfilter "github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/scheduling/test/filter"
	testresponsereceived "github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/requestcontrol/test/responsereceived"
	sourcemetrics "github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/datalayer/source/metrics"
	extractormetrics "github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/datalayer/extractor/metrics"
	sourcenotifications "github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/datalayer/source/notifications"
	requestattributereporter "github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/requestcontrol/requestattributereporter"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/requesthandling/parsers/openai"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/requesthandling/parsers/vllmgrpc"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/requesthandling/parsers/passthrough"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/requesthandling/parsers/vertexai"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/flowcontrol/saturationdetector/concurrency"
	"github.com/llm-d/llm-d-inference-scheduler/pkg/epp/framework/plugins/flowcontrol/saturationdetector/utilization"
)

func init() {
	plugins.RegisterAllPlugins()

	fwkplugin.Register(prefix.PrefixCacheScorerPluginType, prefix.PrefixCachePluginFactory)
	fwkplugin.Register(maxscore.MaxScorePickerType, maxscore.MaxScorePickerFactory)
	fwkplugin.Register(random.RandomPickerType, random.RandomPickerFactory)
	fwkplugin.Register(weightedrandom.WeightedRandomPickerType, weightedrandom.WeightedRandomPickerFactory)
	fwkplugin.Register(single.SingleProfileHandlerType, single.SingleProfileHandlerFactory)
	fwkplugin.Register(kvcacheutilization.KvCacheUtilizationScorerType, kvcacheutilization.KvCacheUtilizationScorerFactory)
	fwkplugin.Register(queuedepth.QueueScorerType, queuedepth.QueueScorerFactory)
	fwkplugin.Register(runningrequests.RunningRequestsSizeScorerType, runningrequests.RunningRequestsSizeScorerFactory)
	fwkplugin.Register(loraaffinity.LoraAffinityScorerType, loraaffinity.LoraAffinityScorerFactory)
	fwkplugin.Register(tokenload.TokenLoadScorerType, tokenload.TokenLoadScorerFactory)
	fwkplugin.Register(globalstrict.GlobalStrictFairnessPolicyType, globalstrict.GlobalStrictFairnessPolicyFactory)
	fwkplugin.Register(roundrobin.RoundRobinFairnessPolicyType, roundrobin.RoundRobinFairnessPolicyFactory)
	fwkplugin.Register(fcfs.FCFSOrderingPolicyType, fcfs.FCFSOrderingPolicyFactory)
	fwkplugin.Register(edf.EDFOrderingPolicyType, edf.EDFOrderingPolicyFactory)
	fwkplugin.Register(slodeadline.SLODeadlineOrderingPolicyType, slodeadline.SLODeadlineOrderingPolicyFactory)
	fwkplugin.Register(usagelimits.StaticUsageLimitPolicyType, usagelimits.StaticPolicyFactory)
	fwkplugin.RegisterAsDefaultProducer(reqdataprodprefix.ApproxPrefixCachePluginType, reqdataprodprefix.ApproxPrefixCacheFactory, attrprefix.PrefixCacheMatchInfoKey)
	fwkplugin.RegisterAsDefaultProducer(inflightload.InFlightLoadProducerType, inflightload.InFlightLoadProducerFactory, attrconcurrency.InFlightLoadKey)
	fwkplugin.RegisterAsDefaultProducer(latencyproducer.LatencyDataProviderPluginType, latencyproducer.PredictedLatencyFactory, attrlatency.LatencyPredictionInfoKey)
	fwkplugin.Register(latencyslo.LatencyAdmissionPluginType, latencyslo.LatencyAdmissionFactory)
	fwkplugin.Register(probabilisticadmitter.Type, probabilisticadmitter.Factory)
	fwkplugin.Register(prefixcacheaffinity.PluginType, prefixcacheaffinity.Factory)
	fwkplugin.Register(sloheadroomtier.PluginType, sloheadroomtier.Factory)
	fwkplugin.Register(latencyscorer.LatencyScorerType, latencyscorer.Factory)
	fwkplugin.Register(testfilter.HeaderBasedTestingFilterType, testfilter.HeaderBasedTestingFilterFactory)
	fwkplugin.Register(testresponsereceived.DestinationEndpointServedVerifierType, testresponsereceived.DestinationEndpointServedVerifierFactory)
	fwkplugin.Register(sourcemetrics.MetricsDataSourceType, sourcemetrics.MetricsDataSourceFactory)
	fwkplugin.Register(extractormetrics.MetricsExtractorType, extractormetrics.CoreMetricsExtractorFactory)
	fwkplugin.Register(sourcenotifications.NotificationSourceType, sourcenotifications.NotificationSourceFactory)
	fwkplugin.Register(sourcenotifications.EndpointNotificationSourceType, sourcenotifications.EndpointSourceFactory)
	fwkplugin.Register(requestattributereporter.RequestAttributeReporterType, requestattributereporter.RequestAttributeReporterPluginFactory)
	fwkplugin.Register(openai.OpenAIParserType, openai.OpenAIParserPluginFactory)
	fwkplugin.Register(vllmgrpc.VllmGRPCParserType, vllmgrpc.VllmGRPCParserPluginFactory)
	fwkplugin.Register(passthrough.PassthroughParserType, passthrough.PassthroughParserPluginFactory)
	fwkplugin.Register(vertexai.VertexAIParserType, vertexai.VertexAIParserPluginFactory)
	fwkplugin.Register(concurrency.ConcurrencyDetectorType, concurrency.ConcurrencyDetectorFactory)
	fwkplugin.Register(utilization.UtilizationDetectorType, utilization.UtilizationDetectorFactory)
}

func TestGuideConfigs(t *testing.T) {
	RegisterFeatureGate("flowControl")
	RegisterFeatureGate("dataLayer")

	tests := []struct {
		name       string
		configText string
		skipReason string
		validate   func(t *testing.T, rawCfg *configapi.EndpointPickerConfig)
	}{
		{
			name: "flow-control",
			configText: `
apiVersion: inference.networking.x-k8s.io/v1alpha1
kind: EndpointPickerConfig
featureGates:
- flowControl
plugins:
- type: queue-scorer
- type: kv-cache-utilization-scorer
- type: prefix-cache-scorer
- type: round-robin-fairness-policy
- type: fcfs-ordering-policy
- type: concurrency-detector
  parameters:
    maxConcurrency: 132
    concurrencyMode: requests
    headroom: 0.0
schedulingProfiles:
- name: default
  plugins:
`,
			validate: func(t *testing.T, rawCfg *configapi.EndpointPickerConfig) {
				require.GreaterOrEqual(t, len(rawCfg.Plugins), 6)
				require.Equal(t, "queue-scorer", rawCfg.Plugins[0].Type)
				require.Equal(t, "kv-cache-utilization-scorer", rawCfg.Plugins[1].Type)
				require.Equal(t, "prefix-cache-scorer", rawCfg.Plugins[2].Type)
				require.Equal(t, "round-robin-fairness-policy", rawCfg.Plugins[3].Type)
				require.Equal(t, "fcfs-ordering-policy", rawCfg.Plugins[4].Type)
				require.Equal(t, "concurrency-detector", rawCfg.Plugins[5].Type)
				require.Contains(t, string(rawCfg.Plugins[5].Parameters), "132")
			},
		},
		{
			name: "optimized-baseline",
			configText: `
apiVersion: inference.networking.x-k8s.io/v1alpha1
kind: EndpointPickerConfig
plugins:
- type: queue-scorer
- type: kv-cache-utilization-scorer
- type: prefix-cache-scorer
- type: no-hit-lru-scorer
schedulingProfiles:
- name: default
  plugins:
  - pluginRef: queue-scorer
    weight: 2
  - pluginRef: kv-cache-utilization-scorer
    weight: 2
  - pluginRef: prefix-cache-scorer
    weight: 3
  - pluginRef: no-hit-lru-scorer
    weight: 2
`,
			validate: func(t *testing.T, rawCfg *configapi.EndpointPickerConfig) {
				require.GreaterOrEqual(t, len(rawCfg.Plugins), 4)
				require.GreaterOrEqual(t, len(rawCfg.SchedulingProfiles[0].Plugins), 4)
				require.Equal(t, 2.0, *rawCfg.SchedulingProfiles[0].Plugins[0].Weight)
				require.Equal(t, 3.0, *rawCfg.SchedulingProfiles[0].Plugins[2].Weight)
			},
		},
		{
			name: "pd-disaggregation",
			configText: `
apiVersion: inference.networking.x-k8s.io/v1alpha1
kind: EndpointPickerConfig
plugins:
- type: disagg-headers-handler
- type: always-disagg-pd-decider
- type: disagg-profile-handler
  parameters:
    deciderPluginName: always-disagg-pd-decider
- type: prefill-filter
- type: decode-filter
- type: prefix-cache-scorer
- type: queue-scorer
- type: kv-cache-utilization-scorer
- type: active-request-scorer
- type: max-score-picker
schedulingProfiles:
- name: prefill
  plugins:
  - pluginRef: prefill-filter
`,
			validate: func(t *testing.T, rawCfg *configapi.EndpointPickerConfig) {
				require.GreaterOrEqual(t, len(rawCfg.Plugins), 10)
				require.Equal(t, "disagg-profile-handler", rawCfg.Plugins[2].Type)
				require.Contains(t, string(rawCfg.Plugins[2].Parameters), "always-disagg-pd-decider")
				require.Equal(t, "prefill", rawCfg.SchedulingProfiles[0].Name)
			},
		},
		{
			name: "precise-prefix-cache-aware",
			skipReason: "Fails to instantiate because the tokenizer plugin requires an actual UDS socket to be available during initialization: dial unix /tmp/tokenizer/tokenizer-uds.socket: connect: no such file or directory",
			configText: `
apiVersion: inference.networking.x-k8s.io/v1alpha1
kind: EndpointPickerConfig
plugins:
  - type: tokenizer
    parameters:
      modelName: Qwen/Qwen3-32B
      udsTokenizerConfig:
        socketFile: /tmp/tokenizer/tokenizer-uds.socket
  - type: endpoint-notification-source
  - type: metrics-data-source
  - type: core-metrics-extractor
  - type: single-profile-handler
  - type: decode-filter
  - type: precise-prefix-cache-scorer
    parameters:
      tokenProcessorConfig:
        blockSize: 64           # must match vLLM --block-size
      speculativeIndexing: true
      indexerConfig:
`,
			validate: func(t *testing.T, rawCfg *configapi.EndpointPickerConfig) {
				require.GreaterOrEqual(t, len(rawCfg.Plugins), 7)
				require.Equal(t, "tokenizer", rawCfg.Plugins[0].Type)
				require.Contains(t, string(rawCfg.Plugins[0].Parameters), "Qwen/Qwen3-32B")
				require.Equal(t, "precise-prefix-cache-scorer", rawCfg.Plugins[6].Type)
			},
		},
		{
			name: "predicted-latency-slo",
			configText: `
apiVersion: inference.networking.x-k8s.io/v1alpha1
kind: EndpointPickerConfig
plugins:
- type: queue-scorer
- type: kv-cache-utilization-scorer
- type: prefix-cache-scorer
- type: metrics-data-source
  parameters:
    insecureSkipVerify: true
    path: /metrics
    scheme: http
- type: core-metrics-extractor
- type: predicted-latency-producer
  parameters:
    streamingMode: true
- type: prefix-cache-affinity-filter
  name: strict-affinity-filter
  parameters:
    affinityThreshold: 0.99
`,
			validate: func(t *testing.T, rawCfg *configapi.EndpointPickerConfig) {
				require.GreaterOrEqual(t, len(rawCfg.Plugins), 7)
				require.Equal(t, "metrics-data-source", rawCfg.Plugins[3].Type)
				require.Equal(t, "predicted-latency-producer", rawCfg.Plugins[5].Type)
				require.Contains(t, string(rawCfg.Plugins[5].Parameters), "true")
				require.Equal(t, "strict-affinity-filter", rawCfg.Plugins[6].Name)
			},
		},
		{
			name: "wide-ep-lws",
			configText: `
apiVersion: inference.networking.x-k8s.io/v1alpha1
kind: EndpointPickerConfig
plugins:
- type: disagg-headers-handler
- type: always-disagg-pd-decider
- type: disagg-profile-handler
  parameters:
    deciderPluginName: always-disagg-pd-decider
- type: prefill-filter
- type: decode-filter
- type: prefix-cache-scorer
- type: queue-scorer
- type: kv-cache-utilization-scorer
- type: active-request-scorer
- type: max-score-picker
schedulingProfiles:
- name: prefill
  plugins:
  - pluginRef: prefill-filter
`,
			validate: func(t *testing.T, rawCfg *configapi.EndpointPickerConfig) {
				require.GreaterOrEqual(t, len(rawCfg.Plugins), 10)
				require.Equal(t, "disagg-profile-handler", rawCfg.Plugins[2].Type)
			},
		},
		{
			name: "tiered-prefix-cache-cpu",
			configText: `
apiVersion: inference.networking.x-k8s.io/v1alpha1
kind: EndpointPickerConfig
plugins:
- type: queue-scorer
- type: kv-cache-utilization-scorer
- type: prefix-cache-scorer
  name: gpu-prefix-cache-scorer
- type: prefix-cache-scorer
  name: cpu-prefix-cache-scorer
  parameters:
    autoTune: false  # vLLM doesn't have the CPU capacity metric to enable autoTune yet
    lruCapacityPerServer: 41000  # Allocating ~100GB for Qwen-32B: 41,000 blocks * 2.5MB/block (based on 160KB/token * 16 block size).
schedulingProfiles:
- name: default
  plugins:
  - pluginRef: queue-scorer
    weight: 2
  - pluginRef: kv-cache-utilization-scorer
    weight: 2.0
`,
			validate: func(t *testing.T, rawCfg *configapi.EndpointPickerConfig) {
				require.GreaterOrEqual(t, len(rawCfg.Plugins), 4)
				require.Equal(t, "gpu-prefix-cache-scorer", rawCfg.Plugins[2].Name)
				require.Equal(t, "cpu-prefix-cache-scorer", rawCfg.Plugins[3].Name)
				require.Contains(t, string(rawCfg.Plugins[3].Parameters), "41000")
			},
		},
		{
			name: "experimental-dp-aware-inferencepool",
			skipReason: "Fails to instantiate: failed to create plugin 'pd-profile-handler': invalid decider plugin type: prefix-based-pd-decider (likely missing explicit decider parameter matching the default)",
			configText: `
apiVersion: inference.networking.x-k8s.io/v1alpha1
kind: EndpointPickerConfig
plugins:
- type: prefill-header-handler
- type: prefill-filter
- type: decode-filter
- type: prefix-cache-scorer
- type: active-request-scorer
- type: queue-scorer
- type: pd-profile-handler
  parameters:
    threshold: 0
    hashBlockSize: 5
schedulingProfiles:
- name: prefill
  plugins:
`,
			validate: func(t *testing.T, rawCfg *configapi.EndpointPickerConfig) {
				require.GreaterOrEqual(t, len(rawCfg.Plugins), 7)
				require.Equal(t, "pd-profile-handler", rawCfg.Plugins[6].Type)
				require.Contains(t, string(rawCfg.Plugins[6].Parameters), "5")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			logger := logr.Discard()
			rawCfg, _, err := LoadRawConfig([]byte(tt.configText), logger)
			if err != nil {
				t.Fatalf("Failed to load raw configuration: %v", err)
			}
			
			handle := igwtestutils.NewTestHandle(context.Background())
			_, err = InstantiateAndConfigure(rawCfg, handle, logger)
			require.NoError(t, err, "Configuration failed to instantiate")

			if tt.validate != nil {
				tt.validate(t, rawCfg)
			}
		})
	}
}
