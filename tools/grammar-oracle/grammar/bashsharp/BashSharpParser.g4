/*
 * Bash# mixed-lane parser (Sprint 384, #1586): Bash 5.3 commands with the
 * Bash# delta and Go regions. The Go productions are imported from the
 * patched grammars-v4 GoParser (go-target/GoParser.g4, BSD-3-Clause; see
 * grammar/NOTICE.md); rules defined here override the imported rules of the
 * same name. Transcribed from the permissive sh/syntax implementation and the
 * sh/grammar delta (delta.ebnf), never from GNU Bash parse.y.
 *
 * Rule names that equal a delta.ebnf production name are delta-owned; the
 * mirror test (delta_mirror_test.go, plan R5) holds them to delta.ebnf.
 *
 * Reserved words (plan R6): the lexer types a word by its spelling alone
 * (W_IF for "if", ...); this grammar decides by position whether such a word
 * is reserved. A keyword type is a keyword only where a rule asks for it (a
 * command position, or the closer of a compound command after a separator);
 * everywhere a word is an argument, `word` takes every word type.
 */

parser grammar BashSharpParser;

options {
    tokenVocab = BashSharpLexer;
    superClass = GoParserBase;
}

import GoParser;

@parser::structmembers {
    bashsharp bool
}

// ---- entry ----

// A script, and the body of a command substitution (README P11).
script
    : linebreak list_body? EOF
    ;

// ---- lists ----

// Right-recursive, so that whether another command follows a separator is
// an LL(1) decision on that command's first token.
list_body
    : and_or (sep_term list_body?)?
    | fence linebreak list_body?
    ;

sep_term
    : (SH_SEMI | SH_AMP) linebreak
    | SH_NEWLINE+
    | REGION_END linebreak
    ;

linebreak
    : SH_NEWLINE*
    ;

// A possibly empty statement list: the body of a Go block or case arm.
compound_list
    : linebreak list_body?
    ;

// A shell compound-command body before a reserved closer (then, fi, do,
// done, esac, "}"): non-empty, and ending where a reserved word may follow
// (R6): after a separator, or right after a compound command ends.
term_list
    : linebreak list_sep
    ;

list_sep
    : and_or sep_term list_sep?
    | fence linebreak list_sep?
    | and_or_compound
    ;

// An and-or list whose last command is a compound command.
and_or_compound
    : (pipeline (AND_IF | OR_IF) linebreak)* W_TIME? W_BANG* (pipe_command (SH_PIPE | PIPE_AMP) linebreak)* compound_command redirect*
    ;

and_or
    : bashsharp_item
    | go_statement
    | pipeline ((AND_IF | OR_IF) linebreak pipeline)*
    ;

// `time` and `!` may stand alone (they then time or negate nothing).
pipeline
    : W_TIME W_BANG* (pipe_command ((SH_PIPE | PIPE_AMP) linebreak pipe_command)*)?
    | W_BANG+ (pipe_command ((SH_PIPE | PIPE_AMP) linebreak pipe_command)*)?
    | pipe_command ((SH_PIPE | PIPE_AMP) linebreak pipe_command)*
    ;

// A pipeline component; Bash also admits `!` after a pipe.
pipe_command
    : W_BANG* command
    ;

command
    : compound_command redirect*
    | function_definition
    | coproc
    | simple_command
    ;

// `coproc [NAME] command`: a name is taken only before a compound command.
coproc
    : W_COPROC (for_name compound_command redirect* | compound_command redirect* | simple_command)
    ;

// ---- simple commands ----

simple_command
    : cmd_prefix (cmd_word cmd_suffix? | agentic_command)?
    | cmd_word cmd_suffix?
    | agentic_command
    ;

cmd_prefix
    : (ASSIGNMENT_WORD | ARRAY_ASSIGN | redirect)+
    ;

cmd_suffix
    : (word | redirect | ARRAY_ASSIGN)+
    ;

// A command name: any word that is not reserved at a command position.
cmd_word
    : WORD
    | AT_WORD
    | W_IN
    ;

// Any other `agentic` word stays an ordinary command (the delta's
// agentic_command), but a standalone "{" after it commits to agentic_block.
agentic_command
    : W_AGENTIC ((word_not_lbrace | redirect | ARRAY_ASSIGN) cmd_suffix?)?
    ;

word_not_lbrace
    : WORD
    | AT_WORD
    | ASSIGNMENT_WORD
    | W_AGENTIC
    | W_IF | W_THEN | W_ELIF | W_ELSE | W_FI | W_DO | W_DONE | W_CASE | W_ESAC | W_WHILE | W_UNTIL | W_FOR
    | W_SELECT | W_IN | W_FUNCTION | W_TIME | W_COPROC | W_BANG | W_RBRACE | W_DLBRACK | W_DRBRACK
    ;

// A word in an argument position: every word type.
word
    : WORD
    | AT_WORD
    | ASSIGNMENT_WORD
    | W_AGENTIC
    | reserved_word
    ;

