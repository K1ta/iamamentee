package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"products/internal/domain"
	"strconv"

	"github.com/elastic/go-elasticsearch/v9"
)

const productSearchIndex = "products"

type (
	elasticErrorResponse struct {
		Err struct {
			Type   string `json:"type"`
			Reason string `json:"reason"`
		} `json:"error"`
		Status int `json:"status"`
	}

	elasticSearchResponse struct {
		Hits struct {
			Hits []struct {
				ID    string  `json:"_id"`
				Score float64 `json:"_score"`
			} `json:"hits"`
		} `json:"hits"`
	}
)

func (e elasticErrorResponse) Error() string {
	return fmt.Sprintf("elasticsearch error %d: (%s) %s", e.Status, e.Err.Type, e.Err.Reason)
}

type SearchStore struct {
	client *elasticsearch.Client
}

func NewSearchStore(addr []string) (*SearchStore, error) {
	cfg := elasticsearch.Config{
		Addresses: addr,
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &SearchStore{client: client}, nil
}

func (s *SearchStore) Index(ctx context.Context, product *domain.Product) error {
	productDoc := map[string]any{
		"name":  product.Name,
		"price": product.Price,
	}
	body, err := json.Marshal(productDoc)
	if err != nil {
		return fmt.Errorf("marshal product doc: %w", err)
	}
	res, err := s.client.Index(
		productSearchIndex,
		bytes.NewReader(body),
		s.client.Index.WithDocumentID(strconv.FormatInt(product.ID, 10)),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("index doc: %w", err)
	}
	res.Body.Close()
	return nil
}

func (s *SearchStore) Search(ctx context.Context, query domain.SearchQuery) ([]int64, error) {
	var must []map[string]any
	if query.Name != "" {
		must = []map[string]any{{"match": map[string]any{"name": query.Name}}}
	}

	filters := make(map[string]any)
	if query.PriceFrom > 0 {
		filters["gte"] = query.PriceFrom
	}
	if query.PriceTo > 0 {
		filters["lte"] = query.PriceTo
	}
	var filter []map[string]any
	if len(filters) > 0 {
		filter = []map[string]any{{"range": map[string]any{"price": filters}}}
	}

	esQuery := map[string]any{
		"_source": false,
		"query": map[string]any{
			"bool": map[string]any{
				"must":   must,
				"filter": filter,
			},
		},
	}

	body, err := json.Marshal(esQuery)
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}
	log.Println("es query:", string(body))

	res, err := s.client.Search(
		s.client.Search.WithIndex(productSearchIndex),
		s.client.Search.WithBody(bytes.NewReader(body)),
		s.client.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var elasticErr elasticErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&elasticErr); err == nil {
			return nil, elasticErr
		}
		return nil, errors.New("unknown elasticsearch error")
	}

	var resp elasticSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	ids := make([]int64, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		id, err := strconv.ParseInt(hit.ID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse id %s: %w", hit.ID, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
