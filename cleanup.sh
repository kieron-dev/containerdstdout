#!/bin/bash

set -euo pipefail

ctr="ctr -n example"

$ctr snapshot rm busybox-snapshot || true
$ctr t kill -s 9 example || true
$ctr t delete example || true
$ctr c delete example || true
