# BOOST ON/OFF

The integration adds two native switches, BOOST 1 and BOOST 2. OFF cancels only that timed mode. It does not turn ventilation power off or alter the saved fan percentages, durations, level or schedule.

The existing start buttons keep their unique IDs and still run button.press automations. New installations disable those legacy buttons by default. Existing users may hide them after switching their dashboard to the two switches. The active-state and remaining-time sensors are unchanged.

## REST API

- POST /api/vent/boost/1 and /api/vent/boost/2: compatible start routes.
- POST /api/vent/boost/1/on and /api/vent/boost/2/on: start aliases.
- POST /api/vent/boost/1/off and /api/vent/boost/2/off: cancel the selected BOOST.

ON requires ventilation power and no conflicting timed mode. Repeating ON on the same active BOOST does not restart its timer. OFF is allowed for an active BOOST even if ventilation power is off. OFF on an already inactive BOOST is a verified no-op.

Start uses the experimentally confirmed numeric write to u6639 (64 for BOOST 1, 128 for BOOST 2). Cancellation is implemented as current_mask & ~selected_bit, preserving every other bit. This cancellation command has not yet been verified on the physical controller. The server returns success only if readback confirms the requested bit state, the corresponding timer (-1 when stopped), unchanged power, and unchanged other mask bits. It writes once and retries only reads. An acknowledgement alone is not success.

Concurrent BOOST writes, including the legacy diagnostic write route, are serialized within one WebServer. This is not a distributed lock: do not run tests from two backend instances or simultaneously change BOOST in the official app. The firmware can also update flags independently. No whole-register reset, guessed timer write, automatic retry of the write, or automatic power-off fallback is used.

## Deployment and live check

Update the backend before the HACS integration. Keep the existing HA config entry and device: do not delete or recreate them. On diomedes, pull main, run go test ./... and build the binary, then restart only ventclear. The Home Assistant app needs a rebuilt image; updating HACS alone does not change the backend.

With BOOST 1 already running, POST /api/vent/boost/1/off, then GET /api/vent/status. Expected: ok=true, verified=true, active=false in the response; boost_1_active=false and boost_1_remaining=null in status; power remains true. A failed readback is an error to investigate, not an instruction to keep repeating writes. The official app remains the fallback for stopping a mode.

## Tests

Run go test -race ./... and go vet ./... for Go. Run python3 -m unittest discover -s tests -p 'test_boost_*.py' -v for Python. Python tests use lightweight Home Assistant interface stubs; they do not replace a live HA installation test.
