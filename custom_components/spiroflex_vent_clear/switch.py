from __future__ import annotations

from typing import Any

from homeassistant.components.switch import SwitchEntity
from homeassistant.core import HomeAssistant
from homeassistant.exceptions import HomeAssistantError
from homeassistant.helpers.entity_platform import AddConfigEntryEntitiesCallback

from .api import SpiroflexApiError
from .const import DOMAIN
from .coordinator import SpiroflexCoordinator
from .entity import SpiroflexEntity


async def async_setup_entry(
    hass: HomeAssistant,
    entry,
    async_add_entities: AddConfigEntryEntitiesCallback,
) -> None:
    """Set up power, BOOST 1, BOOST 2, and Kominek timed switches."""
    async_add_entities(
        [
            SpiroflexPowerSwitch(entry.runtime_data),
            SpiroflexBoostSwitch(entry.runtime_data, 1),
            SpiroflexBoostSwitch(entry.runtime_data, 2),
            SpiroflexFireplaceSwitch(entry.runtime_data),
        ]
    )


class SpiroflexPowerSwitch(SpiroflexEntity, SwitchEntity):
    """Control ventilation power."""

    _attr_name = "Rekuperator"
    _attr_icon = "mdi:hvac"
    _attr_unique_id = f"{DOMAIN}_power"

    @property
    def is_on(self) -> bool:
        return bool(self.coordinator.data.get("power", False))

    async def async_turn_on(self, **kwargs: Any) -> None:
        await self.coordinator.api.async_set_power("on")
        await self.coordinator.async_request_refresh()

    async def async_turn_off(self, **kwargs: Any) -> None:
        await self.coordinator.api.async_set_power("off")
        await self.coordinator.async_request_refresh()


class SpiroflexBoostSwitch(SpiroflexEntity, SwitchEntity):
    """Start or cancel one timed BOOST without switching ventilation power."""

    def __init__(self, coordinator: SpiroflexCoordinator, boost: int) -> None:
        super().__init__(coordinator)
        self._boost = boost
        self._state_key = f"boost_{boost}_active"
        self._other_key = f"boost_{3 - boost}_active"
        self._attr_name = f"BOOST {boost}"
        self._attr_icon = "mdi:fan-clock"
        self._attr_unique_id = f"{DOMAIN}_{coordinator.device_identifier}_boost_{boost}"

    @property
    def is_on(self) -> bool | None:
        value = self.coordinator.data.get(self._state_key)
        return value if isinstance(value, bool) else None

    @property
    def available(self) -> bool:
        if not super().available:
            return False
        data = self.coordinator.data
        own = data.get(self._state_key)
        other = data.get(self._other_key)
        fireplace = data.get("fireplace_active")
        if not all(isinstance(value, bool) for value in (own, other, fireplace)):
            return False
        # An active BOOST MUST stay available so that OFF remains possible.
        return own or (data.get("power") is True and not other and not fireplace)

    async def _async_set_boost(self, on: bool) -> None:
        try:
            await self.coordinator.api.async_set_boost(self._boost, on)
        except SpiroflexApiError as err:
            raise HomeAssistantError(str(err)) from err
        finally:
            # A command can take effect even if its HTTP readback times out.
            # Never optimistically display OFF; reload the device's real state.
            await self.coordinator.async_request_refresh()

    async def async_turn_on(self, **kwargs: Any) -> None:
        await self._async_set_boost(True)

    async def async_turn_off(self, **kwargs: Any) -> None:
        await self._async_set_boost(False)


class SpiroflexFireplaceSwitch(SpiroflexEntity, SwitchEntity):
    """Start or stop the Kominek timed mode without changing power."""

    _attr_name = "Kominek"
    _attr_icon = "mdi:fireplace"

    def __init__(self, coordinator: SpiroflexCoordinator) -> None:
        super().__init__(coordinator)
        self._attr_unique_id = f"{DOMAIN}_{coordinator.device_identifier}_fireplace"

    @property
    def is_on(self) -> bool | None:
        value = self.coordinator.data.get("fireplace_active")
        return value if isinstance(value, bool) else None

    @property
    def available(self) -> bool:
        if not super().available:
            return False
        data = self.coordinator.data
        state = data.get("fireplace_active")
        first = data.get("boost_1_active")
        second = data.get("boost_2_active")
        if not all(isinstance(value, bool) for value in (state, first, second)):
            return False
        # OFF must remain possible even if another mode has been activated.
        return state or (data.get("power") is True and not first and not second)

    async def _async_set_fireplace(self, on: bool) -> None:
        try:
            await self.coordinator.api.async_set_fireplace(on)
        except SpiroflexApiError as err:
            raise HomeAssistantError(str(err)) from err
        finally:
            # The command may have taken effect if the readback timed out.
            await self.coordinator.async_request_refresh()

    async def async_turn_on(self, **kwargs: Any) -> None:
        await self._async_set_fireplace(True)

    async def async_turn_off(self, **kwargs: Any) -> None:
        await self._async_set_fireplace(False)
