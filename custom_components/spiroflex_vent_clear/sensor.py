from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from homeassistant.components.sensor import SensorDeviceClass, SensorEntity, SensorStateClass
from homeassistant.const import PERCENTAGE, UnitOfTemperature, UnitOfVolumeFlowRate
from homeassistant.core import HomeAssistant
from homeassistant.helpers.entity_platform import AddConfigEntryEntitiesCallback

from .const import DOMAIN
from .coordinator import SpiroflexCoordinator
from .entity import SpiroflexEntity


@dataclass(frozen=True)
class SensorDescription:
    key: str
    name: str
    icon: str
    unit: str | None = None
    device_class: SensorDeviceClass | None = None
    state_class: SensorStateClass | None = SensorStateClass.MEASUREMENT


SENSORS = (
    SensorDescription("status", "Stan pracy", "mdi:state-machine"),
    SensorDescription("supply_temperature", "Temperatura nawiewu", "mdi:thermometer", UnitOfTemperature.CELSIUS, SensorDeviceClass.TEMPERATURE),
    SensorDescription("extract_temperature", "Temperatura wyciągu", "mdi:thermometer", UnitOfTemperature.CELSIUS, SensorDeviceClass.TEMPERATURE),
    SensorDescription("intake_temperature", "Temperatura czerpni", "mdi:thermometer", UnitOfTemperature.CELSIUS, SensorDeviceClass.TEMPERATURE),
    SensorDescription("exhaust_temperature", "Temperatura wyrzutni", "mdi:thermometer", UnitOfTemperature.CELSIUS, SensorDeviceClass.TEMPERATURE),
    SensorDescription("room_temperature", "Temperatura pomieszczenia", "mdi:home-thermometer", UnitOfTemperature.CELSIUS, SensorDeviceClass.TEMPERATURE),
    SensorDescription("humidity", "Wilgotność", "mdi:water-percent", PERCENTAGE, SensorDeviceClass.HUMIDITY),
    SensorDescription("co2", "CO₂", "mdi:molecule-co2", "ppm", SensorDeviceClass.CO2),
    SensorDescription("supply_airflow", "Przepływ nawiewu", "mdi:weather-windy", UnitOfVolumeFlowRate.CUBIC_METERS_PER_HOUR),
    SensorDescription("extract_airflow", "Przepływ wywiewu", "mdi:weather-windy", UnitOfVolumeFlowRate.CUBIC_METERS_PER_HOUR),
    SensorDescription("supply_fan_percent", "Nawiew", "mdi:fan", PERCENTAGE),
    SensorDescription("extract_fan_percent", "Wywiew", "mdi:fan", PERCENTAGE),
    SensorDescription("filter_days_remaining", "Pozostało dni do wymiany filtrów", "mdi:air-filter", "d"),
    SensorDescription("supply_filter_usage", "Zużycie filtra nawiewu", "mdi:air-filter", PERCENTAGE),
    SensorDescription("extract_filter_usage", "Zużycie filtra wywiewu", "mdi:air-filter", PERCENTAGE),
    SensorDescription("protect_box_days", "Protect Box - pozostało dni", "mdi:air-filter", "d"),
)


async def async_setup_entry(
    hass: HomeAssistant,
    entry,
    async_add_entities: AddConfigEntryEntitiesCallback,
) -> None:
    """Set up Spiroflex sensors."""
    async_add_entities(
        [SpiroflexSensor(entry.runtime_data, description) for description in SENSORS]
    )


class SpiroflexSensor(SpiroflexEntity, SensorEntity):
    """A sensor backed by the coordinator status."""

    def __init__(
        self,
        coordinator: SpiroflexCoordinator,
        description: SensorDescription,
    ) -> None:
        super().__init__(coordinator)
        self.entity_description = description
        self._attr_name = description.name
        self._attr_icon = description.icon
        self._attr_unique_id = f"{DOMAIN}_{description.key}"
        self._attr_native_unit_of_measurement = description.unit
        self._attr_device_class = description.device_class
        self._attr_state_class = description.state_class

    @property
    def native_value(self) -> Any:
        return self.coordinator.data.get(self.entity_description.key)
