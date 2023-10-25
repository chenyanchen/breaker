package simple

import (
	"context"

	"github.com/chenyanchen/breaker"

	"github.com/chenyanchen/breaker/example"
)

type breakerContentService struct {
	breaker breaker.Breaker

	contentService example.ContentService
}

func NewBreakerContentService(breaker breaker.Breaker, contentService example.ContentService) example.ContentService {
	return &breakerContentService{
		breaker:        breaker,
		contentService: contentService,
	}
}

func (s *breakerContentService) GetContent(ctx context.Context, req *example.GetContentRequest) (*example.GetContentResponse, error) {
	var resp *example.GetContentResponse
	err := s.breaker.Do(func() (err error) {
		resp, err = s.contentService.GetContent(ctx, req)
		return err
	})
	return resp, err
}
