package provider

import (
	"context"
	"sync"
	"testing"
)

func TestCatalogueConcurrentReadersOwnTheirResults(t *testing.T) {
	c, err := NewGenerationCatalog()
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := c.Generations(context.Background(), "Mercedes Benz", "E280")
			if err != nil || len(rows) != 2 {
				t.Errorf("rows=%v err=%v", rows, err)
				return
			}
			rows[0].Generation = "changed by caller"
		}()
	}
	wg.Wait()
	rows, err := c.Generations(context.Background(), "Mercedes-Benz", "E 280")
	if err != nil || rows[0].Generation != "W124 facelift" {
		t.Fatal("caller mutated shared catalogue")
	}
}

func TestCatalogueRejectsBrokenData(t *testing.T) {
	for _, data := range []string{`{`, `[]`, `null`, `[{}]`, `[{"brand":"B","model":"M","generation":"G","production_year_start":2001,"production_year_end":1999}]`} {
		if _, err := parseGenerationCatalog(data); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}
