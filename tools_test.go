package main

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newTestSession connects a new weather server and a new client through in-memory
// transports and returns the client session. The session is closed automatically
// when the test finishes.
func newTestSession(t *testing.T) *mcp.ClientSession {
	t.Helper()
	// In-memory transports connect the server and the client inside the test process,
	// so there is no binary to build nor child process to start and stop. The two
	// ends are interchangeable: one goes to the server and the other to the client.
	st, ct := mcp.NewInMemoryTransports()

	server := newMCPWeatherService()
	if _, err := server.Connect(t.Context(), st, nil); err != nil {
		t.Fatal(err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	session, err := client.Connect(t.Context(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })

	return session
}

// callToolAndValidateResult calls the tool and returns the tool result,
// failing the test if the call or the tool fails.
func callToolAndValidateResult(
	t *testing.T,
	session *mcp.ClientSession,
	name string,
	args map[string]any,
) *mcp.CallToolResult {
	t.Helper()

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("the tool returned an error: %+v", res.Content)
	}

	return res
}

// arrayField returns structured[field] as a slice, failing the test if it is not one.
func arrayField(t *testing.T, structured map[string]any, field string) []any {
	t.Helper()

	values, ok := structured[field].([]any)
	if !ok {
		t.Fatalf("expected %q to be an array, got %T", field, structured[field])
	}

	return values
}

// TestGetCurrentWeather checks that get_current_weather returns both the text result
// and the structured output with all its fields. It calls the real Weather API, so
// it only validates the shape of the response, not the values.
func TestGetCurrentWeather(t *testing.T) {
	session := newTestSession(t)

	res := callToolAndValidateResult(t, session, "get_current_weather", map[string]any{
		"latitude": 6.245, "longitude": -75.571,
	})

	if len(res.Content) == 0 {
		t.Fatal("expected a text content, got none")
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", res.Content[0])
	}
	if !strings.HasPrefix(text.Text, "Current weather:") {
		t.Errorf("unexpected text: %q", text.Text)
	}

	structured, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured content as a map, got %T", res.StructuredContent)
	}
	for _, field := range []string{"temperature", "wind_speed", "humidity", "weather_description"} {
		if _, found := structured[field]; !found {
			t.Errorf("structured content is missing %q", field)
		}
	}
	if desc, _ := structured["weather_description"].(string); desc == "" {
		t.Error("weather_description should not be empty")
	}
}

// TestGetWeatherForecast checks that get_weather_forecast returns one hourly entry per
// field for the requested number of days, and for 7 days when "days" is omitted.
func TestGetWeatherForecast(t *testing.T) {
	session := newTestSession(t)

	tests := []struct {
		name      string
		args      map[string]any
		wantHours int
	}{
		{
			name:      "default days",
			args:      map[string]any{"latitude": 6.245, "longitude": -75.571},
			wantHours: 7 * 24,
		},
		{
			name:      "three days",
			args:      map[string]any{"latitude": 6.245, "longitude": -75.571, "days": 3},
			wantHours: 3 * 24,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := callToolAndValidateResult(t, session, "get_weather_forecast", test.args)
			structured, ok := res.StructuredContent.(map[string]any)
			if !ok {
				t.Fatalf("expected structured content as a map, got %T", res.StructuredContent)
			}

			for _, field := range []string{"time", "temperature", "wind_speed", "humidity", "weather_description"} {
				if got := len(arrayField(t, structured, field)); got != test.wantHours {
					t.Errorf("%q has %d entries, want %d", field, got, test.wantHours)
				}
			}
		})
	}
}

// TestGetWeatherSummary checks that get_weather_summary returns the current weather
// fields together with today's hourly forecast (24 entries).
func TestGetWeatherSummary(t *testing.T) {
	session := newTestSession(t)

	res := callToolAndValidateResult(t, session, "get_weather_summary", map[string]any{
		"latitude": 6.245, "longitude": -75.571,
	})
	structured, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured content as a map, got %T", res.StructuredContent)
	}

	for _, field := range []string{"temperature", "wind_speed", "humidity", "weather_description"} {
		if _, found := structured[field]; !found {
			t.Errorf("structured content is missing %q", field)
		}
	}
	if desc, _ := structured["weather_description"].(string); desc == "" {
		t.Error("weather_description should not be empty")
	}

	forecast, ok := structured["forecast"].(map[string]any)
	if !ok {
		t.Fatalf("expected forecast to be an object, got %T", structured["forecast"])
	}
	// The summary only includes today's forecast: one entry per hour.
	for _, field := range []string{"time", "temperature"} {
		if got := len(arrayField(t, forecast, field)); got != 24 {
			t.Errorf("forecast.%s has %d entries, want 24", field, got)
		}
	}
}
