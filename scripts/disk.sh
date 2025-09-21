#!/usr/bin/env bash
set -euo pipefail

# disk.sh: Manage a set of loopback XFS "disks" for local testing.
#
# Usage:
#   disk.sh prepare <basedir> <count> [sizeMB]
#     - Creates <count> loopback files (default size 2000 MB) under <basedir>/disks
#       and mounts them to <basedir>/mnt/disk1 ... diskN (XFS).
#   disk.sh cleanup <basedir>
#     - Unmounts and detaches all loop devices created under <basedir> and removes the directory.
#
# The script stores state in <basedir>/.loopmeta listing: index,loopdev,imagefile,mountpoint
# so cleanup can deterministically unmount/detach.
#
# Requirements:
#   - sudo privileges (losetup, mkfs.xfs, mount, umount)
#   - xfsprogs installed
#
# Notes:
#   - Existing basedir will be reused unless conflicting mounts exist.
#   - Idempotent prepare: will skip disks already present in metadata.

SCRIPT_NAME="$(basename "$0")"

die() { echo "[ERR] $*" >&2; exit 1; }
info() { echo "[INFO] $*" >&2; }
warn() { echo "[WARN] $*" >&2; }

require_cmd() {
	command -v "$1" >/dev/null 2>&1 || die "command '$1' not found"
}

prepare() {
	local basedir="$1"; local count="$2"; local sizeMB="$3"
	[[ -z "$basedir" || -z "$count" ]] && die "prepare requires <basedir> <count> [sizeMB]"
	[[ ! "$count" =~ ^[0-9]+$ ]] && die "count must be integer"
	[[ -z "$sizeMB" ]] && sizeMB=2000
	[[ ! "$sizeMB" =~ ^[0-9]+$ ]] && die "sizeMB must be integer"

	require_cmd losetup; require_cmd mkfs.xfs; require_cmd dd; require_cmd mount; require_cmd umount
	sudo modprobe loop >/dev/null 2>&1 || true

	mkdir -p "$basedir/disks" "$basedir/mnt"
	local meta="$basedir/.loopmeta"
	touch "$meta"

	# Build an associative map of existing indices to avoid recreation
	declare -A existing
	while IFS=',' read -r idx loopdev img mp; do
		[[ -z "$idx" ]] && continue
		existing[$idx]="$loopdev,$img,$mp"
	done < "$meta"

	for i in $(seq 1 "$count"); do
		if [[ -n "${existing[$i]:-}" ]]; then
			info "disk $i already exists, skipping"
			continue
		fi
		local img="$basedir/disks/img.$i"
		local mp="$basedir/mnt/disk$i"
		info "Creating image $img (${sizeMB}MB)"
		dd if=/dev/zero of="$img" bs=1M count="$sizeMB" status=none || die "dd failed for $img"
		local loopdev
		loopdev=$(sudo losetup --find --show "$img") || die "losetup failed for $img"
		info "Formatting $loopdev as XFS"
		sudo mkfs.xfs -q "$loopdev" || die "mkfs.xfs failed for $loopdev"
		mkdir -p "$mp"
		sudo mount "$loopdev" "$mp" || die "mount failed for $loopdev -> $mp"
		sudo chown "$(id -u):$(id -g)" "$mp" || true
		echo "$i,$loopdev,$img,$mp" >> "$meta"
		info "Mounted $loopdev at $mp"
	done

	info "Prepare complete. Metadata: $meta"
}

cleanup() {
	local basedir="$1"
	[[ -z "$basedir" ]] && die "cleanup requires <basedir>"
	local meta="$basedir/.loopmeta"
	[[ -f "$meta" ]] || { warn "No metadata file ($meta). Attempting best-effort cleanup."; }

	if [[ -f "$meta" ]]; then
		# Process in reverse index order to be neat
		tac "$meta" | while IFS=',' read -r idx loopdev img mp; do
			[[ -z "$idx" ]] && continue
			if mountpoint -q "$mp"; then
				info "Unmounting $mp"
				sudo umount "$mp" || warn "Failed to unmount $mp"
			fi
			if [[ -n "$loopdev" && -e "$loopdev" ]]; then
				info "Detaching $loopdev"
				sudo losetup -d "$loopdev" || warn "Failed to detach $loopdev"
			fi
			if [[ -n "$img" && -f "$img" ]]; then
				info "Removing image $img"
				rm -f "$img" || warn "Failed to remove $img"
			fi
			if [[ -d "$mp" ]]; then
				rmdir "$mp" 2>/dev/null || true
			fi
		done
	fi

	rm -f "$meta" 2>/dev/null || true
	# Remove parent dirs if empty
	rmdir "$basedir/mnt" 2>/dev/null || true
	rmdir "$basedir/disks" 2>/dev/null || true
	rmdir "$basedir" 2>/dev/null || true
	info "Cleanup complete for $basedir"
}

usage() {
	cat <<EOF
Usage:
	$SCRIPT_NAME prepare <basedir> <count> [sizeMB]
	$SCRIPT_NAME cleanup <basedir>

Examples:
	$SCRIPT_NAME prepare /tmp/testenv 6 500   # 6 disks of 500MB
	$SCRIPT_NAME cleanup /tmp/testenv
EOF
}

main() {
	local cmd="${1:-}"; shift || true
	case "$cmd" in
		prepare)
			prepare "${1:-}" "${2:-}" "${3:-}" ;;
		cleanup)
			cleanup "${1:-}" ;;
		""|-h|--help|help)
			usage ;;
		*)
			die "Unknown command: $cmd (try --help)" ;;
	esac
}

main "$@"

