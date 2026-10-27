package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// HTTP response from the Weather API
type WeatherData struct {
	Latitude             float64       `json:"latitude"`
	Longitude            float64       `json:"longitude"`
	Elevation            float64       `json:"elevation"`
	GenerationTimeMs     float64       `json:"generationtime_ms"`
	UTCOffsetSeconds     int           `json:"utc_offset_seconds"`
	Timezone             string        `json:"timezone"`
	TimezoneAbbreviation string        `json:"timezone_abbreviation"`
	Current              *Current      `json:"current,omitempty"`
	CurrentUnits         *CurrentUnits `json:"current_units,omitempty"`
	Hourly               *Hourly       `json:"hourly,omitempty"`
	HourlyUnits          *HourlyUnits  `json:"hourly_units,omitempty"`
}

type Current struct {
	Time             string  `json:"time"`
	Interval         int     `json:"interval"`
	Temperature2m    float64 `json:"temperature_2m"`
	WindSpeed10m     float64 `json:"wind_speed_10m"`
	RelativeHumidity int     `json:"relative_humidity_2m"`
	WeatherCode      int     `json:"weather_code"`
}

type CurrentUnits struct {
	Interval         string `json:"interval"`
	Temperature2m    string `json:"temperature_2m"`
	WindSpeed10m     string `json:"wind_speed_10m"`
	RelativeHumidity string `json:"relative_humidity_2m"`
	WeatherCode      string `json:"weather_code"`
}

type Hourly struct {
	Time             []string  `json:"time"`
	Temperature2m    []float64 `json:"temperature_2m"`
	WindSpeed10m     []float64 `json:"wind_speed_10m"`
	RelativeHumidity []int     `json:"relative_humidity_2m"`
	WeatherCode      []int     `json:"weather_code"`
}

type HourlyUnits struct {
	Time             string `json:"time"`
	Temperature2m    string `json:"temperature_2m"`
	WindSpeed10m     string `json:"wind_speed_10m"`
	RelativeHumidity string `json:"relative_humidity_2m"`
	WeatherCode      string `json:"weather_code"`
}

type APIErrorResponse struct {
	Error  bool   `json:"error"`
	Reason string `json:"reason"`
}

// MCP inputs and outputs for tools
type GetCurrentWeatherInput struct {
	Latitude  float64 `json:"latitude" jsonschema:"The latitude of the location"`
	Longitude float64 `json:"longitude" jsonschema:"The longitude of the location"`
}

type GetCurrentWeatherOutput struct {
	Temperature        float64 `json:"temperature" jsonschema:"Temperature in °C"`
	WindSpeed          float64 `json:"wind_speed" jsonschema:"Wind speed in k/h"`
	Humidity           int     `json:"humidity" jsonschema:"Humidity percentage (%)"`
	WeatherDescription string  `json:"weather_description" jsonschema:"A briefly description of the weather"`
}

type GetWeatherForecastInput struct {
	Latitude  float64 `json:"latitude" jsonschema:"The latitude of the location"`
	Longitude float64 `json:"longitude" jsonschema:"The longitude of the location"`
	Days      int     `json:"days,omitempty" jsonschema:"The number of days to forecast (default: 7)"`
}

type GetWeatherForecastOutput struct {
	Time               []string  `json:"time" jsonschema:"The timestamps corresponding to the weather measurements (ISO8601)"`
	Temperature        []float64 `json:"temperature" jsonschema:"Temperature in °C"`
	WindSpeed          []float64 `json:"wind_speed" jsonschema:"Wind speed in k/h"`
	Humidity           []int     `json:"humidity" jsonschema:"Humidity percentage (%)"`
	WeatherDescription []string  `json:"weather_description" jsonschema:"A briefly description of the weather"`
}

type GetWeatherSummaryInput struct {
	Latitude  float64 `json:"latitude" jsonschema:"The latitude of the location"`
	Longitude float64 `json:"longitude" jsonschema:"The longitude of the location"`
}

