from __future__ import annotations

from homeassistant.components.switch import SwitchEntity
from homeassistant.core import HomeAssistant
from homeassistant.helpers.entity_platform import AddConfigEntryEntitiesCallback

from .const import DOMAIN
from .coordinator import SpiroflexCoordinator
from .entity import SpiroflexEntity


async def async_setup_entry(
    hass: HomeAssistant,
    entry,
    async_add_entities: AddConfigEntryEntitiesCallback,
) -> None:
    """Set up the power switch."""
    async_add_entities([SpiroflexPowerSwitch(entry.runtime_data)])


class SpiroflexPowerSwitch(SpiroflexEntity, SwitchEntity):
    """Control ventilation power."""

    _attr_name = "Rekuperator"
    _attr_icon = "mdi:hvac"
    _attr_unique_id = f"{DOMAIN}_power"

    @property
    def is_on(self) -> bool:
        return bool(self.coordinator.data.get("power", False))

    async def async_turn_on(self, **kwargs) -> None:
        await self.coordinator.api.async_set_power("on")
        await self.coordinator.async_request_refresh()

    async def async_turn_off(self, **kwargs) -> None:
        await self.coordinator.api.async_set_power("off")
        await self.coordinator.async_request_refresh()
