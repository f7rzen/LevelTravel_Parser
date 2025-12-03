package parser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"

	"LevelTravel_Parser/internal/model"
)

type Parser struct {
	Page *rod.Page
}

// Создание парсера с открытием страницы
func NewParser(url string) *Parser {
	l := launcher.New().Headless(true)
	browser := rod.New().ControlURL(l.MustLaunch()).MustConnect()

	page := browser.MustPage(url)
	page.MustWaitLoad()

	fmt.Println("Ждем динамического контента...")
	time.Sleep(5 * time.Second)

	return &Parser{
		Page: page,
	}
}

// Функции Загрузки Страницы
func (p *Parser) ScrollToLoadHotels(targetCount int) {
	lastCount := 0
	for {
		cards, _ := p.Page.Elements("[data-testid='hotel-card']")
		count := len(cards)

		if count >= targetCount {
			fmt.Println("✅ Достигнуто количество отелей:", count)
			return
		}

		if count == lastCount && count > 0 {
			fmt.Println("⚠ Больше отелей не загружается, остановка на:", count)
			return
		}

		if count == 0 && lastCount == 0 {
			_, _ = p.Page.Eval(`window.scrollBy(0, window.innerHeight)`)
		} else if count > 0 {
			lastCard := cards[count-1]
			_ = lastCard.ScrollIntoView()
		}

		lastCount = count
		time.Sleep(2 * time.Second)
	}
}

func (p *Parser) ParseHotels() []model.Hotel {
	// Скроллим и загружаем 100 карточек
	p.ScrollToLoadHotels(100)

	elements, err := p.Page.Elements("[data-testid='hotel-card']")
	if err != nil {
		log.Println("Ошибка поиска карточек отелей:", err)
		return nil
	}

	fmt.Printf("Найдено %d карточек для базового парсинга.\n", len(elements))

	hotels := make([]model.Hotel, 0, len(elements))
	for _, el := range elements {
		hotel := model.Hotel{
			Name:          parseName(el),
			Stars:         parseStars(el),
			Location:      parseLocation(el),
			Rating:        parseRating(el),
			Reviews:       parseReviews(el),
			Photos:        parsePhotos(el),
			URL:           parseURL(el),
			Price:         parsePrice(el),
			BeachType:     parseBeachType(el),
			DistanceToSea: parseDistanceToSea(el),
			BeachLine:     parseBeachLine(el),
		}
		hotels = append(hotels, hotel)

	}

	fmt.Println("Начало параллельного сбора детальных данных...")
	hotels = p.ParseHotelDetailsParallel(hotels)

	return hotels
}

func (p *Parser) ParseHotelDetailsParallel(hotels []model.Hotel) []model.Hotel {
	var wg sync.WaitGroup

	// Ограничитель горутин
	limit := make(chan struct{}, 10)

	fmt.Printf("Запуск %d горутин для сбора деталей (лимит: %d).\n", len(hotels), cap(limit))

	for i := range hotels {
		if hotels[i].URL == "" {
			continue
		}

		wg.Add(1)
		limit <- struct{}{} // Занимаем "слот"

		go func(index int) {
			defer wg.Done()
			defer func() { <-limit }() // Освобождаем "слот"

			fmt.Printf("  -> Открытие страницы для парсинга: %s\n", hotels[index].Name)

			// Создаем новую вкладку/страницу в браузере
			page := p.Page.Browser().MustPage(hotels[index].URL)
			defer page.MustClose() // Обязательно закрываем вкладку

			// Ожидаем загрузки контента
			page.MustWaitLoad()
			time.Sleep(2 * time.Second) // Ждем динамический контент

			hotels[index].Tags = parseTagsDetails(page)

			year, renovation, area := parseDetailsFromDescription(page)
			hotels[index].ConstructionYear = year
			hotels[index].RenovationYear = renovation
			hotels[index].TotalArea = area

			hotels[index].WorkInWinter = parseWorkInWinter(page)

			facts := parseHotelFacts(page)
			hotels[index].HasFreeWiFi = facts["HasFreeWiFi"]
			hotels[index].HasPool = facts["HasPool"]
			hotels[index].HasHeatedPool = facts["HasHeatedPool"]
			hotels[index].HasAquapark = facts["HasAquapark"]
			hotels[index].HasKidsMenu = facts["HasKidsMenu"]
			hotels[index].HasBar = facts["HasBar"]
			hotels[index].IsAirportNear = facts["IsAirportNear"]

			mealString, mealRank := parseMealType(page)
			hotels[index].MealTypeString = mealString
			hotels[index].MealTypeRank = mealRank

			fmt.Printf("  <- Готово: %s\n", hotels[index].Name)

		}(i)
	}

	wg.Wait()
	fmt.Println("✅ Параллельный сбор данных завершен.")
	return hotels
}

