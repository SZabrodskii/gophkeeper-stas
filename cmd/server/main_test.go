package main

import (
	"testing"

	"go.uber.org/fx"
)

func TestDependencyGraph(t *testing.T) {
	if err := fx.ValidateApp(createApp()); err != nil {
		t.Errorf("dependency graph validation failed: %v", err)
	}
}
