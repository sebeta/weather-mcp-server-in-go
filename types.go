package main

// HTTP response from the weather API
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
