package batchhasher_test

import (
	"github.com/massyu/hive.go/batchhasher"
	"sync"
	"testing"

	"github.com/massyu/iota.go/trinary"
)

func BenchmarkBatchHasher_Hash(b *testing.B) {
	batchHasher := batchhasher.NewBatchHasher(243, 81)
	tritsToHash := make(trinary.Trits, 7500)

	b.ResetTimer()

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)

		go func() {
			batchHasher.Hash(tritsToHash)

			wg.Done()
		}()
	}
	wg.Wait()
}
