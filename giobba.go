package giobba

import (
	"fmt"

	"github.com/joeg-ita/giobba/src/usecases"
)

func Giobba() {
	fmt.Println("Giobba")
	giobba := usecases.NewGiobbaStart()
	giobba.Run()

}
