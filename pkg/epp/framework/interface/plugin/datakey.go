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

import "fmt"

// DataKey is the public interface for plugins to express data dependencies.
// It only requires String() for unique identification (format: "DataType/ProducerName").
type DataKey interface {
	String() string
}

// BaseDataKey provides a default implementation of DataKey.
// Concrete key types should embed this struct.
type BaseDataKey struct {
	dataType     string
	producerName string
}

// NewBaseDataKey creates a new BaseDataKey.
// The defaultProducerName is passed as the initial producerName.
func NewBaseDataKey(dataType, defaultProducerName string) BaseDataKey {
	return BaseDataKey{
		dataType:     dataType,
		producerName: defaultProducerName,
	}
}

// WithNonEmptyProducerName returns a copy of the key with the specified producer name
// if the name is not empty, otherwise returns the key unchanged.
func (b BaseDataKey) WithNonEmptyProducerName(name string) BaseDataKey {
	if name != "" {
		b.producerName = name
	}
	return b
}

// String serializes the key to "DataType/ProducerName".
func (b BaseDataKey) String() string {
	return fmt.Sprintf("%s/%s", b.dataType, b.producerName)
}
