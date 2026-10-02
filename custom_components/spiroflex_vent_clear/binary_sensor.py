from __future__ import annotations

from homeassistant.components.binary_sensor import (
    BinarySensorDeviceClass,
    BinarySensorEntity,
)
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
    """Set up Spiroflex binary sensors."""
    async_add_entities([SpiroflexAlarmSensor(entry.runtime_data)])


class SpiroflexAlarmSensor(SpiroflexEntity, BinarySensorEntity):
    """Ventilation alarm."""

    _attr_name = "Alarm"
    _attr_icon = "mdi:alert"
    _attr_device_class = BinarySensorDeviceClass.PROBLEM
    _attr_unique_id = f"{DOMAIN}_alarm"

    @property
    def is_on(self) -> bool:
        return bool(self.coordinator.data.get("alarm", False))
