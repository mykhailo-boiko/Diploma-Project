package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/haradrim/chainorchestra/services/order-service/internal/order"
)

type HTTPProductValidator struct {
	baseURL string
	client  *http.Client
	log     *zap.Logger
}

func NewHTTPProductValidator(baseURL string, log *zap.Logger) *HTTPProductValidator {
	return &HTTPProductValidator{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
		log:     log,
	}
}

func (v *HTTPProductValidator) ValidateProduct(ctx context.Context, productID string) error {
	url := fmt.Sprintf("%s/api/v1/products/%s", v.baseURL, productID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to build product validate request: %w", err)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call inventory: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return order.ErrProductNotFound
	default:
		return errors.New("product validation upstream error")
	}
}
