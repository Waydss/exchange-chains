package simulation

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/L-ka080/DDB-6-case/internal/category"
	"github.com/L-ka080/DDB-6-case/internal/item"
	"github.com/L-ka080/DDB-6-case/internal/user"
	"github.com/L-ka080/DDB-6-case/internal/wish"
)

// simulatedPassword is the shared login password of every simulated user.
const simulatedPassword = "simulation-password"

// Conditions mirrors the CHECK constraint of the items table.
var itemConditions = []item.Condition{
	item.ConditionNew,
	item.ConditionLikeNew,
	item.ConditionGood,
	item.ConditionFair,
	item.ConditionPoor,
}

// itemNamePools maps a category name to candidate item titles. Titles mimic
// real marketplace listing names: product type, brand, and key specs. Item
// titles and wish keywords are drawn from the same pool, so fuzzy title
// matching between a give and a wish within one category works.
var itemNamePools = map[string][]string{
	"Гитары": {
		"Бас-гитара Fender 4 струны",
		"Электрогитара Gibson Les Paul",
		"Акустическая гитара Yamaha F310",
		"Классическая гитара Cordoba C5",
		"Бас-гитара Yamaha 5 струн",
		"Электрогитара Ibanez RG",
		"Акустическая гитара 12 струн",
	},
	"Пианино": {
		"Пианино фортепиано 88 клавиш",
		"Цифровое пианино Yamaha P-125",
		"Синтезатор Casio CT-X 61 клавиша",
		"Рояль акустический",
		"Электронное пианино Roland FP-30",
	},
	"Палатки": {
		"Палатка кемпинг 4 местная",
		"Палатка туристическая 2 местная",
		"Палатка для зимней рыбалки",
		"Тент от дождя туристический",
		"Палатка походная с тамбуром",
	},
	"Барабаны": {
		"Барабанная установка Yamaha 5 предметов",
		"Малый барабан 14 дюймов",
		"Барабан 10 дюймов",
		"Электронная барабанная установка Roland",
	},
	"Клавишные": {
		"MIDI-клавиатура 49 клавиш",
		"Синтезатор Yamaha 61 клавиша",
		"MIDI-контроллер Arturia",
		"Электроорган",
	},
	"Автомобили": {
		"Автомобиль Lada Vesta 1.6",
		"Седан Toyota Camry 2.5",
		"Внедорожник УАЗ Патриот",
		"Хэтчбек Kia Rio 1.6",
		"Автомобиль BMW 3 серии",
	},
	"Гаражи": {
		"Гараж металлический 6х3",
		"Гараж ракушка",
		"Металлический гараж 6 на 4",
		"Бокс гаражный",
	},
	"Телефоны": {
		"iPhone 16 Pro Max 256GB",
		"iPhone 15 128GB",
		"Samsung Galaxy S24 Ultra",
		"Xiaomi Redmi Note 13 Pro",
		"Google Pixel 8 256GB",
		"iPhone 14 128GB",
		"Samsung Galaxy A55",
	},
	"Ноутбуки": {
		"Ноутбук Apple MacBook Air M2",
		"Ноутбук Lenovo ThinkPad T14",
		"Игровой ноутбук ASUS ROG",
		"Ноутбук HP Pavilion 15",
		"Ультрабук Xiaomi Mi Notebook",
	},
	"Телевизоры": {
		"Телевизор Samsung 55 дюймов 4K",
		"Телевизор LG OLED 65",
		"Smart TV Xiaomi 43 дюйма",
		"Телевизор Sony Bravia 50",
	},
	"Холодильники": {
		"Холодильник двухкамерный Indesit",
		"Холодильник LG Door-in-Door",
		"Морозильник Bosch 200 литров",
		"Холодильник Samsung No Frost",
	},
	"Велосипеды": {
		"Велосипед горный Stels 26",
		"Велосипед шоссейный Cube",
		"Велосипед складной 20 дюймов",
		"Велосипед MTB Merida",
	},
	"Книги": {
		"Книга Гарри Поттер и философский камень",
		"Книга Война и мир Толстой",
		"Учебник Высшая математика",
		"Книга Атлант расправил плечи",
		"Сборник рассказов Чехова",
	},
	"Мебель": {
		"Диван угловой мягкий 3 местный",
		"Стол обеденный деревянный",
		"Кресло офисное кожаное",
		"Диван-кровать раскладной",
		"Стол компьютерный",
	},
	"Инструменты": {
		"Дрель Makita 18V",
		"Перфоратор Bosch SDS Plus",
		"Шуруповерт DeWalt 20V",
		"Дрель аккумуляторная Hammer",
	},
	"Садовая техника": {
		"Газонокосилка бензиновая Husqvarna",
		"Триммер электрический 1000W",
		"Культиватор мотоблок",
		"Газонокосилка электрическая",
	},
	"Одежда": {
		"Куртка зимняя мужская North Face",
		"Пальто шерстяное 46 размер",
		"Свитер кашемировый",
		"Куртка демисезонная Columbia",
		"Пуховик женский Adidas",
	},
}

