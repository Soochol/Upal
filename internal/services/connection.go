package services

import (
	"context"

	"github.com/soochol/upal/internal/crypto"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// ConnectionService manages external service connections with secret encryption.
type ConnectionService struct {
	repo repository.ConnectionRepository
	enc  *crypto.Encryptor
}

func NewConnectionService(repo repository.ConnectionRepository, enc *crypto.Encryptor) *ConnectionService {
	return &ConnectionService{repo: repo, enc: enc}
}

// Create encrypts secrets and stores a new connection.
func (s *ConnectionService) Create(ctx context.Context, conn *upal.Connection) error {
	if conn.ID == "" {
		conn.ID = upal.GenerateID("conn")
	}
	if err := s.encryptSecrets(conn); err != nil {
		return err
	}
	return s.repo.Create(ctx, conn)
}

// Get retrieves a connection by ID with secrets still encrypted.
func (s *ConnectionService) Get(ctx context.Context, id string) (*upal.Connection, error) {
	return s.repo.Get(ctx, id)
}

// Resolve retrieves a connection and decrypts its secrets for runtime use.
func (s *ConnectionService) Resolve(ctx context.Context, id string) (*upal.Connection, error) {
	conn, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.decryptSecrets(conn); err != nil {
		return nil, err
	}
	return conn, nil
}

// List returns all connections in safe (masked) form.
func (s *ConnectionService) List(ctx context.Context) ([]upal.ConnectionSafe, error) {
	conns, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	safe := make([]upal.ConnectionSafe, len(conns))
	for i, c := range conns {
		safe[i] = c.Safe()
	}
	return safe, nil
}

// Update encrypts secrets and updates a connection.
func (s *ConnectionService) Update(ctx context.Context, conn *upal.Connection) error {
	if err := s.encryptSecrets(conn); err != nil {
		return err
	}
	return s.repo.Update(ctx, conn)
}

// Delete removes a connection.
func (s *ConnectionService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ConnectionService) encryptSecrets(conn *upal.Connection) error {
	if conn.Password != "" {
		enc, err := s.enc.Encrypt(conn.Password)
		if err != nil {
			return err
		}
		conn.Password = enc
	}
	if conn.Token != "" {
		enc, err := s.enc.Encrypt(conn.Token)
		if err != nil {
			return err
		}
		conn.Token = enc
	}
	return nil
}

func (s *ConnectionService) decryptSecrets(conn *upal.Connection) error {
	if conn.Password != "" {
		dec, err := s.enc.Decrypt(conn.Password)
		if err != nil {
			return err
		}
		conn.Password = dec
	}
	if conn.Token != "" {
		dec, err := s.enc.Decrypt(conn.Token)
		if err != nil {
			return err
		}
		conn.Token = dec
	}
	return nil
}
