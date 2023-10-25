package fallback

import (
	"context"

	"github.com/chenyanchen/breaker"
	"github.com/chenyanchen/breaker/example"
)

type breakerContentService struct {
	breaker breaker.Breaker

	contentService example.ContentService

	defaultContent *example.GetContentResponse
}

func NewBreakerContentService(
	breaker breaker.Breaker,
	contentService example.ContentService,
	defaultContent *example.GetContentResponse,
) example.ContentService {
	return &breakerContentService{
		breaker:        breaker,
		contentService: contentService,
		defaultContent: defaultContent,
	}
}

func (s *breakerContentService) GetContent(ctx context.Context, req *example.GetContentRequest) (*example.GetContentResponse, error) {
	var resp *example.GetContentResponse
	err := s.breaker.Do(func() (err error) {
		resp, err = s.contentService.GetContent(ctx, req)
		return err
	})
	if err == nil {
		return resp, nil
	}

	// do fallback strategy
	if s.defaultContent != nil {
		return s.defaultContent, nil
	}

	return resp, err
}
