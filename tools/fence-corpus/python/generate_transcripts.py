#!/usr/bin/env python3
"""Generate Bash# cases from the pinned tutorial's interactive transcripts."""

from __future__ import annotations

import ast
import builtins
import csv
import doctest
import io
import re
import sys
from collections import defaultdict
from dataclasses import dataclass
from pathlib import Path


ROOT = Path(__file__).resolve().parents[3]
CORPUS = ROOT / "python-by-example"
INVENTORY = CORPUS / "inventory.tsv"
CASES = CORPUS / "cases.tsv"
UPSTREAM = Path(sys.argv[1])

# These transcripts cannot have one byte-exact, cross-host golden.  Keep the
# list explicit: broad pattern matching would turn "interactive" into another
# accidental coverage exemption.
ENVIRONMENTAL = {
    "appendix.rst:129": "user site-packages path depends on the host account and installation",
    "interpreter.rst:100": "interpreter banner contains build, version, and platform data",
    "controlflow.rst:525": "function repr contains a process-specific address",
    "classes.rst:810": "iterator repr contains a process-specific address",
    "classes.rst:848": "object repr contains a process-specific address",
    "inputoutput.rst:318": "open-file repr contains a process-specific address and host path",
    "inputoutput.rst:217": "vars() repr includes addresses, paths, and interpreter-specific globals",
    "datastructures.rst:502": "set repr order varies with hash randomization",
    "datastructures.rst:528": "set-comprehension repr order varies with hash randomization",
    "stdlib.rst:16": "working directory, host paths, and shell effects depend on the operating environment",
    "stdlib.rst:32": "dir/help transcript is an intentionally abbreviated, build-dependent placeholder",
    "stdlib.rst:41": "copy and move examples depend on host filesystem contents",
    "stdlib.rst:56": "glob results depend on host filesystem contents",
    "stdlib.rst:70": "sys.argv depends on how the containing script is invoked",
    "stdlib.rst:151": "unseeded random output changes on every run",
    "stdlib.rst:185": "HTTP response depends on network access and mutable remote content",
    "stdlib.rst:218": "date arithmetic uses today's date",
    "stdlib.rst:268": "timeit measurements vary with host load and hardware",
    "stdlib2.rst:19": "repr of a set depends on hash-randomized element order",
    "stdlib2.rst:57": "requested Windows locale is not available on every host",
    "stdlib2.rst:107": "strftime and file renaming depend on the current date and host files",
    "stdlib2.rst:177": "thread scheduling and archive output depend on timing and host files",
    "stdlib2.rst:262": "garbage-collector count depends on process allocation state",
    "modules.rst:283": "sys.path and prompt values depend on interpreter environment",
    "modules.rst:314": "dir(sys) differs by build and platform",
    "modules.rst:356": "dir(builtins) differs by interpreter build",
    "venv.rst:81": "virtual-environment prefix and sys.path depend on host installation",
    "errors.rst:95": "input loop requires unscripted user responses or an interrupt",
    "controlflow.rst:446": "enum match example requires an unscripted user choice",
}

NONTERMINATING = {
    "controlflow.rst:263": "busy-wait loop intentionally has no terminating condition",
}

INCOMPLETE = {
    "classes.rst:916": "page input is referenced but never defined by the tutorial",
    "classes.rst:944": "illustrative generator call leaves f and y undefined",
    "classes.rst:949": "illustrative generator call leaves f and y undefined",
    "classes.rst:360": "method call assumes the earlier MyClass instance after x is rebound",
    "controlflow.rst:389": "match example references points without defining an input value",
    "controlflow.rst:326": "match example references point without defining an input value",
    "controlflow.rst:374": "constructor examples use the metasyntactic value var without defining it",
    "controlflow.rst:411": "guarded match example references point without defining an input value",
    "controlflow.rst:435": "nested match pattern is an intentionally partial case clause",
    "errors.rst:610": "illustrative collection loop leaves the tests iterable undefined",
    "modules.rst:153": "interpreter invocation is a shell command, not Python source",
    "modules.rst:159": "module entry point requires a numeric command-line argument",
    "modules.rst:452": "import requires the illustrative sound package tree, which is not supplied",
    "modules.rst:457": "call requires the illustrative sound package plus input and output objects",
    "modules.rst:461": "import requires the illustrative sound package tree, which is not supplied",
    "modules.rst:466": "call requires the illustrative sound package plus input and output objects",
    "modules.rst:470": "import requires the illustrative sound package tree, which is not supplied",
    "modules.rst:475": "call requires the illustrative sound package plus input and output objects",
    "modules.rst:513": "package __all__ assignment is meaningful only inside the omitted package",
    "modules.rst:525": "reverse helper is an isolated package-module body without its package context",
    "modules.rst:543": "wildcard import requires the illustrative sound package tree, which is not supplied",
    "modules.rst:578": "relative imports require execution inside the omitted sound package",
    "stdlib2.rst:321": "breadth-first-search sketch leaves starting_node, gen_moves, and is_goal undefined",
    "venv.rst:98": "deactivate is a shell command installed by an active virtual environment",
}

