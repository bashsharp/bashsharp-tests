/*
 * Bash# mixed-lane lexer (Sprint 384, #1586): Bash 5.3 shell text with Go
 * regions. Go regions are lexed by the imported, patched grammars-v4 GoLexer
 * (go-target/GoLexer.g4, BSD-3-Clause; see grammar/NOTICE.md), whose
 * DEFAULT_MODE and NLSEMI modes this grammar extends. Shell text is lexed in
 * mode SH. Transcribed from the permissive sh/syntax implementation and the
 * sh/grammar delta, never from GNU Bash parse.y.
 *
 * The contextual-scanning primitives are enumerated in grammar/bashsharp/README.md
 * (P1-P12). Their state lives in bashsharpparser/lexer_state.go; the rule
 * predicates and actions below name them.
 */

lexer grammar BashSharpLexer;

import GoLexer;

// The contextual-scanning state (bashsharpparser/lexer_state.go).
@lexer::structmembers {
    st lexState
}

// Token types the lexer state machine assigns without a rule of their own
// (README P4-P7): a Go statement-block brace, the brace that closes a
// Go-region block or a brace group inside one, the end of a Go region, and
// a fence's closing line.
//
// A WORD whose exact spelling is a Bash reserved word gets that word's type
// (README P1); so does an assignment-shaped word, and, with Bash# on, a word
// spelled `agentic` or starting with "@". The spelling is all the lexer
// looks at: whether the word is reserved where it stands is decided by the
// parser's position (plan R6), and every parser rule that takes a word takes
// all of these types.
tokens {
    BLOCK_OPEN,
    RBRACE,
    REGION_END,
    FENCE_CLOSE,
    W_IF,
    W_THEN,
    W_ELIF,
    W_ELSE,
    W_FI,
    W_DO,
    W_DONE,
    W_CASE,
    W_ESAC,
    W_WHILE,
    W_UNTIL,
    W_FOR,
    W_SELECT,
    W_IN,
    W_FUNCTION,
    W_TIME,
    W_COPROC,
    W_BANG,
    W_LBRACE,
    W_RBRACE,
    W_DLBRACK,
    W_DRBRACK,
    W_AGENTIC,
    AT_WORD,
    ASSIGNMENT_WORD
}

// ---- Go regions (DEFAULT_MODE, extending the imported GoLexer) ----

// A decorator head (README P8): "@" opens only a decorator region.
AT: '@';

// A typed operand (README P10): a shell expansion used as a Go operand.
SHELL_OPERAND: '$' DOLLAR_BODY -> mode(NLSEMI);

// A shell comment inside a Go region (Bash# accepts "#" comments there).
SH_COMMENT_GO: '#' ~[\r\n]* -> channel(HIDDEN);

// A "{" is classified by the state machine: a statement block (BLOCK_OPEN,
// README P4) or a literal or type body brace (L_CURLY).
L_CURLY: '{' {l.goLCurly()};

mode NLSEMI;

SH_COMMENT_NLSEMI: '#' ~[\r\n]* -> channel(HIDDEN);

// Go semicolon insertion (README P6): at the outer level of a Go region the
// inserted semicolon ends the region (REGION_END) and lexing returns to the
// shell; elsewhere it is EOS. The rule precedes, and so shadows, the imported
// NLSEMI rule EOS, which matches the same input (ANTLR merges imported mode
// rules by position, not by name).
GO_EOS: ([\r\n]+ | ';' | '/*' .*? '*/' | EOF) {l.goEOS()};

// ---- shell text ----

mode SH;

SH_NEWLINE: '\n';

SH_BLANK: [ \t]+ -> channel(HIDDEN);

// Backslash-newline joins lines between words (inside a word, ESCAPE keeps it).
SH_CONT: '\\\n' -> channel(HIDDEN);

// Predicates in this mode come last in their rule: a predicate at the start
// of a rule would keep ANTLR from caching the mode's start state, and every
// token would then be simulated on the ATN. Each one reads the token start
// (TokenStartCharIndex), not the current index.

// "#" starts a comment only at the start of a word (README P2).
SH_COMMENT: '#' ~[\n]* {p.atWordStart()}? -> channel(HIDDEN);

// Bash#: "//" at the start of a word is a comment outside [[ ]] (S6 M14).
SH_SLASH_COMMENT: '//' ~[\n]* {p.slashCommentHere()}? -> channel(HIDDEN);

// A column-one source fence header (README P7). Its body is mode FENCE_BODY,
// up to and including the first line that is exactly the same tilde run.
FENCE_OPEN: '~~~' ~[\n]* '\n' {p.fenceHere()}? {l.fenceOpen()} -> pushMode(FENCE_BODY);

// The arithmetic command (( … )) at a command position or after `for` (README P3).
ARITH_CMD: '((' ARITH_BODY '))' {p.arithCmdHere()}?;

// A compound array assignment NAME=( … ) where an assignment is allowed (README P3).
ARRAY_ASSIGN: NAME ('[' ~[\]\n]* ']')? '+'? '=(' ARRAY_BODY ')' {p.arrayAssignHere()}?;

