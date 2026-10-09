"""BOOST entity/client contract tests with HA stubs, not a live HA instance."""
from __future__ import annotations

import importlib.util
from pathlib import Path
import sys
import types
import unittest
from unittest.mock import AsyncMock, patch

ROOT = Path(__file__).resolve().parents[1] / "custom_components" / "spiroflex_vent_clear"


def load_modules():
    def module(name, **attrs):
        result = types.ModuleType(name)
        for key, value in attrs.items():
            setattr(result, key, value)
        return result

    class BaseEntity:
        def __init__(self, coordinator):
            self.coordinator = coordinator

        @property
        def available(self):
            return self.coordinator.last_update_success

    class HAError(Exception):
        pass

    package = module("boost_contract")
    package.__path__ = [str(ROOT)]
    stubs = {
        "boost_contract": package,
        "aiohttp": module("aiohttp", ClientError=type("ClientError", (Exception,), {}), ClientSession=object),
        "homeassistant": module("homeassistant"),
        "homeassistant.components": module("homeassistant.components"),
        "homeassistant.components.switch": module("homeassistant.components.switch", SwitchEntity=type("SwitchEntity", (), {})),
        "homeassistant.core": module("homeassistant.core", HomeAssistant=object),
        "homeassistant.exceptions": module("homeassistant.exceptions", HomeAssistantError=HAError),
        "homeassistant.helpers": module("homeassistant.helpers"),
        "homeassistant.helpers.entity_platform": module("homeassistant.helpers.entity_platform", AddConfigEntryEntitiesCallback=object),
        "boost_contract.entity": module("boost_contract.entity", SpiroflexEntity=BaseEntity),
        "boost_contract.coordinator": module("boost_contract.coordinator", SpiroflexCoordinator=object),
        "boost_contract.const": module("boost_contract.const", DOMAIN="spiroflex_vent_clear"),
    }
    loaded = {}
    with patch.dict(sys.modules, stubs):
        for name in ("api", "switch"):
            spec = importlib.util.spec_from_file_location(f"boost_contract.{name}", ROOT / f"{name}.py")
            value = importlib.util.module_from_spec(spec)
            sys.modules[spec.name] = value
            spec.loader.exec_module(value)
            loaded[name] = value
    return loaded["api"], loaded["switch"], HAError


api_module, switch_module, HAError = load_modules()


class BoostSwitchTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self):
        self.api = types.SimpleNamespace(base_url="http://test:8088", async_set_boost=AsyncMock())
        self.coordinator = types.SimpleNamespace(
            api=self.api, last_update_success=True,
            data={"power": True, "boost_1_active": True, "boost_2_active": False},
            async_request_refresh=AsyncMock(),
        )
        self.first = switch_module.SpiroflexBoostSwitch(self.coordinator, 1)
        self.second = switch_module.SpiroflexBoostSwitch(self.coordinator, 2)

    def test_active_switch_available_for_stop(self):
        self.assertTrue(self.first.available)
        self.assertTrue(self.first.is_on)
        self.assertFalse(self.second.available)

    def test_stop_stays_available_with_power_off(self):
        self.coordinator.data["power"] = False
        self.assertTrue(self.first.available)

    def test_idle_switches_available(self):
        self.coordinator.data["boost_1_active"] = False
        self.assertTrue(self.first.available)
        self.assertTrue(self.second.available)

    def test_missing_state_not_off(self):
        del self.coordinator.data["boost_1_active"]
        self.assertIsNone(self.first.is_on)
        self.assertFalse(self.first.available)

    def test_failed_update_is_unavailable(self):
        self.coordinator.last_update_success = False
        self.assertFalse(self.first.available)

    async def test_off_calls_only_boost_and_refreshes(self):
        await self.first.async_turn_off()
        self.api.async_set_boost.assert_awaited_once_with(1, False)
        self.coordinator.async_request_refresh.assert_awaited_once()
        self.assertTrue(self.first.is_on)  # No optimistic state assignment.

    async def test_on_calls_boost(self):
        await self.second.async_turn_on()
        self.api.async_set_boost.assert_awaited_once_with(2, True)

    async def test_failed_command_reports_error_and_refreshes(self):
        self.api.async_set_boost.side_effect = api_module.SpiroflexApiError("readback failed")
        with self.assertRaises(HAError):
            await self.first.async_turn_off()
        self.coordinator.async_request_refresh.assert_awaited_once()

    async def test_api_paths_and_timeouts(self):
        api = api_module.SpiroflexApi(None, "test", 8088)
        api._request = AsyncMock(return_value={"ok": True})
        await api.async_start_boost(1)
        api._request.assert_awaited_with("POST", "/api/vent/boost/1", timeout=45)
        await api.async_stop_boost(2)
        api._request.assert_awaited_with("POST", "/api/vent/boost/2/off", timeout=45)

    async def test_empty_ack_is_not_success(self):
        api = api_module.SpiroflexApi(None, "test", 8088)
        api._request = AsyncMock(return_value={})
        with self.assertRaises(api_module.SpiroflexApiError):
            await api.async_stop_boost(1)

    async def test_invalid_argument_does_not_send_request(self):
        api = api_module.SpiroflexApi(None, "test", 8088)
        api._request = AsyncMock()
        for boost in (0, 3, True, "1"):
            with self.assertRaises(ValueError):
                await api.async_set_boost(boost, False)
        api._request.assert_not_awaited()


if __name__ == "__main__":
    unittest.main()
