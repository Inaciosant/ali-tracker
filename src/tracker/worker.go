package tracker

import (
	"context"
	"fmt"
	"sort"
	"strings"

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
	finder      ProductFinder
	bot         Messenger
	chatID      string
	topProducts int
	region      string
	locale      string
	currency    string
}

func NewWorker(finder ProductFinder, bot Messenger, chatID string, topProducts int, region, locale, currency string) *Worker {
	if topProducts <= 0 {
		topProducts = defaultTopProducts
	}

	return &Worker{
		finder:      finder,
		bot:         bot,
		chatID:      strings.TrimSpace(chatID),
		topProducts: topProducts,
		region:      defaultIfEmpty(region, "BR"),
		locale:      defaultIfEmpty(locale, "pt_BR"),
		currency:    defaultIfEmpty(currency, "BRL"),
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

		sort.Slice(products, func(i, j int) bool {
			return products[i].Sales > products[j].Sales
		})

		limit := w.topProducts
		if len(products) < limit {
			limit = len(products)
		}

		sections = append(sections, buildSummary(keyword, products[:limit], w.currency))
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
		builder.WriteString(fmt.Sprintf("   Preco: %s | Nota: %s | Vendas: %d\n", formatPrice(currency, product.PromotionPrice), rating, product.Sales))
		if product.ItemURL != "" {
			builder.WriteString(fmt.Sprintf("   Link: %s\n", product.ItemURL))
		}
	}

	return strings.TrimSpace(builder.String())
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
	if max <= 0 || len(value) <= max {
		return value
	}
	return strings.TrimSpace(value[:max-3]) + "..."
}

func defaultIfEmpty(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