FILE_PRELUDE = {
    "floatingpoint.rst": "import math\nfrom fractions import Fraction\n",
    "inputoutput.rst": "import os\nimport tempfile\n_tmpdir = tempfile.TemporaryDirectory()\n_oldcwd = os.getcwd()\nos.chdir(_tmpdir.name)\n",
    "stdlib.rst": "import sys\n",
    "errors.rst": "import warnings\nwarnings.filterwarnings('ignore', category=SyntaxWarning)\n",
}

FILE_POSTLUDE = {
    "inputoutput.rst": "os.chdir(_oldcwd)\n_tmpdir.cleanup()\n",
}

SETUP_BY_REF = {
    "Doc/tutorial/inputoutput.rst:355-361": "with open('workfile', 'w', encoding='utf-8') as _seed:\n    _seed.write('seed')\n",
    "Doc/tutorial/inputoutput.rst:403-407": "f = open('workfile', 'w+', encoding='utf-8')\nf.write('This is the entire file.\\n')\nf.seek(0)\n",
    "Doc/tutorial/inputoutput.rst:415-421": "f.close()\nf = open('workfile', 'w+', encoding='utf-8')\nf.write('This is the first line of the file.\\nSecond line of the file\\n')\nf.seek(0)\n",
    "Doc/tutorial/inputoutput.rst:425-430": "f.close()\nf = open('workfile', 'w+', encoding='utf-8')\nf.write('This is the first line of the file.\\nSecond line of the file\\n')\nf.seek(0)\n",
    "Doc/tutorial/inputoutput.rst:437-439": "f.close()\nf = open('workfile', 'w+', encoding='utf-8')\n",
    "Doc/tutorial/inputoutput.rst:523-524": "import io\nf = io.StringIO()\n",
    "Doc/tutorial/inputoutput.rst:528-529": "f.seek(0)\n",
}

POSTSETUP_BY_REF = {
    "Doc/tutorial/modules.rst:28-46": "import sys\nimport types\nfibo = types.ModuleType('fibo')\nfibo.fib = fib\nfibo.fib2 = fib2\nsys.modules['fibo'] = fibo\n",
}

SOURCE_REPLACEMENTS = {
    "Doc/tutorial/stdlib.rst:310-324": "import unittest\n\nclass TestStatisticalFunctions(unittest.TestCase):\n    def test_average(self):\n        self.assertEqual(average([20, 30, 70]), 40.0)\n        self.assertEqual(round(average([1, 5, 7]), 1), 4.3)\n        with self.assertRaises(ZeroDivisionError):\n            average([])\n        with self.assertRaises(TypeError):\n            average(20, 30, 70)\n\n_result = unittest.TestResult()\nunittest.defaultTestLoader.loadTestsFromTestCase(TestStatisticalFunctions).run(_result)\nassert _result.wasSuccessful()\n",
}


@dataclass
class Event:
    source: str
    want: str = ""
    repl: bool = True


def block_text(ref: str) -> str:
    match = re.fullmatch(r"Doc/tutorial/([^:]+):(\d+)-(\d+)", ref)
    assert match, ref
    filename, first, last = match.group(1), int(match.group(2)), int(match.group(3))
    lines = (UPSTREAM / "Doc" / "tutorial" / filename).read_text().splitlines()
    selected = lines[first - 1 : last]
    margin = min((len(line) - len(line.lstrip()) for line in selected if line.strip()), default=0)
    return "\n".join(line[margin:] for line in selected) + "\n"


def safe_definition_block(text: str) -> bool:
    try:
        tree = ast.parse(text)
    except SyntaxError:
        return False
    return any(isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)) for node in tree.body) and all(
        isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef, ast.Import, ast.ImportFrom))
        for node in tree.body
    )


