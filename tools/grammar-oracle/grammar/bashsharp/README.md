# Bash# mixed-lane grammar (Sprint 384, #1586, checkpoint 2)

`BashSharpLexer.g4` and `BashSharpParser.g4` are the combined grammar of the
mixed lane: Bash 5.3 shell text, the Bash# delta (`sh/grammar/delta.ebnf`)
and Go regions. They are separate ANTLR lexer and parser grammars. They
import the patched Go grammars in `../go-target/` (patches 0001-0003; see
`../NOTICE.md`), so the Go regions use the same productions as the unit
lane. `gen/generate.sh` generates package `bashsharpparser`.

They were transcribed from the permissive `sh/syntax` implementation (the
start-site decision table in `bashpp_startsites.go`, the Go-region dispatch in
`parser.go` and `bashpp_*.go`) and from the delta. GNU Bash `parse.y` was
never copied.

## Entry points

| Call | Entry rule | Dialect |
|---|---|---|
| `oracle.Recognize("mixed", "bash", src)` | `script` | Bash# off: `sh/syntax` `LangBash`, as `bash -n` |
| `oracle.Recognize("mixed", "bashsharp", src)` | `script` | Bash# on: `sh/syntax` `LangBashPP` |

`script` also parses the body of every command substitution (P11). The result
is the verdict, the diagnostics (CONTRACT.md positions) and the boundary,
which lists the extension sites in source order. Each site is a kind and a
`[start,end)` byte span. The kinds are the reference adapter's: `fence`,
`embed-line`, `decorator`, `at-function`, `agentic-block`,
`agentic-shell-func`, `agentic-typed-func`, `agentic-typed-method`,
`typed-func` and `typed-method`. With Bash# off there are no sites.

## Generated and hand-written files

`bashsharpparser/bashsharp_lexer.go`, `bashsharp_parser.go` and
`go_parser_base.go` are generated, and `bashsharpparser/generated.sha256`
hashes them together with these two grammars. The contextual-scanning state
and the parser predicates are hand-written Go in the same package:
`lexer_state.go` and `parser_support.go`. The grammars name their predicates
and actions. Those files import only the ANTLR runtime and the standard
library.

## Contextual scanning primitives

An ANTLR lexer cannot see parser state (plan R6). Each place where the token
stream depends on context is one of these enumerated primitives:

