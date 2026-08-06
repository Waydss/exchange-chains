package entities

// FIXME Feel free to move and change these structs whatever you like

import (
	"sort"
	"time"
)

type Chain struct {
	Id        uint       `json:"id" gorm:"primaryKey"`
	User      Account    `gorm:"foreignKey:UserId;references:Id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	UserId    uint       `json:"user_id" gorm:"index"`
	RootId    uint       `json:"root"`
	RootWish  WishedItem `json:"root_wish" gorm:"serializer:json"`
	Barters   []Barter   `json:"barters" gorm:"serializer:json"`
	Signature string     `json:"signature" gorm:"uniqueIndex"`
	Complete  bool       `json:"complete"`
	Score     float64    `json:"score"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Barter struct {
	Id         uint       `json:"id"`
	From       uint       `json:"from"`
	To         uint       `json:"to"`
	Give       GivedItem  `json:"give"`
	Wish       WishedItem `json:"wish"`
	FuzzyScore float64    `json:"fuzzy_score"`
	MatchScore float64    `json:"match_score"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Account struct {
	Id           uint
	Name         string
	PublicRating float64
	ShadowRating int
	Wishes       []WishedItem `gorm:"foreignKey:UserID;references:ID"`
	Gives        []GivedItem  `gorm:"foreignKey:UserID;references:ID"`
}

func (a Account) CurrentGive() (GivedItem, bool) {
	if len(a.Gives) == 0 {
		return GivedItem{}, false
	}
	best := a.Gives[0]
	for _, give := range a.Gives[1:] {
		if give.UpdatedAt.After(best.UpdatedAt) {
			best = give
		}
	}
	return best, true
}

func (a Account) CurrentWish() (WishedItem, bool) {
	if len(a.Wishes) == 0 {
		return WishedItem{}, false
	}
	best := a.Wishes[0]
	for _, wish := range a.Wishes[1:] {
		if wish.UpdatedAt.After(best.UpdatedAt) {
			best = wish
		}
	}
	return best, true
}

type GivedItem struct {
	ID        uint
	UserId    uint
	Name      string
	Category  Category
	Price     uint
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WishedItem struct {
	ID        uint
	UserId    uint
	Name      string
	Category  Category
	MinPrice  uint
	MaxPrice  uint
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Previously used for data seeding the database.
// You could join this with other Category struct, or remove it completely
type Category string

const (
	ELECTRONICS                    Category = "electronics"
	SMARTPHONES_ACCESSORIES        Category = "smartphones_accessories"
	COMPUTERS_COMPONENTS           Category = "computers_components"
	SMALL_HOME_APPLIANCES          Category = "small_home_appliances"
	LARGE_HOME_APPLIANCES          Category = "large_home_appliances"
	TVS_AUDIO_VIDEO_EQUIPMENT      Category = "tvs_audio_video_equipment"
	GAMES_GAMING_CONSOLES          Category = "games_gaming_consoles"
	SMART_HOME                     Category = "smart_home"
	MENS_CLOTHING                  Category = "mens_clothing"
	WOMENS_CLOTHING                Category = "womens_clothing"
	KIDS_CLOTHING                  Category = "kids_clothing"
	SHOES                          Category = "shoes"
	BAGS_BACKPACKS                 Category = "bags_backpacks"
	BELTS_GLOVES_HATS              Category = "belts_gloves_hats"
	UNDERWEAR_INTIMATES            Category = "underwear_intimates"
	SPORTSWEAR                     Category = "sportswear"
	COSMETICS_SKINCARE             Category = "cosmetics_skincare"
	PERFUMES_FRAGRANCES            Category = "perfumes_fragrances"
	HAIR_CARE                      Category = "hair_care"
	INTERIOR_DECOR                 Category = "interior_decor"
	BEDDING_HOME_TEXTILES          Category = "bedding_home_textiles"
	KITCHEN_APPLIANCES             Category = "kitchen_appliances"
	LIGHT_FIXTURES                 Category = "light_fixtures"
	PLUMBING_FIXTURES_PARTS        Category = "plumbing_fixtures_parts"
	TOOLS_FASTENERS                Category = "tools_fasteners"
	BUILDING_MATERIALS_IF_ALLOWED  Category = "building_materials_if_allowed"
	AUTOMOTIVE_ACCESSORIES_PARTS   Category = "automotive_accessories_parts"
	TIRES_WHEELS_RIMS              Category = "tires_wheels_rims"
	MOTORCYCLE_BICYCLE_ACCESSORIES Category = "motorcycle_bicycle_accessories"
	GARDEN_SUPPLIES                Category = "garden_supplies"
	GARDEN_TOOLS                   Category = "garden_tools"
	SPORTS_OUTDOOR                 Category = "sports_outdoor"
	FISHING_SUPPLIES               Category = "fishing_supplies"
)

// Previously used for data seeding the database. You can just remove this
var AllCategories []Category = []Category{
	ELECTRONICS,
	SMARTPHONES_ACCESSORIES,
	COMPUTERS_COMPONENTS,
	SMALL_HOME_APPLIANCES,
	LARGE_HOME_APPLIANCES,
	TVS_AUDIO_VIDEO_EQUIPMENT,
	GAMES_GAMING_CONSOLES,
	SMART_HOME,
	MENS_CLOTHING,
	WOMENS_CLOTHING,
	KIDS_CLOTHING,
	SHOES,
	BAGS_BACKPACKS,
	BELTS_GLOVES_HATS,
	UNDERWEAR_INTIMATES,
	SPORTSWEAR,
	COSMETICS_SKINCARE,
	PERFUMES_FRAGRANCES,
	HAIR_CARE,
	INTERIOR_DECOR,
	BEDDING_HOME_TEXTILES,
	KITCHEN_APPLIANCES,
	LIGHT_FIXTURES,
	PLUMBING_FIXTURES_PARTS,
	TOOLS_FASTENERS,
	BUILDING_MATERIALS_IF_ALLOWED,
	AUTOMOTIVE_ACCESSORIES_PARTS,
	TIRES_WHEELS_RIMS,
	MOTORCYCLE_BICYCLE_ACCESSORIES,
	GARDEN_SUPPLIES,
	GARDEN_TOOLS,
	SPORTS_OUTDOOR,
	FISHING_SUPPLIES,
}

// Previously used for data seeding the database. You can just remove this
var AllItemNames = map[string]Category{
	"Wireless Headphones": ELECTRONICS,
	"Bluetooth Speaker":   ELECTRONICS,
	"Smartphone Case":     SMARTPHONES_ACCESSORIES,
	"USB-C Charger":       SMARTPHONES_ACCESSORIES,
	"Laptop Stand":        COMPUTERS_COMPONENTS,
	"Mechanical Keyboard": COMPUTERS_COMPONENTS,
	"Gaming Mouse":        COMPUTERS_COMPONENTS,
	"Monitor Arm":         COMPUTERS_COMPONENTS,
	"Smart Plug":          SMART_HOME,
	"Robot Vacuum":        SMART_HOME,
	"Air Fryer":           KITCHEN_APPLIANCES,
	"Blender":             KITCHEN_APPLIANCES,
	"Coffee Grinder":      KITCHEN_APPLIANCES,
	"Electric Kettle":     KITCHEN_APPLIANCES,
	"Microwave Oven":      KITCHEN_APPLIANCES,
	"Vacuum Cleaner":      SMALL_HOME_APPLIANCES,
	"Steam Iron":          SMALL_HOME_APPLIANCES,
	"Hair Dryer":          HAIR_CARE,
	"Desk Lamp":           LIGHT_FIXTURES,
	"Ceiling Light":       LIGHT_FIXTURES,
	"Faucet Cartridge":    PLUMBING_FIXTURES_PARTS,
	"Shower Head":         PLUMBING_FIXTURES_PARTS,
	"Pipe Wrench":         TOOLS_FASTENERS,
	"Drill Bit Set":       TOOLS_FASTENERS,
	"Hammer Drill":        TOOLS_FASTENERS,
	"Extension Cord":      ELECTRONICS,
	"Screwdriver Set":     TOOLS_FASTENERS,
	"Power Bank":          ELECTRONICS,
	"Car Mount":           AUTOMOTIVE_ACCESSORIES_PARTS,
	"Tire Inflator":       TIRES_WHEELS_RIMS,
	"Motor Oil":           AUTOMOTIVE_ACCESSORIES_PARTS,
	"Bike Helmet":         MOTORCYCLE_BICYCLE_ACCESSORIES,
	"Garden Hose":         GARDEN_SUPPLIES,
	"Pruning Shears":      GARDEN_TOOLS,
	"Shovel":              GARDEN_TOOLS,
	"Fishing Rod":         FISHING_SUPPLIES,
	"Tackle Box":          FISHING_SUPPLIES,
	"Camping Tent":        SPORTS_OUTDOOR,
	"Sleeping Bag":        SPORTS_OUTDOOR,
	"Running Shoes":       SHOES,
	"Sneakers":            SHOES,
	"Sandals":             SHOES,
	"Boots":               SHOES,
	"Backpack":            BAGS_BACKPACKS,
	"Laptop Bag":          BAGS_BACKPACKS,
	"Wallet":              BAGS_BACKPACKS,
	"Belt":                BELTS_GLOVES_HATS,
	"Hat":                 BELTS_GLOVES_HATS,
	"Gloves":              BELTS_GLOVES_HATS,
	"Sunglasses":          BELTS_GLOVES_HATS,
	"Perfume":             PERFUMES_FRAGRANCES,
	"Face Cream":          COSMETICS_SKINCARE,
	"Shampoo":             HAIR_CARE,
	"Body Lotion":         COSMETICS_SKINCARE,
	"T-shirt":             MENS_CLOTHING,
	"Jeans":               MENS_CLOTHING,
	"Hoodie":              MENS_CLOTHING,
	"Jacket":              MENS_CLOTHING,
	"Dress":               WOMENS_CLOTHING,
	"Skirt":               WOMENS_CLOTHING,
	"Socks":               UNDERWEAR_INTIMATES,
	"Underwear":           UNDERWEAR_INTIMATES,
	"Curtains":            BEDDING_HOME_TEXTILES,
	"Bedsheet":            BEDDING_HOME_TEXTILES,
	"Pillow":              BEDDING_HOME_TEXTILES,
	"Blanket":             BEDDING_HOME_TEXTILES,
	"Tablecloth":          BEDDING_HOME_TEXTILES,
	"Storage Box":         INTERIOR_DECOR,
	"Picture Frame":       INTERIOR_DECOR,
	"Wall Shelf":          INTERIOR_DECOR,
	"Bookcase":            INTERIOR_DECOR,
	"Coffee Table":        INTERIOR_DECOR,
	"Office Chair":        INTERIOR_DECOR,
	"Watch":               ELECTRONICS,
	"Smartwatch":          ELECTRONICS,
	"Tablet":              ELECTRONICS,
	"Game Console":        GAMES_GAMING_CONSOLES,
	"Controller":          GAMES_GAMING_CONSOLES,
	"Headphones Stand":    ELECTRONICS,
	"Web Camera":          COMPUTERS_COMPONENTS,
	"Microphone":          COMPUTERS_COMPONENTS,
	"Speaker Stand":       ELECTRONICS,
	"Router":              ELECTRONICS,
	"WiFi Repeater":       ELECTRONICS,
	"Smart Bulb":          SMART_HOME,
	"Thermostat":          SMART_HOME,
	"Kitchen Mixer":       KITCHEN_APPLIANCES,
	"Toaster":             KITCHEN_APPLIANCES,
	"Rice Cooker":         KITCHEN_APPLIANCES,
	"Pressure Cooker":     KITCHEN_APPLIANCES,
	"Frying Pan":          KITCHEN_APPLIANCES,
	"Saucepan":            KITCHEN_APPLIANCES,
	"Cutlery Set":         KITCHEN_APPLIANCES,
	"Plates Set":          KITCHEN_APPLIANCES,
	"Glassware Set":       KITCHEN_APPLIANCES,
	"Dish Rack":           KITCHEN_APPLIANCES,
	"Laundry Basket":      BEDDING_HOME_TEXTILES,
	"Ironing Board":       SMALL_HOME_APPLIANCES,
	"Bed Frame":           BEDDING_HOME_TEXTILES,
	"Mattress":            BEDDING_HOME_TEXTILES,
}

// Previously used for data seeding the database. You can just remove this
var AllItemNamesList = func() []string {
	result := make([]string, 0, len(AllItemNames))
	for name := range AllItemNames {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}()