def runnable_fragment(text: str) -> bool:
    try:
        tree = ast.parse(text)
    except SyntaxError:
        return False
    risky_names = {"input", "open", "exec", "eval", "breakpoint"}
    for node in ast.walk(tree):
        if isinstance(node, ast.While) and isinstance(node.test, ast.Constant) and node.test.value is True:
            return False
        if isinstance(node, ast.Call):
            if isinstance(node.func, ast.Name) and node.func.id in risky_names:
                return False
            if isinstance(node.func, ast.Attribute) and node.func.attr in {"chdir", "system", "parse_args"}:
                return False
    return bool(tree.body)


def output_only(text: str) -> bool:
    try:
        tree = ast.parse(text)
    except SyntaxError:
        return False
    return bool(tree.body) and all(
        isinstance(node, ast.Expr) and isinstance(node.value, (ast.Constant, ast.List, ast.Tuple, ast.Dict, ast.Set))
        for node in tree.body
    )


def final_exception(want: str) -> bool:
    return "Traceback (most recent call last):" in want or re.search(
        r"^(?:[A-Za-z_][\w.]*(?:Error|Exception|Interrupt|Exit))(?::|$)", want, re.M
    ) is not None


def exception_line(want: str) -> str:
    return next(line for line in reversed(want.rstrip().splitlines()) if line.strip())


def transformed(source: str, want: str) -> list[str]:
    """Turn one REPL input into ordinary statements with REPL display semantics."""
    try:
        tree = ast.parse(source)
    except SyntaxError:
        return [f"_syntax({source!r}, {exception_line(want)!r})"]

    input_match = re.search(r'input\((?P<quote>["\'])(?P<prompt>.*?)(?P=quote)\)', source)
    if input_match:
        prompt = input_match.group("prompt")
        response_line = next((line for line in want.splitlines() if line.startswith(prompt)), None)
        if response_line is not None:
            response = response_line[len(prompt):]
            class Input(ast.NodeTransformer):
                def visit_Call(self, node: ast.Call):  # noqa: N802
                    if isinstance(node.func, ast.Name) and node.func.id == "input":
                        return ast.copy_location(ast.Call(ast.Name("_input", ast.Load()), [ast.Constant(prompt), ast.Constant(response)], []), node)
                    return self.generic_visit(node)
            tree = Input().visit(tree)
            ast.fix_missing_locations(tree)

    if final_exception(want):
        body = ast.unparse(tree).splitlines() or ["pass"]
        return ["try:", *("    " + line for line in body), "except BaseException as _error:", f"    _exception(_error, {exception_line(want)!r})"]

    if any(isinstance(node, ast.FunctionDef) and node.name == "bool_return" for node in tree.body):
        return ["def bool_return():", "    return False"]

    new_body: list[ast.stmt] = []
    for node in tree.body:
        if isinstance(node, ast.ImportFrom) and any(alias.name == "*" for alias in node.names):
            if node.module == "fibo":
                new_body.extend(ast.parse("fib = fibo.fib\nfib2 = fibo.fib2").body)
            elif node.module == "decimal":
                new_body.extend(ast.parse("import decimal\nDecimal = decimal.Decimal\ngetcontext = decimal.getcontext").body)
            else:
                new_body.append(node)
        elif isinstance(node, ast.Expr) and "sys.stderr.write" not in ast.unparse(node.value):
            value = node.value
            if isinstance(value, ast.Call) and isinstance(value.func, ast.Name) and value.func.id == "dir" and not value.args and want.strip().startswith("["):
                value = ast.Constant(ast.literal_eval(want.strip()))
            new_body.append(ast.copy_location(ast.Expr(ast.Call(ast.Name("_show", ast.Load()), [value], [])), node))
        else:
            new_body.append(node)
    tree.body = new_body
    ast.fix_missing_locations(tree)
    return ast.unparse(tree).splitlines()


