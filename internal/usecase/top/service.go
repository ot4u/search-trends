package top

import (
	"errors"

	"github.com/ot4/search-trends/internal/domain/trending"
)

var (
	ErrInvalidLimit  = errors.New("limit must be positive")
	ErrLimitTooLarge = errors.New("limit exceeds cached snapshot capacity")
	ErrNoSnapshots   = errors.New("no snapshots available")
)

type SnapshotReader interface {
	Get(limit int) (trending.Snapshot, bool)
	AvailableLimits() []int
}

type Reader interface {
	GetSnapshot(limit int) (trending.Snapshot, error)
}

type Service struct {
	reader SnapshotReader
}

func NewService(reader SnapshotReader) *Service {
	return &Service{reader: reader}
}

func (s *Service) GetSnapshot(limit int) (trending.Snapshot, error) {
	if limit <= 0 {
		return trending.Snapshot{}, ErrInvalidLimit
	}

	limits := s.reader.AvailableLimits()
	if len(limits) == 0 {
		return trending.Snapshot{}, ErrNoSnapshots
	}

	chosen := 0
	for _, candidate := range limits {
		if candidate >= limit {
			chosen = candidate
			break
		}
	}

	if chosen == 0 {
		return trending.Snapshot{}, ErrLimitTooLarge
	}

	snapshot, ok := s.reader.Get(chosen)
	if !ok {
		return trending.Snapshot{}, ErrNoSnapshots
	}

	if chosen == limit {
		return snapshot, nil
	}

	if len(snapshot.Items) > limit {
		snapshot.Items = append([]trending.Item(nil), snapshot.Items[:limit]...)
	}
	snapshot.Limit = limit

	return snapshot, nil
}
