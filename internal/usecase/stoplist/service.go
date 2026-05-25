package stoplist

import (
	"errors"
	"strings"

	domainstoplist "github.com/ot4/search-trends/internal/domain/stoplist"
	"github.com/ot4/search-trends/internal/domain/trending"
)

var (
	ErrEmptyWord       = errors.New("stop-list word is empty after normalization")
	ErrWordMustBeToken = errors.New("stop-list accepts a single normalized token")
)

type Manager interface {
	List() []string
	Add(raw string) (string, bool, error)
	Remove(raw string) (string, bool, error)
}

type Service struct {
	store *domainstoplist.Store
}

func NewService(store *domainstoplist.Store) *Service {
	if store == nil {
		store = domainstoplist.NewStore(nil)
	}

	return &Service{store: store}
}

func (s *Service) List() []string {
	return s.store.List()
}

func (s *Service) Add(raw string) (string, bool, error) {
	word, err := normalizeWord(raw)
	if err != nil {
		return "", false, err
	}

	return word, s.store.Add(word), nil
}

func (s *Service) Remove(raw string) (string, bool, error) {
	word, err := normalizeWord(raw)
	if err != nil {
		return "", false, err
	}

	return word, s.store.Remove(word), nil
}

func normalizeWord(raw string) (string, error) {
	word := trending.NormalizeText(raw)
	if word == "" {
		return "", ErrEmptyWord
	}
	if strings.ContainsRune(word, ' ') {
		return "", ErrWordMustBeToken
	}

	return word, nil
}
