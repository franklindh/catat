package util

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func ParseUUID(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func CreateZeroBalance() pgtype.Numeric {
	// Jika Anda menggunakan math/big
	// return pgtype.Numeric{
	// 	Int:   big.NewInt(0),
	// 	Exp:   0,
	// 	Valid: true,
	// }

	// Atau jika menggunakan cara lain sesuai kebutuhan
	return pgtype.Numeric{Valid: true}
}
