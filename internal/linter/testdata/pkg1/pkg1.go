package pkg1

import (
	"log"
	"os"
)

func mulfunc(i int) (int, error) {
	return i * 2, nil
}

func errCheckFunc() {
	// формулируем ожидания: анализатор должен находить ошибку,
	// описанную в комментарии want

	panic("")   // want "panic func is used"
	log.Fatal() // want "log.Fatal is used outside of main package"
	os.Exit(1)  // want "os.Exit is used outside of main package"
}
