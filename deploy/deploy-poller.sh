#!/bin/bash
# Install this script on lab-server at /usr/local/bin/deploy-poller (owned root, mode 0755).
# Configure authorized_keys for the deploy user with:
#   command="/usr/local/bin/deploy-poller $SSH_ORIGINAL_COMMAND",no-port-forwarding,no-X11-forwarding,no-agent-forwarding <pubkey>
#
# The deploy workflow calls: ssh deploy@lab-server "$VERSION" < ./poller
# $SSH_ORIGINAL_COMMAND becomes $VERSION; the binary arrives on stdin.
set -euo pipefail

VERSION="${1:?usage: deploy-poller <version>}"
APP=poller
SRV="/srv/$APP"
RELEASE_DIR="$SRV/releases/$VERSION"
HEALTH_PORT="${HEALTH_PORT:-8080}"
KEEP_RELEASES=3

# Write binary from stdin to a temp file, then install atomically
mkdir -p "$RELEASE_DIR"
TMPFILE=$(mktemp "$RELEASE_DIR/.tmp.XXXXXX")
trap 'rm -f "$TMPFILE"' EXIT
cat > "$TMPFILE"
install -m 0755 "$TMPFILE" "$RELEASE_DIR/$APP"

# Swap symlink atomically
ln -sfn "$RELEASE_DIR" "$SRV/current"

# Restart service
sudo /bin/systemctl restart "$APP"

# Poll health endpoint
for i in $(seq 1 10); do
  if curl -sf "http://localhost:${HEALTH_PORT}/health" > /dev/null 2>&1; then
    echo "[$APP $VERSION] health check passed (attempt $i)"
    # Prune releases beyond the last $KEEP_RELEASES
    ls -1dt "$SRV/releases"/*/ 2>/dev/null \
      | tail -n +"$((KEEP_RELEASES + 1))" \
      | xargs -r rm -rf
    exit 0
  fi
  sleep 3
done

echo "[$APP $VERSION] health check failed after 30s — service may be unhealthy" >&2
exit 1