| | Primitive | Mechanism |
|---|---|---|
| P0 | Aliases | An empty alias environment: no alias is expanded. A static recognizer has no alias table, and the reference adapter uses none either. |
| P1 | Words | `WORD` is the maximal run of word parts: unquoted characters, `\x`, `'…'`, `"…"`, `$'…'`, `$"…"`, `$name`, `${…}`, `$((…))`, `$(…)`, `$[…]`, backquotes, `<(…)`/`>(…)` and extended globs. Quotes and nested substitutions are balanced recursive lexical fragments. A `$` is a literal only before a character that cannot begin an expansion, so an unterminated `${` is an error. The lexer types a word by its **spelling** only. `if`, `then`, `{`, `}`, `[[`, `!`, … get `W_IF`, `W_THEN`, …. With Bash# on, `agentic` gets `W_AGENTIC` and `@…` gets `AT_WORD`. A `NAME=…` word gets `ASSIGNMENT_WORD`. A backslash-newline inside a word is removed before the spelling is matched. Where an assignment may stand (a command position or declaration-builtin arguments), a word that opens with `NAME[` reads its subscript as a matched pair through the matching `]`, across blanks, operators and newlines (quotes and `$(…)` nest), and then continues the word; end of input inside the pair is an error (Bash 5.3, `sh/syntax`). A `#` at the start of a word always begins a comment, even where a longer `WORD` match would run on through a quote. |
| P2 | Comments | `#` starts a comment only at the start of a word. With Bash# on, `//` at the start of a word outside `[[ ]]` is a comment as well (S6 M14). |
| P3 | Shell positions | The lexer tracks the command position from the token stream: after a separator, an operator or a keyword prefix, and after a pattern's `)`. It also tracks case-pattern positions, for/select word lists, redirection targets, here-document delimiters, `[[ … ]]` (with mode `COND_REGEX` for the operand of `=~`) and declaration builtins. The tracking is used **only** to lex `((…))` and `NAME=(…)`, to decide fence headers (P7), Go-region entry (P5) and the `}` closer of a Go block (P5), and to enter `COND_REGEX`. It never decides a reserved word. |
| P4 | Go statement blocks | In a Go region, a `{` that opens a function body becomes `BLOCK_OPEN`: the lexer tracks a function signature's parameter lists. So does the block of an `if`/`for`/`switch`/`select`/`else` header at the region's outer level. The block's body is shell text in mode `SH`. Any other `{` (a composite literal, or a struct, interface or enum body) stays `L_CURLY`. In a header, a brace after an array, slice, map or struct literal type is a literal, as the spec's composite-literal parsing ambiguity says. After a complete signature, newlines and comment lines before the `{` are hidden: this is the delta's `header_end`. |
| P5 | Go-region entry and block closers | With Bash# on, at a command position, the start-site decision table opens a Go region (mode `DEFAULT_MODE`, the imported GoLexer) for the statements listed below. Inside a Go block, outside any subshell, a standalone `}` word is `RBRACE`, and it closes the innermost brace even in an argument position, as `sh/syntax` does. A `}` glued to the token before it (`"x"}`) is not standalone and closes nothing. |
| P6 | Go-region end | An inserted semicolon at the region's outer level (a newline after a statement-ending token, `;`, or end of input) is `REGION_END`, and lexing returns to the shell. Elsewhere it is the GoLexer's `EOS`. A region also ends, without consuming input, before a `}` that closes the enclosing Go block, before a `)` that closes an enclosing subshell, and before `;;`/`;&`. A case clause head ends at its `:`. |
| P7 | Fences | A column-one header at a command position, validated as `sh/syntax` `recognizeSourceFence` does (`~~~`+ language, optional `as ALIAS`, optional `!RUNNER`, nothing else). Mode `FENCE_BODY` then consumes lines up to and including the first line that is exactly the opener's tilde run. The body is opaque, except that a closed fence without `as ALIAS` declares the functions its language's declaration lines name (`sh/syntax` `bashppRegisterSourceBlockFuncs`), so a later `f()` is a call (P5). |
| P8 | Decorators | A column-one `@name(` opens a decorator region (Go arguments, keyword arguments allowed). The exception is `@name()` followed, possibly after comment lines, by a compound command: that is Bash's function named `@name` (`at_function`). |
| P9 | Here-documents | `<<`/`<<-` queue their delimiter, with quotes removed. After the next shell newline, mode `HEREDOC` consumes the body lines up to each delimiter in turn; `<<-` strips leading tabs. A here-document whose delimiter line never appears is an error at end of input, as in `sh/syntax` (GNU Bash 5.3 only warns). |
| P10 | Typed operands | Inside a Go region, `$name`, `${…}`, `$(…)` and `$((…))` are one `SHELL_OPERAND` token, and Go `operand` admits it. `#` comments are allowed in a Go region. |
| P11 | Substitution re-entry | The body of every `$(…)`, backquote (unescaped first), `<(…)` and `>(…)` is parsed again with entry `script` in the same dialect, with its positions mapped back. A fence inside `$(…)` is therefore still a fence. Arithmetic bodies are not command text. |
| P12 | Embed lines | `embed FORMAT "./path"\|"../path" [as ALIAS] [!RUNNER]` ending its command is an `embed_line`. A left-edge parser predicate checks the token shape. Whether the target file exists is not grammar (the reference phase is `context`). |

### Reserved words (plan R6)

A reserved word is recognized by its **position in the parser grammar**,
never by the lexer. `cmd_word` (a command name) takes `WORD`, `AT_WORD` and
`W_IN`, but no other keyword type. `word` (an argument) takes every word type.
A compound command's closer (`then`, `fi`, `do`, `done`, `esac`, `}`) must
follow a separator or the end of a compound command (`term_list`), because
only a command position admits a reserved word. So `echo if then fi` is one
command, while `then echo` and `{ echo a }` are errors. Keyword predicates on
`WORD` text would be invisible to ANTLR's adaptive prediction past the first
lookahead token. Each `WORD` could then be any keyword, and prediction would
blow up. That is why spelling types are used instead. The only remaining
parser predicates are left-edge checks: `embedLineHere`, the `[[ ]]`
operator checks, and `identIs("enum")`.

