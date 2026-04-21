package domain

type Product struct {
	ItemID          string
	Title           string
	Sales           int
	ItemURL         string
	ImageURL        string
	PromotionPrice  float64
	AverageStarRate *float64
}
