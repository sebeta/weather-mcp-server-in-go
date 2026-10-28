package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const WeatherAPIBase = "https://api.open-meteo.com/v1/forecast"

var descriptionPerWeatherCode = map[string]string{
	"0":  "Clear sky",
	"1":  "Mainly clear",
	"2":  "Partly cloudy",
	"3":  "Overcast",
	"45": "Fog",
	"48": "Depositing rime fog",
	"51": "Light drizzle",
	"53": "Moderate drizzle",
	"55": "Dense drizzle",
	"56": "Light freezing drizzle",
	"57": "Dense freezing drizzle",
	"61": "Slight rain",
	"63": "Moderate rain",
	"65": "Heavy rain",
	"66": "Light freezing rain",
	"67": "Heavy freezing rain",
	"71": "Slight snowfall",
	"73": "Moderate snowfall",
	"75": "Heavy snowfall",
	"77": "Snow grains",
	"80": "Slight rain showers",
	"81": "Moderate rain showers",
	"82": "Violent rain showers",
	"85": "Slight snow showers",
	"86": "Heavy snow showers",
	"95": "Thunderstorm",
	"96": "Thunderstorm with slight hail",
	"97": "Heavy thunderstorm",
	"99": "Thunderstorm with heavy hail",
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// buildWeatherAPIURL builds the Weather API URL with the parameters shared by all tools.
// Any entry in other is added to (or overrides) the query.
func buildWeatherAPIURL(latitude, longitude float64, other map[string]string) string {
	params := url.Values{}
	params.Set("latitude", strconv.FormatFloat(latitude, 'f', 4, 64))
	params.Set("longitude", strconv.FormatFloat(longitude, 'f', 4, 64))
	for k, v := range other {
		params.Set(k, v)
	}

	return WeatherAPIBase + "?" + params.Encode()
}

// fetchWeatherData requests the given Weather API URL and decodes the response.
// Non-200 statuses are returned as errors, using the API's reason when available.
func fetchWeatherData(ctx context.Context, endpoint string) (*WeatherData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating the request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var e APIErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&e); err == nil && e.Reason != "" {
			return nil, errors.New(e.Reason)
		}
		return nil, fmt.Errorf("weather API returned status %d", resp.StatusCode)
	}

	var data WeatherData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("error decoding the response: %w", err)
	}

	return &data, nil
}
