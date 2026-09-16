# Directory package recompilation

Upstream errorcheckandrundir compiles the same ordered package list once for
error checking and again for execution. Keep the remembered dependency map as
the prefix of the current pass. When a path repeats, retain only its preceding
packages, then record that phase's exact source list. This avoids importing a
package into itself, duplicate path registrations and stale main packages
which depend on the package being recompiled.

Do not skip compile phases or merge source lists. Changed source files remain
live inputs to the checker/compiler; conflicting definitions and duplicate
explicit package identities retain their existing errors. Native compiler
arguments, package directives, import order and linkname behavior are untouched.

Focused backend tests cover two full passes, multiple packages, source-list
replacement, unchanged-path source edits, caller slice ownership and root
isolation. Existing frontend duplicate-path refusal and initializer-order
controls remain required. Upstream target accounting belongs to the final
native/interpreted/compiled leaf rather than these focused checks.
