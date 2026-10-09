from __future__ import annotations

from typing import Any

from aiohttp import ClientError, ClientSession


class SpiroflexApiError(Exception):
    """Raised when the Spiroflex API cannot be used."""


class SpiroflexApi:
    """Small async client for the local ventclear REST API."""

    def __init__(self, session: ClientSession, host: str, port: int) -> None:
        self._session = session
        self._base_url = f"http://{host}:{port}"

    async def async_get_status(self) -> dict[str, Any]:
        return await self._request("GET", "/api/vent/status")

    async def async_set_power(self, state: str) -> None:
        await self._request("POST", f"/api/vent/power/{state}")

    async def async_set_level(self, level: int) -> None:
        await self._request("POST", f"/api/vent/level/{level}")

    async def async_set_boost(self, boost: int, on: bool) -> None:
        if type(boost) is not int or boost not in (1, 2):
            raise ValueError("BOOST must be 1 or 2")
        if not isinstance(on, bool):
            raise ValueError("BOOST state must be a boolean")
        # Keep the original start URL compatible with existing button actions.
        path = f"/api/vent/boost/{boost}" + ("" if on else "/off")
        data = await self._request("POST", path, timeout=45)
        if data.get("ok") is not True:
            raise SpiroflexApiError("BOOST command was not confirmed by the backend")

    async def async_start_boost(self, boost: int) -> None:
        await self.async_set_boost(boost, True)

    async def async_stop_boost(self, boost: int) -> None:
        await self.async_set_boost(boost, False)

    async def async_pause(self) -> None:
        await self._request("POST", "/api/vent/pause")

    async def async_set_mode(self, mode: str) -> None:
        await self._request("POST", f"/api/vent/mode/{mode}")

    async def _request(
        self, method: str, path: str, *, timeout: float = 10
    ) -> dict[str, Any]:
        try:
            async with self._session.request(
                method,
                f"{self._base_url}{path}",
                timeout=timeout,
            ) as response:
                text = await response.text()
                if response.status >= 400:
                    raise SpiroflexApiError(f"HTTP {response.status}: {text[:300]}")
                if not text:
                    return {}
                try:
                    data = await response.json(content_type=None)
                except ValueError as err:
                    raise SpiroflexApiError("Invalid JSON response") from err
                if not isinstance(data, dict):
                    raise SpiroflexApiError("Expected a JSON object from Spiroflex API")
                if data.get("ok") is False:
                    raise SpiroflexApiError(data.get("error", "API request failed"))
                return data
        except (ClientError, TimeoutError) as err:
            raise SpiroflexApiError(str(err)) from err

    @property
    def base_url(self) -> str:
        return self._base_url
