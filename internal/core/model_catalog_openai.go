package core

// openAIModelCatalog contains official models served via the OpenAI API platform.
var openAIModelCatalog = map[ModelID]ModelInfo{
	ModelGPT5: {
		Provider:            ProviderOpenAI,
		Vendor:              "OpenAI",
		Name:                "GPT-5",
		ID:                  ModelGPT5,
		InputUSDPerMillion:  1.25,
		OutputUSDPerMillion: 10.00,
		CacheDiscountRate:   0.90, // 90% OFF ($0.125 / MTok)
		SourceLabel:         "OpenAI API pricing",
		SourceURL:           "https://developers.openai.com/api/docs/pricing",
	},
	ModelGPT5Mini: {
		Provider:            ProviderOpenAI,
		Vendor:              "OpenAI",
		Name:                "GPT-5 mini",
		ID:                  ModelGPT5Mini,
		InputUSDPerMillion:  0.25,
		OutputUSDPerMillion: 2.00,
		CacheDiscountRate:   0.90, // 90% OFF ($0.025 / MTok)
		SourceLabel:         "OpenAI API pricing",
		SourceURL:           "https://developers.openai.com/api/docs/pricing",
	},
	ModelGPT4o: {
		Provider:            ProviderOpenAI,
		Vendor:              "OpenAI",
		Name:                "GPT-4o",
		ID:                  ModelGPT4o,
		InputUSDPerMillion:  2.50,
		OutputUSDPerMillion: 10.00,
		CacheDiscountRate:   0.50, // 50% OFF ($1.25 / MTok)
		SourceLabel:         "OpenAI API pricing",
		SourceURL:           "https://developers.openai.com/api/docs/pricing",
	},
	ModelGPT4oMini: {
		Provider:            ProviderOpenAI,
		Vendor:              "OpenAI",
		Name:                "GPT-4o mini",
		ID:                  ModelGPT4oMini,
		InputUSDPerMillion:  0.15,
		OutputUSDPerMillion: 0.60,
		CacheDiscountRate:   0.50, // 50% OFF ($0.075 / MTok)
		SourceLabel:         "OpenAI API pricing",
		SourceURL:           "https://developers.openai.com/api/docs/pricing",
	},
	ModelGPT41: {
		Provider:            ProviderOpenAI,
		Vendor:              "OpenAI",
		Name:                "GPT-4.1",
		ID:                  ModelGPT41,
		InputUSDPerMillion:  2.00,
		OutputUSDPerMillion: 8.00,
		CacheDiscountRate:   0.75, // 75% OFF ($0.50 / MTok)
		SourceLabel:         "OpenAI API pricing",
		SourceURL:           "https://developers.openai.com/api/docs/pricing",
	},
	ModelO1: {
		Provider:            ProviderOpenAI,
		Vendor:              "OpenAI",
		Name:                "o1",
		ID:                  ModelO1,
		InputUSDPerMillion:  15.00,
		OutputUSDPerMillion: 60.00,
		CacheDiscountRate:   0.50, // 50% OFF ($7.50 / MTok)
		SourceLabel:         "OpenAI API pricing",
		SourceURL:           "https://developers.openai.com/api/docs/pricing",
	},
	ModelO3: {
		Provider:            ProviderOpenAI,
		Vendor:              "OpenAI",
		Name:                "o3",
		ID:                  ModelO3,
		InputUSDPerMillion:  2.00,
		OutputUSDPerMillion: 8.00,
		CacheDiscountRate:   0.75, // 75% OFF ($0.50 / MTok)
		SourceLabel:         "OpenAI API pricing",
		SourceURL:           "https://developers.openai.com/api/docs/pricing",
	},
	ModelO3Mini: {
		Provider:            ProviderOpenAI,
		Vendor:              "OpenAI",
		Name:                "o3-mini",
		ID:                  ModelO3Mini,
		InputUSDPerMillion:  1.10,
		OutputUSDPerMillion: 4.40,
		CacheDiscountRate:   0.50, // 50% OFF ($0.55 / MTok)
		SourceLabel:         "OpenAI API pricing",
		SourceURL:           "https://developers.openai.com/api/docs/pricing",
	},
	ModelO4Mini: {
		Provider:            ProviderOpenAI,
		Vendor:              "OpenAI",
		Name:                "o4-mini",
		ID:                  ModelO4Mini,
		InputUSDPerMillion:  1.10,
		OutputUSDPerMillion: 4.40,
		CacheDiscountRate:   0.75, // 75% OFF ($0.275 / MTok)
		SourceLabel:         "OpenAI API pricing",
		SourceURL:           "https://developers.openai.com/api/docs/pricing",
	},
	ModelGPT6Astra: {
		Provider:            ProviderOpenAI,
		Vendor:              "OpenAI",
		Name:                "GPT-6 Astra",
		ID:                  ModelGPT6Astra,
		InputUSDPerMillion:  10.00,
		OutputUSDPerMillion: 50.00,
		CacheDiscountRate:   0.90, // 90% OFF ($1.00 / MTok)
		SourceLabel:         "OpenAI API pricing",
		SourceURL:           "https://developers.openai.com/api/docs/pricing",
	},
	ModelGPT56Sol: {
		Provider:            ProviderOpenAI,
		Vendor:              "OpenAI",
		Name:                "GPT-5.6 Sol",
		ID:                  ModelGPT56Sol,
		InputUSDPerMillion:  4.00,
		OutputUSDPerMillion: 20.00,
		CacheDiscountRate:   0.90, // 90% OFF ($0.40 / MTok)
		SourceLabel:         "OpenAI API pricing",
		SourceURL:           "https://developers.openai.com/api/docs/pricing",
	},
	ModelGPT56Terra: {
		Provider:            ProviderOpenAI,
		Vendor:              "OpenAI",
		Name:                "GPT-5.6 Terra",
		ID:                  ModelGPT56Terra,
		InputUSDPerMillion:  2.00,
		OutputUSDPerMillion: 12.00,
		CacheDiscountRate:   0.90, // 90% OFF ($0.20 / MTok)
		SourceLabel:         "OpenAI API pricing",
		SourceURL:           "https://developers.openai.com/api/docs/pricing",
	},
	ModelGPT56Luna: {
		Provider:            ProviderOpenAI,
		Vendor:              "OpenAI",
		Name:                "GPT-5.6 Luna",
		ID:                  ModelGPT56Luna,
		InputUSDPerMillion:  0.20,
		OutputUSDPerMillion: 1.20,
		CacheDiscountRate:   0.90, // 90% OFF ($0.02 / MTok)
		SourceLabel:         "OpenAI API pricing",
		SourceURL:           "https://developers.openai.com/api/docs/pricing",
	},
}
