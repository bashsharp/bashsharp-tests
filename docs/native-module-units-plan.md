# Preserve compiled module package identity — Sprint 200, story 421

The module recipe for issue20014 already supplies a complete dependency map.
Its compiled path flattened that map, renaming an imported type into main and
changing linker field-tracking output. Emit each listed package through
Bashy's explicit native-unit interface, in dependency order, into a temporary
module using its original module/import paths. Build the original main-package
directory with the exact recipe flags and inherited environment, then execute
with unchanged program arguments. Record every generated unit and source map.
No invented compiler -D flag or upstream recipe change is involved.

Focused tests cover ordered checker dependencies, a nested main package,
original module identity, duplicate units and escaping paths. Bashy's companion
regression exercises actual native field tracking and initialization. Preserve
the existing interpreted route and direct directory compiler phases. Final
acceptance is the one authenticated Sprint 200 upstream leaf.
