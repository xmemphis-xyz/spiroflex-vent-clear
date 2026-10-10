from __future__ import annotations

from typing import Any

import voluptuous as vol
from homeassistant import config_entries
from homeassistant.const import CONF_HOST, CONF_PORT
from homeassistant.helpers.aiohttp_client import async_get_clientsession

from .api import SpiroflexApi, SpiroflexApiError
from .const import DEFAULT_PORT, DOMAIN


class SpiroflexConfigFlow(config_entries.ConfigFlow, domain=DOMAIN):
    """Handle a Spiroflex config flow."""

    VERSION = 1

    async def async_step_user(
        self, user_input: dict[str, Any] | None = None
    ) -> config_entries.ConfigFlowResult:
        """Handle the user step."""
        errors: dict[str, str] = {}

        if user_input is not None:
            host = user_input[CONF_HOST].strip()
            port = int(user_input[CONF_PORT])

            # A reconfigured entry keeps its legacy unique ID, so compare its
            # current endpoint as well when preventing duplicate entries.
            if self._endpoint_in_use(host, port):
                return self.async_abort(reason="already_configured")
            api = SpiroflexApi(
                async_get_clientsession(self.hass), host, port
            )
            try:
                await api.async_get_status()
            except SpiroflexApiError:
                errors["base"] = "cannot_connect"
            else:
                await self.async_set_unique_id(f"{host}:{port}")
                self._abort_if_unique_id_configured()
                return self.async_create_entry(
                    title="Spiroflex ecoVENT Simple",
                    data={CONF_HOST: host, CONF_PORT: port},
                )

        return self.async_show_form(
            step_id="user",
            data_schema=vol.Schema(
                {
                    vol.Required(CONF_HOST): str,
                    vol.Required(CONF_PORT, default=DEFAULT_PORT): vol.Coerce(int),
                }
            ),
            errors=errors,
        )

    def _endpoint_in_use(
        self, host: str, port: int, exclude_entry_id: str | None = None
    ) -> bool:
        """Do not point two config entries at the same backend."""
        return any(
            entry.entry_id != exclude_entry_id
            and str(entry.data.get(CONF_HOST, "")).strip().lower() == host.lower()
            and entry.data.get(CONF_PORT) == port
            for entry in self._async_current_entries()
        )

    async def async_step_reconfigure(
        self, user_input: dict[str, Any] | None = None
    ) -> config_entries.ConfigFlowResult:
        """Move an existing entry to another backend without replacing it."""
        entry = self._get_reconfigure_entry()
        errors: dict[str, str] = {}
        host = entry.data[CONF_HOST]
        port = entry.data[CONF_PORT]

        if user_input is not None:
            host = str(user_input[CONF_HOST]).strip()
            try:
                port = int(user_input[CONF_PORT])
            except (TypeError, ValueError):
                port = 0
            if not host or any(char in host for char in "/?#@") or any(
                char.isspace() for char in host
            ):
                errors[CONF_HOST] = "invalid_host"
            elif not 1 <= port <= 65535:
                errors[CONF_PORT] = "invalid_port"
            elif self._endpoint_in_use(host, port, entry.entry_id):
                errors["base"] = "already_configured"
            else:
                api = SpiroflexApi(
                    async_get_clientsession(self.hass), host, port
                )
                try:
                    await api.async_get_status()
                except SpiroflexApiError:
                    errors["base"] = "cannot_connect"
                else:
                    # DeviceInfo previously used base_url as its identifier.
                    # Freeze that original value before changing the endpoint.
                    # Preserve it on every subsequent reconfiguration as well.
                    device_identifier = entry.data.get(
                        "device_identifier",
                        f"http://{entry.data[CONF_HOST]}:{entry.data[CONF_PORT]}",
                    )
                    if entry.unique_id is not None:
                        await self.async_set_unique_id(entry.unique_id)
                        self._abort_if_unique_id_mismatch()
                    return self.async_update_reload_and_abort(
                        entry,
                        data_updates={
                            CONF_HOST: host,
                            CONF_PORT: port,
                            "device_identifier": device_identifier,
                        },
                        reason="reconfigure_successful",
                    )

        return self.async_show_form(
            step_id="reconfigure",
            data_schema=vol.Schema(
                {
                    vol.Required(CONF_HOST, default=host): str,
                    vol.Required(CONF_PORT, default=port): vol.All(
                        vol.Coerce(int), vol.Range(min=1, max=65535)
                    ),
                }
            ),
            errors=errors,
        )
