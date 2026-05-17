#!/usr/bin/env bash
set -eux pipefail

bash run-rscore.sh

TESTDATA_DIR="public/testdata-2"
RESULTS_DIR="public/testresults"
BIN="./bin/rscore"

mkdir -p "$RESULTS_DIR"

games=()
for f in "$TESTDATA_DIR"/*_pre_nodes.csv; do
    filename="$(basename "$f")"
    game="${filename%_pre_nodes.csv}"
    games+=("$game")
done

for game in "${games[@]}"; do
    pre_nodes="$TESTDATA_DIR/${game}_pre_nodes.csv"
    pre_edges="$TESTDATA_DIR/${game}_pre_edges.csv"
    post_nodes="$TESTDATA_DIR/${game}_post_nodes.csv"
    post_edges="$TESTDATA_DIR/${game}_post_edges.csv"
    output="$RESULTS_DIR/${game}_score.txt"

    echo "=== ${game} PRE ===" > "$output"
    "$BIN" score -n "$pre_nodes" -e "$pre_edges" >> "$output"

    echo "" >> "$output"
    echo "=== ${game} POST ===" >> "$output"
    "$BIN" score -n "$post_nodes" -e "$post_edges" >> "$output"

    echo "Scored $game -> $output"
done