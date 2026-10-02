# spiroflex-vent-clear

## ⚠️  Disclaimer

This repository is intended **solely for educational purposes**.

- Do **not** use this code in production environments or for any real-world applications.
- The code may be incomplete, insecure, or not follow best practices.
- I **do not take any responsibility** for any issues, damages, or consequences resulting from the use, misuse, or inability to use the code provided in this repository.

By using any part of this code, you agree that you are doing so **at your own risk**.

## 📦 Project Overview

**spiroflex-vent-clear** is an experimental integration with the **Spiroflex Vent Clear** ventilation control system. It is implemented as a Go-based application exposing a RESTful API and support for **Amazon Alexa voice commands**.

The application communicates with AWS IoT and Cognito services to authenticate users and interact with VC device. It provides an HTTP API for local control, and optionally, an Alexa Skill endpoint for voice-based interaction.

## 🚀 How to Run

Ensure Go is installed, create `config.yaml` file, then launch the app using:

```bash
go run ./cmd/ventclear
```

## ⚙️ Sample Configuration

Below is a sample `config.yaml` file. All identifiers and values have been changed for privacy and illustrative purposes:

```yaml
region: eu-west-3

cognito:
  username: "user@example.com"
  password: "fake-password"
  user_pool_id: "eu-west-3_examplePool"
  client_id: "abc123exampleclientid"
  identity_pool_id: "eu-west-3:12345678-abcd-ef01-2345-6789abcdef01"

gateway:
  name: demoapigateway

iot:
  name: demoiotendpoint-ats

installation:
  name: "SCP V"

api:
  endpoint: 0.0.0.0:7777
  rest: true
  alexa: true

alexa:
  app_id: amzn1.ask.skill.00000000-0000-0000-0000-000000000000
```

## Home Assistant integration

The repository now contains a custom Home Assistant integration for **Spiroflex ecoVENT Simple**.

### Installation

Using HACS:

1. Open **HACS → Integrations**.
2. Add this repository as a custom repository:
   `https://github.com/xmemphis-xyz/spiroflex-vent-clear`
3. Select category **Integration**.
4. Install **Spiroflex ecoVENT Simple**.
5. Restart Home Assistant.
6. Go to **Settings → Devices & services → Add integration**.
7. Search for **Spiroflex ecoVENT Simple**.
8. Enter the host and port of the server running `ventclear` (default port: `8088`).

The integration creates one Home Assistant device and exposes power control, ventilation level, operating mode, temperatures, humidity, CO₂, airflow, fan percentages, filter status and alarm state.

The Home Assistant integration communicates only with the local REST API exposed by `ventclear`; AWS IoT/Cognito credentials remain on the Go service and are not stored in Home Assistant.
\n