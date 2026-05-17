package tui

import (
	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/amiraminb/coinwarrior/internal/service"
)

var (
	Svc  *service.Service
	Repo repository.Repository
)