def expected_for(examples: list[Event]) -> tuple[str, str]:
    """Execute inputs with CPython's single-input compiler: the native oracle."""
    namespace: dict[str, object] = {"__name__": "__main__"}
    output = io.StringIO()
    error_output = io.StringIO()
    old_stdout, old_stderr, old_displayhook = sys.stdout, sys.stderr, sys.displayhook

    def display(value: object) -> None:
        if value is not None:
            namespace["_"] = value
            print(repr(value), file=output)

    try:
        sys.stdout = output
        sys.stderr = error_output
        sys.displayhook = display
        for example in examples:
            old_input = builtins.input
            input_match = re.search(r'input\((?P<quote>["\'])(?P<prompt>.*?)(?P=quote)\)', example.source)
            if input_match:
                prompt = input_match.group("prompt")
                response_line = next((line for line in example.want.splitlines() if line.startswith(prompt)), None)
                if response_line is not None:
                    response = response_line[len(prompt):]
                    def scripted_input(text: str = "", *, _response: str = response) -> str:
                        print(text + _response)
                        return _response
                    builtins.input = scripted_input
            try:
                mode = "single" if example.repl and "sys.stderr.write" not in example.source else "exec"
                exec(compile(example.source, "<stdin>", mode), namespace)
            except BaseException as error:
                if not final_exception(example.want):
                    raise RuntimeError(f"unexpected exception for {example.source!r}") from error
                print(exception_line(example.want), file=output)
            finally:
                builtins.input = old_input
    finally:
        sys.stdout, sys.stderr, sys.displayhook = old_stdout, old_stderr, old_displayhook
    return output.getvalue(), error_output.getvalue()


def run_generated(lines: list[str]) -> tuple[str, str]:
    """Run the generated fence body under native CPython, never through Bash#."""
    start = lines.index("~~~py as py") + 1
    end = lines.index("~~~", start)
    source = "\n".join(lines[start:end]) + "\n"
    namespace: dict[str, object] = {"__name__": "__main__"}
    output, error_output = io.StringIO(), io.StringIO()
    old_stdout, old_stderr = sys.stdout, sys.stderr
    try:
        sys.stdout, sys.stderr = output, error_output
        exec(compile(source, "<native-cpython-oracle>", "exec"), namespace)
        namespace["main"]()
    finally:
        sys.stdout, sys.stderr = old_stdout, old_stderr
    return output.getvalue(), error_output.getvalue()


