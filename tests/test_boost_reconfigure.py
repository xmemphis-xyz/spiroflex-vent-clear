"""Reconfiguration contract tests using HA stubs, not a live HA instance."""
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
        result.__dict__.update(attrs)
        return result

    class FlowBase:
        def __init_subclass__(cls, **kwargs):
            pass

        def _get_reconfigure_entry(self):
            return self.entry

        def _async_current_entries(self):
            return self.entries

        async def async_set_unique_id(self, value):
            self.unique_id = value

        def _abort_if_unique_id_mismatch(self):
            assert self.unique_id == self.entry.unique_id

        def async_show_form(self, **kwargs):
            return {"type": "form", **kwargs}

        def async_abort(self, **kwargs):
            return {"type": "abort", **kwargs}

        def async_update_reload_and_abort(self, entry, *, data_updates, reason):
            entry.data = {**entry.data, **data_updates}
            self.reloads += 1
            return {"type": "abort", "reason": reason}

    class ApiError(Exception):
        pass

    class FakeApi:
        fail = False
        checks = 0

        def __init__(self, session, host, port):
            self.base_url = f"http://{host}:{port}"

        async def async_get_status(self):
            type(self).checks += 1
            if type(self).fail:
                raise ApiError("offline")
            return {"power": True}

    class Generic:
        @classmethod
        def __class_getitem__(cls, key):
            return cls

    class CoordinatorBase(Generic):
        def __init__(self, *args, **kwargs):
            pass

    class EntityBase(Generic):
        def __init__(self, coordinator):
            self.coordinator = coordinator

    package = module("spiro_reconfigure_contract")
    package.__path__ = [str(ROOT)]
    ce = module("homeassistant.config_entries", ConfigFlow=FlowBase, ConfigFlowResult=dict)
    stubs = {
        "spiro_reconfigure_contract": package,
        "spiro_reconfigure_contract.api": module("api", SpiroflexApi=FakeApi, SpiroflexApiError=ApiError),
        "spiro_reconfigure_contract.const": module("const", DOMAIN="spiroflex_vent_clear", DEFAULT_PORT=8088, DEFAULT_SCAN_INTERVAL=30),
        "voluptuous": module("voluptuous", Schema=lambda data: data, Required=lambda name, **kw: name,
                              Coerce=lambda value: value, All=lambda *args: args, Range=lambda **kwargs: kwargs),
        "homeassistant": module("homeassistant", config_entries=ce),
        "homeassistant.config_entries": ce,
        "homeassistant.const": module("homeassistant.const", CONF_HOST="host", CONF_PORT="port"),
        "homeassistant.core": module("homeassistant.core", HomeAssistant=object),
        "homeassistant.helpers": module("homeassistant.helpers"),
        "homeassistant.helpers.aiohttp_client": module("aiohttp_client", async_get_clientsession=lambda hass: None),
        "homeassistant.helpers.update_coordinator": module("update_coordinator", DataUpdateCoordinator=CoordinatorBase, CoordinatorEntity=EntityBase, UpdateFailed=Exception),
        "homeassistant.helpers.device_registry": module("device_registry", DeviceInfo=lambda **data: data),
    }
    loaded = {}
    with patch.dict(sys.modules, stubs):
        for name in ("config_flow", "coordinator", "entity"):
            spec = importlib.util.spec_from_file_location(f"spiro_reconfigure_contract.{name}", ROOT / f"{name}.py")
            value = importlib.util.module_from_spec(spec)
            sys.modules[spec.name] = value
            spec.loader.exec_module(value)
            loaded[name] = value
    return loaded, FakeApi


modules, Api = load_modules()


class ReconfigureTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self):
        Api.fail = False
        Api.checks = 0
        self.entry = types.SimpleNamespace(
            entry_id="unchanged-entry", unique_id="diomedes:8088",
            data={"host": "diomedes", "port": 8088, "preserve_other_setting": True},
        )
        self.flow = modules["config_flow"].SpiroflexConfigFlow()
        self.flow.hass = object()
        self.flow.entry = self.entry
        self.flow.entries = [self.entry]
        self.flow.reloads = 0

    async def test_show_form_does_not_contact_backend(self):
        result = await self.flow.async_step_reconfigure()
        self.assertEqual(result["step_id"], "reconfigure")
        self.assertEqual(Api.checks, 0)
        self.assertEqual(self.flow.reloads, 0)

    async def test_success_preserves_entry_and_device_identity(self):
        result = await self.flow.async_step_reconfigure({"host": " 192.168.1.7 ", "port": 8088})
        self.assertEqual(result["reason"], "reconfigure_successful")
        self.assertEqual(self.entry.entry_id, "unchanged-entry")
        self.assertEqual(self.entry.unique_id, "diomedes:8088")
        self.assertEqual(self.entry.data["host"], "192.168.1.7")
        self.assertEqual(self.entry.data["device_identifier"], "http://diomedes:8088")
        self.assertTrue(self.entry.data["preserve_other_setting"])
        self.assertEqual(len(self.flow.entries), 1)
        self.assertEqual(self.flow.reloads, 1)
        self.assertEqual(Api.checks, 1)

    async def test_repeated_move_keeps_original_identity(self):
        await self.flow.async_step_reconfigure({"host": "192.168.1.7", "port": 8088})
        await self.flow.async_step_reconfigure({"host": "other-host", "port": 8089})
        self.assertEqual(self.entry.data["device_identifier"], "http://diomedes:8088")
        self.assertEqual(self.entry.data["port"], 8089)

    async def test_failed_connection_does_not_change_entry(self):
        Api.fail = True
        before = dict(self.entry.data)
        result = await self.flow.async_step_reconfigure({"host": "offline", "port": 8088})
        self.assertEqual(result["errors"], {"base": "cannot_connect"})
        self.assertEqual(before, self.entry.data)
        self.assertEqual(self.flow.reloads, 0)

    async def test_duplicate_backend_is_rejected_before_network(self):
        self.flow.entries.append(types.SimpleNamespace(entry_id="other", data={"host": "addon", "port": 8088}))
        result = await self.flow.async_step_reconfigure({"host": "ADDON", "port": 8088})
        self.assertEqual(result["errors"], {"base": "already_configured"})
        self.assertEqual(Api.checks, 0)
        self.assertEqual(self.flow.reloads, 0)

    async def test_new_setup_cannot_duplicate_reconfigured_endpoint(self):
        await self.flow.async_step_reconfigure({"host": "addon", "port": 8088})
        result = await self.flow.async_step_user({"host": "addon", "port": 8088})
        self.assertEqual(result["reason"], "already_configured")
        self.assertEqual(Api.checks, 1)

    async def test_invalid_fields_do_not_contact_backend(self):
        for host, port in (("", 8088), ("http://addon", 8088), ("some host", 8088), ("addon", 0), ("addon", 65536), ("addon", "invalid")):
            with self.subTest(host=host, port=port):
                result = await self.flow.async_step_reconfigure({"host": host, "port": port})
                self.assertTrue(result["errors"])
        self.assertEqual(Api.checks, 0)
        self.assertEqual(self.flow.reloads, 0)

    async def test_same_endpoint_can_be_saved(self):
        result = await self.flow.async_step_reconfigure({"host": "diomedes", "port": 8088})
        self.assertEqual(result["reason"], "reconfigure_successful")

    def test_device_info_uses_stable_identifier_not_new_address(self):
        api = Api(None, "192.168.1.7", 8088)
        coordinator = modules["coordinator"].SpiroflexCoordinator(None, api, device_identifier="http://diomedes:8088")
        entity = modules["entity"].SpiroflexEntity(coordinator)
        self.assertEqual(entity._attr_device_info["identifiers"], {("spiroflex_vent_clear", "http://diomedes:8088")})
        self.assertEqual(coordinator.api.base_url, "http://192.168.1.7:8088")

    def test_legacy_setup_retains_existing_url_identity(self):
        api = Api(None, "diomedes", 8088)
        coordinator = modules["coordinator"].SpiroflexCoordinator(None, api)
        self.assertEqual(coordinator.device_identifier, api.base_url)


if __name__ == "__main__":
    unittest.main()
