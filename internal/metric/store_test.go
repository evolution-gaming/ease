// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metric

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
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
		ids := store.GetIDs()
		assert.ElementsMatch(t, []ID{id1, id2}, ids)
	})

	t.Run("Inserted records exist", func(t *testing.T) {
		// Inserted records Exist!
		assert.True(t, store.Exists(id1))
		assert.True(t, store.Exists(id2))
	})

	t.Run("Inserted records can be retrieved", func(t *testing.T) {
		gotR1, err := store.Get(id1)
		assert.NoError(t, err)
		assert.Equal(t, r1, gotR1)
		gotR2, err := store.Get(id2)
		assert.NoError(t, err)
		assert.Equal(t, r2, gotR2)
	})

	t.Run("Update existing record", func(t *testing.T) {
		new := Record{Name: "new name"}
		// Check that before update the new and old really are not equal.
		old, _ := store.Get(id1)
		assert.NotEqual(t, old, new)

		// Now we do the update.
		err := store.Update(id1, new)
		assert.NoError(t, err)
		// Retrieve updated record an compare, they should be equal.
		updated, _ := store.Get(id1)
		assert.Equal(t, new, updated)
	})

	t.Run("Delete record", func(t *testing.T) {
		id := store.Insert(Record{Name: "delete this record"})
		assert.True(t, store.Exists(id))

		err := store.Delete(id)
		assert.NoError(t, err)
		assert.False(t, store.Exists(id))
	})
}

func Test_Store_SadPath(t *testing.T) {
	store := NewStore()
	nonExistentID := ID(100)

	t.Run("Error retrieving non-existent record", func(t *testing.T) {
		// Check that non existent record is indeed non-existent.
		assert.False(t, store.Exists(nonExistentID))
		_, err := store.Get(nonExistentID)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("Error updating non-existent record", func(t *testing.T) {
		err := store.Update(nonExistentID, Record{Name: "update"})
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("Error deleting non-existent record", func(t *testing.T) {
		err := store.Delete(nonExistentID)
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})
}

func Test_Store_StressInsertDelete(t *testing.T) {
	var wg sync.WaitGroup
	var errCounter atomic.Int64
	store := NewStore()
	// Insert part stressing.
	for i := 0; i < stressIter; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			store.Insert(Record{Name: fmt.Sprintf("iter %d", iter)})
		}(i)
	}
	wg.Wait()

	assert.Len(t, store.records, stressIter)
	assert.Len(t, store.GetIDs(), stressIter)
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

	if cnt := errCounter.Load(); cnt != 0 {
		t.Errorf("Stress Delete caused %d errors", cnt)
	}
}

func Test_Store_StressUpdate(t *testing.T) {
	var wg sync.WaitGroup
	var errCounter atomic.Int64
	store := NewStore()
	id := store.Insert(Record{Name: "first"})

	for i := 0; i < stressIter; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			if err := store.Update(id, Record{Name: fmt.Sprintf("update %d", iter)}); err != nil {
				errCounter.Add(1)
			}
		}(i)
	}
	if cnt := errCounter.Load(); cnt != 0 {
		t.Errorf("Stress Update caused %d errors", cnt)
	}
}
