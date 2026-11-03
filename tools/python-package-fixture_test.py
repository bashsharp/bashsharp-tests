#!/usr/bin/env python3
"""Authored mock checks only; no third-party code, product or Python env runs."""
import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
HELPER = ROOT / 'tools/python-package-fixture.sh'
FIXTURES = ROOT / 'tests/python-packages'
NANO = 'dc54a1a3077cab11d68fac4c5d1cd5c51f5d8c7a'
MINI = '04d809ceab9df28f9adaed044884180159172930'

with tempfile.TemporaryDirectory() as tmp:
    base = Path(tmp).resolve()
    package = base / 'package with spaces'
    package.mkdir()
    bins = base / 'bin'
    bins.mkdir()
    git = bins / 'git'
    git.write_text('#!/bin/sh\n[ "$1" = -C ] && [ "$2" = "$EXPECTED_ROOT" ] && [ "$3" = rev-parse ] && [ "$4" = HEAD ] || exit 90\nprintf "%s\\n" "$MOCK_PIN"\n')
    git.chmod(0o755)
    candidate = bins / 'candidate'
    candidate.write_text('''#!/bin/sh
[ "$PYTHONPATH" = "$EXPECTED_PATH" ] || exit 91
[ "$1" = --bashpp ] || exit 92
[ "$2" = "$EXPECTED_FIXTURE" ] || exit 93
if [ -n "$EXPECTED_PYTHON" ]; then
  [ "$BASHPP_PYTHON" = "$EXPECTED_PYTHON" ] && [ "$PWD" = "$EXPECTED_ROOT" ] || exit 94
fi
case "$2" in
  *nanochat-missing-attribute.bpp) printf '%s\n' 'AttributeError: sprint183_missing_attribute' >&2; exit 1 ;;
esac
printf 'mock fixture executed\n'
''')
    candidate.chmod(0o755)
    env = dict(os.environ, PATH=str(bins) + os.pathsep + os.environ['PATH'],
               BASHY=str(candidate), NANOCHAT_ROOT=str(package), MINISWE_ROOT=str(package),
               EXPECTED_ROOT=str(package), EXPECTED_PATH=str(package),
               EXPECTED_PYTHON='', MOCK_PIN=NANO, PYTHONPATH='ambient-wrong-path')
    def run(name, expected_code, **overrides):
        fixture = FIXTURES / name
        active = dict(env, EXPECTED_FIXTURE=str(fixture), **overrides)
        result = subprocess.run(['bash', str(HELPER), str(fixture), '--bashpp'], env=active,
                                text=True, capture_output=True, timeout=5)
        assert result.returncode == expected_code, (name, result.returncode, result.stderr)
        return result

    assert run('nanochat.bpp', 0).stdout == 'mock fixture executed\n'
    result = run('nanochat-missing-attribute.bpp', 1)
    assert result.stderr == 'AttributeError: sprint183_missing_attribute\n'
    # Preserve the exact errorcheck contract: module lookup failure is not a pass.
    stderr = base / 'stderr'
    for diagnostic, expected in [(result.stderr, 0), ('ModuleNotFoundError: nanochat\n', 1),
                                 ('AttributeError: unrelated\n', 1)]:
        stderr.write_text(diagnostic)
        check = subprocess.run(['bash', '-c', 'source "$1"; check_error_annotations "$2" "$3"',
                                'check', str(ROOT / 'harness/fixture-actions.sh'),
                                str(FIXTURES / 'nanochat-missing-attribute.bpp'), str(stderr)],
                               capture_output=True, timeout=5)
        assert (check.returncode == 0) == (expected == 0)
    assert 'checkout must be' in run('nanochat.bpp', 1, MOCK_PIN='wrong').stderr
    assert 'unavailable' in run('nanochat.bpp', 1, NANOCHAT_ROOT=str(base / 'absent')).stderr
    assert 'unregistered' in run('unregistered.bpp', 1).stderr
    assert 'environment unavailable' in run('mini-swe-agent.bpp', 1, MOCK_PIN=MINI).stderr
    python = package / '.venv/bin/python'
    python.parent.mkdir(parents=True)
    python.write_text('#!/bin/sh\nexit 99\n') # executable prerequisite only, never run
    python.chmod(0o755)
    assert run('mini-swe-agent.bpp', 0, MOCK_PIN=MINI, EXPECTED_PATH=str(package / 'src'),
               EXPECTED_PYTHON=str(python)).stdout == 'mock fixture executed\n'
print('PASS: pinned fixture contexts, paths with spaces, exact diagnostic and prerequisite refusals')
