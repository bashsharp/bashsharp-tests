#!/usr/bin/env ruby
# Grade the interpreted fixture against its existing lowering manifest, including
# exact nonzero statuses and diagnostic bytes. A rejection alone is not a pass.
require 'json'
require 'open3'

module InterpretedFixtureContract
  ROOT = File.expand_path('../..', __dir__)
  HEADER = "id\tcategory\tfixture\texpected_status\tstdout\tstderr\tpublic_test_ref"
  INVENTORIES = {
    'go-profile-cases.tsv' => 'go-profile',
    'profile-additional.tsv' => 'profile-additional'
  }.freeze

  def self.contract(file)
    wanted = File.realpath(file)
    matches = []
    INVENTORIES.each do |manifest, directory|
      root = File.join(ROOT, 'tests/lowering', directory)
      rows = File.readlines(File.join(ROOT, 'docs/lowering', manifest), chomp: true)
                 .reject { |line| line.empty? || line.start_with?('#') }
      raise "invalid manifest header: #{manifest}" unless rows.shift == HEADER
      rows.each do |line|
        fields = line.split("\t", -1)
        raise "invalid manifest row: #{manifest}" unless fields.length == 7
        id, _category, fixture, status, stdout, stderr, reference = fields
        raise "invalid fixture path: #{fixture}" if fixture.start_with?('/') || fixture.split('/').include?('..')
        next unless File.realpath(File.join(root, fixture)) == wanted
        code = Integer(status, 10)
        streams = [JSON.parse(stdout), JSON.parse(stderr)]
        raise "invalid observation contract: #{id}" unless code.between?(0, 255) && streams.all? { |s| s.is_a?(String) } && !reference.empty?
        matches << { root: root, fixture: fixture, status: code, stdout: streams[0].b, stderr: streams[1].b }
      end
    end
    raise "expected exactly one fixture contract, found #{matches.length}: #{file}" unless matches.length == 1
    matches.first
  end

  def self.check(binary, file)
    expected = contract(file)
    binary = File.expand_path(binary) if binary.include?(File::SEPARATOR)
    # The manifests name diagnostics relative to this fixture root. Execute
    # there, as the existing differential runner does; do not erase locations.
    stdout, stderr, status = Open3.capture3(binary, '--bashpp', expected[:fixture], chdir: expected[:root])
    actual = { status: status.exitstatus, stdout: stdout.b, stderr: stderr.b }
    failures = actual.keys.select { |key| actual[key] != expected[key] }
    failures.each { |key| warn "#{file}: #{key}: expected #{expected[key].inspect}, got #{actual[key].inspect}" }
    failures.empty?
  end
end

if $PROGRAM_NAME == __FILE__
  abort 'usage: check-interpreted.rb BASHY FIXTURE' unless ARGV.length == 2
  begin
    exit(InterpretedFixtureContract.check(*ARGV) ? 0 : 1)
  rescue StandardError => error
    warn "fixture contract: #{error.message}"
    exit 1
  end
end
