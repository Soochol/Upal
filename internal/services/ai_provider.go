package services

import (
	"context"
	"fmt"

	"github.com/soochol/upal/internal/crypto"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

type AIProviderService struct {
	repo repository.AIProviderRepository
	enc  *crypto.Encryptor
}

func NewAIProviderService(repo repository.AIProviderRepository, enc *crypto.Encryptor) *AIProviderService {
	return &AIProviderService{repo: repo, enc: enc}
}

// Create generates an ID, handles default logic, encrypts the API key, and persists.
func (s *AIProviderService) Create(ctx context.Context, p *upal.AIProvider) error {
	p.ID = upal.GenerateID("aip")
	if p.IsDefault {
		if err := s.repo.ClearDefault(ctx, p.Category); err != nil {
			return fmt.Errorf("clear default: %w", err)
		}
	}
	if err := s.encryptKey(p); err != nil {
		return err
	}
	return s.repo.Create(ctx, p)
}

// List returns safe views with no API keys exposed.
func (s *AIProviderService) List(ctx context.Context) ([]upal.AIProviderSafe, error) {
	providers, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	safe := make([]upal.AIProviderSafe, len(providers))
	for i, p := range providers {
		safe[i] = p.Safe()
	}
	return safe, nil
}

// Update handles default logic, encrypts the API key, and persists.
func (s *AIProviderService) Update(ctx context.Context, p *upal.AIProvider) error {
	if p.IsDefault {
		if err := s.repo.ClearDefault(ctx, p.Category); err != nil {
			return fmt.Errorf("clear default: %w", err)
		}
	}
	if err := s.encryptKey(p); err != nil {
		return err
	}
	return s.repo.Update(ctx, p)
}

// Delete removes a provider by ID.
func (s *AIProviderService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// SetDefault clears the existing default for the provider's category and marks this one as default.
func (s *AIProviderService) SetDefault(ctx context.Context, id string) error {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.ClearDefault(ctx, p.Category); err != nil {
		return fmt.Errorf("clear default: %w", err)
	}
	p.IsDefault = true
	return s.repo.Update(ctx, p)
}

// Resolve returns a provider with the API key decrypted, for runtime LLM building.
func (s *AIProviderService) Resolve(ctx context.Context, id string) (*upal.AIProvider, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.decryptKey(p); err != nil {
		return nil, err
	}
	return p, nil
}

// ListAll returns full provider objects with decrypted API keys (for building ProviderConfigs).
func (s *AIProviderService) ListAll(ctx context.Context) ([]*upal.AIProvider, error) {
	providers, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range providers {
		if err := s.decryptKey(p); err != nil {
			return nil, err
		}
	}
	return providers, nil
}

func (s *AIProviderService) encryptKey(p *upal.AIProvider) error {
	if p.APIKey == "" {
		return nil
	}
	encrypted, err := s.enc.Encrypt(p.APIKey)
	if err != nil {
		return fmt.Errorf("encrypt api key: %w", err)
	}
	p.APIKey = encrypted
	return nil
}

func (s *AIProviderService) decryptKey(p *upal.AIProvider) error {
	if p.APIKey == "" {
		return nil
	}
	decrypted, err := s.enc.Decrypt(p.APIKey)
	if err != nil {
		return fmt.Errorf("decrypt api key: %w", err)
	}
	p.APIKey = decrypted
	return nil
}
