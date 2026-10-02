# Spiroflex ecoVENT Simple

Home Assistant app that runs the Go REST bridge for Spiroflex Vent Clear / ecoVENT Simple.

The app replaces the standalone `ventclear` process previously running on another server.

## Configuration

Configure the ecoNET connection in the app configuration:

- Region
- Cognito username and password
- Cognito User Pool ID
- Cognito Client ID
- Cognito Identity Pool ID
- API Gateway name
- AWS IoT name
- Installation name and ID

The REST API listens on port 8088.

The configuration is stored by Home Assistant in the app data directory and is not committed to Git.

## Home Assistant integration

After starting the app, configure the **Spiroflex ecoVENT Simple** HACS integration to connect to the Home Assistant host on port 8088.

For example:

```
Host: homeassistant.local
Port: 8088
```

The Go service communicates directly with ecoNET/AWS IoT. The Home Assistant integration communicates only with its local REST API.

## Testing

Before removing the existing service from the other server, verify the REST status endpoint and the power/level controls.

Keep the existing service available until the Home Assistant app has been tested successfully.
