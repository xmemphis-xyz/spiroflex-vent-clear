from __future__ import annotations

from homeassistant.components.select import SelectEntity
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
    """Set up Spiroflex selects."""
    async_add_entities(
        [
            SpiroflexLevelSelect(entry.runtime_data),
            SpiroflexModeSelect(entry.runtime_data),
        ]
    )


class SpiroflexLevelSelect(SpiroflexEntity, SelectEntity):
    """Select the ventilation level."""

    _attr_name = "Bieg"
    _attr_icon = "mdi:fan"
    _attr_unique_id = f"{DOMAIN}_level"
    _attr_options = ["1", "2", "3", "Pauza"]

    @property
    def current_option(self) -> str | None:
        data = self.coordinator.data
        if data.get("paused"):
            return "Pauza"
        level = data.get("level")
        return str(level) if level is not None else None

    async def async_select_option(self, option: str) -> None:
        if option == "Pauza":
            await self.coordinator.api.async_pause()
        else:
            await self.coordinator.api.async_set_level(int(option))
        await self.coordinator.async_request_refresh()


class SpiroflexModeSelect(SpiroflexEntity, SelectEntity):
    """Select the operating mode."""

    _attr_name = "Tryb pracy"
    _attr_icon = "mdi:calendar-clock"
    _attr_unique_id = f"{DOMAIN}_mode"
    _attr_options = ["Harmonogram", "Manual"]

    @property
    def current_option(self) -> str | None:
        return "Harmonogram" if self.coordinator.data.get("mode") == "schedule" else "Manual"

    async def async_select_option(self, option: str) -> None:
        mode = "schedule" if option == "Harmonogram" else "manual"
        await self.coordinator.api.async_set_mode(mode)
        await self.coordinator.async_request_refresh()
