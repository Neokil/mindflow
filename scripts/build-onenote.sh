#!/usr/bin/env bash
# Compile the native format adapter; no .NET runtime is needed by the server.
set -euo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
case "$(uname -s)" in
  Darwin) platform=osx ;;
  Linux) platform=linux ;;
  *) printf 'Unsupported build platform\n' >&2; exit 1 ;;
esac
case "$(uname -m)" in
  arm64|aarch64) architecture=arm64 ;;
  x86_64) architecture=x64 ;;
  *) printf 'Unsupported architecture\n' >&2; exit 1 ;;
esac
export DOTNET_CLI_HOME="$root/.cache/dotnet-home"
export NUGET_PACKAGES="$root/.cache/nuget"
export DOTNET_GENERATE_ASPNET_CERTIFICATE=false
# A locked restore prevents silent transitive dependency changes.
dotnet publish internal/workspace/onenote/codec/OneNoteCodec.csproj \
  --configuration Release --runtime "$platform-$architecture" \
  --output "$root/.cache/e2e" -p:RestoreLockedMode=true
