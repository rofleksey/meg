package meg

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func CheckPasswordHash(hashedPassword, pass string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(pass))
	return err == nil
}
