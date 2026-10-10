# Move an existing integration to the Home Assistant app

Integration version 0.1.6 adds Reconfigure. This is a HACS integration update only; the existing app/backend 0.1.6 does not need rebuilding.

1. Verify the new backend with GET /api/vent/status on the Home Assistant host at port 8088.
2. Install integration 0.1.6 and restart Home Assistant Core once to load the new Python code.
3. Open Settings > Devices & services > Spiroflex ecoVENT Simple. Use the menu of the existing config entry and choose Reconfigure.
4. Enter the new host (without http://) and port 8088, then submit.
5. Verify that the existing device and entity IDs are unchanged and test power/BOOST control from HA before stopping the old bridge process.

The flow tests the new endpoint before saving. It updates and reloads the same config entry, retaining its unique ID and all unrelated data. The original URL-based device identifier is saved separately from the connection address and reused by every entity. This avoids a duplicate device when moving from a separate host to the app. Subsequent moves preserve that same identifier. Entity unique IDs are not changed.

Only change the endpoint to another bridge serving the SAME ventilation unit. The current API status response does not expose an immutable controller identity; successful connectivity alone does not prove controller identity.

The tests in tests/test_boost_reconfigure.py cover identity preservation, repeated moves, duplicate endpoints, invalid inputs and failed connections using lightweight HA interface stubs. They do not replace a live Home Assistant migration test. Do not delete/re-add the integration or edit .storage manually.

Keep ecoNET passwords out of screenshots and Git. If a password was shared accidentally, rotate it and update the app options and any bridge configuration still in use.
