package util

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func randomEmail() string {
	domains := []string{"gmail.com", "yahoo.com", "hotmail.com", "outlook.com", "example.com"}
	domain := domains[rand.Intn(len(domains))]

	return fmt.Sprintf("test_%s@%s", uuid.New().String()[:8], domain)
}

func randomName() string {
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

func randomEmailPg() pgtype.Text {
	return pgtype.Text{
		String: randomEmail(),
		Valid:  true,
	}
}

func randomNamePg() pgtype.Text {
	return pgtype.Text{
		String: randomName(),
		Valid:  true,
	}
}

func GetRandomEmail() pgtype.Text {
	return randomEmailPg()
}

func GetRandomName() pgtype.Text {
	return randomNamePg()
}
