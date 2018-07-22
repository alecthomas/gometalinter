package main

import (
	"fmt"
	"github.com/nbutton23/zxcvbn-go"
)

func main() {
	password := "Testaaatyhg890l33t"

	passwordStenght := zxcvbn.PasswordStrength(password, nil)

	fmt.Printf(
		`Password score    (0-4): %d
Estimated entropy (bit): %f
Estimated time to crack: %s%s`,
		passwordStenght.Score,
		passwordStenght.Entropy,
		passwordStenght.CrackTimeDisplay, "\n",
	)
}
