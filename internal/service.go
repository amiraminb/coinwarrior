package internal

import "github.com/amiraminb/coinwarrior/internal/repository"

type Service struct {
	repo repository.Repository
}

func NewService(repo repository.Repository) *Service {
	return &Service{repo: repo}
}
