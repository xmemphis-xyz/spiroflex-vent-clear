# Changelog

## 0.1.5

- Added native BOOST 1/2 switches with ON and OFF. An active switch stays available for cancellation.
- Added per-BOOST cancellation endpoints. Clear only the selected bit of u6639; do not write power, base mode, fan setpoints or configured duration.
- Verify both the selected state flag and its timer after every command. Report unconfirmed changes as errors, not success.
- Serialize BOOST writes within each backend process; repeated ON does not restart a running timer.
- Preserve legacy start buttons and their unique IDs for existing automations. They are disabled by default for new installations.
- Allow up to 45 seconds for BOOST HTTP calls and refresh actual HA state after errors as well as successful commands.
- Added Go unit tests, Python entity/client contract tests and CI.

Hardware note: cancellation by clearing the selected bit requires live validation on the user's controller. Tests cover the software contract, not the physical device. Upgrade the backend before installing this integration version. See docs/boost-controls.md.

## 0.1.1

- Removed the unnecessary Home Assistant version restriction from HACS metadata.

## 0.1.0

Initial Home Assistant release.

- Added HACS-compatible Home Assistant integration.
- Added local REST communication between Home Assistant and `ventclear`.
- Added power, level, pause and operating-mode controls.
- Added temperature, humidity, fan and filter sensors.
- Added alarm binary sensor.
- Removed unsupported CO2, airflow and Protect Box entities from the Home Assistant integration.
- Added VentClear integration icon.
