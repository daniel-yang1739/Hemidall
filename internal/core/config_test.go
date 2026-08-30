package core

import (
	"reflect"
	"testing"
)

func TestUnmarshalHostConfig(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *HostConfig
		wantErr bool
	}{
		{
			name: "Valid JSON overriding defaults",
			data: []byte(`{
				"global_context_window": 150000,
				"daily_quota_rpd": 10000,
				"exchange_rate": 30.5,
				"compression_threshold": 0.8,
				"model_overrides": {
					"gemini-1.5-pro": {
						"context_window": 2000000,
						"cache_discount_rate": 0.5
					}
				}
			}`),
			want: &HostConfig{
				GlobalContextWindow:  150000,
				DailyQuotaRPD:        10000,
				ExchangeRate:         30.5,
				CompressionThreshold: 0.8,
				ModelOverrides: map[string]ModelOverrideConfig{
					"gemini-1.5-pro": {
						ContextWindow:     2000000,
						CacheDiscountRate: 0.5,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Empty JSON keeps defaults",
			data: []byte(`{}`),
			want: &HostConfig{
				GlobalContextWindow:  DefaultFallbackAgentWindow,
				DailyQuotaRPD:        DefaultDailyQuotaRPDPro,
				ExchangeRate:         DefaultUSDtoTWDExchangeRate,
				CompressionThreshold: 0.5,
				ModelOverrides:       map[string]ModelOverrideConfig{},
			},
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			data:    []byte(`{ invalid json`),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalHostConfig(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalHostConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalHostConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