reserved_word
    : W_IF | W_THEN | W_ELIF | W_ELSE | W_FI | W_DO | W_DONE | W_CASE | W_ESAC | W_WHILE | W_UNTIL | W_FOR
    | W_SELECT | W_IN | W_FUNCTION | W_TIME | W_COPROC | W_BANG | W_LBRACE | W_RBRACE | W_DLBRACK | W_DRBRACK
    ;

redirect
    : IO_NUMBER? (SH_LESS | SH_GREAT | DGREAT | LESSAND | GREATAND | LESSGREAT | CLOBBER | AMP_GREAT | AMP_DGREAT | TLESS) word
    | IO_NUMBER? (DLESS | DLESSDASH) word
    ;

// ---- compound commands ----

compound_command
    : block
    | subshell
    | for_clause
    | case_clause
    | if_clause
    | while_clause
    | until_clause
    | select_clause
    | ARITH_CMD
    | cond_command
    ;

// A brace group, and a Go statement block (BLOCK_OPEN, README P4) whose body
// is shell text: the delta's `block`. Inside a Go block a standalone "}" is
// RBRACE and closes the innermost brace without a separator (README P5).
block
    : BLOCK_OPEN compound_list RBRACE
    | W_LBRACE term_list W_RBRACE
    | W_LBRACE linebreak list_body RBRACE
    ;

subshell
    : SH_LPAREN linebreak list_body SH_RPAREN
    ;

if_clause
    : W_IF term_list W_THEN term_list (W_ELIF term_list W_THEN term_list)* (W_ELSE term_list)? W_FI
    ;

while_clause
    : W_WHILE term_list do_group
    ;

until_clause
    : W_UNTIL term_list do_group
    ;

do_group
    : W_DO term_list W_DONE
    ;

// Bash also admits a brace group as the body of for and select.
for_clause
    : W_FOR for_name linebreak (W_IN word* sequential_sep)? linebreak loop_body
    | W_FOR for_name sequential_sep linebreak loop_body
    | W_FOR ARITH_CMD sequential_sep? linebreak loop_body
    ;

select_clause
    : W_SELECT for_name linebreak (W_IN word* sequential_sep)? linebreak loop_body
    | W_SELECT for_name sequential_sep linebreak loop_body
    ;

loop_body
    : do_group
    | block
    ;

for_name
    : WORD
    | AT_WORD
    | ASSIGNMENT_WORD
    | W_AGENTIC
    ;

sequential_sep
    : SH_SEMI linebreak
    | SH_NEWLINE+
    ;

// Every case item but the last ends with ;; ;& or ;;&.
case_clause
    : W_CASE word linebreak W_IN linebreak case_item* case_item_last? W_ESAC
    ;

case_item
    : SH_LPAREN? pattern (SH_PIPE pattern)* SH_RPAREN compound_list (DSEMI | SEMI_AMP | DSEMI_AMP) linebreak
    ;

case_item_last
    : SH_LPAREN? pattern (SH_PIPE pattern)* SH_RPAREN linebreak list_sep?
    ;

// A pattern is any word except `esac`, which closes the case.
pattern
    : WORD
    | AT_WORD
    | ASSIGNMENT_WORD
    | W_AGENTIC
    | W_IF | W_THEN | W_ELIF | W_ELSE | W_FI | W_DO | W_DONE | W_CASE | W_WHILE | W_UNTIL | W_FOR
    | W_SELECT | W_IN | W_FUNCTION | W_TIME | W_COPROC | W_BANG | W_LBRACE | W_RBRACE | W_DLBRACK | W_DRBRACK
    ;

cond_command
    : W_DLBRACK linebreak cond_or linebreak W_DRBRACK
    ;

cond_or
    : cond_and (OR_IF linebreak cond_and)*
    ;

cond_and
    : cond_not (AND_IF linebreak cond_not)*
    ;

cond_not
    : W_BANG linebreak cond_not
    | cond_primary
    ;

// A unary operator needs its operand: [[ -z ]] is an error.
cond_primary
    : SH_LPAREN linebreak cond_or linebreak SH_RPAREN
    | {p.condUnaryHere()}? WORD cond_word
    | cond_word (SH_LESS | SH_GREAT | {p.condBinaryHere()}? (WORD | ASSIGNMENT_WORD)) cond_word
    | {!p.condUnaryWord()}? cond_word
    ;

// A [[ ]] operand: any word but the closing "]]".
cond_word
    : WORD
    | AT_WORD
    | ASSIGNMENT_WORD
    | W_AGENTIC
    | W_IF | W_THEN | W_ELIF | W_ELSE | W_FI | W_DO | W_DONE | W_CASE | W_ESAC | W_WHILE | W_UNTIL | W_FOR
    | W_SELECT | W_IN | W_FUNCTION | W_TIME | W_COPROC | W_LBRACE | W_RBRACE | W_DLBRACK
    ;

// ---- functions ----

function_definition
    : at_function
    | shell_function
    ;

// The delta's shell_function: `function NAME [()]` or `NAME ()`, then a
// compound command (Bash admits any compound body, not only a block).
shell_function
    : W_FUNCTION fname (SH_LPAREN SH_RPAREN)? linebreak compound_command redirect*
    | fname SH_LPAREN SH_RPAREN linebreak compound_command redirect*
    ;

