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
    """Set up the power switch and two independently controllable BOOST modes."""
    async_add_entities(
        [
            SpiroflexPowerSwitch(entry.runtime_data),
            SpiroflexBoostSwitch(entry.runtime_data, 1),
            SpiroflexBoostSwitch(entry.runtime_data, 2),
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
        self._attr_unique_id = f"{DOMAIN}_{coordinator.api.base_url}_boost_{boost}"

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
        if not isinstance(own, bool) or not isinstance(other, bool):
            return False
        # An active BOOST MUST stay available so that OFF remains possible.
        return own or (data.get("power") is True and not other)

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
