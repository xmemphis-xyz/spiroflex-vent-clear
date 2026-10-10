from __future__ import annotations

from datetime import timedelta
import logging
from typing import Any

from homeassistant.core import HomeAssistant
from homeassistant.helpers.update_coordinator import DataUpdateCoordinator, UpdateFailed

from .api import SpiroflexApi, SpiroflexApiError
from .const import DEFAULT_SCAN_INTERVAL, DOMAIN

_LOGGER = logging.getLogger(__name__)


class SpiroflexCoordinator(DataUpdateCoordinator[dict[str, Any]]):
    """Coordinate polling of the Spiroflex API."""

    def __init__(
        self,
        hass: HomeAssistant,
        api: SpiroflexApi,
        *,
        device_identifier: str | None = None,
    ) -> None:
        self.api = api
        self.device_identifier = device_identifier or api.base_url
        super().__init__(
            hass,
            _LOGGER,
            name=DOMAIN,
            update_interval=timedelta(seconds=DEFAULT_SCAN_INTERVAL),
        )

    async def _async_update_data(self) -> dict[str, Any]:
        try:
            return await self.api.async_get_status()
        except SpiroflexApiError as err:
            raise UpdateFailed(str(err)) from err
