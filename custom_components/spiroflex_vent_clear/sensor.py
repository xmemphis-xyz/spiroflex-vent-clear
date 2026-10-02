from __future__ import annotations

from typing import Any

from homeassistant.components.sensor import (
    SensorDeviceClass,
    SensorEntity,
    SensorEntityDescription,
    SensorStateClass,
)
from homeassistant.const import PERCENTAGE, UnitOfTemperature, UnitOfVolumeFlowRate
from homeassistant.core import HomeAssistant
from homeassistant.helpers.entity_platform import AddConfigEntryEntitiesCallback

from .const import DOMAIN
from .coordinator import SpiroflexCoordinator
from .entity import SpiroflexEntity


SENSORS = (
    SensorEntityDescription(
        key="status",
        name="Stan pracy",
        icon="mdi:state-machine",
        state_class=None,
    ),
    SensorEntityDescription(
        key="supply_temperature",
        name="Temperatura nawiewu",
        icon="mdi:thermometer",
        native_unit_of_measurement=UnitOfTemperature.CELSIUS,
        device_class=SensorDeviceClass.TEMPERATURE,
    ),
    SensorEntityDescription(
        key="extract_temperature",
        name="Temperatura wyciągu",
        icon="mdi:thermometer",
        native_unit_of_measurement=UnitOfTemperature.CELSIUS,
        device_class=SensorDeviceClass.TEMPERATURE,
    ),
    SensorEntityDescription(
        key="intake_temperature",
        name="Temperatura czerpni",
        icon="mdi:thermometer",
        native_unit_of_measurement=UnitOfTemperature.CELSIUS,
        device_class=SensorDeviceClass.TEMPERATURE,
    ),
    SensorEntityDescription(
        key="exhaust_temperature",
        name="Temperatura wyrzutni",
        icon="mdi:thermometer",
        native_unit_of_measurement=UnitOfTemperature.CELSIUS,
        device_class=SensorDeviceClass.TEMPERATURE,
    ),
    SensorEntityDescription(
        key="room_temperature",
        name="Temperatura pomieszczenia",
        icon="mdi:home-thermometer",
        native_unit_of_measurement=UnitOfTemperature.CELSIUS,
        device_class=SensorDeviceClass.TEMPERATURE,
    ),
    SensorEntityDescription(
        key="humidity",
        name="Wilgotność",
        icon="mdi:water-percent",
        native_unit_of_measurement=PERCENTAGE,
        device_class=SensorDeviceClass.HUMIDITY,
    ),
    SensorEntityDescription(
        key="co2",
        name="CO₂",
        icon="mdi:molecule-co2",
        native_unit_of_measurement="ppm",
        device_class=SensorDeviceClass.CO2,
    ),
    SensorEntityDescription(
        key="supply_airflow",
        name="Przepływ nawiewu",
        icon="mdi:weather-windy",
        native_unit_of_measurement=UnitOfVolumeFlowRate.CUBIC_METERS_PER_HOUR,
    ),
    SensorEntityDescription(
        key="extract_airflow",
        name="Przepływ wywiewu",
        icon="mdi:weather-windy",
        native_unit_of_measurement=UnitOfVolumeFlowRate.CUBIC_METERS_PER_HOUR,
    ),
    SensorEntityDescription(
        key="supply_fan_percent",
        name="Nawiew",
        icon="mdi:fan",
        native_unit_of_measurement=PERCENTAGE,
    ),
    SensorEntityDescription(
        key="extract_fan_percent",
        name="Wywiew",
        icon="mdi:fan",
        native_unit_of_measurement=PERCENTAGE,
    ),
    SensorEntityDescription(
        key="filter_days_remaining",
        name="Pozostało dni do wymiany filtrów",
        icon="mdi:air-filter",
        native_unit_of_measurement="d",
    ),
    SensorEntityDescription(
        key="supply_filter_usage",
        name="Zużycie filtra nawiewu",
        icon="mdi:air-filter",
        native_unit_of_measurement=PERCENTAGE,
    ),
    SensorEntityDescription(
        key="extract_filter_usage",
        name="Zużycie filtra wywiewu",
        icon="mdi:air-filter",
        native_unit_of_measurement=PERCENTAGE,
    ),
    SensorEntityDescription(
        key="protect_box_days",
        name="Protect Box - pozostało dni",
        icon="mdi:air-filter",
        native_unit_of_measurement="d",
    ),
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
        description: SensorEntityDescription,
    ) -> None:
        super().__init__(coordinator)
        self.entity_description = description
        self._attr_unique_id = f"{DOMAIN}_{description.key}"

    @property
    def native_value(self) -> Any:
        return self.coordinator.data.get(self.entity_description.key)
