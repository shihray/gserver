package selector

import (
	"testing"

	"github.com/shihray/gserver/registry"
)

func TestStrategies(t *testing.T) {
	testData := []*registry.Service{
		&registry.Service{
			Name:    "test1",
			ID:      "test1-1",
			Address: "10.0.0.1",
		},
		&registry.Service{
			Name:    "test1",
			ID:      "test1-3",
			Address: "10.0.0.3",
		},
	}

	for name, strategy := range map[string]Strategy{"random": Random, "roundrobin": RoundRobin} {
		next := strategy(testData)
		counts := make(map[string]int)

		for i := 0; i < 100; i++ {
			node, err := next()
			if err != nil {
				t.Fatal(err)
			}
			counts[node.ID]++
		}

		t.Logf("%s: %+v\n", name, counts)
	}
}
