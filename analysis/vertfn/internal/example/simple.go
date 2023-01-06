package example

import "log"

func (s *Oven) WarmUp() *Oven {
	s.temperature += 42
	return s
}

func mixFlour() {
	log.Println("mixing flour~~")
}

func MakePancake() {
	visitStore()

	mixFlour()

	FindOven().WarmUp().Bake()

	Enjoy("pancake")
}

func buyFlour() {
	log.Println("bought flour")
}

func visitStore() {
	buyFlour()
}

type Oven struct {
	temperature float32
}

func FindOven() *Oven { return &Oven{} }

func (s *Oven) Bake() {
	log.Println("done!")
}

func Enjoy(food string) {
	log.Println(food + " is so good")
}
