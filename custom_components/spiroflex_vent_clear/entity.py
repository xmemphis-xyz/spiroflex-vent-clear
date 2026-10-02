from __future__ import annotations

from homeassistant.helpers.device_registry import DeviceInfo
from homeassistant.helpers.update_coordinator import CoordinatorEntity

from .const import DOMAIN
from .coordinator import SpiroflexCoordinator


class SpiroflexEntity(CoordinatorEntity[SpiroflexCoordinator]):
    """Base entity for Spiroflex ecoVENT Simple."""

    _attr_has_entity_name = True

    def __init__(self, coordinator: SpiroflexCoordinator) -> None:
        super().__init__(coordinator)
        self._attr_device_info = DeviceInfo(
            identifiers={(DOMAIN, coordinator.api.base_url)},
            name="Spiroflex ecoVENT Simple",
            manufacturer="Spiroflex",
            model="Vent Clear / ecoVENT Simple",
        )
