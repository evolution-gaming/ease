// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Centralised store of various encode metrics.

package metric

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrRecordNotFound = errors.New("record not found")

type ID int64

type Store struct {
	mu      sync.RWMutex
	records map[ID]Record
	next    ID
}

func NewStore() *Store {
	return &Store{
		records: make(map[ID]Record),
	}
}

func (s *Store) Insert(r Record) ID {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[s.next] = r
	id := s.next
	s.next++

	return id
}

func (s *Store) Get(id ID) (Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.records[id]
	if !ok {
		return r, fmt.Errorf("getting record: %w", ErrRecordNotFound)
	}

	return r, nil
}

func (s *Store) Exists(id ID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.records[id]

	return exists
}

func (s *Store) GetIDs() []ID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]ID, 0, len(s.records))
	for id := range s.records {
		ids = append(ids, id)
	}
	return ids
}

func (s *Store) Update(id ID, r Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.records[id]; !exists {
		return fmt.Errorf("updating record: %w", ErrRecordNotFound)
	}

	s.records[id] = r
	return nil
}

func (s *Store) Delete(id ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.records[id]; !exists {
		return fmt.Errorf("deleting record: %w", ErrRecordNotFound)
	}

	delete(s.records, id)
	return nil
}

// Record contains metrics for a single encode.
type Record struct {
	Name             string
	SourceFile       string
	CompressedFile   string
	VQMResultFile    string
	Cmd              string
	HStime           string
	HUtime           string
	HElapsed         string
	Stime            time.Duration
	Utime            time.Duration
	Elapsed          time.Duration
	MaxRss           int64
	VideoDuration    float64
	AvgEncodingSpeed float64

	PSNRMin          float64
	PSNRMax          float64
	PSNRMean         float64
	PSNRHarmonicMean float64
	PSNRStDev        float64
	PSNRVariance     float64

	MS_SSIMMin          float64
	MS_SSIMMax          float64
	MS_SSIMMean         float64
	MS_SSIMHarmonicMean float64
	MS_SSIMStDev        float64
	MS_SSIMVariance     float64

	VMAFMin          float64
	VMAFMax          float64
	VMAFMean         float64
	VMAFHarmonicMean float64
	VMAFStDev        float64
	VMAFVariance     float64

	BitrateMin  float64
	BitrateMax  float64
	BitrateMean float64
}
