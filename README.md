# Weather MCP server in Go

A small [Model Context Protocol](https://modelcontextprotocol.io) (MCP) server written in Go with the [official Go SDK](https://github.com/modelcontextprotocol/go-sdk).
It exposes weather tools backed by the free [Open-Meteo](https://open-meteo.com) API, so no API key is needed.
It talks to MCP clients (IDEs, agents) over stdio.

Built as a learning project for getting started with MCP in Go.
It was based on the [weather MCP server](https://github.com/kodekloudhub/MCP-For-Beginners/tree/main/Weather-MCP-Server) of
the repo [MCP for beginners](https://github.com/kodekloudhub/MCP-For-Beginners).

## Installation

Requirements: the Go version declared in `go.mod` or newer.

```bash
go build -o weather-mcp-server .
```

Then register the binary in your MCP client using its **absolute path**. Replace `<absolute-path>` below with the directory where you built it (for example, the output of `pwd`). In Zed (`settings.json`):

```json
{
  "context_servers": {
    "weather": {
      "command": "<absolute-path>/weather-mcp-server",
      "args": [],
      "env": {}
    }
  }
}
```

Or in Claude Code:

```bash
claude mcp add weather -- <absolute-path>/weather-mcp-server
```

To try it without a client, use the MCP Inspector:

```bash
npx @modelcontextprotocol/inspector ./weather-mcp-server
```

## Tools

All tools take coordinates as input. Temperatures are in °C, wind speed in km/h and humidity in %.

| Tool | Input | Returns |
|---|---|---|
| `get_current_weather` | `latitude`, `longitude` | Current temperature, wind speed, humidity and a weather description. |
| `get_weather_forecast` | `latitude`, `longitude`, `days` (optional, default 7) | Hourly temperature, wind speed, humidity and weather description for the requested days. |
| `get_weather_summary` | `latitude`, `longitude` | Current weather plus today's hourly temperature forecast. |
