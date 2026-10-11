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

Only when activation state, duration and cancel write have been verified
on this controller should a writable Home Assistant switch be enabled.
The switch will belong after the two BOOST switches. If the fireplace
mode is to compensate a cooker hood, ensure sufficient combustion air
independently; software control is not a safety interlock.
