from __future__ import annotations

from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant
from homeassistant.helpers.aiohttp_client import async_get_clientsession

from .api import SpiroflexApi
from .const import CONF_HOST, CONF_PORT, PLATFORMS
from .coordinator import SpiroflexCoordinator


async def async_setup_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Set up Spiroflex ecoVENT Simple from a config entry."""
    api = SpiroflexApi(
        async_get_clientsession(hass),
        entry.data[CONF_HOST],
        entry.data[CONF_PORT],
    )
    coordinator = SpiroflexCoordinator(
        hass, api, device_identifier=entry.data.get("device_identifier")
    )
    await coordinator.async_config_entry_first_refresh()

    entry.runtime_data = coordinator
    await hass.config_entries.async_forward_entry_setups(entry, PLATFORMS)
    return True


async def async_unload_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Unload a Spiroflex config entry."""
    return await hass.config_entries.async_unload_platforms(entry, PLATFORMS)
