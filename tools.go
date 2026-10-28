package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
