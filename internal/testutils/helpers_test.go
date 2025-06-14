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

package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringPtr(t *testing.T) {
	original := "hello"
	ptr := ToPtr(original)
	if ptr == nil {
		t.Fatal("StringPtr returned nil")
	}
	if *ptr != original {
		t.Fatalf("expected %q, got %q", original, *ptr)
	}

	// Ensure the pointer value is independent of the original variable
	original = "world"
	if *ptr == original {
		t.Error("pointer value changed with the original variable")
	}
}

func TestIsPointer(t *testing.T) {
	value := "test"
	ptr := ToPtr(value)

	assert.IsType(t, *ptr, value)
}