type GetWeatherSummaryOutput struct {
	Temperature        float64 `json:"temperature" jsonschema:"Temperature in °C"`
	WindSpeed          float64 `json:"wind_speed" jsonschema:"Wind speed in k/h"`
	Humidity           int     `json:"humidity" jsonschema:"Humidity percentage (%)"`
	WeatherDescription string  `json:"weather_description" jsonschema:"A briefly description of the weather"`
	Forecast           struct {
		Time        []string  `json:"time" jsonschema:"The timestamps corresponding to the weather measurements (ISO8601)"`
		Temperature []float64 `json:"temperature" jsonschema:"Temperature in °C"`
	} `json:"forecast" jsonschema:"The hourly forecast for the current day, including timestamps (ISO8601) and temperarures in °C"`
}

// GetCurrentWeather is the handler of the get_current_weather tool. It returns the
// current temperature, wind speed, humidity and a description of the weather for
// the given coordinates, both as structured output and as a human-readable text.
func GetCurrentWeather(
	ctx context.Context,
	mcpReq *mcp.CallToolRequest,
	input GetCurrentWeatherInput,
) (*mcp.CallToolResult, *GetCurrentWeatherOutput, error) {
	endpoint := buildWeatherAPIURL(
		input.Latitude,
		input.Longitude,
		map[string]string{
			"current": "temperature_2m,wind_speed_10m,relative_humidity_2m,weather_code",
		},
	)
	data, err := fetchWeatherData(ctx, endpoint)
	if err != nil {
		return nil, nil, err
	}
	if data.Current == nil {
		return nil, nil, errors.New("the API response has no current weather data")
	}

	output := GetCurrentWeatherOutput{
		Temperature:        data.Current.Temperature2m,
		WindSpeed:          data.Current.WindSpeed10m,
		Humidity:           data.Current.RelativeHumidity,
		WeatherDescription: descriptionPerWeatherCode[strconv.Itoa(data.Current.WeatherCode)],
	}

	text := fmt.Sprintf(
		"Current weather: %s. Temperature %.1f °C, wind %.1f km/h, humidity %d%%.",
		output.WeatherDescription,
		output.Temperature,
		output.WindSpeed,
		output.Humidity,
	)

	result := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}

	return result, &output, nil
}

// GetWeatherForecast is the handler of the get_weather_forecast tool. It returns the
// hourly temperature, wind speed, humidity and weather description for the given
// coordinates over the requested number of days (7 when not specified).
func GetWeatherForecast(
	ctx context.Context,
	mcpReq *mcp.CallToolRequest,
	input GetWeatherForecastInput,
) (*mcp.CallToolResult, *GetWeatherForecastOutput, error) {
	urlParams := map[string]string{
		"hourly": "temperature_2m,wind_speed_10m,relative_humidity_2m,weather_code",
	}
	// Days is optional: only send "forecast_days" when the caller sent a positive value.
	// Otherwise Open-Meteo applies its own default of 7 days.
	if input.Days > 0 {
		urlParams["forecast_days"] = strconv.Itoa(input.Days)
	}

	endpoint := buildWeatherAPIURL(
		input.Latitude,
		input.Longitude,
		urlParams,
	)

	data, err := fetchWeatherData(ctx, endpoint)
	if err != nil {
		return nil, nil, err
	}
	if data.Hourly == nil {
		return nil, nil, errors.New("the API response has no current weather data")
	}

	weatherDescriptions := func(codes []int) []string {
		var descriptions []string
		for _, c := range codes {
			descriptions = append(descriptions, descriptionPerWeatherCode[strconv.Itoa(c)])
		}
		return descriptions
	}(data.Hourly.WeatherCode)

	output := GetWeatherForecastOutput{
		Time:               data.Hourly.Time,
		Temperature:        data.Hourly.Temperature2m,
		WindSpeed:          data.Hourly.WindSpeed10m,
		Humidity:           data.Hourly.RelativeHumidity,
		WeatherDescription: weatherDescriptions,
	}

	// I deliberately decided not to include a tool result. When the tool result
	// is nil, the SDK fulfills it based on the output.
	return nil, &output, nil
}

