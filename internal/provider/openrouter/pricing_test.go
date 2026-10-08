package openrouter

import (
	"context"
	"net/http"
	"testing"

	"github.com/rewdy/genifer/internal/provider"
)

func TestClassifyPricing(t *testing.T) {
	perImage := endpointsResponse{Endpoints: []struct {
		Pricing []pricingLine `json:"pricing"`
	}{{Pricing: []pricingLine{
		{Billable: "output_image", Unit: "image", CostUSD: 0.018},
		{Billable: "input_image", Unit: "image", CostUSD: 0},
	}}}}
	if p := classifyPricing(perImage); p.Unit != provider.PricePerImage || p.USD != 0.018 {
		t.Errorf("per-image: got %+v", p)
	}

	token := endpointsResponse{Endpoints: []struct {
		Pricing []pricingLine `json:"pricing"`
	}{{Pricing: []pricingLine{
		{Billable: "output_image", Unit: "token", CostUSD: 0.00003},
		{Billable: "input_text", Unit: "token", CostUSD: 0.000005},
	}}}}
	if p := classifyPricing(token); p.Unit != provider.PricePerToken {
		t.Errorf("per-token: got %+v", p)
	}

	tiered := endpointsResponse{Endpoints: []struct {
		Pricing []pricingLine `json:"pricing"`
	}{{Pricing: []pricingLine{
		{Billable: "output_image", Unit: "image", CostUSD: 0.041, Variant: "768"},
		{Billable: "output_image", Unit: "image", CostUSD: 0.607, Variant: "4k"},
	}}}}
	if p := classifyPricing(tiered); p.Unit != provider.PricePerImageTiered || p.USD != 0.041 {
		t.Errorf("tiered (want lowest 0.041): got %+v", p)
	}

	free := endpointsResponse{Endpoints: []struct {
		Pricing []pricingLine `json:"pricing"`
	}{{Pricing: []pricingLine{
		{Billable: "output_image", Unit: "image", CostUSD: 0},
	}}}}
	if p := classifyPricing(free); p.Unit != provider.PriceFree {
		t.Errorf("free: got %+v", p)
	}

	empty := endpointsResponse{}
	if p := classifyPricing(empty); p.Unit != provider.PriceUnknown {
		t.Errorf("empty: got %+v", p)
	}
}

func TestPricingEndpointCall(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"a/b","endpoints":[{"pricing":[{"billable":"output_image","unit":"image","cost_usd":0.02}]}]}`))
	}))
	defer srv.Close()
	p, err := c.Pricing(context.Background(), "a/b")
	if err != nil {
		t.Fatalf("Pricing: %v", err)
	}
	if p.Unit != provider.PricePerImage || p.USD != 0.02 {
		t.Errorf("got %+v", p)
	}
}

func TestGenerateCapturesCost(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// b64 for "img" is "aW1n"
		w.Write([]byte(`{"data":[{"b64_json":"aW1n","media_type":"image/png"}],"usage":{"cost":0.0123}}`))
	}))
	defer srv.Close()
	res, err := c.Generate(context.Background(), provider.GenerateRequest{Model: "m", Prompt: "p"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if res.CostUSD != 0.0123 {
		t.Errorf("CostUSD = %v, want 0.0123", res.CostUSD)
	}
}
