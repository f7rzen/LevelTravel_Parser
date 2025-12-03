package main

import (
	"fmt"

	"LevelTravel_Parser/internal/parser"
)

func main() {
	url := "https://level.travel/search/Moscow-RU-to-Any-TR-departure-08.04.2026-for-7-nights-2-adults-0-kids-1..5-stars-package-type"

	p := parser.NewParser(url)

	hotels := p.ParseHotels()

	fmt.Println("\n=== Итог ===")
	fmt.Println("Отелей собрано:", len(hotels))

	parser.SaveToJSON(hotels, "hotels.json")
}
