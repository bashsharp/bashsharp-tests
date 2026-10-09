#!/usr/bin/env ruby
# Exercise case discovery without provisioning or running foreign toolchains.
runner = File.expand_path('../../tools/lowering/differential.rb', __dir__)
source = File.read(runner).split(/^options = /, 2).first
eval(source, TOPLEVEL_BINDING, runner)
rows = cases
abort "missing current null-safety regression" unless rows.any? { |row| row[:family] == 'null-safety' && row[:id] == 'reassign-after-narrow' && row[:expectation] == 'reject' }
abort "wrong ledger size" unless rows.length == 34
abort "wrong runtime denominator" unless rows.count { |row| row[:expectation] == 'run' } == 18
abort "wrong rejection denominator" unless rows.count { |row| row[:expectation] == 'reject' } == 16
puts 'current lowering ledger PASS: 34 cases, 18 runtime, 16 rejection'
