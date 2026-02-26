package services

import (
	"context"

	"github.com/soochol/upal/internal/crypto"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

type ConnectionService struct {
	repo repository.ConnectionRepository
	enc  *crypto.Encryptor
}

func NewConnectionService(repo repository.ConnectionRepository, enc *crypto.Encryptor) *ConnectionService {
	return &ConnectionService{repo: repo, enc: enc}
}

func (s *ConnectionService) Create(ctx context.Context, conn *upal.Connection) error {
	if conn.ID == "" {
		conn.ID = upal.GenerateID("conn")
	}
	if err := s.encryptSecrets(conn); err != nil {
		return err
	}
	return s.repo.Create(ctx, conn)
}

func (s *ConnectionService) Get(ctx context.Context, id string) (*upal.Connection, error) {
	return s.repo.Get(ctx, id)
}

// Resolve retrieves a connection with secrets decrypted for runtime use.
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

func (s *ConnectionService) Update(ctx context.Context, conn *upal.Connection) error {
	if err := s.encryptSecrets(conn); err != nil {
		return err
	}
	return s.repo.Update(ctx, conn)
}

func (s *ConnectionService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ConnectionService) encryptSecrets(conn *upal.Connection) error {
	return s.transformSecrets(conn, s.enc.Encrypt)
}

func (s *ConnectionService) decryptSecrets(conn *upal.Connection) error {
	return s.transformSecrets(conn, s.enc.Decrypt)
}

func (s *ConnectionService) transformSecrets(conn *upal.Connection, fn func(string) (string, error)) error {
	for _, field := range []*string{&conn.Password, &conn.Token} {
		if *field == "" {
			continue
		}
		val, err := fn(*field)
		if err != nil {
			return err
		}
		*field = val
	}
	return nil
}
