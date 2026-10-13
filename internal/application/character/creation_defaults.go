package character

import (
	"context"
	"fmt"

	"mcgame-server/internal/domain/character"
)

type CreationDefaultsProvider interface {
	NewCharacterDefaults(context.Context) (character.NewCharacterDefaults, error)
}

type CreationPackageWriter interface {
	CreateWithPackage(context.Context, *character.Character, *character.CreationPackage) error
}

func (s *Service) SetCreationDefaults(provider CreationDefaultsProvider, writer CreationPackageWriter) {
	s.creationDefaults = provider
	s.creationWriter = writer
}

func (s *Service) prepareCreationPackage(ctx context.Context, char *character.Character) (*character.CreationPackage, error) {
	defaults := character.DefaultNewCharacterDefaults()
	if s.creationDefaults != nil {
		var err error
		defaults, err = s.creationDefaults.NewCharacterDefaults(ctx)
		if err != nil {
			return nil, err
		}
	}
	defaults = defaults.Clone()
	if err := defaults.Validate(); err != nil {
		return nil, err
	}
	applyStarterRewards(char, defaults)
	return s.buildCreationPackage(char, defaults)
}

func (s *Service) persistCreation(ctx context.Context, char *character.Character, bundle *character.CreationPackage) error {
	if s.creationWriter != nil {
		return s.creationWriter.CreateWithPackage(ctx, char, bundle)
	}
	if s.creationDefaults != nil || len(bundle.Items) != 0 || len(bundle.Pets) != 0 {
		return fmt.Errorf("atomic creation package persistence is unavailable")
	}
	return s.repo.Create(ctx, char)
}
