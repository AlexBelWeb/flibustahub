package app

import (
	"context"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/services/catalog"
	"github.com/alexbelweb/flibustahub/internal/services/personal"
)

func (s *Service) personalSvc() (*personal.Service, error) {
	st := s.snap()
	if st.catalog == nil {
		return nil, apperr.New(apperr.CodeDBOpenFailed, nil)
	}
	return personal.New(st.catalog, s.Logger()), nil
}

func (s *Service) PersonalSnapshot(ctx context.Context) (personal.Snapshot, error) {
	p, err := s.personalSvc()
	if err != nil {
		return personal.Snapshot{}, err
	}
	return p.Snapshot(ctx)
}

func (s *Service) SetWorkRating(ctx context.Context, id int64, rating *int) error {
	p, err := s.personalSvc()
	if err != nil {
		return err
	}
	return p.SetRating(ctx, id, rating)
}

func (s *Service) SetWorkComment(ctx context.Context, id int64, comment string) error {
	p, err := s.personalSvc()
	if err != nil {
		return err
	}
	return p.SetComment(ctx, id, comment)
}

func (s *Service) SetWorkWantToRead(ctx context.Context, id int64, want bool) error {
	p, err := s.personalSvc()
	if err != nil {
		return err
	}
	return p.SetWantToRead(ctx, id, want)
}

func (s *Service) ExportPersonal(ctx context.Context, path string) (personal.ExportResult, error) {
	p, err := s.personalSvc()
	if err != nil {
		return personal.ExportResult{}, err
	}
	return p.Export(ctx, path)
}

func (s *Service) PreviewPersonalImport(ctx context.Context, path string) (personal.ImportReport, error) {
	p, err := s.personalSvc()
	if err != nil {
		return personal.ImportReport{}, err
	}
	return p.PreviewImport(ctx, path)
}

func (s *Service) ImportPersonal(ctx context.Context, path string) (personal.ImportReport, error) {
	p, err := s.personalSvc()
	if err != nil {
		return personal.ImportReport{}, err
	}
	return p.Import(ctx, path)
}

func (s *Service) GetHome(ctx context.Context) (catalog.HomeDashboard, error) {
	c, err := s.catalogSvc()
	if err != nil {
		return catalog.HomeDashboard{}, err
	}
	return c.Home(ctx)
}
