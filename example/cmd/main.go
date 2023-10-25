package main

import (
	"context"
	"fmt"

	"github.com/chenyanchen/breaker"
	"github.com/chenyanchen/breaker/example"
	"github.com/chenyanchen/breaker/example/simple"
)

func main() {
	contentService := &contentService{}

	breakerContentService := simple.NewBreakerContentService(breaker.NewGoogleBreaker(), contentService)

	response, err := breakerContentService.GetContent(context.Background(), &example.GetContentRequest{})
	if err != nil {
		panic(err)
	}

	fmt.Println("response:", response)
}

type contentService struct{}

func (*contentService) GetContent(context.Context, *example.GetContentRequest) (*example.GetContentResponse, error) {
	return &example.GetContentResponse{}, nil
}
