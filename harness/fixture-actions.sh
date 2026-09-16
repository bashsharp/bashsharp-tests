# Fixture action and diagnostic annotations use the source's comment syntax.
# Go uses //; Bash++ shell sources use #.
action_of() {
  local file="$1" a
  a="$(sed -n '1,10p' "$file" | grep -m1 -oE '^(//|#) *(run|errorcheck|build)' | sed -E 's@^(//|#) *@@')"
  [ -n "$a" ] || a=run
  printf '%s' "$a"
}

check_error_annotations() {
  local file="$1" errf="$2" pat missing=0
  while IFS= read -r pat; do
    [ -n "$pat" ] || continue
    grep -qE -- "$pat" "$errf" || { echo "      missing diagnostic matching: $pat"; missing=1; }
  done < <(grep -oE '(//|#) *ERROR +"[^"]*"' "$file" | sed -E 's@(//|#) *ERROR +"(.*)"@\2@')
  return $missing
}