DSEMI_AMP: ';;&';
DSEMI: ';;';
SEMI_AMP: ';&';
SH_SEMI: ';';
AND_IF: '&&';
OR_IF: '||';
PIPE_AMP: '|&';
SH_PIPE: '|';
AMP_DGREAT: '&>>';
AMP_GREAT: '&>';
SH_AMP: '&';
SH_LPAREN: '(';
SH_RPAREN: ')';
DLESSDASH: '<<-';
TLESS: '<<<';
DLESS: '<<';
LESSAND: '<&';
LESSGREAT: '<>';
DGREAT: '>>';
GREATAND: '>&';
CLOBBER: '>|';
SH_LESS: '<';
SH_GREAT: '>';

// A file-descriptor number directly before a redirection operator.
IO_NUMBER: [0-9]+ {p.redirectFollows()}?;

// A shell word (README P1): the maximal run of word parts. Its spelling
// alone selects a reserved-word type (see tokens); the parser decides by
// position whether that word is reserved (R6).
// A backslash with nothing after it (end of input) is a literal word part.
WORD: (WORD_PART+ '\\'? | '\\') {l.shWord()};

fragment WORD_PART: WORD_CHAR | ESCAPE | SQ_STRING | DQ_STRING | DOLLAR | BACKQUOTE | PROC_SUBST | EXTGLOB;

fragment WORD_CHAR: ~[ \t\n;&|()<>'"`$\\];

// A backslash quotes the next character.
fragment ESCAPE: '\\' .;

fragment SQ_STRING: '\'' ~'\''* '\'';

fragment DQ_STRING: '"' (~["\\$`] | ESCAPE | DQ_DOLLAR | BACKQUOTE)* '"';

fragment BACKQUOTE: '`' (~[`\\] | ESCAPE)* '`';

fragment PROC_SUBST: [<>] '(' SUBST_BODY ')';

// An extended glob ?( ) *( ) +( ) @( ) !( ), parsed as sh/syntax does.
fragment EXTGLOB: [?*+@!] '(' EXTGLOB_BODY ')';

fragment EXTGLOB_BODY: (~[()'"\\$`\n] | ESCAPE | SQ_STRING | DQ_STRING | DOLLAR | BACKQUOTE | '(' EXTGLOB_BODY ')')*;

// "$" begins an expansion; it is a literal "$" only before a character that
// cannot begin one (README P1), so an unterminated "${" is an error.
fragment DOLLAR: '$' (DOLLAR_BODY | '\'' (~['\\] | ESCAPE)* '\'' | DQ_STRING | {p.dollarLiteral(false)}?);

fragment DQ_DOLLAR: '$' (DOLLAR_BODY | {p.dollarLiteral(true)}?);

fragment DOLLAR_BODY
    : NAME
    | [0-9@*#?$!-]
    | '{' BRACE_BODY '}'
    | '((' ARITH_BODY '))'
    | '(' SUBST_BODY ')'
    | '[' (~[\]'"\\$`] | ESCAPE | SQ_STRING | DQ_STRING | DOLLAR | BACKQUOTE)* ']'
    ;

fragment NAME: [a-zA-Z_] [a-zA-Z_0-9]*;

fragment BRACE_BODY: (~[}'"\\$`] | ESCAPE | SQ_STRING | DQ_STRING | DOLLAR | BACKQUOTE)*;

fragment ARITH_BODY: (~[()'"\\$`] | ESCAPE | SQ_STRING | DQ_STRING | DOLLAR | BACKQUOTE | '(' ARITH_BODY ')')*;

// A command substitution body is delimited here and parsed again with the
// parser entry `script` (README P11).
fragment SUBST_BODY: (~[()'"\\$`] | ESCAPE | SQ_STRING | DQ_STRING | DOLLAR | BACKQUOTE | '(' SUBST_BODY ')')*;

fragment ARRAY_BODY: (~[()'"\\$`] | ESCAPE | SQ_STRING | DQ_STRING | DOLLAR | BACKQUOTE)*;

// ---- the right operand of =~ inside [[ ]] (README P3) ----

// Parentheses group the regular expression and may hold blanks and
// operator characters; the operand ends at a blank outside them.
mode COND_REGEX;

REGEX_BLANK: [ \t]+ -> channel(HIDDEN);

REGEX_WORD: REGEX_PART+ -> type(WORD), popMode;

fragment REGEX_PART: ~[ \t\n()'"\\$`] | ESCAPE | SQ_STRING | DQ_STRING | DOLLAR | BACKQUOTE | '(' REGEX_PAREN* ')';

fragment REGEX_PAREN: ~[()'"\\$`] | ESCAPE | SQ_STRING | DQ_STRING | DOLLAR | BACKQUOTE | '(' REGEX_PAREN* ')';

// ---- here-document bodies (README P9) ----

mode HEREDOC;

HEREDOC_LINE: ~[\n]* ('\n' | EOF) {l.heredocLine()} -> channel(HIDDEN);

// ---- fence bodies (README P7) ----

mode FENCE_BODY;

FENCE_LINE: ~[\n]* ('\n' | EOF) {l.fenceLine()};
