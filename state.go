package main

import (
	"github.com/horsedevours/blog-aggregator/internal/config"
	"github.com/horsedevours/blog-aggregator/internal/database"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}
