package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// main serves a MCP weather server over stdin/stdout until the client disconnects.
func main() {
	server := newMCPWeatherService()
	err := server.Run(context.Background(), &mcp.StdioTransport{})
	if err != nil {
		log.Fatal(err)
	}
}

// newMCPWeatherService creates a new weather MCP server and registers the tools in it.
func newMCPWeatherService() *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "weather-mcp-server",
			Version: "0.0.1",
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

	return server
}
