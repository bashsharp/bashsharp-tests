#!/usr/bin/env ruby
# Sprint 197 decorator acceptance battery (story bb5ee9f84a12), following the
# agentic runner's bounded-product-check shape; no provider, compiler or
# foreign suite. tests/manifest.tsv is the status of record: planned cases
# are executed and reported PLANNED — never PASS, never fatal — so this
# runner cannot fake product evidence before the S197.2 engine slice lands.
# Dialect-off parity with independent GNU Bash 5.3 is fatal on every run.
require 'open3'
require 'timeout'

require_supported = false
ARGV.each do |arg|
  if arg == '--require-supported'
    require_supported = true
  else
    abort "decorators: unknown option #{arg}"
  end
end

root = File.expand_path('../..', __dir__)
fixtures = File.join(root, 'tests/decorators')
binary = File.expand_path(ENV.fetch('BASHY_BIN', File.join(root, '../bashy/bin/bash')))
oracle_candidates = ENV.key?('BASH53') ? [ENV.fetch('BASH53')] : ENV.fetch('PATH', '').split(File::PATH_SEPARATOR).map { |dir| File.join(dir, 'bash') }
oracle = oracle_candidates.map { |path| File.expand_path(path) }.uniq.find do |path|
  File.executable?(path) && Open3.capture2e(path, '--version').first.include?('version 5.3')
end
abort 'decorators: GNU Bash 5.3 not found on PATH; set BASH53=/path/to/bash' unless oracle
[binary, oracle].each { |path| abort "decorators: missing executable #{path}" unless File.executable?(path) }

rows = File.readlines(File.join(fixtures, 'cases.tsv'), chomp: true).reject { |line| line.empty? || line.start_with?('#') }.map { |line| line.split("\t", -1) }
abort 'decorators: expected 16 unique cases' unless rows.size == 16 && rows.map(&:first).uniq.size == 16
abort 'decorators: malformed case ledger' unless rows.all? { |row| row.size == 5 }

# S197.1 layout fixtures are graded by the harness's generic loop, not here.
layout = %w[call-forms.bpp compound-bodies.bpp declaration-layout.bpp]
actual = Dir.glob(File.join(fixtures, '*.bpp')).map { |path| File.basename(path) }.sort
abort 'decorators: unlisted or missing fixture' unless actual == (rows.map { |row| row[1] } + layout).sort

# Declared status comes from the manifest, never from the outcome.
manifest = {}
File.readlines(File.join(root, 'tests/manifest.tsv'), chomp: true).each do |line|
  next if line.empty? || line.start_with?('#')
  cols = line.split("\t", -1)
  manifest[cols[0]] = cols[1] if cols.size >= 2
end

if require_supported
  planned_entries = manifest.select { |k, v| k.start_with?('tests/decorators/') && v == 'planned' }
  abort "decorators: --require-supported is on but manifest has planned entries" unless planned_entries.empty?
end

STDERR_CLASSES = {
  'empty'    => nil,
  'undef'    => /EDECO-UNDEF/,
  'self'     => /EDECO-SELF/,
  'cycle'    => /EDECO-CYCLE/,
  'sig'      => /EDECO-SIG/,
  'reserved' => /EDECO-RESERVED/,
}.freeze
abort 'decorators: unknown stderr class in ledger' unless rows.all? { |row| STDERR_CLASSES.key?(row[4]) }

executions = 0
run = lambda do |program, flags, entry, source, parse_only = false|
  env = {'BASHY_HINTS' => 'off', 'BASHY_BASHPP' => nil, 'BASHY_AGENTIC' => nil, 'BASHY_ADVICE' => nil, 'BASH_ENV' => nil, 'ENV' => nil}
  command = [program, *flags, *(parse_only ? ['-n'] : [])]
  text = File.binread(File.join(fixtures, source))
  input = ''
  case entry
  when 'file' then command += [source]
  when 'stdin' then command += ['-s', '--']; input = text
  when '-c' then command += ['-c', text, 'decorator-fixture']
  end
  executions += 1
  # Bound each real process; kill the process group on timeout, including tools.
  Open3.popen3(env, *command, chdir: fixtures, pgroup: true) do |stdin, stdout, stderr, waiter|
    writer = Thread.new { begin stdin.write(input); rescue Errno::EPIPE; ensure stdin.close; end }
    out = Thread.new { stdout.read }
    err = Thread.new { stderr.read }
    begin
      Timeout.timeout(15) do
        status = waiter.value
        writer.join
        [out.value, err.value, status.exitstatus]
      end
    rescue Timeout::Error
      Process.kill('KILL', -waiter.pid) rescue Errno::ESRCH
      abort "decorators: timeout #{source}/#{entry}/#{flags.join(' ')}"
    end
  end
end

failures = []
planned_report = []
ratchet_candidates = []
supported_pass = 0
rows.each do |id, source, expected_status, expected_stdout, expected_stderr|
  status_declared = manifest["tests/decorators/#{source}"] || 'supported'
  want_out = expected_stdout == 'empty' ? '' : File.binread(File.join(fixtures, expected_stdout))
  product_mismatches = []
  %w[file stdin -c].each do |entry|
    label = "#{id}/#{entry}"

    out, err, status = run.call(binary, ['--bashpp'], entry, source)
    status_ok = expected_status == 'nonzero' ? status && status != 0 : status == Integer(expected_status)
    pattern = STDERR_CLASSES.fetch(expected_stderr)
    stderr_ok = pattern.nil? ? err.empty? : err.match?(pattern)
    unless status_ok && out == want_out && stderr_ok
      product_mismatches << "#{label}: exit=#{status.inspect}, stdout=#{out.inspect}, stderr=#{err.inspect}"
    end

    # Classic/POSIX verdicts come from the independent GNU oracle and are
    # fatal regardless of manifest status: decorator grammar must stay
    # invisible with the dialect off, engine slice or no engine slice. Only
    # the never-claimed compatibility case executes in these modes; feature
    # bodies must never become accidental ordinary commands during this check.
    [[['--no-bashpp'], []], [['--posix', '--no-bashpp'], ['--posix']], [['--posix', '--bashpp'], []]].each do |flags, oracle_flags|
      parse_only = id != 'near-miss-shell'
      got = run.call(binary, flags, entry, source, parse_only)
      ref = run.call(oracle, oracle_flags, entry, source, parse_only)
      equal = parse_only ? (got[2] == 0) == (ref[2] == 0) : got == ref
      failures << "#{label}/#{flags.join(' ')}: product=#{got.inspect}, GNU=#{ref.inspect}" unless equal
    end
  end

  if product_mismatches.empty?
    if status_declared == 'planned'
      ratchet_candidates << id
      puts "decorators: #{id} — planned case now PASSES (ratchet candidate; manifest flip is the manager's move)"
    else
      supported_pass += 1
      puts "decorators: checked #{id}"
    end
  elsif status_declared == 'planned'
    planned_report << id
    puts "decorators: #{id} — PLANNED (engine slice not built; #{product_mismatches.size}/3 product entries failing)"
  else
    failures.concat(product_mismatches)
    puts "decorators: #{id} — FAILED (supported)"
  end
end

abort failures.join("\n") unless failures.empty?
summary = "decorators: dialect matrix PASS against GNU 5.3; #{supported_pass} supported case(s) passed; #{executions} product/oracle executions"
summary += "; #{planned_report.size} PLANNED (no decorator product evidence claimed): #{planned_report.join(', ')}" unless planned_report.empty?
summary += "; ratchet candidates: #{ratchet_candidates.join(', ')}" unless ratchet_candidates.empty?
puts summary