// generateBatch produces a deterministic batch of users, items, and wishes.
// userID starts at userStart and increases by one per user; item and wish ids
// are assigned sequentially after itemStart and wishStart.
func generateBatch(
	rng *rand.Rand,
	categories []category.Category,
	passwordHash string,
	opts SeedOptions,
	userStart int64,
	itemStart int64,
	wishStart int64,
	count int,
) ([]user.User, []item.Item, []wish.WishedItem) {
	users := make([]user.User, 0, count)
	items := make([]item.Item, 0, count*opts.MaxGives)
	wishes := make([]wish.WishedItem, 0, count*opts.MaxWishes)

	now := time.Now().UTC()
	nextItemID := itemStart + 1
	nextWishID := wishStart + 1

	for i := 0; i < count; i++ {
		userID := userStart + int64(i)

		users = append(users, user.User{
			ID:           userID,
			Login:        fmt.Sprintf("user_%08d", userID),
			PasswordHash: passwordHash,
			Name:         fmt.Sprintf("Пользователь %d", userID),
			PublicRating: rng.Float64() * 5,
			ShadowRating: rng.IntN(10),
			CreatedAt:    now,
			UpdatedAt:    now,
		})

		for g := 0; g < rng.IntN(opts.MaxGives-opts.MinGives+1)+opts.MinGives; g++ {
			itemCategory := categories[rng.IntN(len(categories))]
			wantedCategory := categories[rng.IntN(len(categories))]
			priceMin := int64(rng.IntN(100_000)) + 1

			items = append(items, item.Item{
				ID:               nextItemID,
				UserID:           userID,
				Title:            randomItemTitle(rng, itemCategory.Name),
				Description:      "",
				Condition:        itemConditions[rng.IntN(len(itemConditions))],
				CategoryID:       itemCategory.ID,
				PriceMin:         priceMin,
				PriceMax:         priceMin,
				WantedCategoryID: wantedCategory.ID,
				Status:           item.StatusSearching,
				CreatedAt:        now,
				UpdatedAt:        now,
			})
			nextItemID++
		}

		for w := 0; w < rng.IntN(opts.MaxWishes-opts.MinWishes+1)+opts.MinWishes; w++ {
			wishCategory := categories[rng.IntN(len(categories))]
			priceMin := int64(rng.IntN(50_000)) + 1

			wishes = append(wishes, wish.WishedItem{
				ID:            nextWishID,
				UserID:        userID,
				CategoryID:    wishCategory.ID,
				PriceMin:      priceMin,
				PriceMax:      priceMin + int64(rng.IntN(500_000)),
				TitleKeywords: randomItemTitle(rng, wishCategory.Name),
				CreatedAt:     now,
				UpdatedAt:     now,
			})
			nextWishID++
		}
	}

	return users, items, wishes
}

// randomItemTitle returns a title from the pool of the category. Unknown
// categories fall back to a generic deterministic title.
func randomItemTitle(rng *rand.Rand, categoryName string) string {
	pool, ok := itemNamePools[categoryName]
	if !ok || len(pool) == 0 {
		return fmt.Sprintf("Предмет категории %s", categoryName)
	}
	return pool[rng.IntN(len(pool))]
}
