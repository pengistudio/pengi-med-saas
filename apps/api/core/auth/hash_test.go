package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "mySuperSecretPassword123!"

	hash, err := HashPassword(password)

	// Verificar que no hay error al hashear
	if err != nil {
		t.Fatalf("Se esperaba nil error, se obtuvo: %v", err)
	}

	// Verificar que el hash no esté vacío
	if hash == "" {
		t.Errorf("El hash devuelto está vacío")
	}

	// Verificar que el hash es diferente al password original
	if hash == password {
		t.Errorf("El hash y la contraseña original son iguales. Esto no debería pasar.")
	}

	// Un hash de bcrypt siempre empieza con "$2a$", "$2b$" o "$2x$"
	if len(hash) < 4 || (hash[:4] != "$2a$" && hash[:4] != "$2b$" && hash[:4] != "$2x$") {
		t.Errorf("El formato del hash no parece ser de bcrypt, hash obtenido: %v", hash)
	}
}

func TestCompareHashAndPassword(t *testing.T) {
	password := "mySuperSecretPassword123!"
	wrongPassword := "WrongPassword!123"

	// Hashear la contraseña válida
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Error al preparar el test (hashear contraseña): %v", err)
	}

	// Caso de éxito: Comparar el hash con la contraseña correcta
	isValid := CompareHashAndPassword(hash, password)
	if !isValid {
		t.Errorf("Se esperaba que CompareHashAndPassword devolviera true para la contraseña correcta")
	}

	// Caso de fallo: Comparar el hash con una contraseña incorrecta
	isValid = CompareHashAndPassword(hash, wrongPassword)
	if isValid {
		t.Errorf("Se esperaba que CompareHashAndPassword devolviera false para una contraseña incorrecta")
	}
}
