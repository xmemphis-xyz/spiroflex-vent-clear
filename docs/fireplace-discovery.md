# Kominek (fireplace) timed-mode protocol discovery

Spiroflex Vent Clear Deco / ecoVENT Simple has a third timed mode **Kominek**,
in addition to BOOST 1 and BOOST 2. The manufacturer's user manual confirms
it is separately configured, but does not provide ecoNET register IDs.

Do not extrapolate the third control flag from 64 and 128; a candidate
register or bit is not the same thing as a verified writable command.

A read-only endpoint is available after the backend is rebuilt:

`GET /api/diagnostics/fireplace`

The older `GET /api/diagnostics/parameter-table` endpoint can be used
**without upgrading the backend**, and is sufficient to map the mode.

1. With all timed modes OFF, save a baseline:

   `curl -fsS http://HOME_ASSISTANT_IP:8088/api/diagnostics/parameter-table | jq -S 'map(select(.[0] != null)) | map({key: .[0], value: .[2]}) | from_entries' > /tmp/kominek-off.json`

2. Activate **Kominek** in the official application. Wait 5–10 seconds.
3. Save the active values:

   `curl -fsS http://HOME_ASSISTANT_IP:8088/api/diagnostics/parameter-table | jq -S 'map(select(.[0] != null)) | map({key: .[0], value: .[2]}) | from_entries' > /tmp/kominek-on.json`

4. Compare:

   `diff -u /tmp/kominek-off.json /tmp/kominek-on.json`

Focus on `u6639` (mask), `u7429` (candidate countdown), `u7158`
(candidate configured duration), and `u6202/u6203` (fan percentages).
A screenshot or reading of the actual "Kominek" settings should also
be collected to confirm what the countdown represents.

Observed on 2026-10-11 from the user's before/after parameter-table diff:
- `u6639`: 0 -> **256** (Kominek mask bit confirmed on activation)
- `u7429`: -1 -> **10** (third timed-mode countdown confirmed)
- `u6202`: 40% -> **100%** (supply fan)
- `u6203`: 40% -> **10%** (extract fan)

Backend v0.1.7 adds POST /api/vent/fireplace/on and /off. It uses one
u6639 mask write (256 to start, &^256 to stop), preserving all other
bits, and waits for controller readback of both the state flag and timer.
START is blocked when BOOST 1/2 or any unknown timed mode is active.
Repeated ON must not reset the active timer; inactive OFF is a no-op.
The Go code tests these invariants.

HA v0.1.7 adds a native ON/OFF Kominek switch after BOOST 1 and BOOST 2,
binary sensor "Kominek aktywny", and remaining-time sensor. The addon
must be upgraded to v0.1.7 before upgrading HACS, because the previous
backend does not expose this command or third-mode status fields.

**Live ON/OFF from the new REST endpoint still requires device testing.**
The recorded diff proves manual activation from the official app, not
that REST writes are accepted. Do not blindly retry a timed-out POST:
read the current state first. Keep the hood automation disabled until
manual ON and OFF are verified in Home Assistant.

If the fireplace mode is used to assist a cooker hood, ensure adequate
combustion air independently; software control is not a safety interlock.
