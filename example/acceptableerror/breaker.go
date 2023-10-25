package acceptableerror

import (
	"context"
	"errors"

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
	getContentFn := func() (err error) {
		resp, err = s.contentService.GetContent(ctx, req)
		return err
	}

	// handle acceptable errors
	getContentFn = handleAcceptableErrors(getContentFn, example.ErrContentNotFound)

	err := s.breaker.Do(getContentFn)
	return resp, err
}

func handleAcceptableErrors(fn func() error, acceptableErrors ...error) func() error {
	return func() error {
		err := fn()
		if err == nil {
			return nil
		}

		for _, target := range acceptableErrors {
			if errors.Is(err, target) {
				// TODO: do something, like log
				return nil
			}
		}

		return err
	}
}
