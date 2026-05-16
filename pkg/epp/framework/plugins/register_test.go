package plugins

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/llm-d/llm-d-router/pkg/epp/framework/interface/plugin"
	"github.com/llm-d/llm-d-router/pkg/epp/framework/plugins/requestcontrol/dataproducer/inflightload"
)

func TestRegisterAllPluginsRegistersInflightLoadDefaultProducer(t *testing.T) {
	RegisterAllPlugins()

	require.Contains(t, plugin.Registry, inflightload.InFlightLoadProducerType)
}
