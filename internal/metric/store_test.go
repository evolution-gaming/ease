// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metric

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/evolution-gaming/ease/internal/it"
)

// Number of iterations for stress scenarios.
var stressIter int = 1_000_000

func Test_Store_HappyPath(t *testing.T) {
	store := NewStore()

	var id1, id2 ID
	var r1, r2 Record
	r1 = Record{Name: "first"}
	r2 = Record{Name: "second"}

	// Insertion works as expected.
	id1 = store.Insert(r1)
	id2 = store.Insert(r2)

	t.Run("Retrieve all inserted IDs", func(t *testing.T) {
		want := slices.Sorted(slices.Values([]ID{id1, id2}))
		got := slices.Sorted(slices.Values(store.GetIDs()))
		it.Should(t, slices.Equal(got, want), "got = %v, want = %v", got, want)
	})

	t.Run("Inserted records exist", func(t *testing.T) {
		// Inserted records Exist!
		it.Should(t, store.Exists(id1), "record should exist: %v", id1)
		it.Should(t, store.Exists(id2), "record should exist: %v", id2)
	})

	t.Run("Inserted records can be retrieved", func(t *testing.T) {
		gotR1, err := store.Get(id1)
		it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)
		it.Should(t, gotR1 == r1, "got = %v, want = %v", gotR1, r1)
		gotR2, err := store.Get(id2)
		it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)
		it.Should(t, gotR2 == r2, "got = %v, want = %v", gotR2, r2)
	})

	t.Run("Update existing record", func(t *testing.T) {
		newRecord := Record{Name: "new name"}
		// Check that before update the new and old really are not equal.
		old, _ := store.Get(id1)
		it.Should(t, old != newRecord, "got = %v, want != %v", old, newRecord)

		// Now we do the update.
		err := store.Update(id1, newRecord)
		it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)
		// Retrieve updated record an compare, they should be equal.
		updated, _ := store.Get(id1)
		it.Should(t, updated == newRecord, "got = %v, want = %v", updated, newRecord)
	})

	t.Run("Delete record", func(t *testing.T) {
		id := store.Insert(Record{Name: "delete this record"})
		it.Should(t, store.Exists(id), "record should exist: %v", id)

		err := store.Delete(id)
		it.Should(t, err == nil, "Unexpected error: got = %v, want = nil", err)
		it.Should(t, !store.Exists(id), "record should not exist: %v", id)
	})
}

func Test_Store_SadPath(t *testing.T) {
	store := NewStore()
	nonExistentID := ID(100)

	t.Run("Error retrieving non-existent record", func(t *testing.T) {
		// Check that non existent record is indeed non-existent.
		it.Should(t, !store.Exists(nonExistentID), "record should not exist: %v", nonExistentID)
		_, err := store.Get(nonExistentID)
		it.Should(t, errors.Is(err, ErrRecordNotFound), "got = %v, want = %v", err, ErrRecordNotFound)
	})

	t.Run("Error updating non-existent record", func(t *testing.T) {
		err := store.Update(nonExistentID, Record{Name: "update"})
		it.Should(t, errors.Is(err, ErrRecordNotFound), "got = %v, want = %v", err, ErrRecordNotFound)
	})

	t.Run("Error deleting non-existent record", func(t *testing.T) {
		err := store.Delete(nonExistentID)
		it.Should(t, errors.Is(err, ErrRecordNotFound), "got = %v, want = %v", err, ErrRecordNotFound)
	})
}

func Test_Store_StressInsertDelete(t *testing.T) {
	var wg sync.WaitGroup
	var errCounter atomic.Int64
	store := NewStore()
	// Insert part stressing.
	for i := range stressIter {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			store.Insert(Record{Name: fmt.Sprintf("iter %d", iter)})
		}(i)
	}
	wg.Wait()

	it.Should(t, len(store.records) == stressIter, "got = %v, want = %v", len(store.records), stressIter)
	it.Should(t, len(store.GetIDs()) == stressIter, "got = %v, want = %v", len(store.GetIDs()), stressIter)
	// Delete part stressing.
	for _, id := range store.GetIDs() {
		wg.Add(1)
		go func(id ID) {
			defer wg.Done()
			if err := store.Delete(id); err != nil {
				errCounter.Add(1)
			}
		}(id)
	}
	wg.Wait()

	cnt := errCounter.Load()
	it.Should(t, cnt == 0, "Stress Delete caused %d errors", cnt)
}

func Test_Store_StressUpdate(t *testing.T) {
	var wg sync.WaitGroup
	var errCounter atomic.Int64
	store := NewStore()
	id := store.Insert(Record{Name: "first"})

	for i := range stressIter {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			if err := store.Update(id, Record{Name: fmt.Sprintf("update %d", iter)}); err != nil {
				errCounter.Add(1)
			}
		}(i)
	}
	wg.Wait()

	cnt := errCounter.Load()
	it.Should(t, cnt == 0, "Stress Update caused %d errors", cnt)
}
