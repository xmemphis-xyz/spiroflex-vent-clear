#!/usr/bin/with-contenv bashio

if [ ! -f /data/options.json ]; then
  bashio::log.error "Home Assistant app configuration is missing."
  exit 1
fi

export SPIROFLEX_CONFIG_FILE="/data/options.json"

bashio::log.info "Starting Spiroflex ecoVENT Simple REST bridge"
exec /usr/bin/ventclear
