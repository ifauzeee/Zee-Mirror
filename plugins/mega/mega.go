package mega

import (
	"zee-mirror/internal/config"
)

func init() {
}
type MegaEngine struct {
	Config *config.Config
}

func NewMegaEngine(cfg *config.Config) *MegaEngine {
	return &MegaEngine{Config: cfg}
}
