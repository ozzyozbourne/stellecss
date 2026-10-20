package web

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/ozzyozbourne/stellecss/internal/bundle"
	"github.com/ozzyozbourne/stellecss/internal/config"
)

// keepBundles is how many past stylesheets stay servable. A page that loaded
// a hash just before an update must still be able to fetch it.
const keepBundles = 8

// store owns the config file, the current bundle and the live-refresh
// subscribers. Every change goes through [store.apply], which validates,
// rebuilds, writes the config and output, then notifies subscribers, in that
// order, so a subscriber never sees a hash it cannot fetch.
type store struct {
	path string

	mu     sync.RWMutex
	cfg    *config.Config
	b      *bundle.Bundle
	byHash map[string][]byte
	order  []string
	subs   map[chan string]struct{}
}

func openStore(path string) (*store, error) {
	c, err := config.Load(path)
	if errors.Is(err, fs.ErrNotExist) {
		c = config.Default()
		err = c.Save(path)
	}
	if err != nil {
		return nil, err
	}
	s := &store{path: path, byHash: map[string][]byte{}, subs: map[chan string]struct{}{}}
	return s, s.apply(c)
}

// snapshot returns the current config (a private copy) and bundle.
func (s *store) snapshot() (*config.Config, *bundle.Bundle) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Clone(), s.b
}

func (s *store) css(hash string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.byHash[hash]
	return b, ok
}

// apply makes c the current config.
func (s *store) apply(c *config.Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	b, err := bundle.Build(c)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := c.Save(s.path); err != nil {
		return err
	}
	if err := writeOutput(s.path, c, b); err != nil {
		return err
	}
	s.cfg, s.b = c.Clone(), b
	if _, ok := s.byHash[b.Hash]; !ok {
		s.byHash[b.Hash] = b.CSS
		s.order = append(s.order, b.Hash)
		if len(s.order) > keepBundles {
			delete(s.byHash, s.order[0])
			s.order = s.order[1:]
		}
	}
	for ch := range s.subs {
		// Drop a pending hash in favour of the newest; only the latest
		// stylesheet matters to a subscriber.
		select {
		case <-ch:
		default:
		}
		ch <- b.Hash
	}
	return nil
}

func (s *store) subscribe() (ch chan string, cancel func()) {
	ch = make(chan string, 1)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	return ch, func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}
}

func writeOutput(cfgPath string, c *config.Config, b *bundle.Bundle) error {
	if c.Bundle.Output == "" {
		return nil
	}
	out := filepath.Join(filepath.Dir(cfgPath), c.Bundle.Output)
	if !c.Bundle.Overwrite {
		if _, err := os.Stat(out); err == nil {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	return os.WriteFile(out, b.CSS, 0o644)
}
