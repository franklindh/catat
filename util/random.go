package util

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

func RandomName() string {
	firstNames := []string{
		"John", "Jane", "Michael", "Sarah", "David", "Emily", "Robert", "Lisa",
		"William", "Jennifer", "Thomas", "Michelle", "Charles", "Amy", "Daniel",
		"Angela", "Matthew", "Jessica", "Christopher", "Ashley", "James", "Amanda",
		"Brian", "Melissa", "Kevin", "Stephanie", "Mark", "Nicole", "Steven",
		"Rachel", "Paul", "Heather", "Timothy", "Elizabeth", "Jason", "Megan",
	}

	lastNames := []string{
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller",
		"Davis", "Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez",
		"Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin",
		"Lee", "Perez", "Thompson", "White", "Harris", "Sanchez", "Clark",
		"Ramirez", "Lewis", "Robinson", "Walker", "Young", "Allen", "King",
	}

	firstName := firstNames[rand.Intn(len(firstNames))]
	lastName := lastNames[rand.Intn(len(lastNames))]

	return fmt.Sprintf("%s %s", firstName, lastName)
}

func RandomString(n int) string {
	var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1)
}

func RandomBalance() pgtype.Numeric {
	maxAmountInSmallestUnit := int64(1000000000000)
	randomAmountInSmallestUnit := rand.Int63n(maxAmountInSmallestUnit)

	return pgtype.Numeric{
		Int:   big.NewInt(randomAmountInSmallestUnit),
		Exp:   -4,
		Valid: true,
	}
}

func RandomNumeric(min, max int64) string {
	return fmt.Sprintf("%d.0000", min+rand.Int63n(max-min+1))
}

func RandomEmail() string {
	return fmt.Sprintf("%s@email.com", RandomString(6))
}

func RandomGoogleID() string {
	return fmt.Sprintf("googleid_%s", RandomString(10))
}

func PgxUUIDToGoogleUUID(pgxUUID pgtype.UUID) uuid.UUID {
	if !pgxUUID.Valid {
		return uuid.Nil
	}
	return pgxUUID.Bytes
}

func GoogleUUIDToPgxUUID(googleUUID uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: googleUUID,
		Valid: googleUUID != uuid.Nil,
	}
}

func RandomPasetoKey() string {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func RandomPasetoKeyString() string {
	return base64.StdEncoding.EncodeToString(make([]byte, 24))[:32]
}
