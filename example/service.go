package example

import (
	"context"
	"errors"
)

type ContentService interface {
	GetContent(ctx context.Context, req *GetContentRequest) (*GetContentResponse, error)
}

type (
	GetContentRequest  struct{}
	GetContentResponse struct{}
)

var ErrContentNotFound = errors.New("content not found")
