#!/bin/bash
cat $1| jq '{ server: .data.tls.result.handshake_log.server_key_exchange.ecdh_params, client: .data.tls.result.handshake_log.client_key_exchange.ecdh_params}' | grep -v "null"
