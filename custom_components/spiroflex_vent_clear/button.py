from __future__ import annotations

from homeassistant.components.button import ButtonEntity, ButtonEntityDescription
from homeassistant.core import HomeAssistant
from homeassistant.helpers.entity_platform import AddConfigEntryEntitiesCallback

from .const import DOMAIN
from .coordinator import SpiroflexCoordinator
from .entity import SpiroflexEntity

# Preserve unique IDs and button.press automations from 0.1.4. New installations
# use the ON/OFF switches; previously enabled buttons remain user-controlled.
BOOST_BUTTONS = (
    ButtonEntityDescription(key="boost_1", name="BOOST 1 - uruchom", icon="mdi:fan-plus", entity_registry_enabled_default=False),
    ButtonEntityDescription(key="boost_2", name="BOOST 2 - uruchom", icon="mdi:fan-plus", entity_registry_enabled_default=False),
)


async def async_setup_entry(
    hass: HomeAssistant,
    entry,
    async_add_entities: AddConfigEntryEntitiesCallback,
) -> None:
    async_add_entities(
        [SpiroflexBoostButton(entry.runtime_data, description, index + 1)
         for index, description in enumerate(BOOST_BUTTONS)]
    )


class SpiroflexBoostButton(SpiroflexEntity, ButtonEntity):
    def __init__(self, coordinator: SpiroflexCoordinator, description: ButtonEntityDescription, boost: int) -> None:
        super().__init__(coordinator)
        self.entity_description = description
        self._boost = boost
        self._attr_unique_id = f"{DOMAIN}_{description.key}"

    @property
    def available(self) -> bool:
        if not super().available:
            return False
        data = self.coordinator.data
        return data.get("power") is True and data.get("boost_1_active") is False and data.get("boost_2_active") is False

    async def async_press(self) -> None:
        try:
            await self.coordinator.api.async_start_boost(self._boost)
        finally:
            await self.coordinator.async_request_refresh()
