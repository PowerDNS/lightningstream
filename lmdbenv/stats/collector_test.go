package stats

import (
	"fmt"
	"testing"

	"github.com/PowerDNS/lightningstream/lmdbenv"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/prometheus/client_golang/prometheus"
)

func TestCollector(t *testing.T) {
	err := lmdbenv.TestEnv(func(env *lmdb.Env) error {
		names := []string{"foo", "bar"}
		err := env.Update(func(txn *lmdb.Txn) error {
			for _, name := range names {
				if _, err := txn.CreateDBI(name); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		c := NewCollector(true)
		c.AddTarget("test", names, env)
		ch := make(chan prometheus.Metric, 1000) // buffer large enough for Collect
		c.Collect(ch)
		close(ch)

		var metrics []prometheus.Metric
		for m := range ch {
			metrics = append(metrics, m)
		}

		if len(metrics) < 14 {
			return fmt.Errorf("too few metrics: %d", len(metrics))
		}

		return nil
	})
	if err != nil {
		t.Errorf("returned error: %v", err)
	}
}