// GetWeatherSummary is the handler of the get_weather_summary tool. It returns the
// current weather for the given coordinates together with today's hourly
// temperature forecast.
func GetWeatherSummary(
	ctx context.Context,
	mcpReq *mcp.CallToolRequest,
	input GetWeatherSummaryInput,
) (*mcp.CallToolResult, *GetWeatherSummaryOutput, error) {
	endpoint := buildWeatherAPIURL(
		input.Latitude,
		input.Longitude,
		map[string]string{
			"current":       "temperature_2m,wind_speed_10m,relative_humidity_2m,weather_code",
			"hourly":        "temperature_2m",
			"forecast_days": "1",
		},
	)
	data, err := fetchWeatherData(ctx, endpoint)
	if err != nil {
		return nil, nil, err
	}
	if data.Current == nil || data.Hourly == nil {
		return nil, nil, errors.New("the API response has no current or hourly weather data")
	}

	output := GetWeatherSummaryOutput{
		Temperature:        data.Current.Temperature2m,
		WindSpeed:          data.Current.WindSpeed10m,
		Humidity:           data.Current.RelativeHumidity,
		WeatherDescription: descriptionPerWeatherCode[strconv.Itoa(data.Current.WeatherCode)],
	}
	output.Forecast.Time = data.Hourly.Time
	output.Forecast.Temperature = data.Hourly.Temperature2m

	// I deliberately decided not to include a tool result. When the tool result
	// is nil, the SDK fulfills it based on the output.
	return nil, &output, nil
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

// plainArrays replaces the nullable ["null", "array"] type of slices with a
// plain "array" in the schema and, recursively, in its nested properties,
// for better portability across clients.
func plainArrays(schema *jsonschema.Schema) {
	for _, prop := range schema.Properties {
		if slices.Contains(prop.Types, "array") {
			prop.Types = nil
			prop.Type = "array"
		}
		plainArrays(prop)
	}
}

/*
	NOTE: The forecastOutputSchema and summaryOutputSchema functions exist because
	of a warning in the MCP Inspector, specifically:

	"`type` is an array (["null","array"]). This is legal JSON Schema, and some model
	providers recommend it for nullable fields, but it is less portable: a client that
	maps tool schemas onto a single-`type` dialect (such as the OpenAPI subset used
	for Gemini function declarations) may reject the tool or drop the constraint."

	I found that, when an output has a slice field, the SDK infers its schema with
	jsonschema-go, which treats every Go slice as "array or null" (a nil slice
	marshals to JSON null), so the output schema gets `"type": ["null", "array"]`.
	The same happens with pointers (*T becomes "T or null"). Since that behavior
	lives in the inference, it cannot be changed from the output struct itself.

	The workaround is to provide the OutputSchema explicitly in mcp.Tool: the schema
	is still inferred from the output type (so descriptions from the `jsonschema`
	tags and any new field are kept), but plainArrays then replaces the nullable
	type with a plain "array", including in nested properties (summary's forecast).

	Trade-off: with a plain "array", `null` is no longer valid output, so the slices
	returned by the tools must not be nil.
*/

// forecastOutputSchema infers the output schema from GetWeatherForecastOutput.
func forecastOutputSchema() (*jsonschema.Schema, error) {
	schema, err := jsonschema.For[GetWeatherForecastOutput](nil)
	if err != nil {
		return nil, err
	}
	plainArrays(schema)
	return schema, nil
}

// summaryOutputSchema infers the output schema from GetWeatherSummaryOutput.
func summaryOutputSchema() (*jsonschema.Schema, error) {
	schema, err := jsonschema.For[GetWeatherSummaryOutput](nil)
	if err != nil {
		return nil, err
	}
	plainArrays(schema)
	return schema, nil
}

// main registers the weather tools in the MCP server and serves it over
// stdin/stdout until the client disconnects.
func main() {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "weather-mcp-server",
			Version: "1.0.0",
		},
		nil,
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "get_current_weather",
			Description: "Get current weather data for a specific location",
		},
		GetCurrentWeather,
	)

	forecastSchema, err := forecastOutputSchema()
	if err != nil {
		log.Fatal(err)
	}
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:         "get_weather_forecast",
			Description:  "Get hourly weather forecast for a specific location",
			OutputSchema: forecastSchema,
		},
		GetWeatherForecast,
	)

	summarySchema, err := summaryOutputSchema()
	if err != nil {
		log.Fatal(err)
	}
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:         "get_weather_summary",
			Description:  "Get a summary of current weather and today's forecast for a location",
			OutputSchema: summarySchema,
		},
		GetWeatherSummary,
	)

	err = server.Run(context.Background(), &mcp.StdioTransport{})
	if err != nil {
		log.Fatal(err)
	}
}
