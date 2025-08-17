#!/usr/bin/env bash

set -eu -o pipefail

# This script sets up a GitHub Action runner environment to exercise different
# conditions in tests for the installation script.

declare -r IS_LINUX="${IS_LINUX:?missing IS_LINUX}"
declare -r IS_MACOS="${IS_MACOS:?missing IS_MACOS}"
declare -r WANT_GH="${WANT_GH:-''}"
declare -r WANT_CURL="${WANT_CURL:-''}"
declare -r WANT_JQ="${WANT_JQ:-''}"

echo >&2 "IS_LINUX='${IS_LINUX}' IS_MACOS='${IS_MACOS}' WANT_GH='${WANT_GH}' WANT_CURL='${WANT_CURL}' WANT_JQ='${WANT_JQ}'"

function has_command() {
	command -v "${1:?missing cmd}" >/dev/null 2>&1
}

if [[ ${IS_LINUX} == 'true' ]]; then
	if [[ ${WANT_GH} == 'true' ]]; then
		has_command gh || (sudo apt-get update && sudo apt-get install gh)
	elif [[ ${WANT_GH} == 'false' ]]; then
		has_command gh && sudo apt-get remove gh
	fi

	if [[ ${WANT_CURL} == 'true' ]]; then
		has_command curl || (sudo apt-get update && sudo apt-get install curl)
	elif [[ ${WANT_CURL} == 'false' ]]; then
		has_command curl && sudo apt-get remove curl
	fi

	if [[ ${WANT_JQ} == 'true' ]]; then
		has_command jq || (sudo apt-get update && sudo apt-get install jq)
	elif [[ ${WANT_JQ} == 'false' ]]; then
		has_command jq && sudo apt-get remove jq
	fi

	exit 0
fi

if [[ ${IS_MACOS} == 'true' ]]; then
	if [[ ${WANT_GH} == 'true' ]]; then
		has_command gh || brew install gh
	elif [[ ${WANT_GH} == 'false' ]]; then
		has_command gh && brew remove --force gh
	fi

	if [[ ${WANT_CURL} == 'true' ]]; then
		has_command curl || brew install curl
	elif [[ ${WANT_CURL} == 'false' ]]; then
		has_command curl && brew remove --force curl
	fi

	if [[ ${WANT_JQ} == 'true' ]]; then
		has_command jq || brew install jq
	elif [[ ${WANT_JQ} == 'false' ]]; then
		has_command jq && brew remove --force jq
	fi

	exit 0
fi