fname
    : WORD
    | W_AGENTIC
    | W_IN
    ;

// `@name()` and a compound body: Bash's own function named @name (delta
// at_function and indented_at_function). The lexer types such a name AT_WORD
// with Bash# on; with Bash# off it is a WORD and a shell_function.
at_function
    : AT_WORD SH_LPAREN SH_RPAREN linebreak compound_command redirect*
    ;

// ---- the Bash# delta ----

// A fence is a list item of its own: its closing line ends it (list_body).
bashsharp_item
    : embed_line
    | decorated_decl
    | agentic_decl
    ;

// A source fence (README P7): the header line through the first exact closer.
fence
    : FENCE_OPEN FENCE_CLOSE
    ;

// `embed FORMAT "./path" [as ALIAS] [!RUNNER]` (README P12).
embed_line
    : {p.embedLineHere()}? WORD WORD WORD word*
    ;

decorated_decl
    : decorator_line+ function_decl
    ;

// A decorator commits at a column-one `@name(` and ends its line, or ends at
// ";" (README P8): the decorator region's REGION_END.
decorator_line
    : decorator REGION_END linebreak
    ;

decorator
    : decorator_head arguments
    ;

decorator_head
    : AT decorator_name (DOT decorator_name)*
    ;

decorator_name
    : IDENTIFIER
    | BREAK | CASE | CHAN | CONST | CONTINUE | DEFAULT | DEFER | ELSE | FALLTHROUGH | FOR | FUNC | GO | GOTO | IF
    | IMPORT | INTERFACE | MAP | PACKAGE | RANGE | RETURN | SELECT | STRUCT | SWITCH | TYPE | VAR
    ;

function_decl
    : typed_func
    | typed_method
    | agentic_shell_func
    | agentic_typed_func
    | agentic_typed_method
    | shell_function
    ;

agentic_decl
    : agentic_block
    | agentic_shell_func
    | agentic_typed_func
    | agentic_typed_method
    | agentic_command
    ;

agentic_block
    : W_AGENTIC block
    ;

agentic_shell_func
    : W_AGENTIC W_FUNCTION fname SH_LPAREN SH_RPAREN linebreak block
    ;

agentic_typed_func
    : W_AGENTIC typed_func
    ;

agentic_typed_method
    : W_AGENTIC typed_method
    ;

// Typed functions and methods: a Go signature (with Bash# default
// parameters) and a block whose body is shell text.
typed_func
    : FUNC IDENTIFIER typeParameters? signature block
    ;

typed_method
    : FUNC receiver IDENTIFIER signature block
    ;

// ---- Go regions (README P5): the statements a start site opens ----

go_statement
    : typed_func
    | typed_method
    | declaration
    | importDecl
    | deferStmt
    | goStmt
    | returnStmt
    | ifStmt
    | forStmt
    | switchStmt
    | selectStmt
    | simpleStmt
    ;

// ---- Bash# extensions of imported Go productions ----

// Default parameter values: `func f(n int = 3)`.
parameterDecl
    : identifierList? ELLIPSIS? type_ (ASSIGN expression)?
    ;

// Keyword arguments: `f(name: value)`, after any positional arguments.
arguments
    : L_PAREN (argument_list ELLIPSIS? COMMA?)? R_PAREN
    ;

argument_list
    : (type_ (COMMA expressionList)? | expressionList) (COMMA keyword_argument)*
    | keyword_argument (COMMA keyword_argument)*
    ;

keyword_argument
    : IDENTIFIER COLON expression
    ;

// An import path is an interpreted string: backquotes at a mixed start site
// are shell command substitution (S6 M16).
importPath
    : INTERPRETED_STRING_LIT
    ;

// A shell expansion is a Go operand (typed operands, README P10).
operand
    : literal
    | operandName typeArgs?
    | L_PAREN expression R_PAREN
    | SHELL_OPERAND
    ;

// Enums: `type Color enum { Red; Green }`.
typeDef
    : IDENTIFIER typeParameters? (enum_type | type_)
    ;

enum_type
    : {p.identIs("enum")}? IDENTIFIER L_CURLY (IDENTIFIER eos)* R_CURLY
    ;

// Switch and select bodies hold case clauses whose statements are shell text.
exprSwitchStmt
    : SWITCH (expression? | simpleStmt? eos expression?) BLOCK_OPEN linebreak exprCaseClause* RBRACE
    ;

exprCaseClause
    : exprSwitchCase COLON compound_list
    ;

typeSwitchStmt
    : SWITCH (typeSwitchGuard | eos typeSwitchGuard | simpleStmt eos typeSwitchGuard) BLOCK_OPEN linebreak typeCaseClause* RBRACE
    ;

typeCaseClause
    : typeSwitchCase COLON compound_list
    ;

selectStmt
    : SELECT BLOCK_OPEN linebreak commClause* RBRACE
    ;

commClause
    : commCase COLON compound_list
    ;
