package contactinquiry

import (
	"database/sql"
	"fmt"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(in CreateContactInquiryInput) (string, error) {
	var id string
	err := s.db.QueryRow(`
		INSERT INTO contact_inquiries
			(first_name, last_name, email, phone, message)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, in.FirstName, in.LastName, in.Email, in.Phone, in.Message).Scan(&id)
	return id, err
}

func (s *Store) List(limit, offset int) ([]ContactInquiry, error) {
	rows, err := s.db.Query(`
		SELECT id, first_name, last_name, email, phone, message, is_read, created_at::text
		FROM contact_inquiries
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inquiries []ContactInquiry
	for rows.Next() {
		var inquiry ContactInquiry
		if err := rows.Scan(
			&inquiry.ID, &inquiry.FirstName, &inquiry.LastName,
			&inquiry.Email, &inquiry.Phone, &inquiry.Message,
			&inquiry.IsRead, &inquiry.CreatedAt,
		); err != nil {
			return nil, err
		}
		inquiries = append(inquiries, inquiry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return inquiries, nil
}

func (s *Store) Get(id string) (*ContactInquiry, error) {
	var inquiry ContactInquiry
	err := s.db.QueryRow(`
		SELECT id, first_name, last_name, email, phone, message, is_read, created_at::text
		FROM contact_inquiries
		WHERE id = $1
	`, id).Scan(
		&inquiry.ID, &inquiry.FirstName, &inquiry.LastName,
		&inquiry.Email, &inquiry.Phone, &inquiry.Message,
		&inquiry.IsRead, &inquiry.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &inquiry, nil
}

func (s *Store) Count() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM contact_inquiries`).Scan(&count)
	return count, err
}

func validateLimit(page, limit int) (int, int, error) {
	if page < 1 {
		return 0, 0, fmt.Errorf("page must be greater than zero")
	}
	if limit < 1 || limit > 100 {
		return 0, 0, fmt.Errorf("limit must be between 1 and 100")
	}
	return limit, (page - 1) * limit, nil
}
