package domain

type Product struct {
	ItemID          string
	Title           string
	Sales           int
	ItemURL         string
	ImageURL        string
	OriginalPrice   float64
	PromotionPrice  float64
	AverageStarRate *float64
}