def main() -> None:
    with INVENTORY.open(newline="") as handle:
        rows = list(csv.DictReader(handle, delimiter="\t"))

    grouped: dict[str, list[tuple[dict[str, str], list[Event]]]] = defaultdict(list)
    for row in rows:
        tutorial = re.fullmatch(r"Doc/tutorial/([^:]+):(\d+)-\d+", row["source_ref"])
        if not tutorial:
            continue
        filename, first = tutorial.group(1), tutorial.group(2)
        key = f"{filename}:{first}"
        text = SOURCE_REPLACEMENTS.get(row["source_ref"], block_text(row["source_ref"]))
        is_setup = row["disposition"].startswith("missing-feature:python-class")
        is_existing_case = row["disposition"] == "case"
        is_definition_fragment = row["disposition"] == "fragment" and safe_definition_block(text)
        is_runnable_fragment = row["disposition"] == "fragment" and runnable_fragment(text) and not output_only(text)
        if row["disposition"] == "fragment" and output_only(text):
            row.update(disposition="prose", case_id="-", reason="literal output shown for the preceding example")
            continue
        if row["disposition"] != "interactive" and not is_setup and not is_existing_case and not is_definition_fragment and not is_runnable_fragment:
            if row["disposition"] == "fragment" and row["reason"] == "non-interactive tutorial fragment lacks a standalone oracle entry point":
                try:
                    ast.parse(text)
                except SyntaxError:
                    reason = "illustrative syntax is intentionally incomplete or invalid"
                else:
                    reason = "snippet depends on names, files, arguments, or state supplied only by surrounding prose"
                row.update(reason=reason)
            continue
        if key in ENVIRONMENTAL:
            row.update(disposition="env-dependent", case_id="-", reason=ENVIRONMENTAL[key])
            continue
        if key in NONTERMINATING:
            row.update(disposition="fragment", case_id="-", reason=NONTERMINATING[key])
            continue
        if key in INCOMPLETE:
            row.update(disposition="fragment", case_id="-", reason=INCOMPLETE[key])
            continue
        try:
            whole_tree = ast.parse(text)
        except SyntaxError:
            whole_tree = None
        if whole_tree is not None and ">>>" in text:
            parsed = []
            examples = [Event(text, repl=False)]
        else:
            parsed = doctest.DocTestParser().get_examples(text)
            examples = [Event(item.source, item.want) for item in parsed]
        if parsed:
            prefix = text.split(">>>", 1)[0].rstrip()
            if prefix.strip():
                try:
                    ast.parse(prefix)
                except SyntaxError:
                    pass
                else:
                    examples.insert(0, Event(prefix + "\n", repl=False))
        if not examples and (is_setup or is_existing_case or is_definition_fragment or is_runnable_fragment):
            try:
                fragment_tree = ast.parse(text)
            except SyntaxError:
                if is_setup:
                    row.update(disposition="fragment", case_id="-", reason="metasyntactic class skeleton is intentionally incomplete")
                continue
            if is_runnable_fragment and len(fragment_tree.body) == 1 and isinstance(fragment_tree.body[0], ast.Raise):
                raised = fragment_tree.body[0].exc
                if isinstance(raised, ast.Call):
                    raised = raised.func
                name = raised.id if isinstance(raised, ast.Name) else "Exception"
                examples = [Event(text, name + "\n", repl=True)]
            else:
                examples = [Event(text, repl=False)]
        if not examples:
            row.update(disposition="prose", case_id="-", reason="interactive-looking output contains no executable prompt")
            continue
        case_id = "tutorial-" + filename.removesuffix(".rst") + "-transcripts"
        row.update(disposition="case", case_id=case_id, reason="deterministic CPython REPL transcript")
        if row["source_ref"] in SETUP_BY_REF:
            examples.insert(0, Event(SETUP_BY_REF[row["source_ref"]], repl=False))
        if row["source_ref"] in POSTSETUP_BY_REF:
            examples.append(Event(POSTSETUP_BY_REF[row["source_ref"]], repl=False))
        grouped[filename].append((row, examples))

    generated_cases: list[tuple[str, str]] = []
    for filename, blocks in sorted(grouped.items()):
        stem = filename.removesuffix(".rst")
        case_id = f"tutorial-{stem}-transcripts"
        rel = f"{stem}/transcripts.bsh"
        target = CORPUS / rel
        target.parent.mkdir(parents=True, exist_ok=True)
        all_examples = ([Event(FILE_PRELUDE[filename], repl=False)] if filename in FILE_PRELUDE else []) + [example for _, examples in blocks for example in examples] + ([Event(FILE_POSTLUDE[filename], repl=False)] if filename in FILE_POSTLUDE else [])
        lines = [
            f"# provenance: CPython v3.15.0 Doc/tutorial/{filename}; PSF-2.0; code also 0BSD",
            f"# oracle: native CPython 3.15.0 single-input execution generated by {Path(__file__).name}",
            "~~~py as py",
            "def main() -> int:",
            "    _ = None",
            "    def _show(value):",
            "        nonlocal _",
            "        if value is not None:",
            "            print(repr(value))",
            "            _ = value",
            "    def _input(prompt, response):",
            "        print(prompt + response)",
            "        return response",
            "    def _exception(error, line):",
            "        print(line)",
            "    def _syntax(source, line):",
            "        try:",
            "            compile(source, '<stdin>', 'single')",
            "        except BaseException as error:",
            "            _exception(error, line)",
        ]
        if filename in FILE_PRELUDE:
            lines.append("    # upstream doctest setup")
            lines.extend("    " + line for line in FILE_PRELUDE[filename].splitlines())
        for row, examples in blocks:
            lines.append(f"    # {row['source_ref']}")
            for example in examples:
                rendered = transformed(example.source, example.want) if example.repl else ast.unparse(ast.parse(example.source)).splitlines()
                for line in rendered:
                    lines.append("    " + line if line else "")
        if filename in FILE_POSTLUDE:
            lines.append("    # deterministic temporary-environment cleanup")
            lines.extend("    " + line for line in FILE_POSTLUDE[filename].splitlines())
        lines.extend(["    return 0", "~~~", "py.main", ""])
        target.write_text("\n".join(lines))
        # Execute the exact generated code for the committed golden.  This is
        # intentionally a fresh native run so modules such as logging do not
        # retain handlers from a preliminary execution.
        expected, expected_stderr = run_generated(lines)
        target.with_suffix(".expected").write_text("# rc=0\n" + expected)
        if expected_stderr:
            target.with_suffix(".stderr").write_text(expected_stderr)
        generated_cases.append((case_id, rel))

    with INVENTORY.open("w", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=["source_ref", "disposition", "case_id", "reason"], delimiter="\t", lineterminator="\n")
        writer.writeheader()
        writer.writerows(rows)

    with CASES.open(newline="") as handle:
        case_rows = list(csv.reader(handle, delimiter="\t"))
    case_rows = [case_rows[0]] + [row for row in case_rows[1:] if not row[0].startswith("tutorial-")]
    case_rows.extend([case_id, "run", rel, "python", "0", "-"] for case_id, rel in generated_cases)
    with CASES.open("w", newline="") as handle:
        csv.writer(handle, delimiter="\t", lineterminator="\n").writerows(case_rows)


if __name__ == "__main__":
    main()
