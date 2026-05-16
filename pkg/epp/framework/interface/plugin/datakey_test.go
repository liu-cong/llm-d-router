/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testDataKey struct {
	dataType     string
	producerName string
}

func (t testDataKey) DataType() string     { return t.dataType }
func (t testDataKey) ProducerName() string { return t.producerName }
func (t testDataKey) WithNonEmptyProducerName(name string) DataKey {
	if name != "" {
		t.producerName = name
	}
	return t
}
func (t testDataKey) String() string {
	return t.dataType + "/" + t.producerName
}

func newTestDataKey(dataType, defaultProducerName string) testDataKey {
	return testDataKey{
		dataType:     dataType,
		producerName: defaultProducerName,
	}
}

func TestDataKey_String(t *testing.T) {
	tests := []struct {
		name     string
		key      DataKey
		expected string
	}{
		{
			name:     "Unscoped uses DefaultProducerType",
			key:      newTestDataKey("KeyA", "ProdTypeA"),
			expected: "KeyA/ProdTypeA",
		},
		{
			name:     "Scoped uses ProducerName",
			key:      newTestDataKey("KeyA", "ProdTypeA").WithNonEmptyProducerName("ProdNameA"),
			expected: "KeyA/ProdNameA",
		},
		{
			name:     "Scoped with empty name does not override",
			key:      newTestDataKey("KeyA", "ProdTypeA").WithNonEmptyProducerName(""),
			expected: "KeyA/ProdTypeA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cast to FrameworkDataKey to call String() if we want, or use SerializeDataKey.
			// Since testDataKey implements String(), but interface DataKey doesn't have it,
			// we must cast or call SerializeDataKey.
			if stringer, ok := tt.key.(interface{ String() string }); ok {
				assert.Equal(t, tt.expected, stringer.String())
			} else {
				t.Fatalf("key does not implement String()")
			}
		})
	}
}


