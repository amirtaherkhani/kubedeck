# macOS node exporter

This installs Prometheus node exporter as a per-user macOS LaunchAgent. It
listens on port `9101` because Rancher Desktop already forwards port `9100` to
the Linux `lima-rancher-desktop` VM.

Install or upgrade the pinned exporter:

```bash
./host/macos/node-exporter/install.sh
```

The LaunchAgent starts at login and is restarted if it exits. Its files are:

- Binary: `~/.local/opt/node_exporter-1.11.1/node_exporter`
- Symlink: `~/.local/bin/node_exporter`
- LaunchAgent: `~/Library/LaunchAgents/com.prometheus.node-exporter.plist`
- Logs: `~/Library/Logs/node-exporter.log` and
  `~/Library/Logs/node-exporter.err.log`

The thermal collector is disabled because it reports no CPU power status on
this Mac. The dashboard uses the CPU, load, memory, disk, filesystem, and
network collectors.

Verify native Darwin metrics:

```bash
curl -fsS http://127.0.0.1:9101/metrics | \
  grep -E 'node_exporter_build_info|node_uname_info'
```

Remove the service without deleting downloaded binaries:

```bash
launchctl bootout "gui/$(id -u)" \
  "$HOME/Library/LaunchAgents/com.prometheus.node-exporter.plist"
rm "$HOME/Library/LaunchAgents/com.prometheus.node-exporter.plist"
```