func parseName(el *rod.Element) string {
	nameEl, err := el.Element("[data-testid='hotel-card__title']")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(nameEl.MustText())
}

func parseStars(el *rod.Element) int {
	starsContainer, err := el.Element(".hotelCardStars-module__starsContainer__H3sFD")
	if err != nil {
		return 0
	}

	starIcons, err := starsContainer.Elements("svg.hotelCardStars-module__starIcon__LT1hY")
	if err != nil {
		return 0
	}

	return len(starIcons)
}

func parseLocation(el *rod.Element) string {
	locEl, err := el.Element(".hotelCardLocation-module__locationDesktop__A6M4m")
	if err != nil {
		return ""
	}

	text, _ := locEl.Text()
	return strings.TrimSpace(text)
}

func parseRating(el *rod.Element) float64 {
	rEl, err := el.Element(".HotelRatingSquare__RatingText-sc-1v3txw8-1")
	if err != nil {
		return 0
	}

	text, err := rEl.Text()
	if err != nil {
		return 0
	}

	rating, _ := strconv.ParseFloat(strings.TrimSpace(text), 64)
	return rating
}

func parseReviews(el *rod.Element) int {
	rEl, err := el.Element(".HotelRatingSquare__ReviewsCount-sc-1v3txw8-2")
	if err != nil {
		return 0
	}

	span, err := rEl.Element("span")
	if err != nil {
		return 0
	}

	text, _ := span.Text()
	cleaned := strings.ReplaceAll(text, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")

	n, _ := strconv.Atoi(strings.TrimSpace(cleaned))
	return n
}

func parsePhotos(el *rod.Element) []string {
	photos := []string{}
	photoEls, err := el.Elements("img")
	if err != nil {
		return photos
	}
	for _, img := range photoEls {
		src, _ := img.Attribute("src")
		if src != nil && *src != "" {
			photos = append(photos, *src)
		}
	}
	return photos
}

func parseURL(el *rod.Element) string {
	linkEl, err := el.Element("a")
	if err != nil {
		return ""
	}
	href, _ := linkEl.Attribute("href")
	if href != nil {
		return "https://level.travel" + *href
	}
	return ""
}

func parsePrice(el *rod.Element) int {
	priceEl, err := el.Element(".hotelCardPrice-module__styledPrice__sFPQW")
	if err != nil {
		return 0
	}

	text, err := priceEl.Text()
	if err != nil {
		return 0
	}

	cleaned := strings.ReplaceAll(text, "от", "")
	cleaned = strings.ReplaceAll(cleaned, "₽", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "\u00a0", "")

	price, _ := strconv.Atoi(cleaned)
	return price
}

func parseBeachType(el *rod.Element) string {
	labels, _ := el.Elements(".hotelFeature-module__label__8UW3c")
	for _, l := range labels {
		txt, _ := l.Text()
		txt = strings.TrimSpace(txt)

		if txt == "песок" || txt == "галька" || txt == "пес./гал." || txt == "платф." {
			return txt
		}
	}
	return ""
}

func parseDistanceToSea(el *rod.Element) int {
	labels, _ := el.Elements(".hotelFeature-module__label__8UW3c")
	for _, l := range labels {
		txt, _ := l.Text()
		txt = strings.TrimSpace(txt)

		if strings.Contains(txt, "м") {
			clean := strings.ReplaceAll(txt, "м", "")
			clean = strings.TrimSpace(clean)
			n, _ := strconv.Atoi(clean)
			return n
		}
	}
	return 0
}

func parseBeachLine(el *rod.Element) int {
	items, _ := el.Elements(".hotelFeature-module__feature__5iqtF")

	for _, item := range items {
		pathEl, _ := item.Element("svg path")
		if pathEl == nil {
			continue
		}

		d, _ := pathEl.Attribute("d")
		if d == nil {
			continue
		}

		path := *d

		switch {
		case strings.Contains(path, "H14.9c-1.034") || strings.Contains(path, "v-4.174H14.9"):
			return 1
		case strings.Contains(path, "H6.096v-.856") || strings.Contains(path, "d=\"M3.099"):
			return 2
		case strings.Contains(path, "c0-.902-.738") || strings.Contains(path, "1.845h1.058"):
			return 3
		}
	}

	return 0
}

func parseTagsDetails(page *rod.Page) []string {
	tagElements, err := page.Elements(`.Labels_labelList__x3tfP`)
	if err != nil {
		return []string{}
	}

	tags := []string{}
	for _, li := range tagElements {
		txt, _ := li.Text()
		txt = strings.TrimSpace(txt)
		if txt != "" {
			tags = append(tags, txt)
		}
	}

	return tags
}

func parseDetailsFromDescription(page *rod.Page) (year, renovation, area int) {
	el, err := page.Element(".AboutHotel_hotelDescription__ETBJW")
	if err != nil {
		log.Printf("Описание отеля не найдено на странице: %s", page.MustInfo().URL)
		return 0, 0, 0
	}

	text, err := el.Text()
	if err != nil {
		return 0, 0, 0
	}

	reYear := regexp.MustCompile(`Построен:\s*(\d{4})\s*г\.`)
	matchYear := reYear.FindStringSubmatch(text)
	if len(matchYear) > 1 {
		year, _ = strconv.Atoi(matchYear[1])
	}

	reRenovation := regexp.MustCompile(`Реновация:\s*(\d{4})\s*г\.`)
	matchRenovation := reRenovation.FindStringSubmatch(text)
	if len(matchRenovation) > 1 {
		renovation, _ = strconv.Atoi(matchRenovation[1])
	}

	reArea := regexp.MustCompile(`Общая площадь:\s*([\d\s]+)\s*(?:м2|кв\.м\.)`)
	matchArea := reArea.FindStringSubmatch(text)
	if len(matchArea) > 1 {
		cleanedAreaString := strings.ReplaceAll(matchArea[1], " ", "")

		area, _ = strconv.Atoi(cleanedAreaString)
	}

	return year, renovation, area
}

func parseWorkInWinter(page *rod.Page) bool {
	winterClosingPhrase := "Отель закрывается на зимний период"

	descriptionEl, _ := page.Elements(".AboutHotel_importantEventDescription__mXqvw")

	if len(descriptionEl) == 0 {
		return true
	}

	text, err := descriptionEl[0].Text()
	if err != nil {
		return true
	}

	if strings.Contains(strings.ToLower(text), strings.ToLower(winterClosingPhrase)) {
		return false
	}

	return true
}

func parseHotelFacts(page *rod.Page) map[string]bool {
	facts := map[string]bool{
		"HasFreeWiFi":   false,
		"HasPool":       false,
		"HasHeatedPool": false,
		"HasAquapark":   false,
		"HasKidsMenu":   false,
		"HasBar":        false,
		"IsAirportNear": false,
	}

	keywords := map[string]string{
		"HasFreeWiFi":   "Бесплатный Wi-Fi",
		"HasHeatedPool": "Подогреваемый бассейн",
		"HasAquapark":   "Аквапарк",
		"HasKidsMenu":   "Детское меню",
		"HasBar":        "Бар",
		"IsAirportNear": "Рядом аэропорт",
	}

	factsContainer, err := page.Element(".HotelFactsContent_features__8xr0p")
	if err != nil {
		return facts
	}

	titleElements, err := factsContainer.Elements(".HotelFactsContent_featureTitle__gud53")
	if err != nil {
		return facts
	}

	foundTitles := make(map[string]struct{})
	for _, el := range titleElements {
		text, _ := el.Text()
		foundTitles[strings.TrimSpace(text)] = struct{}{}
	}

	for factKey, keyword := range keywords {
		if _, ok := foundTitles[keyword]; ok {
			facts[factKey] = true
		}
	}

	if _, ok := foundTitles["Бассейн"]; ok {
		facts["HasPool"] = true
	}

	if facts["HasHeatedPool"] {
		facts["HasPool"] = true
	}

	return facts
}

func getMealRank(mealType string) int {

	mealTypeLower := strings.ToLower(mealType)

	switch {
	case strings.Contains(mealTypeLower, "ультра всё включено"):
		return 6
	case strings.Contains(mealTypeLower, "всё включено"):
		return 5
	case strings.Contains(mealTypeLower, "завтрак, обед, ужин +"):
		return 5
	case strings.Contains(mealTypeLower, "завтрак, обед, ужин"):
		return 4
	case strings.Contains(mealTypeLower, "завтрак и ужин"):
		return 3
	case strings.Contains(mealTypeLower, "завтрак"):
		return 2
	case strings.Contains(mealTypeLower, "ужин"):
		return 1
	case strings.Contains(mealTypeLower, "без питания"):
		return 0
	default:
		return -1
	}
}

func parseMealType(page *rod.Page) (meal string, rank int) {
	mealEl, err := page.Element(".MatrixCell_mealText__x_puB")
	if err != nil {
		return "", -1
	}

	text, _ := mealEl.Text()
	meal = strings.TrimSpace(text)
	rank = getMealRank(meal)

	return meal, rank
}

func SaveToJSON(hotels []model.Hotel, filename string) {
	data := model.Result{
		Source:    "Level.Travel",
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Hotels:    hotels,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Println("JSON Marshal error:", err)
		return
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		log.Println("Ошибка записи файла:", err)
		return
	}

	fmt.Printf("✅ Данные сохранены в %s\n", filename)
}
