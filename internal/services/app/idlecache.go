package app

import "github.com/alexbelweb/flibustahub/internal/db"

func (s *Service) catalogDB() *db.DB {
	if s == nil {
		return nil
	}
	return s.Catalog()
}

func (s *Service) WindowAway() {
	if c := s.catalogDB(); c != nil {
		c.WindowAway()
	}
}

func (s *Service) WindowBack() {
	if c := s.catalogDB(); c != nil {
		c.WindowBack()
	}
}

func (s *Service) holdCache() {
	if c := s.catalogDB(); c != nil {
		c.BeginHeavy()
	}
}

func (s *Service) releaseAfterHeavy() {
	if c := s.catalogDB(); c != nil {
		c.EndHeavy()
	}
}
