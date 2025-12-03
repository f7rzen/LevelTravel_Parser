package model

type Hotel struct {
	Name          string   `json:"name"`            //Название
	Stars         int      `json:"stars"`           //Звезды
	Location      string   `json:"location"`        //Локация
	Rating        float64  `json:"rating"`          //Рейтинг отеля
	Reviews       int      `json:"reviews"`         //Количество оценок
	Photos        []string `json:"photos"`          //Фотографии
	URL           string   `json:"url"`             //Ссылка
	Price         int      `json:"price"`           //Цена
	BeachType     string   `json:"beach_type"`      //Тип пляжа
	DistanceToSea int      `json:"distance_to_sea"` //Расстояние до моря
	BeachLine     int      `json:"beach_line"`      //Линияя береговая

	Tags             []string `json:"tags"`
	ConstructionYear int      `json:"construction_year"` // Год постройки
	RenovationYear   int      `json:"renovation_year"`   // Год реновации
	TotalArea        int      `json:"total_area"`        // Общая площадь
	WorkInWinter     bool     `json:"work_in_winter"`    // Работает ли зимой
	HasFreeWiFi      bool     `json:"has_free_wifi"`     //Имеет ли бесплатный WiFi
	HasPool          bool     `json:"has_pool"`          //Имеет ли басейн
	HasHeatedPool    bool     `json:"has_heated_pool"`   //Имеет ли подогреваемый басейн
	HasAquapark      bool     `json:"has_aquapark"`      //Имеет ли аквапарк
	HasKidsMenu      bool     `json:"has_kids_menu"`     //Имеет ли детское меню
	HasBar           bool     `json:"has_bar"`           //Имеет ли бар
	IsAirportNear    bool     `json:"is_airport_near"`   //Рядом ли аэропорт
	MealTypeString   string   `json:"meal_type_string"`  // Тип питания
	MealTypeRank     int      `json:"meal_type_rank"`    // Рейтинг питания
}

type Result struct {
	Source    string  `json:"source"`
	Timestamp string  `json:"timestamp"`
	Hotels    []Hotel `json:"hotels"`
}
