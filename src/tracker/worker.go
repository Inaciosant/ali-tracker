package tracker

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"ali-tracker/src/domain"
)

const defaultTopProducts = 5

type ProductFinder interface {
	SearchByKeyword(ctx context.Context, keyword, region, locale, currency string) ([]domain.Product, error)
}

type Messenger interface {
	SendMessage(chatID, message string) error
}

type Worker struct {
	finder             ProductFinder
	bot                Messenger
	chatID             string
	topProducts        int
	minDiscountPercent float64
	region             string
	locale             string
	currency           string
}

func NewWorker(
	finder ProductFinder,
	bot Messenger,
	chatID string,
	topProducts int,
	minDiscountPercent float64,
	region, locale, currency string,
) *Worker {
	if topProducts <= 0 {
		topProducts = defaultTopProducts
	}
	if minDiscountPercent < 0 {
		minDiscountPercent = 0
	}

	return &Worker{
		finder:             finder,
		bot:                bot,
		chatID:             strings.TrimSpace(chatID),
		topProducts:        topProducts,
		minDiscountPercent: minDiscountPercent,
		region:             defaultIfEmpty(region, "BR"),
		locale:             defaultIfEmpty(locale, "pt_BR"),
		currency:           defaultIfEmpty(currency, "BRL"),
	}
}

func (w *Worker) Run(ctx context.Context, keywords []string) error {
	if w.finder == nil {
		return fmt.Errorf("finder is nil")
	}
	if w.bot == nil {
		return fmt.Errorf("bot is nil")
	}
	if w.chatID == "" {
		return fmt.Errorf("chat id is empty")
	}

	filteredKeywords := normalizeKeywords(keywords)
	if len(filteredKeywords) == 0 {
		return fmt.Errorf("no keywords provided")
	}

	var sections []string
	for _, keyword := range filteredKeywords {
		products, err := w.finder.SearchByKeyword(ctx, keyword, w.region, w.locale, w.currency)
		if err != nil {
			sections = append(sections, fmt.Sprintf("Busca: %s\nErro: %v", keyword, err))
			continue
		}

		if len(products) == 0 {
			sections = append(sections, fmt.Sprintf("Busca: %s\nNenhum produto encontrado.", keyword))
			continue
		}

		filteredProducts := filterByDiscount(products, w.minDiscountPercent)
		if len(filteredProducts) == 0 {
			sections = append(
				sections,
				fmt.Sprintf("Busca: %s\nNenhuma promocao com desconto minimo de %.1f%%.", keyword, w.minDiscountPercent),
			)
			continue
		}

		sort.Slice(filteredProducts, func(i, j int) bool {
			return filteredProducts[i].Sales > filteredProducts[j].Sales
		})

		limit := w.topProducts
		if len(filteredProducts) < limit {
			limit = len(filteredProducts)
		}

		sections = append(sections, buildSummary(keyword, filteredProducts[:limit], w.currency))
	}

	message := "Resumo de buscas AliExpress :\n\n" + strings.Join(sections, "\n\n")
	return w.bot.SendMessage(w.chatID, message)
}

func buildSummary(keyword string, products []domain.Product, currency string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Busca: %s\n", keyword))

	for i, product := range products {
		rating := "n/a"
		if product.AverageStarRate != nil {
			rating = fmt.Sprintf("%.1f", *product.AverageStarRate)
		}

		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, truncate(product.Title, 90)))
		discount := discountPercent(product)
		priceLabel := formatPrice(currency, product.PromotionPrice)
		if product.OriginalPrice > 0 && product.OriginalPrice > product.PromotionPrice {
			priceLabel = fmt.Sprintf("%s (antes %s)", formatPrice(currency, product.PromotionPrice), formatPrice(currency, product.OriginalPrice))
		}
		builder.WriteString(fmt.Sprintf("   Preco: %s | Desconto: %.1f%% | Nota: %s | Vendas: %d\n", priceLabel, discount, rating, product.Sales))
		if product.ItemURL != "" {
			builder.WriteString(fmt.Sprintf("   Link: %s\n", product.ItemURL))
		}
	}

	return strings.TrimSpace(builder.String())
}

func filterByDiscount(products []domain.Product, minDiscount float64) []domain.Product {
	if minDiscount <= 0 {
		return products
	}

	filtered := make([]domain.Product, 0, len(products))
	for _, product := range products {
		if discountPercent(product) >= minDiscount {
			filtered = append(filtered, product)
		}
	}
	return filtered
}

func discountPercent(product domain.Product) float64 {
	if product.OriginalPrice <= 0 || product.PromotionPrice <= 0 {
		return 0
	}
	if product.PromotionPrice >= product.OriginalPrice {
		return 0
	}
	return ((product.OriginalPrice - product.PromotionPrice) / product.OriginalPrice) * 100
}

func normalizeKeywords(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	output := make([]string, 0, len(values))

	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		output = append(output, trimmed)
	}

	return output
}

func formatPrice(currency string, value float64) string {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "USD":
		return fmt.Sprintf("$%.2f", value)
	case "BRL":
		return fmt.Sprintf("R$%.2f", value)
	case "EUR":
		return fmt.Sprintf("EUR %.2f", value)
	default:
		if strings.TrimSpace(currency) == "" {
			return fmt.Sprintf("%.2f", value)
		}
		return fmt.Sprintf("%.2f %s", value, strings.ToUpper(currency))
	}
}

func truncate(value string, max int) string {
	if max <= 0 || utf8.RuneCountInString(value) <= max {
		return value
	}
	if max <= 3 {
		return "..."
	}

	runes := []rune(value)
	return strings.TrimSpace(string(runes[:max-3])) + "..."
}

func defaultIfEmpty(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
