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
    async_add_entities([SpiroflexAlarmSensor(entry.runtime_data), SpiroflexBoostSensor(entry.runtime_data, 1), SpiroflexBoostSensor(entry.runtime_data, 2)])


class SpiroflexAlarmSensor(SpiroflexEntity, BinarySensorEntity):
    """Ventilation alarm."""

    _attr_name = "Alarm"
    _attr_icon = "mdi:alert"
    _attr_device_class = BinarySensorDeviceClass.PROBLEM
    _attr_unique_id = f"{DOMAIN}_alarm"

    @property
    def is_on(self) -> bool:
        return bool(self.coordinator.data.get("alarm", False))


class SpiroflexBoostSensor(SpiroflexEntity, BinarySensorEntity):
    """Whether a timed BOOST mode is running."""

    def __init__(self, coordinator: SpiroflexCoordinator, boost: int) -> None:
        super().__init__(coordinator)
        self._boost = boost
        self._attr_name = f"BOOST {boost} aktywny"
        self._attr_icon = "mdi:fan"
        self._attr_unique_id = f"{DOMAIN}_boost_{boost}_active"

    @property
    def is_on(self) -> bool:
        return bool(self.coordinator.data.get(f"boost_{self._boost}_active", False))