### Bash# start sites (P5)

Transcribed from `RecognizeStartSite` and the `parser.go` dispatch:

- `var`/`const` + a non-keyword identifier, and `const (`. `type` + a
  non-keyword identifier only with a complete type shape: an alias, a
  `struct`/`interface`/`enum` body whose `}` closes before end of input, a
  type with parentheses or braces, or one type word ending the command.
  `type foo` and `type foo bar baz` stay the shell builtin.
- `import` + anything (only an interpreted-string path is valid; S6 M16), and
  `package` (always an error; S6 M15).
- `func NAME(`, `func NAME[…](` (type parameters without a shell operator
  character; `[T int | float64]` is the S6 M11 gap) and `func (` with a
  non-empty list. This covers typed functions, methods and an immediately
  called literal.
- `go` + a literal or a selector call.
- A short declaration `a, b := …` with a right-hand side, the `:=` followed
  by a blank (`ech:=o` and `x :=1` are shell words).
- A call: `f(…)` with arguments or a dotted callee. A plain `f()` qualifies
  only for a declared typed function or a predeclared builtin followed by a
  statement end (`sh/syntax` `bashppCallable`).
- A send `ch <- v`, a receive `<-ch`, and `*p = …`. At top level, an
  assignment `a = f(…)` with a call on the right.
- Inside a Go block only: `return`, `defer` + literal or call, any assignment
  or `++`/`--`, Go `if` (a standalone `{` and no `then` on the line), Go
  `for` (not `for NAME in`/`for NAME;`/`for ((`), `switch … {`,
  `select {`, and in a switch or select body `case …:`/`default:`.

The shell escapes `command var …` and `"var" …` are not command positions for
these sites. A non-command position, a quoted keyword and an indented
decorator stay shell.

## Coverage map (plan feature inventory)

| Surface | Productions | Evidence |
|---|---|---|
| Bash 5.3 commands, lists, pipelines, `time`, `!`, coproc, functions, compound commands (`{}`, `()`, if, while, until, for, `for ((…))`, select, case with `;;`/`;&`/`;;&`, `((…))`, `[[ … ]]`), redirections, here-documents, words, quoting, expansions and substitutions | `script` … `cond_command`, P1-P3, P9, P11 | 250/250 startsite rows (bash dialect); `TestMixedPrimitives` |
| Mixed Go declarations, statements and expressions | `go_statement` (imported Go rules), P4-P6, P10 | S6 M01-M17; delta fixtures |
| Keyword arguments and defaults | `arguments`/`argument_list`/`keyword_argument`; `parameterDecl` | `tests/decorators/arguments.bpp` |
| Enums | `typeDef`/`enum_type` | `TestMixedPrimitives` |
| Decorators and `agentic` | `decorated_decl`, `decorator_line`, `decorator`, `at_function`, `agentic_*` | `tests/decorators/*.bpp`, delta fixtures |
| Fences and embeds | `fence` (P7), `embed_line` (P12) | `delta/accept/boundaries.bsh`, fence first-closer rejects |

`delta_mirror_test.go` holds every delta-owned production to `delta.ebnf`
(plan R5). Each EBNF production is mirrored by a same-named rule, implemented
by named lexical primitives, or replaced by named base-shell or imported Go
productions. Where the grammar is deliberately wider, the widening is
documented: a function body is any compound command, not only a block.

## Known limits (open, not waived)

- Arithmetic (`$((…))`, `((…))`, `$[…]`, `let`) and the inside of `${…}`
  are balanced but opaque. GNU Bash checks them only when expanding.
  `sh/syntax` parses them and refuses more (for example `$((a b))`, `${1a}`,
  zsh/mksh-only forms).
- A `$(…)` body with an unbalanced `)` (a case pattern) or a here-document
  holding `)` is not delimited correctly. Expansions inside here-document
  bodies are not parsed.
- The island lane (Go payloads of fences and embeds) and the bounded mutation
  set belong to checkpoints 3-4.
