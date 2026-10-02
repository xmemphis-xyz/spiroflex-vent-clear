# Spiroflex ecoVENT Simple

Home Assistant integration and local REST bridge for **Spiroflex Vent Clear / ecoVENT Simple** ventilation units.

The project consists of:

- a Go service that communicates with the Spiroflex ecoNET/AWS IoT backend;
- a local REST API for controlling and reading the ventilation unit;
- a custom Home Assistant integration that communicates only with that local REST API.

> **Status:** early release. The integration is being developed against a real ecoVENT Simple installation. Some device functions, including BOOST 1/2 configuration, are still being mapped and are not yet exposed.

## Features

The current Home Assistant integration provides:

- ventilation power control;
- ventilation level 1–3;
- pause;
- operating mode: **Harmonogram** / **Manual**;
- operating status;
- supply, extract, intake, exhaust and room temperatures;
- humidity;
- supply and extract fan percentages;
- filter replacement countdown;
- supply and extract filter usage;
- alarm state.

Unsupported or unverified parameters are intentionally not exposed.

## Architecture

```
Home Assistant
      |
      | local HTTP
      v
ventclear REST API
      |
      | MQTT / AWS IoT
      v
Spiroflex ecoNET
```

AWS Cognito and IoT credentials stay in the Go service configuration. They are not stored in Home Assistant.

## Home Assistant installation with HACS

After the repository is public:

1. Open **HACS → Integrations**.
2. Search for **Spiroflex ecoVENT Simple**.
3. Install the integration.
4. Restart Home Assistant.
5. Go to **Settings → Devices & services → Add integration**.
6. Search for **Spiroflex ecoVENT Simple**.
7. Enter the host and port of the machine running `ventclear`.

The default REST API port is **8088**.

### Manual HACS repository

If the repository has not yet been added to the HACS default repository list, add it as a custom repository:

```
https://github.com/xmemphis-xyz/spiroflex-vent-clear
```

Select **Integration** as the category.

## Running the Go service

Create a local `config.yaml` containing the required ecoNET credentials and installation information.

Example structure:

```yaml
region: eu-west-3

cognito:
  username: "user@example.com"
  password: "your-password"
  user_pool_id: "eu-west-3_example"
  client_id: "your-client-id"
  identity_pool_id: "eu-west-3:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

gateway:
  name: your-api-gateway

iot:
  name: your-iot-endpoint

installation:
  name: "ecoVENT Simple"
  id: "your-installation-id"

api:
  endpoint: "0.0.0.0:8088"
  rest: true
```

**Do not commit `config.yaml`.** It is included in `.gitignore`.

Run:

```bash
go run ./cmd/ventclear
```

Or build a binary:

```bash
go build -o ventclear ./cmd/ventclear
```

## Local API

The REST bridge currently exposes:

```
GET  /api/vent/status

POST /api/vent/power/on
POST /api/vent/power/off

POST /api/vent/level/1
POST /api/vent/level/2
POST /api/vent/level/3
POST /api/vent/pause

POST /api/vent/mode/schedule
POST /api/vent/mode/manual
```

Diagnostic endpoints are also available for protocol development.

## Development

The Home Assistant integration lives under:

```
custom_components/spiroflex_vent_clear/
```

The Go REST API lives under:

```
api/
```

The ecoNET communication layer lives under:

```
econet/
```

Pull requests and issue reports are welcome.

## Disclaimer

This project is an independent community project and is not affiliated with or endorsed by Spiroflex. Use it at your own risk. Test changes carefully before using them to control a ventilation system.
