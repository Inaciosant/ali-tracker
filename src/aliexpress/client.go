package aliexpress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ali-tracker/src/domain"
)

const defaultHost = "aliexpress-datahub.p.rapidapi.com"

type Client struct {
	apiKey     string
	apiHost    string
	httpClient *http.Client
}

type requestVariant struct {
	region   string
	locale   string
	currency string
}

var errNoResults = errors.New("no results")

func NewClient(apiKey, apiHost string) *Client {
	host := strings.TrimSpace(apiHost)
	if host == "" {
		host = defaultHost
	}

	return &Client{
		apiKey:  strings.TrimSpace(apiKey),
		apiHost: host,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *Client) SearchByKeyword(ctx context.Context, keyword, region, locale, currency string) ([]domain.Product, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, fmt.Errorf("rapidapi key is empty")
	}
	if strings.TrimSpace(keyword) == "" {
		return nil, fmt.Errorf("keyword is empty")
	}

	endpoints := []string{"item_search", "item_search_2"}
	variants := buildVariants(region, locale, currency)

	var errMessages []string
	hadNoResults := false
	for _, endpoint := range endpoints {
		for _, variant := range variants {
			products, err := c.fetchProducts(ctx, endpoint, keyword, variant)
			if err == nil {
				return products, nil
			}
			if errors.Is(err, errNoResults) {
				hadNoResults = true
				continue
			}
			errMessages = append(errMessages, err.Error())
		}
	}

	if hadNoResults {
		return []domain.Product{}, nil
	}

	if len(errMessages) == 0 {
		return nil, fmt.Errorf("failed to search by keyword")
	}

	return nil, fmt.Errorf("failed to search by keyword: %s", errMessages[len(errMessages)-1])
}

func buildVariants(region, locale, currency string) []requestVariant {
	variants := []requestVariant{
		{defaultIfEmpty(region, "BR"), defaultIfEmpty(locale, "pt_BR"), defaultIfEmpty(currency, "BRL")},
		{"BR", "en_US", "USD"},
		{"US", "en_US", "USD"},
	}

	out := make([]requestVariant, 0, len(variants))
	seen := make(map[string]struct{}, len(variants))
	for _, v := range variants {
		key := strings.ToUpper(strings.TrimSpace(v.region)) + "|" + strings.TrimSpace(v.locale) + "|" + strings.ToUpper(strings.TrimSpace(v.currency))
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	return out
}

func (c *Client) fetchProducts(ctx context.Context, endpoint, keyword string, variant requestVariant) ([]domain.Product, error) {
	url := fmt.Sprintf("https://%s/%s", c.apiHost, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	query := req.URL.Query()
	query.Set("q", keyword)
	query.Set("keyword", keyword)
	query.Set("keywords", keyword)
	query.Set("region", variant.region)
	query.Set("locale", variant.locale)
	query.Set("currency", variant.currency)
	query.Set("sort", "salesDesc")
	req.URL.RawQuery = query.Encode()

	req.Header.Set("X-RapidAPI-Key", c.apiKey)
	req.Header.Set("X-RapidAPI-Host", c.apiHost)
	req.Header.Set("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("%s (%s/%s/%s) returned %d: %s", endpoint, variant.region, variant.locale, variant.currency, res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload searchResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode %s json: %w", endpoint, err)
	}

	if payload.Result.Status.Code != 0 && payload.Result.Status.Code != 200 {
		if payload.Result.Status.Code == 5008 {
			return nil, errNoResults
		}
		return nil, fmt.Errorf("%s (%s/%s/%s) payload status %d (%s)", endpoint, variant.region, variant.locale, variant.currency, payload.Result.Status.Code, payload.Result.Status.Data)
	}

	products := make([]domain.Product, 0, len(payload.Result.ResultList))
	for _, entry := range payload.Result.ResultList {
		item := entry.Item
		originalPrice := 0.0
		promotionPrice := 0.0
		hasPromo := false

		if item.SKU.Def.Price.Valid {
			originalPrice = item.SKU.Def.Price.Value
			promotionPrice = originalPrice
		}
		if item.SKU.Def.PromotionPrice.Valid {
			promotionPrice = item.SKU.Def.PromotionPrice.Value
			hasPromo = true
		}

		if !hasPromo && !item.SKU.Def.Price.Valid {
			promotionPrice = 0
		}

		var averageStarRate *float64
		if item.AverageStarRate.Valid {
			value := item.AverageStarRate.Value
			averageStarRate = &value
		}

		products = append(products, domain.Product{
			ItemID:          item.ItemID,
			Title:           item.Title,
			Sales:           item.Sales.Value,
			ItemURL:         normalizeURL(item.ItemURL),
			ImageURL:        normalizeURL(item.Image),
			OriginalPrice:   originalPrice,
			PromotionPrice:  promotionPrice,
			AverageStarRate: averageStarRate,
		})
	}

	return products, nil
}

func normalizeURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "//") {
		return "https:" + trimmed
	}
	return trimmed
}

func defaultIfEmpty(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

type searchResponse struct {
	Result struct {
		Status struct {
			Code int    `json:"code"`
			Data string `json:"data"`
		} `json:"status"`
		ResultList []struct {
			Item struct {
				ItemID  string  `json:"itemId"`
				Title   string  `json:"title"`
				Sales   flexInt `json:"sales"`
				ItemURL string  `json:"itemUrl"`
				Image   string  `json:"image"`
				SKU     struct {
					Def struct {
						Price          flexFloat `json:"price"`
						PromotionPrice flexFloat `json:"promotionPrice"`
					} `json:"def"`
				} `json:"sku"`
				AverageStarRate flexFloat `json:"averageStarRate"`
			} `json:"item"`
		} `json:"resultList"`
	} `json:"result"`
}

type flexInt struct {
	Value int
}

func (v *flexInt) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		v.Value = 0
		return nil
	}

	if strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"") {
		raw = strings.Trim(raw, "\"")
	}

	if raw == "" {
		v.Value = 0
		return nil
	}

	if parsed, err := strconv.Atoi(raw); err == nil {
		v.Value = parsed
		return nil
	}

	if parsedFloat, err := strconv.ParseFloat(raw, 64); err == nil {
		v.Value = int(parsedFloat)
		return nil
	}

	return fmt.Errorf("invalid int value: %s", raw)
}

type flexFloat struct {
	Value float64
	Valid bool
}

func (v *flexFloat) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		v.Value = 0
		v.Valid = false
		return nil
	}

	if strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"") {
		raw = strings.Trim(raw, "\"")
	}

	if raw == "" {
		v.Value = 0
		v.Valid = false
		return nil
	}

	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fmt.Errorf("invalid float value: %s", raw)
	}

	v.Value = parsed
	v.Valid = true
	return nil
}
