#!/usr/bin/env ruby
require 'tmpdir'
require 'open3'
require_relative '../../tools/lowering/check-interpreted'

root = InterpretedFixtureContract::ROOT
checker = File.join(root, 'tools/lowering/check-interpreted.rb')
negative = File.join(root, 'tests/lowering/go-profile/const/typed-overflow-neg.bpp')
expected = InterpretedFixtureContract.contract(negative)

Dir.mktmpdir('fixture-contract') do |dir|
  binary = File.join(dir, 'stub')
  mutations = {
    'exact expected rejection' => [2, expected[:stdout], expected[:stderr], true],
    'invalid code accepted' => [0, expected[:stdout], expected[:stderr], false],
    'wrong nonzero status' => [1, expected[:stdout], expected[:stderr], false],
    'missing diagnostic' => [2, '', '', false],
    'wrong source location' => [2, '', expected[:stderr].sub('line 3:', 'line 4:'), false],
    'extra stdout' => [2, 'unexpected', expected[:stderr], false]
  }
  mutations.each do |label, (code, stdout, stderr, pass)|
    File.write(binary, "#!/usr/bin/env ruby\n" +
      "abort 'wrong source context' unless Dir.pwd == #{expected[:root].inspect} && ARGV == ['--bashpp', #{expected[:fixture].inspect}]\n" +
      "STDOUT.write(#{stdout.inspect}); STDERR.write(#{stderr.inspect}); exit #{code}\n")
    File.chmod(0o755, binary)
    _out, err, status = Open3.capture3('ruby', checker, binary, negative)
    abort "FAIL #{label}: #{err}" unless status.success? == pass
  end
  unknown = File.join(dir, 'unknown.bpp')
  File.write(unknown, 'echo unknown')
  _, _, status = Open3.capture3('ruby', checker, binary, unknown)
  abort 'FAIL unregistered fixture accepted' if status.success?

  actions = File.join(root, 'harness/fixture-actions.sh')
  fixture = File.join(root, 'tests/python-packages/nanochat-missing-attribute.bpp')
  out, err, status = Open3.capture3('bash', '-c', 'source "$1"; action_of "$2"', 'actions', actions, fixture)
  abort "FAIL shell errorcheck annotation: #{err}" unless status.success? && out == 'errorcheck'
  error_file = File.join(dir, 'stderr')
  [
    ["AttributeError: module has no attribute sprint183_missing_attribute\n", true],
    ["ModuleNotFoundError: no module nanochat.execution\n", false],
    ["AttributeError: missing unrelated_attribute\n", false]
  ].each do |diagnostic, pass|
    File.write(error_file, diagnostic)
    _, _, status = Open3.capture3('bash', '-c', 'source "$1"; check_error_annotations "$2" "$3"', 'actions', actions, fixture, error_file)
    abort 'FAIL shell negative diagnostic classification' unless status.success? == pass
  end
end
puts 'PASS: exact statuses/streams, source context, missing contracts and negative diagnostic mutations'
